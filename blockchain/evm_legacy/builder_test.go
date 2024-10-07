package evm_legacy_test

import (
	"fmt"
	"testing"

	"github.com/openweb3-io/crosschain/blockchain/evm/builder"
	"github.com/openweb3-io/crosschain/blockchain/evm_legacy"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/test-go/testify/require"
)

func TestBuilderLegacyTransfer(t *testing.T) {
	// EVM legacy re-uses the EVM builder, but uses a different tx-input.
	// This ensures that the builder properly typecasts/converts to the evm input, avoiding any panic.
	b, _ := evm_legacy.NewTxBuilder(&xc.ChainConfig{})

	from := xc.Address("0x724435CC1B2821362c2CD425F2744Bd7347bf299")
	to := xc.Address("0x3ad57b83B2E3dC5648F32e98e386935A9B10bb9F")
	amount := xc.NewBigIntFromUint64(100)

	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(t, err)

	input := evm_legacy.NewTxInput()

	fmt.Println("--- ", input.GetBlockchain())
	fmt.Printf("--- %T\n", input)

	input.GasTipCap = builder.GweiToWei(evm_legacy.DefaultMaxTipCapGwei - 1)
	trans, err := b.NewTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, trans)

	trans, err = b.NewTokenTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, trans)

	trans, err = b.NewNativeTransfer(args, input)
	require.NoError(t, err)
	require.NotNil(t, trans)
}
