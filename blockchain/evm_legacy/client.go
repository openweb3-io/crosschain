package evm_legacy

import (
	"context"
	"fmt"

	evmclient "github.com/openweb3-io/crosschain/blockchain/evm/client"
	"github.com/openweb3-io/crosschain/blockchain/evm/tx"
	evminput "github.com/openweb3-io/crosschain/blockchain/evm/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xclient "github.com/openweb3-io/crosschain/client"
	"github.com/openweb3-io/crosschain/factory/blockchains/registry"
	xc "github.com/openweb3-io/crosschain/types"
)

type Client struct {
	evmClient *evmclient.Client
}

var _ xclient.IClient = &Client{}

type TxInput evminput.TxInput

var _ xc.TxInput = &TxInput{}

func init() {
	registry.RegisterTxBaseInput(&TxInput{})
}

func NewTxInput() *TxInput {
	return &TxInput{}
}

func (input *TxInput) GetBlockchain() xc.Blockchain {
	return xc.BlockchainEVMLegacy
}

func (input *TxInput) SetGasFeePriority(other xc.GasFeePriority) error {
	return ((*evminput.TxInput)(input)).SetGasFeePriority(other)
}
func (input *TxInput) IndependentOf(other xc.TxInput) (independent bool) {
	return ((*evminput.TxInput)(input)).IndependentOf(other)
}
func (input *TxInput) SafeFromDoubleSend(other ...xc.TxInput) (independent bool) {
	return ((*evminput.TxInput)(input)).SafeFromDoubleSend(other...)
}

func NewClient(cfg *xc.ChainConfig) (*Client, error) {
	evmClient, err := evmclient.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		evmClient: evmClient,
	}, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc.TxInput, error) {
	asset, _ := args.GetAsset()

	nativeAsset := client.evmClient.Chain
	zero := xc.NewBigIntFromUint64(0)
	result := NewTxInput()
	result.GasPrice = zero

	// Nonce
	nonce, err := client.evmClient.GetNonce(ctx, args.GetFrom())
	if err != nil {
		return result, err
	}
	result.Nonce = nonce

	if nativeAsset.NoGasFees {
		result.GasPrice = zero
	} else {
		// legacy gas fees
		baseFee, err := client.evmClient.EthClient.SuggestGasPrice(ctx)
		if err != nil {
			return result, err
		}
		result.GasPrice = xc.BigInt(*baseFee).ApplyGasPriceMultiplier(nativeAsset)
	}
	builder, err := NewTxBuilder(client.evmClient.Chain)
	if err != nil {
		return nil, fmt.Errorf("could not prepare to simulate legacy: %v", err)
	}
	tf, err := builder.NewTransfer(args, result)
	if err != nil {
		return nil, fmt.Errorf("could not prepare to simulate legacy: %v", err)
	}
	gasLimit, err := client.evmClient.SimulateGasWithLimit(ctx, args.GetFrom(), tf.(*tx.Tx), asset)
	if err != nil {
		return nil, err
	}
	result.GasLimit = gasLimit

	return result, nil
}

func (client *Client) FetchLegacyTxInput(ctx context.Context, from xc.Address, to xc.Address, asset xc.IAsset) (xc.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, xc.NewBigIntFromUint64(1), xcbuilder.WithAsset(asset))
	return client.FetchTransferInput(ctx, args)
}

func (client *Client) BroadcastTx(ctx context.Context, txInput xc.Tx) error {
	return client.evmClient.BroadcastTx(ctx, txInput)
}

func (client *Client) FetchLegacyTxInfo(ctx context.Context, txHash xc.TxHash) (*xc.LegacyTxInfo, error) {
	return client.evmClient.FetchLegacyTxInfo(ctx, txHash)
}

func (client *Client) FetchTxInfo(ctx context.Context, txHash xc.TxHash) (xclient.TxInfo, error) {
	return client.evmClient.FetchTxInfo(ctx, txHash)
}

func (client *Client) FetchNativeBalance(ctx context.Context, address xc.Address) (*xc.BigInt, error) {
	return client.evmClient.FetchNativeBalance(ctx, address)
}

func (client *Client) FetchBalance(ctx context.Context, address xc.Address) (*xc.BigInt, error) {
	return client.evmClient.FetchBalance(ctx, address)
}

func (client *Client) FetchBalanceForAsset(ctx context.Context, address xc.Address, contractAddress xc.ContractAddress) (*xc.BigInt, error) {
	return client.evmClient.FetchBalanceForAsset(ctx, address, contractAddress)
}

func (client *Client) EstimateGas(ctx context.Context, tx xc.Tx) (*xc.BigInt, error) {
	return client.evmClient.EstimateGas(ctx, tx)
}
