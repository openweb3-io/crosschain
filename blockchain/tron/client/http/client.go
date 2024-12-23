package http

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/openweb3-io/crosschain/blockchain/tron"
	httpclient "github.com/openweb3-io/crosschain/blockchain/tron/http_client"
	xcbuilder "github.com/openweb3-io/crosschain/builder"

	"github.com/btcsuite/btcutil/base58"
	xcclient "github.com/openweb3-io/crosschain/client"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/pkg/errors"
)

var _ xcclient.IClient = &Client{}

const TRANSFER_EVENT_HASH_HEX = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
const TX_TIMEOUT = 2 * time.Hour

type Client struct {
	cfg    *xc_types.ChainConfig
	client *httpclient.Client
}

func NewClient(
	cfg *xc_types.ChainConfig,
) (*Client, error) {
	endpoint := cfg.Client.URL

	client, err := httpclient.NewHttpClient(endpoint)
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg,
		client,
	}, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc_types.TxInput, error) {
	input := new(tron.TxInput)

	dummyTx, err := client.client.CreateTransaction(ctx, string(args.GetFrom()), string(args.GetTo()), 5)

	if err != nil {
		return nil, err
	}

	input.RefBlockBytes = dummyTx.RawData.RefBlockBytes
	input.RefBlockHash = dummyTx.RawData.RefBlockHashBytes
	// set timeout period
	input.Timestamp = time.Now().Unix()
	input.Expiration = time.Now().Add(TX_TIMEOUT).Unix()

	return input, nil
}

func (client *Client) FetchLegacyTxInput(ctx context.Context, from xc_types.Address, to xc_types.Address, asset xc_types.IAsset) (xc_types.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, xc_types.NewBigIntFromUint64(1), xcbuilder.WithAsset(asset))
	return client.FetchTransferInput(ctx, args)
}

func (client *Client) BroadcastTx(ctx context.Context, _tx xc_types.Tx) error {
	tx := _tx.(*tron.Tx)
	bz, err := tx.Serialize()
	if err != nil {
		return err
	}

	if _, err := client.client.BroadcastHex(ctx, hex.EncodeToString(bz)); err != nil {
		return err
	}

	return nil
}

func (client *Client) FetchLegacyTxInfo(ctx context.Context, txHash xc_types.TxHash) (*xc_types.LegacyTxInfo, error) {
	tx, err := client.client.GetTransactionByID(ctx, string(txHash))
	if err != nil {
		return nil, err
	}

	info, err := client.client.GetTransactionInfoByID(ctx, string(txHash))
	if err != nil {
		return nil, err
	}

	block, err := client.client.GetBlockByNum(ctx, info.BlockNumber)
	if err != nil {
		return nil, err
	}

	var from xc_types.Address
	var to xc_types.Address
	var amount xc_types.BigInt
	sources, destinations := deserialiseTransactionEvents(info.Logs)
	// If we cannot retrieve transaction events, we can infer that the TX is a native transfer
	if len(sources) == 0 && len(destinations) == 0 {
		from, to, amount, err = deserialiseNativeTransfer(tx)
		if err != nil {
			return nil, err
		}

		source := new(xc_types.LegacyTxInfoEndpoint)
		source.Address = from
		source.Amount = amount
		source.Asset = string(client.cfg.Chain)
		source.NativeAsset = client.cfg.Chain

		destination := new(xc_types.LegacyTxInfoEndpoint)
		destination.Address = to
		destination.Amount = amount
		destination.Asset = string(client.cfg.Chain)
		destination.NativeAsset = client.cfg.Chain

		sources = append(sources, source)
		destinations = append(destinations, destination)
	}

	txInfo := &xc_types.LegacyTxInfo{
		BlockHash:       block.BlockId,
		TxID:            string(txHash),
		ExplorerURL:     client.cfg.ExplorerURL + fmt.Sprintf("/transaction/%s", string(txHash)),
		From:            from,
		To:              to,
		ContractAddress: xc_types.ContractAddress(info.ContractAddress),
		Amount:          amount,
		Fee:             xc_types.NewBigIntFromUint64(uint64(info.Fee)),
		BlockIndex:      int64(info.BlockNumber),
		BlockTime:       int64(info.BlockTimeStamp / 1000),
		Confirmations:   0,
		Status:          xc_types.TxStatusSuccess,
		Sources:         sources,
		Destinations:    destinations,
		Time:            int64(info.BlockTimeStamp),
		TimeReceived:    0,
		Error:           "",
	}

	return txInfo, nil
}

func (client *Client) FetchTxInfo(ctx context.Context, txHashStr xc_types.TxHash) (*xcclient.TxInfo, error) {
	legacyTx, err := client.FetchLegacyTxInfo(ctx, txHashStr)
	if err != nil {
		return nil, err
	}

	// remap to new tx
	return xcclient.TxInfoFromLegacy(client.cfg.Chain, legacyTx, xcclient.Account), nil
}

func (a *Client) FetchBalance(ctx context.Context, address xc_types.Address) (*xc_types.BigInt, error) {
	account, err := a.client.GetAccount(ctx, string(address))
	if err != nil {
		return nil, err
	}
	balance := xc_types.NewBigIntFromUint64(account.Balance)
	return &balance, nil
}

