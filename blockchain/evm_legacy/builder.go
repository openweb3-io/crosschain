package evm_legacy

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	evmaddress "github.com/openweb3-io/crosschain/blockchain/evm/address"
	evmbuilder "github.com/openweb3-io/crosschain/blockchain/evm/builder"
	evminput "github.com/openweb3-io/crosschain/blockchain/evm/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
)

var DefaultMaxTipCapGwei uint64 = 5

// TxBuilder for EVM
type TxBuilder evmbuilder.TxBuilder

var _ xcbuilder.TxBuilder = &TxBuilder{}
var _ xcbuilder.TxTokenBuilder = &TxBuilder{}
var _ xcbuilder.TxXTransferBuilder = &TxBuilder{}

// NewTxBuilder creates a new EVM TxBuilder
func NewTxBuilder(cfg *xc.ChainConfig) (TxBuilder, error) {
	builder, err := evmbuilder.NewTxBuilder(cfg)
	if err != nil {
		return TxBuilder{}, err
	}
	builder = builder.WithTxBuilder(&LegacyEvmTxBuilder{})

	return TxBuilder(builder), nil
}

// supports evm before london merge
type LegacyEvmTxBuilder struct {
}

var _ evmbuilder.GethTxBuilder = &LegacyEvmTxBuilder{}

func parseInput(input xc.TxInput) (*TxInput, error) {
	switch input := input.(type) {
	case *TxInput:
		return input, nil
	case *evminput.TxInput:
		return (*TxInput)(input), nil
	default:
		return nil, fmt.Errorf("invalid input type %T", input)
	}
}

func (*LegacyEvmTxBuilder) BuildTxWithPayload(chain *xc.ChainConfig, to xc.Address, value xc.BigInt, data []byte, inputRaw xc.TxInput) (xc.Tx, error) {
	address, err := evmaddress.FromHex(to)
	if err != nil {
		return nil, err
	}
	chainID := new(big.Int).SetInt64(chain.ChainID)
	input, err := parseInput(inputRaw)
	if err != nil {
		return nil, err
	}
	// Protection from setting very high gas tip
	// TODO

	return &Tx{
		EthTx: types.NewTransaction(
			input.Nonce,
			address,
			value.Int(),
			input.GasLimit,
			input.GasPrice.Int(),
			data,
		),
		Signer: types.LatestSignerForChainID(chainID),
	}, nil
}

func (txBuilder TxBuilder) NewTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	// type cast back to evm input, which is expected by the evm builder
	inputEvm := (*evminput.TxInput)(input.(*TxInput))
	return evmbuilder.TxBuilder(txBuilder).NewTransfer(args, inputEvm)
}

func (txBuilder TxBuilder) NewNativeTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	inputEvm := (*evminput.TxInput)(input.(*TxInput))
	return evmbuilder.TxBuilder(txBuilder).NewNativeTransfer(args, inputEvm)
}

func (txBuilder TxBuilder) NewTokenTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	inputEvm := (*evminput.TxInput)(input.(*TxInput))
	return evmbuilder.TxBuilder(txBuilder).NewTokenTransfer(args, inputEvm)
}

func (txBuilder TxBuilder) NewTask(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	inputEvm := (*evminput.TxInput)(input.(*TxInput))
	return evmbuilder.TxBuilder(txBuilder).NewTask(args, inputEvm)
}
