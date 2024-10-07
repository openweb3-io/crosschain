package btc

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/openweb3-io/crosschain/blockchain/btc/address"
	"github.com/openweb3-io/crosschain/blockchain/btc/params"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/sirupsen/logrus"
)

const TxVersion int32 = 2

// TxBuilder for Bitcoin
type TxBuilder struct {
	Chain          *xc.ChainConfig
	Params         *chaincfg.Params
	AddressDecoder address.AddressDecoder
	// isBch  bool
}

var _ xcbuilder.TxBuilder = &TxBuilder{}

// NewTxBuilder creates a new Bitcoin TxBuilder
func NewTxBuilder(cfg *xc.ChainConfig) (TxBuilder, error) {
	params, err := params.GetParams(cfg)
	if err != nil {
		return TxBuilder{}, err
	}
	return TxBuilder{
		Chain:          cfg,
		Params:         params,
		AddressDecoder: &address.BtcAddressDecoder{},
		// isBch:  native.Chain == xc.BCH,
	}, nil
}

func (txBuilder TxBuilder) WithAddressDecoder(decoder address.AddressDecoder) TxBuilder {
	txBuilder.AddressDecoder = decoder
	return txBuilder
}

// Old transfer interface
func (txBuilder TxBuilder) NewTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	asset, _ := args.GetAsset()
	if asset == nil {
		asset = txBuilder.Chain
	}

	switch asset := asset.(type) {
	case *xc.ChainConfig:
		return txBuilder.NewNativeTransfer(args, input)
	case *xc.TokenAssetConfig:
		return txBuilder.NewTokenTransfer(args, input)
	default:
		return nil, fmt.Errorf("NewTransfer not implemented for %T", asset)
	}
}

// NewNativeTransfer creates a new transfer for a native asset
func (txBuilder TxBuilder) NewNativeTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	asset, _ := args.GetAsset()
	if asset == nil {
		asset = txBuilder.Chain
	}

	var local_input *tx_input.TxInput
	var ok bool
	if local_input, ok = (input.(*tx_input.TxInput)); !ok {
		return &tx.Tx{}, errors.New("xc.TxInput is not from a bitcoin chain")
	}
	// Only need to save min utxo for the transfer.
	totalSpend := local_input.SumUtxo()

	gasPrice := local_input.GasPricePerByte
	// 255 for bitcoin, 300 for bch
	estimatedTxBytesLength := xc.NewBigIntFromUint64(uint64(255 * len(local_input.UnspentOutputs)))
	if xc.NativeAsset(txBuilder.Chain.Chain) == xc.BCH {
		estimatedTxBytesLength = xc.NewBigIntFromUint64(uint64(300 * len(local_input.UnspentOutputs)))
	}
	fee := gasPrice.Mul(&estimatedTxBytesLength)

	amount := args.GetAmount()
	transferAmountAndFee := amount.Add(&fee)
	unspentAmountMinusTransferAndFee := totalSpend.Sub(&transferAmountAndFee)
	recipients := []tx.Recipient{
		{
			To:    args.GetTo(),
			Value: amount,
		},
		{
			To:    args.GetFrom(),
			Value: unspentAmountMinusTransferAndFee,
		},
	}

	msgTx := wire.NewMsgTx(TxVersion)

	for _, input := range local_input.UnspentOutputs {
		hash := chainhash.Hash{}
		copy(hash[:], input.Hash)
		msgTx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&hash, input.Index), nil, nil))
	}

	// Outputs
	for _, recipient := range recipients {
		addr, err := txBuilder.AddressDecoder.Decode(recipient.To, txBuilder.Params)
		if err != nil {
			return nil, err
		}
		script, err := txscript.PayToAddrScript(addr)
		if err != nil {
			logrus.WithError(err).WithField("to", recipient.To).Error("trying paytoaddr")
			return nil, err
		}
		value := recipient.Value.Int().Int64()
		if value < 0 {
			diff := local_input.SumUtxo().Sub(&amount)
			return nil, fmt.Errorf("not enough funds for fees, estimated fee is %s but only %s is left after transfer",
				fee.ToHuman(asset.GetDecimals()).String(), diff.ToHuman(asset.GetDecimals()).String(),
			)
		}
		msgTx.AddTxOut(wire.NewTxOut(value, script))
	}

	tx := tx.Tx{
		MsgTx: msgTx,

		From:   args.GetFrom(),
		To:     args.GetTo(),
		Amount: amount,
		Input:  local_input,

		Recipients: recipients,
	}
	return &tx, nil
}

// NewTokenTransfer creates a new transfer for a token asset
func (txBuilder TxBuilder) NewTokenTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	return nil, errors.New("not implemented")
}
