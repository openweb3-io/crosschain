package solana

import (
	solana_sdk "github.com/gagliardetto/solana-go"
	"github.com/openweb3-io/crosschain/types"
)

type Tx struct {
	*solana_sdk.Transaction
	// opts       *types.TransferArgs
	signatures []types.TxSignature
}

func (tx *Tx) Serialize() ([]byte, error) {
	return tx.Transaction.MarshalBinary()
}

func (tx *Tx) Hash() types.TxHash {
	return types.TxHash(tx.signatures[0])
}

func (tx *Tx) GetSignatures() []types.TxSignature {
	return tx.signatures
}
