package solana

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	solana_sdk "github.com/gagliardetto/solana-go"
	ata "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"

	_types "github.com/openweb3-io/crosschain/types"
)

type SolanaApi struct {
	endpoint string
	chainId  *big.Int
	client   *rpc.Client
}

func New(endpoint string, chainId *big.Int) *SolanaApi {
	if endpoint == "" {
		endpoint = rpc.MainNetBeta_RPC
	}
	client := rpc.New(endpoint)
	return &SolanaApi{endpoint, chainId, client}
}

func (a *SolanaApi) EstimateGas(ctx context.Context, input *_types.TransferArgs) (*big.Int, error) {

	tx, err := a.buildTransction(ctx, input)
	if err != nil {
		return nil, err
	}

	feeCalc, err := a.client.GetFeeForMessage(ctx, tx.Message.ToBase64(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	fee := uint64(*feeCalc.Value)

	return big.NewInt(int64(fee)), nil
}

func (a *SolanaApi) BuildTransaction(ctx context.Context, input *_types.TransferArgs) (_types.Tx, error) {
	solanaTx, err := a.buildTransction(ctx, input)
	if err != nil {
		return nil, err
	}

	solanaTx, err = a.signTx(ctx, input, solanaTx)
	if err != nil {
		return nil, err
	}

	signatures := make([]_types.TxSignature, len(solanaTx.Signatures))
	for i, signature := range solanaTx.Signatures {
		signatures[i] = signature[:]
	}

	return &Tx{
		Transaction: solanaTx,
		signatures:  signatures,
	}, nil
}

func (a *SolanaApi) signTx(ctx context.Context, input *_types.TransferArgs, tx *solana_sdk.Transaction) (*solana_sdk.Transaction, error) {
	rawData, err := tx.Message.MarshalBinary()
	if err != nil {
		return nil, err
	}

	if input.FeePayer != nil {
		_signatureFee, err := input.FeePayerSigner.Sign(ctx, rawData)
		if err != nil {
			return nil, err
		}
		tx.Signatures = append(tx.Signatures, solana_sdk.Signature(_signatureFee))
	}

	_signature, err := input.Signer.Sign(ctx, rawData)
	if err != nil {
		return nil, err
	}
	tx.Signatures = append(tx.Signatures, solana_sdk.Signature(_signature))

	return tx, nil
}

func (a *SolanaApi) BroadcastSignedTx(ctx context.Context, _tx _types.Tx) error {
	tx := _tx.(*Tx)
	solanaTx := tx.Transaction

	payload, err := solanaTx.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = a.client.SendRawTransaction(ctx, payload)
	return err
}

func (a *SolanaApi) buildTransction(ctx context.Context, input *_types.TransferArgs) (*solana_sdk.Transaction, error) {
	recipient := solana_sdk.MustPublicKeyFromBase58(input.To)
	sender := solana_sdk.MustPublicKeyFromBase58(input.From)

	if input.FeePayer != nil && input.From == *input.FeePayer {
		return nil, fmt.Errorf("no need to set feepayer")
	}

	recent, err := a.client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	var tx *solana_sdk.Transaction

	opts := []solana_sdk.TransactionOption{}
	if input.FeePayer != nil {
		feePayer := solana_sdk.MustPublicKeyFromBase58(*input.FeePayer)
		opts = append(opts, solana_sdk.TransactionPayer(feePayer))
	}

	if input.ContractAddress == nil {

		balance, err := a.client.GetBalance(ctx, sender, rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}
		if input.Amount.Int().Cmp(big.NewInt(int64(balance.Value))) > 0 {
			return nil, fmt.Errorf("insuffiecent amount, balance: %v, amount: %v", balance.Value, input.Amount.String())
		}

		tx, err = solana_sdk.NewTransaction(
			[]solana_sdk.Instruction{
				system.NewTransferInstruction(
					input.Amount.Uint64(),
					sender,
					recipient,
				).Build(),
			},
			recent.Value.Blockhash,
			opts...,
		)
		if err != nil {
			return nil, err
		}
	} else {
		instr := []solana_sdk.Instruction{}
		mint := solana_sdk.MustPublicKeyFromBase58(*input.ContractAddress)

		createInst, err := ata.NewCreateInstruction(sender, recipient, mint).ValidateAndBuild()
		if err != nil {
			return nil, err
		}

		senderAssociated, _, _ := solana_sdk.FindAssociatedTokenAddress(sender, mint)
		recipientAssociated, _, _ := solana_sdk.FindAssociatedTokenAddress(recipient, mint)

		if _, err := a.client.GetAccountInfo(ctx, recipientAssociated); err != nil {
			instr = append(instr, createInst)
		}

		balance, err := a.client.GetTokenAccountBalance(ctx, senderAssociated, rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}

		balanceAmount, err := strconv.ParseInt(balance.Value.Amount, 10, 64)
		if err != nil {
			return nil, err
		}

		if input.Amount.Int().Cmp(big.NewInt(balanceAmount)) > 0 {
			return nil, fmt.Errorf("insuffiecent amount, balance: %v, amount: %v", balance.Value, input.Amount.String())
		}

		transInst, err := token.NewTransferCheckedInstruction(
			input.Amount.Uint64(),
			uint8(input.TokenDecimals),
			senderAssociated,
			mint,
			recipientAssociated,
			sender,
			nil,
		).ValidateAndBuild()
		if err != nil {
			return nil, err
		}

		instr = append(instr, transInst)
		tx, err = solana_sdk.NewTransaction(instr, recent.Value.Blockhash, opts...)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

func (a *SolanaApi) GetBalance(ctx context.Context, address string, contractAddress *string) (*big.Int, error) {
	balance := big.NewInt(0)

	addr := solana_sdk.MustPublicKeyFromBase58(address)
	if contractAddress == nil {
		out, err := a.client.GetBalance(ctx, addr, rpc.CommitmentFinalized)
		if err != nil {
			return balance, err
		}
		balance = big.NewInt(int64(out.Value))
	} else {
		mint := solana_sdk.MustPublicKeyFromBase58(*contractAddress)
		associated, _, err := solana_sdk.FindAssociatedTokenAddress(addr, mint)
		if err != nil {
			return balance, err
		}
		out, err := a.client.GetTokenAccountBalance(ctx, associated, rpc.CommitmentFinalized)
		if err != nil {
			return balance, err
		}
		amount, err := strconv.ParseInt(out.Value.Amount, 10, 64)
		if err != nil {
			return balance, err
		}
		balance = big.NewInt(amount)
	}

	return balance, nil
}
