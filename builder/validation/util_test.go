package validation_test

import (
	"testing"

	"github.com/openweb3-io/crosschain/builder/validation"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/require"
)

func mustToBlockchain(human string) xc_types.BigInt {
	dec, err := xc_types.NewAmountHumanReadableFromStr(human)
	if err != nil {
		panic(err)
	}
	return dec.ToBlockchain(18)
}

func TestStakingAmount(t *testing.T) {

	div, err := validation.Count32EthChunks(mustToBlockchain("32"))
	require.NoError(t, err)
	require.EqualValues(t, 1, div)
	div, err = validation.Count32EthChunks(mustToBlockchain("96"))
	require.NoError(t, err)
	require.EqualValues(t, 3, div)

	_, err = validation.Count32EthChunks(mustToBlockchain("48"))
	require.Error(t, err)

	_, err = validation.Count32EthChunks(mustToBlockchain("32.00001"))
	require.Error(t, err)

	_, err = validation.Count32EthChunks(mustToBlockchain("31"))
	require.Error(t, err)

	_, err = validation.Count32EthChunks(mustToBlockchain("0"))
	require.Error(t, err)
}
