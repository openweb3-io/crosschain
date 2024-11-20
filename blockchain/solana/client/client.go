package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/openweb3-io/crosschain/blockchain/solana/builder"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx_input"
	solana_types "github.com/openweb3-io/crosschain/blockchain/solana/types"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xcclient "github.com/openweb3-io/crosschain/client"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Client struct {
	cfg    *xc.ChainConfig
	client *rpc.Client
}

var _ xcclient.IClient = &Client{}
var _ xcclient.StakingClient = &Client{}

func NewClient(cfg *xc.ChainConfig) (*Client, error) {
	endpoint := cfg.Client.URL
	if endpoint == "" {
		endpoint = rpc.MainNetBeta_RPC
	}
	client := rpc.New(endpoint)
	return &Client{cfg: cfg, client: client}, nil
}

func (client *Client) FetchBaseInput(ctx context.Context, fromAddr xc.Address) (*tx_input.TxInput, error) {
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

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc.TxInput, error) {
	txInput, err := client.FetchBaseInput(ctx, args.GetFrom())
	if err != nil {
		return nil, err
	}

	asset, _ := args.GetAsset()
	if asset == nil {
		return txInput, nil
	}

	mint, err := solana.PublicKeyFromBase58(string(asset.GetContract()))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid mint address: %s", string(asset.GetContract()))
	}

	mintInfo, err := client.client.GetAccountInfo(ctx, mint)
	if err != nil {
		return nil, err
	}
	txInput.TokenProgram = mintInfo.Value.Owner

	// get account info - check if to is an owner or ata
	accountTo, err := solana.PublicKeyFromBase58(string(args.GetTo()))
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
		ataToStr, err := solana_types.FindAssociatedTokenAddress(string(args.GetTo()), string(asset.GetContract()), mintInfo.Value.Owner)
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
	if asset.GetContract() != "" {
		tokenAccounts, err := client.GetTokenAccountsByOwner(ctx, string(args.GetFrom()), string(asset.GetContract()))
		if err != nil {
			return nil, err
		}
		zero := xc.NewBigIntFromInt64(0)

		for _, acc := range tokenAccounts {
			amount := xc.NewBigIntFromStr(acc.Info.Parsed.Info.TokenAmount.Amount)
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
		txInput.PrioritizationFee = xc.NewBigIntFromUint64(
			priority_fee_sum / priority_fee_count,
		)
	} else {
		// default 100
		txInput.PrioritizationFee = xc.NewBigIntFromUint64(
			100,
		)
	}
	// apply multiplier
	txInput.PrioritizationFee = txInput.PrioritizationFee.ApplyGasPriceMultiplier(client.cfg)

	return txInput, nil
}

