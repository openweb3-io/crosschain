package http

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/openweb3-io/crosschain/blockchain/tron/tx_input"
	"google.golang.org/protobuf/types/known/anypb"

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
	input := new(tx_input.TxInput)

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
	sources, destinations := deserializeTransactionEvents(info.Logs)
	// If we cannot retrieve transaction events, we can infer that the TX is a native transfer
	if len(sources) == 0 && len(destinations) == 0 {
		from, to, amount, err = deserializeNativeTransfer(tx)
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
		Fee:             xc_types.NewBigIntFromUint64(info.Fee),
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
	balance := xc_types.NewBigIntFromInt64(account.Balance)
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
	var createAccountFee xc_types.BigInt

	for _, v := range params.ChainParameter {
		switch v.Key {
		case "getTransactionFee":
			transactionFee = xc_types.NewBigIntFromInt64(v.Value)
		case "getEnergyFee":
			energyFee = xc_types.NewBigIntFromInt64(v.Value)
		case "getCreateAccountFee":
			createAccountFee = xc_types.NewBigIntFromInt64(v.Value)
			// TODO: parameter returns 0.1 trx, not correctly 1 trx, fix it in future
			createAccountFee = xc_types.NewBigIntFromInt64(1000000)
		}
	}

	newAccount, err := isNewAccount(ctx, client, _tx.Args.GetTo())
	if err != nil {
		return nil, err
	}

	accountResource, err := client.client.GetAccountResource(ctx, string(_tx.Args.GetFrom()))
	if err != nil {
		return nil, err
	}
	accountAvailableBandwidth := accountResource.FreeNetLimit - accountResource.FreeNetUsed
	accountAvailableBandwidth += accountResource.NetLimit - accountResource.NetUsed
	availableBandwidth := xc_types.NewBigIntFromInt64(accountAvailableBandwidth)

	actualBandwidthUsage := bandwidthUsage.Sub(&availableBandwidth)
	zero := xc_types.NewBigIntFromInt64(0)
	if actualBandwidthUsage.Cmp(&zero) < 0 {
		actualBandwidthUsage = xc_types.NewBigIntFromInt64(0)
	}

	if asset == nil || asset.GetContract() == "" {
		totalCost := transactionFee.Mul(&actualBandwidthUsage)
		if newAccount {
			totalCost = totalCost.Add(&createAccountFee)
		}
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
		if newAccount {
			newAccountFee := xc_types.NewBigIntFromInt64(25000)
			energyUsage = energyUsage.Add(&newAccountFee)
		}

		accountAvailableEnergy := accountResource.EnergyLimit - accountResource.EnergyUsed
		availableEnergy := xc_types.NewBigIntFromInt64(accountAvailableEnergy)
		actualEnergyUsage := energyUsage.Sub(&availableEnergy)
		zero := xc_types.NewBigIntFromInt64(0)
		if actualEnergyUsage.Cmp(&zero) < 0 {
			actualEnergyUsage = xc_types.NewBigIntFromInt64(0)
		}

		energyCost := energyFee.Mul(&actualEnergyUsage)
		bandwidthCost := transactionFee.Mul(&actualBandwidthUsage)
		totalCost := bandwidthCost.Add(&energyCost)
		return &totalCost, nil
	}
}

func (client *Client) FetchStakeInput(ctx context.Context, address xc_types.Address, resource tx_input.Resource, amount xc_types.BigInt) (xc_types.StakeTxInput, error) {
	input := new(tx_input.StakingInput)

	dummyTx, err := client.client.FreezeBalanceV2(ctx, string(address), httpclient.Resource(resource), amount.Int())

	if err != nil {
		return nil, err
	}

	input.RefBlockBytes = dummyTx.RawData.RefBlockBytes
	input.RefBlockHash = dummyTx.RawData.RefBlockHashBytes
	// set timeout period
	input.Timestamp = time.Now().Unix()
	input.Expiration = time.Now().Add(TX_TIMEOUT).Unix()

	input.Resource = resource

	return input, nil
}

func (client *Client) FetchUnstakeInput(ctx context.Context, address xc_types.Address, resource tx_input.Resource, amount xc_types.BigInt) (xc_types.UnstakeTxInput, error) {
	input := new(tx_input.UnstakingInput)

	dummyTx, err := client.client.UnfreezeBalanceV2(ctx, string(address), httpclient.Resource(resource), amount.Int())

	if err != nil {
		return nil, err
	}

	input.RefBlockBytes = dummyTx.RawData.RefBlockBytes
	input.RefBlockHash = dummyTx.RawData.RefBlockHashBytes
	// set timeout period
	input.Timestamp = time.Now().Unix()
	input.Expiration = time.Now().Add(TX_TIMEOUT).Unix()

	input.Resource = resource

	return input, nil
}