func (client *Client) FetchBalanceForAsset(ctx context.Context, address xc_types.Address, contractAddress xc_types.ContractAddress) (*xc_types.BigInt, error) {
	a, err := client.client.ReadTrc20Balance(ctx, string(address), string(contractAddress))
	if err != nil {
		return nil, err
	}

	return (*xc_types.BigInt)(a), nil
}

func (client *Client) EstimateGasFee(ctx context.Context, tx xc_types.Tx) (amount *xc_types.BigInt, err error) {
	_tx := tx.(*tron.Tx)

	txRawData, err := tx.Serialize()
	if err != nil {
		return nil, err
	}
	txSize := int64(len(txRawData) - 1) // actual tx size is less than serialized size by 1 byte

	// signatures also consume bandwidth, so we need to add them
	signatures := len(tx.GetSignatures())
	if signatures == 0 {
		return nil, errors.New("transaction has no signatures")
	}
	signaturesSize := int64(signatures * 65) // every signature is 65 bytes
	totalSize := txSize + signaturesSize
	bandwidthUsage := xc_types.NewBigIntFromInt64(totalSize)

	asset, _ := _tx.Args.GetAsset()

	params, err := client.client.GetChainParameters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get chain params")
	}

	var transactionFee xc_types.BigInt
	var energyFee xc_types.BigInt

	for _, v := range params.ChainParameter {
		if v.Key == "getTransactionFee" {
			transactionFee = xc_types.NewBigIntFromInt64(v.Value)
		} else if v.Key == "getEnergyFee" {
			energyFee = xc_types.NewBigIntFromInt64(v.Value)
		}
	}

	// TODO:
	// consider using wallet/getaccountresource to get the current free bandwidth and energy balance of the from account,
	// so we can get more accurate fee, but it will increase the number of API calls

	if asset == nil || asset.GetContract() == "" {
		totalCost := transactionFee.Mul(&bandwidthUsage)
		return &totalCost, nil
	} else {
		params := []map[string]any{
			{
				"address": _tx.Args.GetTo(),
			},
			{
				"uint256": _tx.Args.GetAmount().String(),
			},
		}
		b, _ := json.Marshal(params)

		estimate, err := client.client.EstimateEnergy(
			context.Background(),
			string(_tx.Args.GetFrom()),
			string(asset.GetContract()),
			"transfer(address,uint256)",
			string(b),
			0,
		)
		if err != nil {
			return nil, err
		}

		energyUsage := xc_types.NewBigIntFromInt64(estimate.EnergyRequired)
		energyCost := energyFee.Mul(&energyUsage)
		bandwidthCost := transactionFee.Mul(&bandwidthUsage)
		totalCost := bandwidthCost.Add(&energyCost)
		return &totalCost, nil
	}
}

func deserialiseTransactionEvents(log []*httpclient.Log) ([]*xc_types.LegacyTxInfoEndpoint, []*xc_types.LegacyTxInfoEndpoint) {
	sources := make([]*xc_types.LegacyTxInfoEndpoint, 0)
	destinations := make([]*xc_types.LegacyTxInfoEndpoint, 0)

	for _, event := range log {
		source := new(xc_types.LegacyTxInfoEndpoint)
		destination := new(xc_types.LegacyTxInfoEndpoint)
		source.NativeAsset = xc_types.TRX
		destination.NativeAsset = xc_types.TRX

		// The addresses in the TVM omits the prefix 0x41, so we add it here to allow us to parse the addresses
		eventContractB58 := base58.CheckEncode(event.Address, 0x41)
		eventSourceB58 := base58.CheckEncode(event.Topics[1][12:], 0x41)      // Remove padding
		eventDestinationB58 := base58.CheckEncode(event.Topics[2][12:], 0x41) // Remove padding
		eventMethodBz := event.Topics[0]

		eventValue := new(big.Int)
		eventValue.SetString(hex.EncodeToString(event.Data), 16) // event value is returned as a padded big int hex

		if hex.EncodeToString(eventMethodBz) != strings.TrimPrefix(TRANSFER_EVENT_HASH_HEX, "0x") {
			continue
		}

		source.ContractAddress = xc_types.ContractAddress(eventContractB58)
		destination.ContractAddress = xc_types.ContractAddress(eventContractB58)

		source.Address = xc_types.Address(eventSourceB58)
		source.Amount = xc_types.NewBigIntFromUint64(eventValue.Uint64())
		destination.Address = xc_types.Address(eventDestinationB58)
		destination.Amount = xc_types.NewBigIntFromUint64(eventValue.Uint64())

		sources = append(sources, source)
		destinations = append(destinations, destination)
	}

	return sources, destinations
}

func deserialiseNativeTransfer(tx *httpclient.GetTransactionIDResponse) (xc_types.Address, xc_types.Address, xc_types.BigInt, error) {
	if len(tx.RawData.Contract) != 1 {
		return "", "", xc_types.BigInt{}, fmt.Errorf("unsupported transaction")
	}

	contract := tx.RawData.Contract[0]

	if contract.Type != "TransferContract" {
		return "", "", xc_types.BigInt{}, fmt.Errorf("unsupported transaction")
	}
	transferContract, err := contract.AsTransferContract()
	if err != nil {
		return "", "", xc_types.BigInt{}, fmt.Errorf("invalid transfer-contract: %v", err)
	}

	from := xc_types.Address(transferContract.Owner)
	to := xc_types.Address(transferContract.To)
	amount := transferContract.Amount

	return from, to, xc_types.NewBigIntFromUint64(uint64(amount)), nil
}
