package builder_test

import (
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/openweb3-io/crosschain/blockchain/solana/builder"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx_input"
	"github.com/openweb3-io/crosschain/blockchain/solana/types"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/require"
)

type TxInput = tx_input.TxInput
type Tx = tx.Tx

/*
func TestNewTxBuilder(t *testing.T) {
	// txBuilder, err := builder.NewTxBuilder(&xc_types.TokenAssetConfig{Asset: "USDC"})
	require.NoError(t, err)
	require.NotNil(t, txBuilder)
	require.Equal(t, "USDC", txBuilder.Asset.(*xc_types.TokenAssetConfig).Asset)
}*/

func TestNewNativeTransfer(t *testing.T) {
	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	args, err := xcbuilder.NewTransferArgs(
		xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb"),
		xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11"),
		xc_types.NewBigIntFromUint64(1200000), // 1.2 SOL
	)
	require.NoError(t, err)

	input := &tx_input.TxInput{}
	tx, err := builder.NewNativeTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx := tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x2), solTx.Message.Instructions[0].ProgramIDIndex) // system tx
}

func TestNewNativeTransferErr(t *testing.T) {

	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})

	args, err := xcbuilder.NewTransferArgs(
		xc_types.Address("from"),
		xc_types.Address("to"),
		xc_types.BigInt{},
	)
	require.NoError(t, err)

	input := &TxInput{}
	tx, err := builder.NewNativeTransfer(args, input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 3")

	args.SetFrom(xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb"))
	// fails on parsing to
	tx, err = builder.NewNativeTransfer(args, input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 2")
}

func TestNewTokenTransfer(t *testing.T) {
	contract := xc_types.ContractAddress("4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU")
	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 USDC

	ataToStr, _ := types.FindAssociatedTokenAddress(string(to), string(contract), solana.TokenProgramID)
	ataTo := solana.MustPublicKeyFromBase58(ataToStr)

	args, err := xcbuilder.NewTransferArgs(
		from,
		to,
		amount,
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract:    contract,
			Decimals:    6,
			ChainConfig: &xc_types.ChainConfig{},
		}),
	)
	require.NoError(t, err)

	// transfer to existing ATA
	input := &TxInput{}
	tx, err := builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx := tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
	require.Equal(t, ataTo, solTx.Message.AccountKeys[2])                       // destination

	// transfer to non-existing ATA: create
	args, err = xcbuilder.NewTransferArgs(
		from,
		to,
		amount,
	)
	require.NoError(t, err)

	input = &TxInput{
		ShouldCreateATA: true,
	}
	tx, err = builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 2, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x7), solTx.Message.Instructions[0].ProgramIDIndex)
	require.Equal(t, uint16(0x8), solTx.Message.Instructions[1].ProgramIDIndex)
	require.Equal(t, ataTo, solTx.Message.AccountKeys[1])

	// transfer directly to ATA
	args.SetTo(xc_types.Address(ataToStr))

	input = &TxInput{
		ToIsATA: true,
	}
	tx, err = builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
	require.Equal(t, ataTo, solTx.Message.AccountKeys[2])                       // destination

	// invalid: direct to ATA, but ToIsATA: false
	args.SetTo(xc_types.Address(ataToStr))
	input = &TxInput{
		ToIsATA: false,
	}
	tx, err = builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
	require.NotEqual(t, ataTo, solTx.Message.AccountKeys[2])                    // destination
}

func validateTransferChecked(tx *solana.Transaction, instr *solana.CompiledInstruction) (*token.TransferChecked, error) {
	accs, _ := instr.ResolveInstructionAccounts(&tx.Message)
	inst, _ := token.DecodeInstruction(accs, instr.Data)
	transferChecked := *inst.Impl.(*token.TransferChecked)
	if len(transferChecked.Signers) > 0 {
		return &transferChecked, fmt.Errorf("should not send multisig transfers")
	}
	return &transferChecked, nil
}
func getTokenTransferAmount(tx *solana.Transaction, instr *solana.CompiledInstruction) uint64 {
	transferChecked, err := validateTransferChecked(tx, instr)
	if err != nil {
		panic(err)
	}
	return *transferChecked.Amount
}