func (a *Client) EstimateGasFee(ctx context.Context, _tx xc.Tx) (*xc.BigInt, error) {
	tx := _tx.(*tx.Tx)
	solanaTx := tx.SolTx

	feeCalc, err := a.client.GetFeeForMessage(ctx, solanaTx.Message.ToBase64(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	value := *feeCalc.Value

	fee := xc.NewBigIntFromUint64(value)
	return &fee, nil
}

func (a *Client) BroadcastTx(ctx context.Context, _tx xc.Tx) error {
	tx := _tx.(*tx.Tx)
	solanaTx := tx.SolTx

	payload, err := solanaTx.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = a.client.SendRawTransaction(ctx, payload)
	return err
}

func (a *Client) FetchBalance(ctx context.Context, address xc.Address) (*xc.BigInt, error) {
	addr := solana.MustPublicKeyFromBase58(string(address))
	out, err := a.client.GetBalance(ctx, addr, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	balance := xc.NewBigIntFromUint64(out.Value)
	return &balance, nil
}

func (a *Client) FetchBalanceForAsset(ctx context.Context, address xc.Address, contractAddress xc.ContractAddress) (*xc.BigInt, error) {
	addr := solana.MustPublicKeyFromBase58(string(address))

	mint := solana.MustPublicKeyFromBase58(string(contractAddress))
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
	balance := xc.NewBigIntFromInt64(amount)
	return &balance, nil

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

// FetchLegacyTxInfo returns tx info for a Solana tx
func (client *Client) FetchLegacyTxInfo(ctx context.Context, txHash xc.TxHash) (*xc.LegacyTxInfo, error) {
	result := &xc.LegacyTxInfo{}

	txSig, err := solana.SignatureFromBase58(string(txHash))
	if err != nil {
		return nil, err
	}
	// confusingly, '0' is the latest version, which comes after 'legacy' (no version).
	maxVersion := uint64(0)
	res, err := client.client.GetTransaction(
		ctx,
		txSig,
		&rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64,
			Commitment:                     rpc.CommitmentFinalized,
			MaxSupportedTransactionVersion: &maxVersion,
		},
	)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Transaction == nil {
		return nil, errors.New("invalid transaction in response")
	}

	solTx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(res.Transaction.GetBinary()))
	if err != nil {
		return nil, err
	}
	tx := tx.NewTxFrom(solTx)
	meta := res.Meta
	if res.BlockTime != nil {
		result.BlockTime = res.BlockTime.Time().Unix()
	}

	if res.Slot > 0 {
		result.BlockIndex = int64(res.Slot)
		if res.BlockTime != nil {
			result.BlockTime = int64(*res.BlockTime)
		}

		recent, err := client.client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			// ignore
			logrus.WithError(err).Warn("failed to get latest blockhash")
		} else {
			result.Confirmations = int64(recent.Context.Slot) - result.BlockIndex
		}
	}
	result.Fee = xc.NewBigIntFromUint64(meta.Fee)

	result.TxID = string(txHash)
	result.ExplorerURL = client.cfg.ExplorerURL + "/tx/" + result.TxID + "?cluster=" + client.cfg.Network

	sources := []*xc.LegacyTxInfoEndpoint{}
	dests := []*xc.LegacyTxInfoEndpoint{}

	for _, instr := range tx.GetSystemTransfers() {
		from := instr.GetFundingAccount().PublicKey.String()
		to := instr.GetRecipientAccount().PublicKey.String()
		amount := xc.NewBigIntFromUint64(*instr.Lamports)
		sources = append(sources, &xc.LegacyTxInfoEndpoint{
			Address: xc.Address(from),
			Amount:  amount,
		})
		dests = append(dests, &xc.LegacyTxInfoEndpoint{
			Address: xc.Address(to),
			Amount:  amount,
		})
	}
	for _, instr := range tx.GetVoteWithdraws() {
		from := instr.GetWithdrawAuthorityAccount().PublicKey.String()
		to := instr.GetRecipientAccount().PublicKey.String()
		amount := xc.NewBigIntFromUint64(*instr.Lamports)
		sources = append(sources, &xc.LegacyTxInfoEndpoint{
			Address: xc.Address(from),
			Amount:  amount,
		})
		dests = append(dests, &xc.LegacyTxInfoEndpoint{
			Address: xc.Address(to),
			Amount:  amount,
		})
	}
	for _, instr := range tx.GetStakeWithdraws() {
		from := instr.GetStakeAccount().PublicKey.String()
		to := instr.GetRecipientAccount().PublicKey.String()
		amount := xc.NewBigIntFromUint64(*instr.Lamports)
		sources = append(sources, &xc.LegacyTxInfoEndpoint{
			Address: xc.Address(from),
			Amount:  amount,
		})
		dests = append(dests, &xc.LegacyTxInfoEndpoint{
			Address: xc.Address(to),
			Amount:  amount,
		})
	}
	for _, instr := range tx.GetTokenTransferCheckeds() {
		from := instr.GetOwnerAccount().PublicKey.String()
		toTokenAccount := instr.GetDestinationAccount().PublicKey
		contract := xc.ContractAddress(instr.GetMintAccount().PublicKey.String())
		to := xc.Address(toTokenAccount.String())
		// Solana doesn't keep full historical state, so we can't rely on always being able to lookup the account.
		tokenAccountInfo, err := client.LookupTokenAccount(ctx, toTokenAccount)
		if err != nil {
			logrus.WithError(err).Warn("failed to lookup token account")
		} else {
			to = xc.Address(tokenAccountInfo.Parsed.Info.Owner)
		}

		amount := xc.NewBigIntFromUint64(*instr.Amount)
		sources = append(sources, &xc.LegacyTxInfoEndpoint{
			Address:         xc.Address(from),
			Amount:          amount,
			ContractAddress: contract,
		})
		dests = append(dests, &xc.LegacyTxInfoEndpoint{
			Address:         xc.Address(to),
			Amount:          amount,
			ContractAddress: contract,
		})
	}
	for _, instr := range tx.GetTokenTransfers() {
		from := instr.GetOwnerAccount().PublicKey.String()
		toTokenAccount := instr.GetDestinationAccount().PublicKey
		tokenAccountInfo, err := client.LookupTokenAccount(ctx, toTokenAccount)

		to := xc.Address(toTokenAccount.String())
		contract := xc.ContractAddress("")
		// Solana doesn't keep full historical state, so we can't rely on always being able to lookup the account.
		if err != nil {
			logrus.WithError(err).Warn("failed to lookup token account")
		} else {
			to = xc.Address(tokenAccountInfo.Parsed.Info.Owner)
			contract = xc.ContractAddress(tokenAccountInfo.Parsed.Info.Mint)
		}

		amount := xc.NewBigIntFromUint64(*instr.Amount)
		sources = append(sources, &xc.LegacyTxInfoEndpoint{
			Address:         xc.Address(from),
			Amount:          amount,
			ContractAddress: contract,
		})
		dests = append(dests, &xc.LegacyTxInfoEndpoint{
			Address:         xc.Address(to),
			Amount:          amount,
			ContractAddress: contract,
		})
	}
	for _, instr := range tx.GetDelegateStake() {
		xcStake := &xcclient.Stake{
			Account:   instr.GetStakeAccount().PublicKey.String(),
			Validator: instr.GetVoteAccount().PublicKey.String(),
			Address:   instr.GetStakeAuthority().PublicKey.String(),
			// Needs to be looked up from separate instruction
			Balance: xc.BigInt{},
		}
		for _, createAccount := range tx.GetCreateAccounts() {
			if createAccount.NewAccount.Equals(instr.GetStakeAccount().PublicKey) {
				xcStake.Balance = xc.NewBigIntFromUint64(createAccount.Lamports)
			}
		}

		result.AddStakeEvent(xcStake)
	}
	for _, instr := range tx.GetDeactivateStakes() {
		xcStake := &xcclient.Unstake{
			Account: instr.GetStakeAccount().PublicKey.String(),
			Address: instr.GetStakeAuthority().PublicKey.String(),

			// Needs to be looked up
			Balance:   xc.BigInt{},
			Validator: "",
		}
		stakeAccountInfo, err := client.LookupStakeAccount(ctx, instr.GetStakeAccount().PublicKey)
		if err != nil {
			logrus.WithError(err).Warn("failed to lookup stake account")
		} else {
			xcStake.Validator = stakeAccountInfo.Parsed.Info.Stake.Delegation.Voter
			xcStake.Balance = xc.NewBigIntFromStr(stakeAccountInfo.Parsed.Info.Stake.Delegation.Stake)
		}
		result.AddStakeEvent(xcStake)
	}

	if len(sources) > 0 {
		result.From = sources[0].Address
	}
	if len(dests) > 0 {
		result.To = dests[0].Address
		result.Amount = dests[0].Amount
		result.ContractAddress = dests[0].ContractAddress
	}

	result.Sources = sources
	result.Destinations = dests

	return result, nil
}

func (client *Client) LookupTokenAccount(ctx context.Context, tokenAccount solana.PublicKey) (solana_types.TokenAccountInfo, error) {
	var accountInfo solana_types.TokenAccountInfo
	info, err := client.client.GetAccountInfoWithOpts(ctx, tokenAccount, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentFinalized,
		Encoding:   "jsonParsed",
	})
	if err != nil {
		return solana_types.TokenAccountInfo{}, err
	}
	accountInfo, err = solana_types.ParseRpcData(info.Value.Data)
	if err != nil {
		return solana_types.TokenAccountInfo{}, err
	}
	return accountInfo, nil
}

func (client *Client) LookupStakeAccount(ctx context.Context, stakeAccount solana.PublicKey) (solana_types.StakeAccount, error) {
	info, err := client.client.GetAccountInfoWithOpts(ctx, stakeAccount, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentFinalized,
		Encoding:   "jsonParsed",
	})
	if err != nil {
		return solana_types.StakeAccount{}, err
	}
	var stakeAccountInfo solana_types.StakeAccount
	err = json.Unmarshal(info.Value.Data.GetRawJSON(), &stakeAccountInfo)
	if err != nil {
		return solana_types.StakeAccount{}, err
	}
	return stakeAccountInfo, nil
}

func (client *Client) FetchLegacyTxInput(ctx context.Context, from xc.Address, to xc.Address, asset xc.IAsset) (xc.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, xc.NewBigIntFromUint64(1), xcbuilder.WithAsset(asset))
	return client.FetchTransferInput(ctx, args)
}
