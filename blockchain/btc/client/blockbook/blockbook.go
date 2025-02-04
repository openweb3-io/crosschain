package blockbook

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/openweb3-io/crosschain/blockchain/btc/address"
	"github.com/openweb3-io/crosschain/blockchain/btc/params"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"

	xclient "github.com/openweb3-io/crosschain/client"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type BlockbookClient struct {
	httpClient http.Client
	cfg        *xc.ChainConfig
	Chaincfg   *chaincfg.Params
	Url        string
	decoder    address.AddressDecoder
}

var _ xclient.IClient = &BlockbookClient{}
var _ address.WithAddressDecoder = &BlockbookClient{}

func NewClient(cfg *xc.ChainConfig) (*BlockbookClient, error) {
	httpClient := http.Client{}
	chaincfg, err := params.GetParams(cfg)
	if err != nil {
		return &BlockbookClient{}, err
	}
	url := cfg.Client.URL
	url = strings.TrimSuffix(url, "/")
	decoder := address.NewAddressDecoder()

	return &BlockbookClient{
		httpClient,
		cfg,
		chaincfg,
		url,
		decoder,
	}, nil
}

func (client *BlockbookClient) LatestBlock(ctx context.Context) (uint64, error) {
	var stats StatsResponse

	err := client.get(ctx, "/api/v2", &stats)
	if err != nil {
		return 0, err
	}

	return uint64(stats.Backend.Blocks), nil
}

func (client *BlockbookClient) BroadcastTx(ctx context.Context, tx xc.Tx) error {
	serial, err := tx.Serialize()
	if err != nil {
		return fmt.Errorf("bad tx: %v", err)
	}

	postData := hex.EncodeToString(serial)
	err = client.post(ctx, "/api/v2/sendtx/", "text/plain", []byte(postData), nil)
	if err != nil {
		return err
	}

	return nil
}

func (txBuilder *BlockbookClient) WithAddressDecoder(decoder address.AddressDecoder) address.WithAddressDecoder {
	txBuilder.decoder = decoder
	return txBuilder
}

const BitcoinCashPrefix = "bitcoincash:"

func (client *BlockbookClient) UnspentOutputs(ctx context.Context, addr xc.Address) ([]tx_input.Output, error) {
	var data UtxoResponse
	var formattedAddr string = string(addr)
	if client.cfg.Chain == xc.BCH {
		if !strings.HasPrefix(string(addr), BitcoinCashPrefix) {
			formattedAddr = fmt.Sprintf("%s%s", BitcoinCashPrefix, addr)
		}
	}

	err := client.get(ctx, fmt.Sprintf("api/v2/utxo/%s", formattedAddr), &data)
	if err != nil {
		return nil, err
	}

	// TODO try filtering using confirmed UTXO only for target amount, using heuristic as fallback.
	data = tx_input.FilterUnconfirmedHeuristic(data)
	btcAddr, err := client.decoder.Decode(addr, client.Chaincfg)
	if err != nil {
		return nil, err
	}
	script, err := txscript.PayToAddrScript(btcAddr)
	if err != nil {
		return nil, err
	}

	outputs := tx_input.NewOutputs(data, script)

	return outputs, nil
}

func (client *BlockbookClient) EstimateFee(ctx context.Context) (xc.BigInt, error) {
	var data EstimateFeeResponse
	// fee estimate for last N blocks
	blocks := 6
	err := client.get(ctx, fmt.Sprintf("/api/v2/estimatefee/%d", blocks), &data)
	if err != nil {
		return xc.BigInt{}, err
	}

	btcPerKb, err := decimal.NewFromString(data.Result)
	if err != nil {
		return xc.BigInt{}, err
	}
	// convert to BTC/byte
	BtcPerB := btcPerKb.Div(decimal.NewFromInt(1000))
	// convert to sats/byte
	satsPerB := xc.AmountHumanReadable(BtcPerB).ToBlockchain(client.cfg.GetDecimals())

	satsPerByte := tx_input.LegacyFeeFilter(client.cfg, satsPerB.Uint64(), client.cfg.ChainGasMultiplier, client.cfg.ChainMaxGasPrice)

	return xc.NewBigIntFromUint64(satsPerByte), nil
}

