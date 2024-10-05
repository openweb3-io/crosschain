package client

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/pkg/errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/openweb3-io/crosschain/blockchain/solana/builder"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx_input"
	solana_types "github.com/openweb3-io/crosschain/blockchain/solana/types"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
)

type Client struct {
	Asset  types.IAsset
	client *rpc.Client
}

func NewClient(asset types.IAsset) (*Client, error) {
	cfg := asset.GetChain()
	endpoint := cfg.URL
	if endpoint == "" {
		endpoint = rpc.MainNetBeta_RPC
	}
	client := rpc.New(endpoint)
	return &Client{Asset: asset, client: client}, nil
}

func (client *Client) FetchBaseInput(ctx context.Context, fromAddr types.Address) (*tx_input.TxInput, error) {
	txInput := tx_input.NewTxInput()

	// get recent block hash (i.e. nonce)
	recent, err := client.client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("could not get latest blockhash: %v", err)
	}
	if recent == nil || recent.Value == nil {
		return nil, errors.New("error fetching latest blockhash")
	}
	txInput.RecentBlockHash = recent.Value.Blockhash

	return txInput, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (types.TxInput, error) {
	txInput, err := client.FetchBaseInput(ctx, args.From)
	if err != nil {
		return nil, err
	}

	if args.ContractAddress == nil {
		return txInput, nil
	}

	contract := *args.ContractAddress

	mint, err := solana.PublicKeyFromBase58(string(*args.ContractAddress))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid mint address: %s", string(contract))
	}

	mintInfo, err := client.client.GetAccountInfo(ctx, mint)
	if err != nil {
		return nil, err
	}
	txInput.TokenProgram = mintInfo.Value.Owner

	// get account info - check if to is an owner or ata
	accountTo, err := solana.PublicKeyFromBase58(string(args.To))
	if err != nil {
		return nil, err
	}

	// Determine if destination is a token account or not by
	// trying to lookup a token balance
	_, err = client.client.GetTokenAccountBalance(ctx, accountTo, rpc.CommitmentFinalized)
	if err != nil {
		txInput.ToIsATA = false
	} else {
		txInput.ToIsATA = true
	}

	// for tokens, get ata account info
	ataTo := accountTo
	if !txInput.ToIsATA {
		ataToStr, err := solana_types.FindAssociatedTokenAddress(string(args.To), string(contract), mintInfo.Value.Owner)
		if err != nil {
			return nil, err
		}
		ataTo = solana.MustPublicKeyFromBase58(ataToStr)
	}

	_, err = client.client.GetAccountInfo(ctx, ataTo)
	if err != nil {
		// if the ATA doesn't exist yet, we will create when sending tokens
		txInput.ShouldCreateATA = true
	}

	// Fetch all token accounts as if they are utxo
	if contract != "" {
		tokenAccounts, err := client.GetTokenAccountsByOwner(ctx, string(args.From), string(contract))
		if err != nil {
			return nil, err
		}
		zero := types.NewBigIntFromInt64(0)

		for _, acc := range tokenAccounts {
			amount := types.NewBigIntFromStr(acc.Info.Parsed.Info.TokenAmount.Amount)
			if amount.Cmp(&zero) > 0 {
				txInput.SourceTokenAccounts = append(txInput.SourceTokenAccounts, &tx_input.TokenAccount{
					Account: acc.Account.Pubkey,
					Balance: amount,
				})
			}
		}

		sort.Slice(txInput.SourceTokenAccounts, func(i, j int) bool {
			return txInput.SourceTokenAccounts[i].Balance.Cmp(&txInput.SourceTokenAccounts[j].Balance) > 0
		})
		if len(txInput.SourceTokenAccounts) > builder.MaxTokenTransfers {
			txInput.SourceTokenAccounts = txInput.SourceTokenAccounts[:builder.MaxTokenTransfers]
		}

		if len(tokenAccounts) == 0 {
			// no balance
			return nil, errors.New("no balance to send solana token")
		}
	}

	// fetch priority fee info
	accountsToLock := solana.PublicKeySlice{}
	accountsToLock = append(accountsToLock, mint)
	fees, err := client.client.GetRecentPrioritizationFees(ctx, accountsToLock)
	if err != nil {
		return txInput, fmt.Errorf("could not lookup priority fees: %v", err)
	}
	priority_fee_count := uint64(0)
	// start with 100 min priority fee, then average in the recent priority fees paid.
	priority_fee_sum := uint64(100)
	for _, fee := range fees {
		if fee.PrioritizationFee > 0 {
			priority_fee_sum += fee.PrioritizationFee
			priority_fee_count += 1
		}
	}
	if priority_fee_count > 0 {
		txInput.PrioritizationFee = types.NewBigIntFromUint64(
			priority_fee_sum / priority_fee_count,
		)
	} else {
		// default 100
		txInput.PrioritizationFee = types.NewBigIntFromUint64(
			100,
		)
	}
	// apply multiplier
	txInput.PrioritizationFee = txInput.PrioritizationFee.ApplyGasPriceMultiplier(client.Asset.GetChain())

	return txInput, nil
}

