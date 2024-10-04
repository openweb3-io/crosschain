package solana

import (
	"github.com/openweb3-io/crosschain/types"
)

type TxBuilder struct {
}

func NewTxBuilder() *TxBuilder {
	return &TxBuilder{}
}

func (b *TxBuilder) NewTransfer(input types.TxInput) (types.Tx, error) {
	txInput := input.(*TxInput)

	if txInput.ContractAddress != nil {
		return b.NewTokenTransfer()
	} else {
		return b.NewNativeTransfer()
	}
}

func (b *TxBuilder) NewTokenTransfer() (types.Tx, error) {
	return nil, nil
}

func (b *TxBuilder) NewNativeTransfer() (types.Tx, error) {
	return nil, nil
}
