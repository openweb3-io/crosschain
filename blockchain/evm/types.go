package evm

import (
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/openweb3-io/crosschain/types"
)

type Tx struct {
	*eth_types.Transaction
	nonce      uint64
	signatures []types.TxSignature
}

func (tx *Tx) Serialize() ([]byte, error) {
	return tx.Transaction.MarshalBinary()
}

func (tx *Tx) Hash() types.TxHash {
	return types.TxHash(tx.Transaction.Hash().Bytes())
}

func (tx *Tx) GetSignatures() []types.TxSignature {
	return tx.signatures
}
