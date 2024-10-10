package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	xcbuilder "github.com/openweb3-io/crosschain/builder"

	"github.com/btcsuite/btcutil/base58"
	tronClient "github.com/fbsobreira/gotron-sdk/pkg/client"
	tronApi "github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	xcclient "github.com/openweb3-io/crosschain/client"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

var _ xcclient.IClient = &Client{}

const TRANSFER_EVENT_HASH_HEX = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
const TX_TIMEOUT = 2 * time.Hour

type Client struct {
	cfg    *xc_types.ChainConfig
	client *tronClient.GrpcClient
}

func NewClient(
	cfg *xc_types.ChainConfig,
) (*Client, error) {
	endpoint := cfg.URL

	if endpoint == "" {
		endpoint = "grpc.trongrid.io:50051"
	}

	client := tronClient.NewGrpcClient(endpoint)

	if err := client.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		log.Fatalf("error dial rpc, err %v", err)
	}

	return &Client{
		cfg,
		client,
	}, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc_types.TxInput, error) {
	input := new(TxInput)

	asset, _ := args.GetAsset()
	var err error
	var tx *tronApi.TransactionExtention
	if asset != nil {
		tx, err = client.client.TRC20Send(string(args.GetFrom()), string(args.GetTo()), string(asset.GetContract()), args.GetAmount().Int(), 0)
	} else {
		tx, err = client.client.Transfer(string(args.GetFrom()), string(args.GetTo()), args.GetAmount().Int().Int64())
	}

	if err != nil {
		return nil, err
	}
	dummyTx := tx.Transaction

	input.RefBlockBytes = dummyTx.RawData.RefBlockBytes
	input.RefBlockHash = dummyTx.RawData.RefBlockHash
	// set timeout period
	input.Timestamp = time.Now().Unix()
	input.Expiration = time.Now().Add(TX_TIMEOUT).Unix()

	return input, nil
}

func (a *Client) FetchBalance(ctx context.Context, address xc_types.Address) (*xc_types.BigInt, error) {
	account, err := a.client.GetAccount(string(address))
	if err != nil {
		return nil, err
	}
	balance := xc_types.NewBigIntFromInt64(account.Balance)
	return &balance, nil
}

func (a *Client) FetchBalanceForAsset(ctx context.Context, address xc_types.Address, contractAddress xc_types.ContractAddress) (*xc_types.BigInt, error) {
	balance, err := a.client.TRC20ContractBalance(string(address), string(contractAddress))
	if err != nil {
		return nil, err
	}
	return (*xc_types.BigInt)(balance), nil
}

func (a *Client) EstimateGasFee(ctx context.Context, tx xc_types.Tx) (amount *xc_types.BigInt, err error) {
	_tx := tx.(*Tx)

	bandwithUsage := xc_types.NewBigIntFromInt64(200)
	/*
		if txInput.Gas != nil {
			bandwithUsage = *txInput.Gas
		}
	*/

	params, err := a.client.Client.GetChainParameters(ctx, &tronApi.EmptyMessage{})
	if err != nil {
		return nil, errors.Wrap(err, "get chain params")
	}

	var transactionFee xc_types.BigInt
	var energyFee xc_types.BigInt

	for _, v := range params.ChainParameter {
		if v.Key == "getTransactionFee" {
			transactionFee = xc_types.NewBigIntFromInt64(v.Value)
		}
		if v.Key == "getEnergyFee" {
			energyFee = xc_types.NewBigIntFromInt64(v.Value)
		}
	}

	asset, _ := _tx.args.GetAsset()
	if asset == nil || asset.GetContract() == "" {
		//普通trx转账只需要带宽
		totalCost := (&transactionFee).Mul(&bandwithUsage)
		return &totalCost, nil
	} else {
		estimate, err := a.client.EstimateEnergy(
			string(_tx.args.GetFrom()),
			string(asset.GetContract()),
			"transfer(address,uint256)",
			fmt.Sprintf(`[{"address": "%s"},{"uint256": "%v"}]`, _tx.args.GetTo(), _tx.args.GetAmount()),
			0, "", 0,
		)
		if err != nil {
			return nil, err
		}

		energyUsage := xc_types.NewBigIntFromInt64(estimate.EnergyRequired)
		bandwidthCost := transactionFee.Mul(&bandwithUsage)
		energyCost := energyFee.Add(&energyUsage)
		totalCost := bandwidthCost.Add(&energyCost)

		return &totalCost, nil
	}

}

func (a *Client) BroadcastTx(ctx context.Context, _tx xc_types.Tx) error {
	tx := _tx.(*Tx)
	if _, err := a.client.Broadcast(tx.tronTx); err != nil {
		return err
	}

	return nil
}

func (client *Client) FetchLegacyTxInfo(ctx context.Context, txHash xc_types.TxHash) (*xc_types.LegacyTxInfo, error) {
	tx, err := client.client.GetTransactionByID(string(txHash))
	if err != nil {
		return nil, err
	}

	info, err := client.client.GetTransactionInfoByID(string(txHash))
	if err != nil {
		return nil, err
	}

	block, err := client.client.GetBlockByNum(info.BlockNumber)
	if err != nil {
		return nil, err
	}

	var from xc_types.Address
	var to xc_types.Address
	var amount xc_types.BigInt
	sources, destinations := deserialiseTransactionEvents(info.Log)
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

	return &xc_types.LegacyTxInfo{
		BlockHash:       string(block.Blockid),
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
	}, nil
}

func deserialiseTransactionEvents(log []*core.TransactionInfo_Log) ([]*xc_types.LegacyTxInfoEndpoint, []*xc_types.LegacyTxInfoEndpoint) {
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

func deserialiseNativeTransfer(tx *core.Transaction) (xc_types.Address, xc_types.Address, xc_types.BigInt, error) {
	if len(tx.RawData.Contract) != 1 {
		return "", "", xc_types.BigInt{}, fmt.Errorf("unsupported transaction")
	}

	contract := tx.RawData.Contract[0]

	if contract.Type != core.Transaction_Contract_TransferContract {
		return "", "", xc_types.BigInt{}, fmt.Errorf("unsupported transaction")
	}

	transferContract := &core.TransferContract{}
	err := proto.Unmarshal(contract.Parameter.Value, transferContract)
	if err != nil {
		return "", "", xc_types.BigInt{}, fmt.Errorf("invalid transfer-contract: %v", err)
	}

	from := xc_types.Address(transferContract.OwnerAddress)
	to := xc_types.Address(transferContract.ToAddress)
	amount := transferContract.Amount

	return from, to, xc_types.NewBigIntFromUint64(uint64(amount)), nil
}

func (client *Client) FetchLegacyTxInput(ctx context.Context, from xc_types.Address, to xc_types.Address, asset xc_types.IAsset) (xc_types.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, xc_types.NewBigIntFromUint64(1), xcbuilder.WithAsset(asset))
	return client.FetchTransferInput(ctx, args)
}
