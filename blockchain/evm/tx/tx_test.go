package tx_test

import (
	"testing"

	"github.com/openweb3-io/crosschain/blockchain/evm/tx"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/require"
)

func TestTxHashEmpty(t *testing.T) {
	tx := tx.Tx{}
	require.Equal(t, xc_types.TxHash(""), tx.Hash())
}

func TestTxSighashesEmpty(t *testing.T) {
	tx := tx.Tx{}
	sighashes, err := tx.Sighashes()
	require.NotNil(t, sighashes)
	require.EqualError(t, err, "transaction not initialized")
}

func TestTxAddSignatureEmpty(t *testing.T) {
	tx := tx.Tx{}
	err := tx.AddSignatures([]xc_types.TxSignature{}...)
	require.EqualError(t, err, "transaction not initialized")
}
