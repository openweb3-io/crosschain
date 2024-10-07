package evm_legacy

import (
	evmtx "github.com/openweb3-io/crosschain/blockchain/evm/tx"
	xc "github.com/openweb3-io/crosschain/types"
)

// Tx for EVM
type Tx = evmtx.Tx

var _ xc.Tx = &Tx{}