func (client *BlockbookClient) FetchLegacyTxInfo(ctx context.Context, txHash xc.TxHash) (*xc.LegacyTxInfo, error) {
	var data TransactionResponse
	txWithInfo := &xc.LegacyTxInfo{
		Amount: xc.NewBigIntFromUint64(0), // prevent nil pointer exception
		Fee:    xc.NewBigIntFromUint64(0),
	}

	expectedTo := ""

	err := client.get(ctx, "/api/v2/tx/"+string(txHash), &data)
	if err != nil {
		return txWithInfo, err
	}
	latestBlock, err := client.LatestBlock(ctx)
	if err != nil {
		return txWithInfo, err
	}

	txWithInfo.Fee = xc.NewBigIntFromStr(data.Fees)
	timestamp := time.Unix(data.BlockTime, 0)
	if data.BlockHeight > 0 {
		txWithInfo.BlockTime = timestamp.Unix()
		txWithInfo.BlockIndex = int64(data.BlockHeight)
		txWithInfo.BlockHash = data.BlockHash
		txWithInfo.Confirmations = int64(latestBlock) - int64(data.BlockHeight) + 1
		txWithInfo.Status = xc.TxStatusSuccess
	}
	txWithInfo.TxID = string(txHash)

	sources := []*xc.LegacyTxInfoEndpoint{}
	destinations := []*xc.LegacyTxInfoEndpoint{}

	// build Tx
	txObject := &tx.Tx{
		Input:      tx_input.NewTxInput(),
		Recipients: []tx.Recipient{},
		MsgTx:      &wire.MsgTx{},
		Signed:     true,
	}
	inputs := []tx.Input{}
	// btc chains the native asset and asset are the same
	asset := client.cfg.Chain

	for _, in := range data.Vin {
		hash, _ := hex.DecodeString(in.TxID)
		// sigScript, _ := hex.DecodeString(in.ScriptHex)

		input := tx.Input{
			Output: tx_input.Output{
				Outpoint: tx_input.Outpoint{
					Hash:  hash,
					Index: uint32(in.Vout),
				},
				Value: xc.NewBigIntFromStr(in.Value),
				// PubKeyScript: []byte{},
			},
			// SigScript: sigScript,
			// Address: xc.Address(in.Addresses[0]),
		}
		if len(in.Addresses) > 0 {
			input.Address = xc.Address(in.Addresses[0])
		}
		txObject.Input.UnspentOutputs = append(txObject.Input.UnspentOutputs, input.Output)
		inputs = append(inputs, input)
		sources = append(sources, &xc.LegacyTxInfoEndpoint{
			Address:         input.Address,
			Amount:          input.Value,
			ContractAddress: "",
			NativeAsset:     xc.NativeAsset(asset),
			Asset:           string(asset),
		})
	}

	for _, out := range data.Vout {
		recipient := tx.Recipient{
			// To:    xc.Address(out.Recipient),
			Value: xc.NewBigIntFromStr(out.Value),
		}
		if len(out.Addresses) > 0 {
			recipient.To = xc.Address(out.Addresses[0])
		}
		txObject.Recipients = append(txObject.Recipients, recipient)
	}

	// detect from, to, amount
	from, _ := tx.DetectFrom(inputs)
	to, amount, _ := txObject.DetectToAndAmount(from, expectedTo)
	for _, out := range data.Vout {
		if len(out.Addresses) > 0 {
			addr := out.Addresses[0]
			endpoint := &xc.LegacyTxInfoEndpoint{
				Address:     xc.Address(addr),
				Amount:      xc.NewBigIntFromStr(out.Value),
				NativeAsset: xc.NativeAsset(asset),
				Asset:       string(asset),
			}
			if addr != from {
				// legacy endpoint drops 'change' movements
				destinations = append(destinations, endpoint)
			} else {
				txWithInfo.AddDroppedDestination(endpoint)
			}
		}
	}

	// from
	// to
	// amount
	txWithInfo.From = xc.Address(from)
	txWithInfo.To = xc.Address(to)
	txWithInfo.Amount = amount
	txWithInfo.Sources = sources
	txWithInfo.Destinations = destinations

	return txWithInfo, nil
}