func (client *Client) FetchWithdrawInput(ctx context.Context, ownerAddress xc_types.Address) (xc_types.WithdrawTxInput, error) {
	input := new(tx_input.WithdrawInput)

	dummyTx, err := client.client.WithdrawExpireUnfreeze(ctx, string(ownerAddress))

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

// FetchDelegatingTx and FetchUnDelegatingTx
// Currently lacks an abstraction layer for delegating resources,
// so return the transaction object without going through the Input + Builder process
func (client *Client) FetchDelegatingTx(ctx context.Context, ownerAddress, receiverAddress xc_types.Address, resource tx_input.Resource, amount xc_types.BigInt) (xc_types.Tx, error) {
	dummyTx, err := client.client.DelegateResource(ctx, string(ownerAddress), string(receiverAddress), httpclient.Resource(resource), amount.Int())
	if err != nil {
		return nil, err
	}

	ownerAddressBytes, err := common.DecodeCheck(string(ownerAddress))
	if err != nil {
		return nil, err
	}

	receiverAddressBytes, err := common.DecodeCheck(string(receiverAddress))
	if err != nil {
		return nil, err
	}

	dummyParams := dummyTx.RawData.Contract[0].Parameter.Value

	params := &core.DelegateResourceContract{
		OwnerAddress:    ownerAddressBytes,
		ReceiverAddress: receiverAddressBytes,
		Balance:         dummyParams.Balance,
	}

	if dummyParams.Resource == httpclient.ResourceEnergy {
		params.Resource = core.ResourceCode_ENERGY
	} else {
		params.Resource = core.ResourceCode_BANDWIDTH
	}

	if dummyParams.Lock != nil {
		params.Lock = *dummyParams.Lock
	}

	if dummyParams.LockPeriod != nil {
		params.LockPeriod = *dummyParams.LockPeriod
	}

	contract := &core.Transaction_Contract{}
	contract.Type = core.Transaction_Contract_DelegateResourceContract
	param, err := anypb.New(params)
	if err != nil {
		return nil, err
	}
	contract.Parameter = param

	tx := &core.Transaction{}
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: dummyTx.RawData.RefBlockBytes,
		RefBlockHash:  dummyTx.RawData.RefBlockHashBytes,
		Expiration:    int64(dummyTx.RawData.Expiration),
		Timestamp:     int64(dummyTx.RawData.Timestamp),
	}

	return &tron.Tx{
		TronTx: tx,
	}, nil
}

func (client *Client) FetchUnDelegatingTx(ctx context.Context, ownerAddress, receiverAddress xc_types.Address, resource tx_input.Resource, amount xc_types.BigInt) (xc_types.Tx, error) {
	dummyTx, err := client.client.UnDelegateResource(ctx, string(ownerAddress), string(receiverAddress), httpclient.Resource(resource), amount.Int())
	if err != nil {
		return nil, err
	}

	ownerAddressBytes, err := common.DecodeCheck(string(ownerAddress))
	if err != nil {
		return nil, err
	}

	receiverAddressBytes, err := common.DecodeCheck(string(receiverAddress))
	if err != nil {
		return nil, err
	}

	dummyParams := dummyTx.RawData.Contract[0].Parameter.Value

	params := &core.UnDelegateResourceContract{
		OwnerAddress:    ownerAddressBytes,
		ReceiverAddress: receiverAddressBytes,
		Balance:         dummyParams.Balance,
	}

	if dummyParams.Resource == httpclient.ResourceEnergy {
		params.Resource = core.ResourceCode_ENERGY
	} else {
		params.Resource = core.ResourceCode_BANDWIDTH
	}

	contract := &core.Transaction_Contract{}
	contract.Type = core.Transaction_Contract_UnDelegateResourceContract
	param, err := anypb.New(params)
	if err != nil {
		return nil, err
	}
	contract.Parameter = param

	tx := &core.Transaction{}
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: dummyTx.RawData.RefBlockBytes,
		RefBlockHash:  dummyTx.RawData.RefBlockHashBytes,
		Expiration:    int64(dummyTx.RawData.Expiration),
		Timestamp:     int64(dummyTx.RawData.Timestamp),
	}

	return &tron.Tx{
		TronTx: tx,
	}, nil
}

func (client *Client) EstimateGasFeeIncludeResourceUsage(ctx context.Context, tx xc_types.Tx) (amount *xc_types.BigInt, err error) {
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
	var createAccountFee xc_types.BigInt

	for _, v := range params.ChainParameter {
		switch v.Key {
		case "getTransactionFee":
			transactionFee = xc_types.NewBigIntFromInt64(v.Value)
		case "getEnergyFee":
			energyFee = xc_types.NewBigIntFromInt64(v.Value)
		case "getCreateAccountFee":
			createAccountFee = xc_types.NewBigIntFromInt64(v.Value)
			// TODO: parameter returns 0.1 trx, not correctly 1 trx, fix it in future
			createAccountFee = xc_types.NewBigIntFromInt64(1000000)
		}
	}

	newAccount, err := isNewAccount(ctx, client, _tx.Args.GetTo())
	if err != nil {
		return nil, err
	}

	if asset == nil || asset.GetContract() == "" {
		totalCost := transactionFee.Mul(&bandwidthUsage)
		if newAccount {
			totalCost = totalCost.Add(&createAccountFee)
		}
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
		if newAccount {
			newAccountFee := xc_types.NewBigIntFromInt64(25000)
			energyUsage = energyUsage.Add(&newAccountFee)
		}

		energyCost := energyFee.Mul(&energyUsage)
		bandwidthCost := transactionFee.Mul(&bandwidthUsage)
		totalCost := bandwidthCost.Add(&energyCost)
		return &totalCost, nil
	}
}

func isNewAccount(ctx context.Context, client *Client, address xc_types.Address) (bool, error) {
	newAccount := false
	_, err := client.client.GetAccount(ctx, string(address))
	if err != nil {
		if strings.Contains(err.Error(), "could not find account") {
			newAccount = true
		} else {
			return false, err
		}
	}
	return newAccount, nil
}

func deserializeTransactionEvents(log []*httpclient.Log) ([]*xc_types.LegacyTxInfoEndpoint, []*xc_types.LegacyTxInfoEndpoint) {
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

func deserializeNativeTransfer(tx *httpclient.GetTransactionIDResponse) (xc_types.Address, xc_types.Address, xc_types.BigInt, error) {
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

	return from, to, xc_types.NewBigIntFromUint64(amount), nil
}
