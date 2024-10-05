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
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/require"
)

type TxInput = tx_input.TxInput
type Tx = tx.Tx

func TestNewTxBuilder(t *testing.T) {

	txBuilder, err := builder.NewTxBuilder(&xc_types.TokenAssetConfig{Asset: "USDC"})
	require.NoError(t, err)
	require.NotNil(t, txBuilder)
	require.Equal(t, "USDC", txBuilder.Asset.(*xc_types.TokenAssetConfig).Asset)
}

func TestNewNativeTransfer(t *testing.T) {
	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 SOL
	input := &tx_input.TxInput{
		From:   from,
		To:     to,
		Amount: amount,
	}
	tx, err := builder.NewNativeTransfer(input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx := tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x2), solTx.Message.Instructions[0].ProgramIDIndex) // system tx
}

func TestNewNativeTransferErr(t *testing.T) {

	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})

	from := xc_types.Address("from") // fails on parsing from
	to := xc_types.Address("to")
	amount := xc_types.BigInt{}
	input := &TxInput{
		From:   from,
		To:     to,
		Amount: amount,
	}
	tx, err := builder.NewNativeTransfer(input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 3")

	from = xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	input.From = from
	// fails on parsing to
	tx, err = builder.NewNativeTransfer(input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 2")
}

func TestNewTokenTransfer(t *testing.T) {

	contract := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	builder, _ := builder.NewTxBuilder(&xc_types.TokenAssetConfig{
		Contract:    contract,
		Decimals:    6,
		ChainConfig: &xc_types.ChainConfig{},
	})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 USDC

	ataToStr, _ := types.FindAssociatedTokenAddress(string(to), string(contract), solana.TokenProgramID)
	ataTo := solana.MustPublicKeyFromBase58(ataToStr)

	// transfer to existing ATA
	input := &TxInput{
		From:   from,
		To:     to,
		Amount: amount,
	}
	tx, err := builder.NewTokenTransfer(input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx := tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
	require.Equal(t, ataTo, solTx.Message.AccountKeys[2])                       // destination

	// transfer to non-existing ATA: create
	input = &TxInput{
		ShouldCreateATA: true,
		From:            from,
		To:              to,
		Amount:          amount,
	}
	tx, err = builder.NewTokenTransfer(input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 2, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x7), solTx.Message.Instructions[0].ProgramIDIndex)
	require.Equal(t, uint16(0x8), solTx.Message.Instructions[1].ProgramIDIndex)
	require.Equal(t, ataTo, solTx.Message.AccountKeys[1])

	// transfer directly to ATA
	to = xc_types.Address(ataToStr)
	input = &TxInput{
		ToIsATA: true,
		From:    from,
		To:      to,
		Amount:  amount,
	}
	tx, err = builder.NewTokenTransfer(input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x4), solTx.Message.Instructions[0].ProgramIDIndex) // token tx
	require.Equal(t, ataTo, solTx.Message.AccountKeys[2])                       // destination

	// invalid: direct to ATA, but ToIsATA: false
	to = xc_types.Address(ataToStr)
	input = &TxInput{
		ToIsATA: false,
		From:    from,
		To:      to,
		Amount:  amount,
	}
	tx, err = builder.NewTokenTransfer(input)
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

	contract := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	builder, _ := builder.NewTxBuilder(&xc_types.TokenAssetConfig{
		Contract:    contract,
		Decimals:    6,
		ChainConfig: &xc_types.ChainConfig{},
	})
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
		From:   from,
		To:     to,
		Amount: amountTooBig,
	}
	_, err = builder.NewTokenTransfer(input)
	require.ErrorContains(t, err, "cannot send")

	input.Amount = amountExact
	tx, err := builder.NewTokenTransfer(input)
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
	input.Amount = amountSmall1
	tx, err = builder.NewTokenTransfer(input)
	require.NoError(t, err)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[0]))

	// amountSmall2 should just have 2 instruction (first 100, second 50)
	input.Amount = amountSmall2
	tx, err = builder.NewTokenTransfer(input)
	require.NoError(t, err)
	solTx = tx.(*Tx).SolTx
	require.Equal(t, 2, len(solTx.Message.Instructions))
	require.EqualValues(t, 100, getTokenTransferAmount(solTx, &solTx.Message.Instructions[0]))
	require.EqualValues(t, 50, getTokenTransferAmount(solTx, &solTx.Message.Instructions[1]))

	// amountSmall3 should just have 3 instruction (first 100, second 100)
	input.Amount = amountSmall3
	tx, err = builder.NewTokenTransfer(input)
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
	input := &TxInput{
		From:   from,
		To:     to,
		Amount: amount,
	}
	tx, err := txBuilder.NewTokenTransfer(input)
	require.Nil(t, tx)
	require.EqualError(t, err, "asset does not have a contract")

	// invalid from, to
	txBuilder, _ = builder.NewTxBuilder(&xc_types.TokenAssetConfig{
		Contract: "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU",
		Decimals: 6,
	})
	from = xc_types.Address("from")
	to = xc_types.Address("to")
	amount = xc_types.BigInt{}
	input = &TxInput{
		From:   from,
		To:     to,
		Amount: amount,
	}
	tx, err = txBuilder.NewTokenTransfer(input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 3")

	from = xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	input.From = from
	tx, err = txBuilder.NewTokenTransfer(input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 2")

	// invalid asset config
	txBuilder, _ = builder.NewTxBuilder(&xc_types.TokenAssetConfig{
		Contract: "contract",
		Decimals: 6,
	})
	tx, err = txBuilder.NewTokenTransfer(input)
	require.Nil(t, tx)
	require.EqualError(t, err, "invalid length, expected 32, got 6")
}

func TestNewTransfer(t *testing.T) {

	builder, _ := builder.NewTxBuilder(&xc_types.ChainConfig{})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 SOL
	input := &TxInput{
		From:   from,
		To:     to,
		Amount: amount,
	}
	tx, err := builder.NewTransfer(input)
	require.NoError(t, err)
	require.NotNil(t, tx)
	solTx := tx.(*Tx).SolTx
	require.Equal(t, 0, len(solTx.Signatures))
	require.Equal(t, 1, len(solTx.Message.Instructions))
	require.Equal(t, uint16(0x2), solTx.Message.Instructions[0].ProgramIDIndex) // system tx
}

func TestNewTransferAsToken(t *testing.T) {

	builder, _ := builder.NewTxBuilder(&xc_types.TokenAssetConfig{
		Contract:    "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU",
		Decimals:    6,
		ChainConfig: &xc_types.ChainConfig{},
	})
	from := xc_types.Address("Hzn3n914JaSpnxo5mBbmuCDmGL6mxWN9Ac2HzEXFSGtb")
	to := xc_types.Address("BWbmXj5ckAaWCAtzMZ97qnJhBAKegoXtgNrv9BUpAB11")
	amount := xc_types.NewBigIntFromUint64(1200000) // 1.2 SOL

	type testcase struct {
		txInput               *TxInput
		expectedSourceAccount string
	}
	testcases := []testcase{
		{
			txInput: &TxInput{
				RecentBlockHash: solana.HashFromBytes([]byte{1, 2, 3, 4}),
				From:            from,
				To:              to,
				Amount:          amount,
			},
			expectedSourceAccount: "DvSgNMRxVSMBpLp4hZeBrmQo8ZRFne72actTZ3PYE3AA",
		},
		{
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
		tx, err := builder.NewTransfer(v.txInput)
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