func (a *Client) EstimateGas(ctx context.Context, _tx types.Tx) (*types.BigInt, error) {
	tx := _tx.(*tx.Tx)
	solanaTx := tx.SolTx

	feeCalc, err := a.client.GetFeeForMessage(ctx, solanaTx.Message.ToBase64(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	value := *feeCalc.Value

	fee := types.NewBigIntFromUint64(value)
	return &fee, nil
}

func (a *Client) SubmitTx(ctx context.Context, _tx types.Tx) error {
	tx := _tx.(*tx.Tx)
	solanaTx := tx.SolTx

	payload, err := solanaTx.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = a.client.SendRawTransaction(ctx, payload)
	return err
}

func (a *Client) FetchBalance(ctx context.Context, address types.Address, contractAddress *types.Address) (*types.BigInt, error) {
	addr := solana.MustPublicKeyFromBase58(string(address))
	if contractAddress == nil {
		out, err := a.client.GetBalance(ctx, addr, rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}
		balance := types.NewBigIntFromUint64(out.Value)
		return &balance, nil
	} else {
		mint := solana.MustPublicKeyFromBase58(string(*contractAddress))
		associated, _, err := solana.FindAssociatedTokenAddress(addr, mint)
		if err != nil {
			return nil, err
		}
		out, err := a.client.GetTokenAccountBalance(ctx, associated, rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseInt(out.Value.Amount, 10, 64)
		if err != nil {
			return nil, err
		}
		balance := types.NewBigIntFromInt64(amount)
		return &balance, nil
	}
}

type TokenAccountWithInfo struct {
	// We need to manually parse TokenAccountInfo
	Info *solana_types.TokenAccountInfo
	// Account is what's returned by solana client
	Account *rpc.TokenAccount
}

// Get all token accounts for a given token that are owned by an address.
func (client *Client) GetTokenAccountsByOwner(ctx context.Context, addr string, contract string) ([]*TokenAccountWithInfo, error) {
	address, err := solana.PublicKeyFromBase58(addr)
	if err != nil {
		return nil, err
	}
	mint, err := solana.PublicKeyFromBase58(contract)
	if err != nil {
		return nil, err
	}

	conf := rpc.GetTokenAccountsConfig{
		Mint: &mint,
	}
	opts := rpc.GetTokenAccountsOpts{
		Commitment: rpc.CommitmentFinalized,
		// required to be able to parse extra data as json
		Encoding: "jsonParsed",
	}
	out, err := client.client.GetTokenAccountsByOwner(ctx, address, &conf, &opts)
	if err != nil || out == nil {
		return nil, err
	}
	tokenAccounts := []*TokenAccountWithInfo{}
	for _, acc := range out.Value {
		var accountInfo solana_types.TokenAccountInfo
		accountInfo, err = solana_types.ParseRpcData(acc.Account.Data)
		if err != nil {
			return nil, err
		}
		tokenAccounts = append(tokenAccounts, &TokenAccountWithInfo{
			Info:    &accountInfo,
			Account: acc,
		})
	}
	return tokenAccounts, nil
}
