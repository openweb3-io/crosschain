package bitcoin_cash

import (
	"github.com/openweb3-io/crosschain/blockchain/btc"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
)

// TxBuilder for Bitcoin
type TxBuilder struct {
	btc.TxBuilder
}

var _ xcbuilder.FullTransferBuilder = &TxBuilder{}

// NewTxBuilder creates a new Bitcoin TxBuilder
func NewTxBuilder(cfg *xc.ChainConfig) (TxBuilder, error) {
	txBuilder, err := btc.NewTxBuilder(cfg)
	if err != nil {
		return TxBuilder{}, err
	}
	return TxBuilder{
		TxBuilder: txBuilder.WithAddressDecoder(&BchAddressDecoder{}),
	}, nil
}

func (txBuilder TxBuilder) Transfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	return txBuilder.NewTransfer(args.GetFrom(), args.GetTo(), args.GetAmount(), input)
}

func (txBuilder TxBuilder) NewTransfer(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	txObj, err := txBuilder.TxBuilder.NewTransfer(from, to, amount, input)
	if err != nil {
		return txObj, err
	}
	return txObj.(*tx.Tx), nil
}

func (txBuilder TxBuilder) NewNativeTransfer(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	txObj, err := txBuilder.TxBuilder.NewNativeTransfer(from, to, amount, input)
	if err != nil {
		return txObj, err
	}
	return txObj.(*tx.Tx), nil
}

func (txBuilder TxBuilder) NewTokenTransfer(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	txObj, err := txBuilder.TxBuilder.NewTokenTransfer(from, to, amount, input)
	if err != nil {
		return txObj, err
	}
	return txObj.(*tx.Tx), nil
}