func (client *BlockbookClient) FetchTxInfo(ctx context.Context, txHashStr xc.TxHash) (*xclient.TxInfo, error) {
	legacyTx, err := client.FetchLegacyTxInfo(ctx, txHashStr)
	if err != nil {
		return nil, err
	}
	chain := client.cfg.Chain

	// delete the fee to avoid double counting.
	// the new model will calculate fees from the difference of inflows/outflows
	legacyTx.Fee = xc.NewBigIntFromUint64(0)

	// add back the change movements
	legacyTx.Destinations = append(legacyTx.Destinations, legacyTx.GetDroppedBtcDestinations()...)

	// remap to new tx
	return xclient.TxInfoFromLegacy(chain, legacyTx, xclient.Utxo), nil
}

func (client *BlockbookClient) FetchBalance(ctx context.Context, address xc.Address) (*xc.BigInt, error) {
	allUnspentOutputs, err := client.UnspentOutputs(ctx, address)
	amount := xc.NewBigIntFromUint64(0)
	if err != nil {
		return nil, err
	}
	for _, unspent := range allUnspentOutputs {
		amount = amount.Add(&unspent.Value)
	}
	return &amount, nil
}

func (client *BlockbookClient) FetchBalanceForAsset(ctx context.Context, address xc.Address, contractAddress xc.ContractAddress) (*xc.BigInt, error) {
	return nil, errors.New("not implemented")
}

func (client *BlockbookClient) FetchNativeBalance(ctx context.Context, address xc.Address) (*xc.BigInt, error) {
	return client.FetchBalance(ctx, address)
}
func (client *BlockbookClient) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc.TxInput, error) {
	input := tx_input.NewTxInput()
	allUnspentOutputs, err := client.UnspentOutputs(ctx, args.GetFrom())
	if err != nil {
		return input, err
	}
	input.UnspentOutputs = allUnspentOutputs
	gasPerByte, err := client.EstimateFee(ctx)
	input.GasPricePerByte = gasPerByte
	if err != nil {
		return input, err
	}

	return input, nil
}

func (client *BlockbookClient) FetchLegacyTxInput(ctx context.Context, from xc.Address, to xc.Address, asset xc.IAsset) (xc.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, xc.NewBigIntFromUint64(1), xcbuilder.WithAsset(asset))
	return client.FetchTransferInput(ctx, args)
}

func (client *BlockbookClient) EstimateGasFee(ctx context.Context, tx xc.Tx) (*xc.BigInt, error) {
	return nil, nil
}

func (client *BlockbookClient) get(ctx context.Context, path string, resp interface{}) error {
	path = strings.TrimPrefix(path, "/")
	url := fmt.Sprintf("%s/%s", client.Url, path)
	logrus.WithFields(logrus.Fields{
		"url": url,
	}).Debug("get")
	res, err := client.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("blockbook get failed: %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 && res.StatusCode != 201 {
		var errResponse ErrorResponse
		err = json.Unmarshal(body, &errResponse)
		if err == nil {
			return fmt.Errorf("failed to get %s: %s", path, errResponse.Error)
		}
		return fmt.Errorf("failed to get %s: code=%d", path, res.StatusCode)
	}

	if resp != nil {
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *BlockbookClient) post(ctx context.Context, path string, contentType string, input []byte, resp interface{}) error {
	path = strings.TrimPrefix(path, "/")
	url := fmt.Sprintf("%s/%s", client.Url, path)
	logrus.WithFields(logrus.Fields{
		"url":  url,
		"body": string(input),
	}).Debug("post")
	res, err := client.httpClient.Post(url, contentType, bytes.NewReader(input))
	if err != nil {
		return fmt.Errorf("blockbook post failed: %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 && res.StatusCode != 201 {
		var errResponse ErrorResponse
		err = json.Unmarshal(body, &errResponse)
		if err == nil {
			return fmt.Errorf("failed to post %s: %s", path, errResponse.Error)
		}
		return fmt.Errorf("failed to post %s: code=%d", path, res.StatusCode)
	}

	if resp != nil {
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}
	}
	return nil
}