func TestNewMultiTokenTransfer(t *testing.T) {

	contract := xc_types.ContractAddress("4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU")
	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amountTooBig := xc_types.NewBigIntFromUint64(500)
	amountExact := xc_types.NewBigIntFromUint64(300)
	amountSmall1 := xc_types.NewBigIntFromUint64(100)
	amountSmall2 := xc_types.NewBigIntFromUint64(150)
	amountSmall3 := xc_types.NewBigIntFromUint64(200)

	ataToStr, err := types.FindAssociatedTokenAddress(string(to), string(contract), solana.TokenProgramID)
	require.NoError(t, err)
	ataTo := solana.MustPublicKeyFromBase58(ataToStr)

	args, err := xcbuilder.NewTransferArgs(
		from,
		to,
		amountTooBig,
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract:    contract,
			Decimals:    6,
			ChainConfig: &xc_types.ChainConfig{},
		}),
	)
	require.NoError(t, err)

	// transfer to existing ATA
	input := &TxInput{
		SourceTokenAccounts: []*tx_input.TokenAccount{
			{
				Account: solana.PublicKey{},
				Balance: xc_types.NewBigIntFromUint64(100),
			},
			{
				Account: solana.PublicKey{},
				Balance: xc_types.NewBigIntFromUint64(100),
			},
			{
				Account: solana.PublicKey{},
				Balance: xc_types.NewBigIntFromUint64(100),
			},
		},
	}
	_, err = builder.NewTokenTransfer(args, input)
	require.ErrorContains(t, err, "cannot send")

	args.SetAmount(amountExact)
	tx, err := builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	solTx := tx.(*Tx).SolTx

	_, err = validateTransferChecked(solTx, &solTx.Message.Instructions[0])
	require.NoError(t, err)

	require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
	require.Equal(t, ataTo, solTx.Message.AccountKeys[2])                       // destination
	require.Equal(t, 3, len(solTx.Message.Instructions))
	// exactAmount should have 3 instructions, 100 amount each
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[0]))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[1]))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[2]))

	// amountSmall1 should just have 1 instruction (fits 1 token balance exact)
	args.SetAmount(amountSmall1)
	tx, err = builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[0]))

	// amountSmall2 should just have 2 instruction (first 100, second 50)
	args.SetAmount(amountSmall2)
	tx, err = builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 2, len(solTx.Message.Instructions))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[0]))
	require.EqualValues(t, 50, getTokenTransferAmount(solTx, &solTx.Message.Instructions[1]))

	// amountSmall3 should just have 3 instruction (first 100, second 100)	input.Amount =
	args.SetAmount(amountSmall3)
	tx, err = builder.NewTokenTransfer(args, input)
	require.NoError(t, err)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 2, len(solTx.Message.Instructions))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[0]))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[1]))

}

func TestNewTokenTransferErr(t *testing.T) {

	// invalid asset
	txBuilder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("from")
	to := xc_types.Address("to")
	amount := xc_types.BigInt{}

	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(t, err)

	input := &TxInput{}
	tx, err := txBuilder.NewTokenTransfer(args, input)
	require.Nil(t, tx)
	require.EqualError(t, err, "asset does not have a contract")

	// invalid from, to
	from = xc_types.Address("from")
	to = xc_types.Address("to")
	amount = xc_types.BigInt{}

	args, err = xcbuilder.NewTransferArgs(from, to, amount,
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract: "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU",
			Decimals: 6,
		}),
	)
	require.NoError(t, err)

	input = &TxInput{}
	tx, err = txBuilder.NewTokenTransfer(args, input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 3")

	from = xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	args.SetFrom(from)
	tx, err = txBuilder.NewTokenTransfer(args, input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 2")

	// invalid asset config
	args.SetAsset(&xc_types.TokenAssetConfig{
		Contract: "contract",
		Decimals: 6,
	})

	txBuilder, _ = builder.NewTxBuilder(&xc_types.ChainConfig{})
	tx, err = txBuilder.NewTokenTransfer(args, input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 6")
}

func TestNewTransfer(t *testing.T) {
	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 SOL

	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(t, err)

	input := &TxInput{}
	tx, err := builder.NewTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx := tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x2), solTx.Message.Instructions[0].ProgramIDIndex) // system tx
}

func TestNewTransferAsToken(t *testing.T) {
	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 SOL

	args, err := xcbuilder.NewTransferArgs(
		from,
		to,
		amount,
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract:    "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU",
			Decimals:    6,
			ChainConfig: &xc_types.ChainConfig{},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	type testcase struct {
		args                  *xcbuilder.TransferArgs
		txInput               *TxInput
		expectedSourceAccount string
	}
	testcases := []testcase{
		{
			args: args,
			txInput: &TxInput{
				RecentBlockHash: solana.HashFromBytes([]byte{1, 2, 3, 4}),
			},
			expectedSourceAccount: "DvSgNMRxVSMBpLp4hZeBrmQo8ZRFne72actTZ3PYE3AA",
		},
		{
			args: args,
			txInput: &TxInput{
				RecentBlockHash: solana.HashFromBytes([]byte{1, 2, 3, 4}),
				SourceTokenAccounts: []*tx_input.TokenAccount{
					{
						Account: solana.MustPublicKeyFromBase58("gCr8Xc43gEKntp7pjsBNq8qFHeUUdie2D7TrfbzPMJP"),
					},
				},
			},
			// should use new source account specified in txInput
			expectedSourceAccount: "gCr8Xc43gEKntp7pjsBNq8qFHeUUdie2D7TrfbzPMJP",
		},
	}
	for _, v := range testcases {
		tx, err := builder.NewTransfer(v.args, v.txInput)
		require.Nil(t, err)
		require.NotNil(t, tx)
		solTx := tx.(*Tx).SolTx
		require.Equal(t, 0, len(solTx.Signatures))
		require.Equal(t, 1, len(solTx.Message.Instructions))
		require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
		tokenTf, err := validateTransferChecked(solTx, &solTx.Message.Instructions[0])
		require.NoError(t, err)
		require.Equal(t, v.expectedSourceAccount, tokenTf.Accounts[0].PublicKey.String())
	}
}
