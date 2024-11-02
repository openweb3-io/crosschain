package tonapi

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/openweb3-io/crosschain/blockchain/ton"
	tonaddress "github.com/openweb3-io/crosschain/blockchain/ton/address"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	tontx "github.com/openweb3-io/crosschain/blockchain/ton/tx"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	xcclient "github.com/openweb3-io/crosschain/client"
	xc_types "github.com/openweb3-io/crosschain/types"
	_tonapi "github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"go.uber.org/zap"
)

type TextComment struct {
	Text string `json:"text"`
}

type Value struct {
	SumType string       `json:"sum_type"`
	OpCode  int          `json:"op_code"`
	Value   *TextComment `json:"value"`
}

type ForwardPayload struct {
	IsRight bool   `json:"is_right"`
	Value   *Value `json:"value"`
}

type JettonTransferPayload struct {
	QueryId             int64           `json:"query_id"`
	Amount              string          `json:"amount"`
	Destination         string          `json:"destination"`
	ResponseDestination string          `json:"response_destination"`
	CustomPayload       any             `json:"custom_payload"`
	ForwardTonAmount    string          `json:"forward_ton_amount"`
	ForwardPayload      *ForwardPayload `json:"forward_payload"`
}

type Client struct {
	cfg    *xc_types.ChainConfig
	Client *_tonapi.Client
}

var _ xcclient.IClient = &Client{}

func NewClient(cfg *xc_types.ChainConfig) (*Client, error) {
	var url = cfg.Client.URL
	if url == "" {
		url = _tonapi.TonApiURL
	}

	tonApi, err := _tonapi.NewClient(
		url,
		_tonapi.WithToken(cfg.Client.Auth),
	)
	if err != nil {
		return nil, err
	}

	return &Client{cfg, tonApi}, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc_types.TxInput, error) {
	acc, err := client.Client.GetAccount(ctx, _tonapi.GetAccountParams{
		AccountID: string(args.GetFrom()),
	})
	if err != nil {
		return nil, err
	}

	seq, err := client.Client.GetAccountSeqno(ctx, _tonapi.GetAccountSeqnoParams{
		AccountID: string(args.GetFrom()),
	})
	if err != nil {
		return nil, err
	}

	input := &ton.TxInput{
		Timestamp:       time.Now().Unix(),
		AccountStatus:   ton.AccountStatus(acc.Status),
		TonBalance:      xc_types.NewBigIntFromInt64(acc.GetBalance()),
		Seq:             uint32(seq.Seqno),
		EstimatedMaxFee: xc_types.NewBigIntFromInt64(0), // TODO
	}

	if pubKey, ok := args.GetPublicKey(); ok {
		input.PublicKey = pubKey
	}

	memo, _ := args.GetMemo()

	asset, _ := args.GetAsset()
	if asset != nil && asset.GetContract() != "" {
		input.TokenWallet, err = client.GetJettonWallet(ctx, args.GetFrom(), asset.GetContract())
		if err != nil {
			return input, err
		}

		maxFee, err := client.EstimateMaxFee(ctx, args.GetFrom(), args.GetTo(), input.TokenWallet, asset.GetDecimals(), memo, input.Seq)
		if err != nil {
			return input, err
		}
		input.EstimatedMaxFee = *maxFee
	}

	return input, nil
}

func (client *Client) GetJettonWallet(ctx context.Context, from xc_types.Address, contract xc_types.ContractAddress) (xc_types.Address, error) {
	// fromAddr, _ := address.ParseAddr(string(from))
	// contractAddr, _ := address.ParseAddr(string(contract))

	result, err := client.Client.GetAccountJettonBalance(ctx, _tonapi.GetAccountJettonBalanceParams{
		AccountID: string(from),
		JettonID:  string(contract),
	})
	if err != nil {
		return "", err
	}

	/*
		token := jetton.NewJettonMasterClient(client.lclient, contractAddr)
		tokenWallet, err := token.GetJettonWallet(ctx, fromAddr)
		if err != nil {
			return "", err
		}
	*/

	return xc_types.Address(result.WalletAddress.Address), nil
}

func (client *Client) EstimateMaxFee(ctx context.Context, from xc_types.Address, to xc_types.Address, jettonWalletAddress xc_types.Address, tokenDecimals int32, memo string, seq uint32) (*xc_types.BigInt, error) {
	fromAddr, _ := address.ParseAddr(string(from))
	toAddr, _ := address.ParseAddr(string(to))
	jettonWalletAddr, err := tonaddress.ParseAddress(jettonWalletAddress, "")
	if err != nil {
		return nil, err
	}
	amountTlb, _ := tlb.FromNano(big.NewInt(1), int(tokenDecimals))

	example, err := ton.BuildJettonTransfer(
		10,
		fromAddr,
		jettonWalletAddr,
		toAddr,
		amountTlb,
		tlb.MustFromTON("1.0"),
		memo,
	)
	if err != nil {
		return nil, err
	}

	seqnoFetcher := func(ctx context.Context, subWallet uint32) (uint32, error) {
		return seq, nil
	}

	w, err := wallet.FromAddress(seqnoFetcher, fromAddr, wallet.V4R2)
	if err != nil {
		return nil, err
	}

	cellBuilder, err := w.BuildMessages(ctx, false, []*wallet.Message{example})
	if err != nil {
		return nil, err
	}

	tx := tontx.NewTx(fromAddr, cellBuilder, nil)
	sighashes, err := tx.Sighashes()
	if err != nil {
		return nil, err
	}
	if len(sighashes) != 1 {
		return nil, errors.New("invalid sighashes")
	}

	privateKey := make([]byte, 64)
	signature := ed25519.Sign(privateKey, sighashes[0])

	err = tx.AddSignatures(signature)
	if err != nil {
		return nil, err
	}

	b, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	res, err := client.Client.EmulateMessageToWallet(ctx, &_tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(b),
		/*Params: []tonapi.EmulateMessageToWalletReqParamsItem{
			{
				Address: string(from),
			},
		},*/
	}, _tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "could not estimate fee")
	}

	gas := xc_types.NewBigIntFromInt64(res.Event.Extra * -1)

	return &gas, nil
}

func (a *Client) FetchBalanceForAsset(ctx context.Context, ownerAddress xc_types.Address, contractAddress xc_types.ContractAddress) (*xc_types.BigInt, error) {
	jettonBalance, err := a.Client.GetAccountJettonBalance(ctx, _tonapi.GetAccountJettonBalanceParams{
		AccountID: string(ownerAddress),
		JettonID:  string(contractAddress),
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetJettonWallet failed")
	}

	amount := xc_types.NewBigIntFromStr(jettonBalance.Balance)
	return &amount, nil
}

func (a *Client) FetchBalance(ctx context.Context, address xc_types.Address) (*xc_types.BigInt, error) {
	account, err := a.Client.GetAccount(ctx, _tonapi.GetAccountParams{
		AccountID: string(address),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get balance failed")
	}

	balance := xc_types.NewBigIntFromInt64(account.Balance)
	return &balance, nil
}

func (a *Client) EstimateGasFee(ctx context.Context, tx xc_types.Tx) (*xc_types.BigInt, error) {
	if len(tx.GetSignatures()) == 0 {
		// add a mock sig
		sighashes, err := tx.Sighashes()
		if err != nil {
			return nil, err
		}
		if len(sighashes) != 1 {
			return nil, errors.New("invalid sighashes")
		}

		privateKey := make([]byte, 64)
		signature := ed25519.Sign(privateKey, sighashes[0])

		err = tx.AddSignatures(signature)
		if err != nil {
			return nil, err
		}
	}

	boc, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	res, err := a.Client.EmulateMessageToWallet(ctx, &_tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(boc),
	}, _tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "EmulateMessageToWallet failed")
	}

	gas := xc_types.NewBigIntFromInt64(res.Event.Extra * -1)
	return &gas, nil
}

func (client *Client) EstimateGas1(ctx context.Context, args *xcbuilder.TransferArgs) (*xc_types.BigInt, error) {
	input, err := client.FetchTransferInput(ctx, args)
	if err != nil {
		return nil, err
	}

	txBuilder, err := ton.NewTxBuilder(client.cfg)
	if err != nil {
		return nil, err
	}

	tx, err := txBuilder.NewTransfer(args, input)
	if err != nil {
		return nil, err
	}

	if len(tx.GetSignatures()) == 0 {
		// add a mock sig
		sighashes, err := tx.Sighashes()
		if err != nil {
			return nil, err
		}
		if len(sighashes) != 1 {
			return nil, errors.New("invalid sighashes")
		}

		privateKey := make([]byte, 64)
		signature := ed25519.Sign(privateKey, sighashes[0])

		err = tx.AddSignatures(signature)
		if err != nil {
			return nil, err
		}
	}

	boc, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	res, err := client.Client.EmulateMessageToWallet(ctx, &_tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(boc),
	}, _tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "EmulateMessageToWallet failed")
	}

	gas := xc_types.NewBigIntFromInt64(res.Event.Extra * -1)
	return &gas, nil
}

func (a *Client) BroadcastTx(ctx context.Context, _tx xc_types.Tx) error {
	tx := _tx.(*tontx.Tx)

	st := time.Now()
	defer func() {
		zap.S().Info("broadcast transaction", zap.Duration("cost", time.Since(st)))
	}()

	payload, err := tx.Serialize()
	if err != nil {
		return err
	}

	_, err = a.Client.SendMessage(ctx, payload)
	if err != nil {
		return errors.Wrap(err, "SendMessage failed")
	}

	return nil
}

func (client *Client) FetchLegacyTxInput(ctx context.Context, from xc_types.Address, to xc_types.Address, asset xc_types.IAsset) (xc_types.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, xc_types.NewBigIntFromUint64(1), xcbuilder.WithAsset(asset))
	return client.FetchTransferInput(ctx, args)
}

// Returns transaction info - legacy/old endpoint
func (client *Client) FetchLegacyTxInfo(ctx context.Context, txHash xc_types.TxHash) (*xc_types.LegacyTxInfo, error) {
	chainInfo, err := client.Client.GetRawMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := client.FetchTonTxByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	sources := []*xc_types.LegacyTxInfoEndpoint{}
	dests := []*xc_types.LegacyTxInfoEndpoint{}
	chain := client.cfg.Chain

	totalFee := xc_types.NewBigIntFromInt64(tx.TotalFees)

	for _, msg := range tx.OutMsgs {
		if msg.Bounced {
			// if the message bounced, do no add endpoints
		} else {
			memo := ""

			if msg.DecodedBody != nil && msg.DecodedOpName.Value == "text_comment" {
				var body struct {
					Text string `json:"text"`
				}
				_ = json.Unmarshal(msg.DecodedBody, &body)

				memo = string(body.Text)
			}

			if msg.Destination.IsSet() && msg.Destination.Value.Address != "" && msg.Value != 0 {
				// addr, err := client.substituteOrParse(addrBook, *)
				addr, err := tonaddress.ParseAddress(xc_types.Address(msg.Destination.Value.Address), "")
				if err != nil {
					return nil, fmt.Errorf("invalid address %s: %v", msg.Destination.Value.Address, err)
				}
				value := xc_types.NewBigIntFromInt64(msg.Value)
				dests = append(dests, &xc_types.LegacyTxInfoEndpoint{
					Address:         xc_types.Address(addr.String()),
					ContractAddress: "",
					Amount:          value,
					NativeAsset:     chain,
					Memo:            memo,
				})
			}
			if msg.Source.IsSet() && msg.Source.Value.Address != "" && msg.Value != 0 {
				addr, err := tonaddress.ParseAddress(xc_types.Address(msg.Source.Value.Address), "")
				if err != nil {
					return nil, fmt.Errorf("invalid address %v: %v", msg.Source, err)
				}
				value := xc_types.NewBigIntFromInt64(msg.Value)
				sources = append(sources, &xc_types.LegacyTxInfoEndpoint{
					Address:         xc_types.Address(addr.String()),
					ContractAddress: "",
					Amount:          value,
					NativeAsset:     chain,
					Memo:            memo,
				})
			}
		}
	}

	switch tx.InMsg.Value.MsgType {
	case _tonapi.MessageMsgTypeIntMsg:
		{
			b, _ := json.MarshalIndent(tx, "", "\t")
			fmt.Printf("tx.InMsg.Value.DecodedBody: %v\n", string(b))

			sources = append(sources, &xc_types.LegacyTxInfoEndpoint{
				// this is the token wallet of the sender/owner
				Address: xc_types.Address(tx.InMsg.Value.Source.Value.Address),
				Amount:  xc_types.NewBigIntFromInt64(tx.InMsg.Value.Value),
				// Asset:           tf.Jetton.Symbol,
				ContractAddress: "",
				NativeAsset:     chain,
			})

			dests = append(dests, &xc_types.LegacyTxInfoEndpoint{
				// this is the token wallet of the sender/owner
				Address: xc_types.Address(tx.InMsg.Value.Destination.Value.Address),
				Amount:  xc_types.NewBigIntFromInt64(tx.InMsg.Value.Value),
				// Asset:           tf.Jetton.Symbol,
				ContractAddress: "",
				NativeAsset:     chain,
			})
		}
	}

	jettonSources, jettonDests, err := client.detectJettonMovements(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("could not detect jetton movements: %v", err)
	}

	block, err := client.Client.GetBlockchainBlock(ctx, _tonapi.GetBlockchainBlockParams{
		BlockID: tx.Block,
	})
	if err != nil {
		return nil, err
	}

	sources = append(sources, jettonSources...)
	dests = append(dests, jettonDests...)
	info := &xc_types.LegacyTxInfo{
		BlockHash:     block.Shard,
		BlockIndex:    int64(block.Seqno),
		BlockTime:     tx.Utime, // block.GenUtime
		Confirmations: int64(chainInfo.Last.Seqno - block.Seqno),
		// Use the InMsg hash as this can be determined offline,
		// whereas the tx.Hash is determined by the chain after submitting.
		TxID:        tontx.Normalize(tx.InMsg.Value.Hash),
		ExplorerURL: "",

		Sources:      sources,
		Destinations: dests,
		Fee:          totalFee,
		From:         "",
		To:           "",
		ToAlt:        "",
		Amount:       xc_types.BigInt{},

		// unused fields
		ContractAddress: "",
		FeeContract:     "",
		Time:            0,
		TimeReceived:    0,
	}
	if len(info.Sources) > 0 {
		info.From = info.Sources[0].Address
	}
	if len(info.Destinations) > 0 {
		info.To = info.Destinations[0].Address
		info.Amount = info.Destinations[0].Amount
	}

	return info, nil
}

// This detects any JettonMessage in the nest of "InternalMessage"
// This may need to be expanded as Jetton transfer could be nested deeper in more 'InternalMessages'
func (client *Client) detectJettonMovements(ctx context.Context, tx *_tonapi.Transaction) ([]*xc_types.LegacyTxInfoEndpoint, []*xc_types.LegacyTxInfoEndpoint, error) {
	if tx.InMsg.Value.MsgType == _tonapi.MessageMsgTypeIntMsg {
		return nil, nil, nil
	}

	boc, err := hex.DecodeString(tx.InMsg.Value.RawBody.Value)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid base64: %v", err)
	}

	inMsg, err := cell.FromBOC(boc)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid boc: %v", err)
	}

	fmt.Printf("msgType: %v\n", tx.InMsg.Value.MsgType)

	internalMsg := &tlb.InternalMessage{}
	nextMsg, err := inMsg.BeginParse().LoadRefCell()
	if err != nil {
		err = tlb.LoadFromCell(internalMsg, inMsg.BeginParse())
	} else {
		err = tlb.LoadFromCell(internalMsg, nextMsg.BeginParse())
	}

	if err != nil {
		err = tlb.LoadFromCell(internalMsg, inMsg.BeginParse())
	}

	if err != nil {
		return nil, nil, nil
	}
	if internalMsg.DstAddr == nil {
		return nil, nil, nil
	}

	next := internalMsg.Body
	for next != nil {
		sources, dests, ok, err := client.ParseJetton(ctx, next, internalMsg.DstAddr)
		if err != nil {
			return nil, nil, fmt.Errorf("%v", err)
		}
		if ok {
			return sources, dests, nil
		}
		nextMsg, err := next.BeginParse().LoadRefCell()
		if err != nil {
			break
		} else {
			next = nextMsg
		}
	}

	return nil, nil, nil
}

func ParseBlock(block string) (string, string, error) {
	parts := strings.Split(block, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid block format: %s", block)
	}
	return parts[0], parts[1], nil
}

// Prioritize getting tx by msg-hash as it's deterministic offline.  Fallback to using chain-calculated tx hash.
func (client *Client) FetchTonTxByHash(ctx context.Context, txHash xc_types.TxHash) (_ *_tonapi.Transaction, err error) {
	transaction, _ := client.Client.GetBlockchainTransactionByMessageHash(ctx, _tonapi.GetBlockchainTransactionByMessageHashParams{
		MsgID: string(txHash),
	})

	if transaction == nil {
		transaction, err = client.Client.GetBlockchainTransaction(ctx, _tonapi.GetBlockchainTransactionParams{
			TransactionID: url.QueryEscape(string(txHash)),
		})

		if err != nil {
			return nil, err
		}

		if transaction == nil {
			return nil, fmt.Errorf("no TON transaction found by %s", txHash)
		}
	}

	return transaction, nil
}

func (client *Client) ParseJetton(ctx context.Context, c *cell.Cell, tokenWallet *address.Address) ([]*xc_types.LegacyTxInfoEndpoint, []*xc_types.LegacyTxInfoEndpoint, bool, error) {
	net := client.cfg.Network
	jettonTfMaybe := &jetton.TransferPayload{}
	err := tlb.LoadFromCell(jettonTfMaybe, c.BeginParse())
	if err != nil {
		// give up here - no jetton movement(s)
		logrus.WithError(err).Debug("no jetton transfer detected")
		return nil, nil, false, nil
	}
	memo, ok := ton.ParseComment(jettonTfMaybe.ForwardPayload)
	// fmt.Println("memo ", memo, ok)
	if !ok {
		memo, _ = ton.ParseComment(jettonTfMaybe.CustomPayload)
	}
	tf, err := client.LookupTransferForTokenWallet(ctx, tokenWallet)
	if err != nil {
		return nil, nil, false, err
	}
	masterAddr, err := tonaddress.ParseAddress(xc_types.Address(tf.Jetton.Address), net)
	if err != nil {
		return nil, nil, false, err
	}
	// The native jetton structure is confusingly inconsistent in that it uses the 'tokenWallet' for the sourceAddress,
	// but uses the owner account for the destinationAddress.  But in the /jetton/transfers endpoint, it is reported
	// using the owner address.  So we use that.
	ownerAddr, err := tonaddress.ParseAddress(xc_types.Address(tf.Sender.Value.Address), "")
	if err != nil {
		return nil, nil, false, err
	}

	chain := client.cfg.Chain
	amount := xc_types.BigInt(*jettonTfMaybe.Amount.Nano())
	sources := []*xc_types.LegacyTxInfoEndpoint{
		{
			// this is the token wallet of the sender/owner
			Address: xc_types.Address(ownerAddr.String()),
			Amount:  amount,
			// Asset:           tf.Jetton.Symbol,
			ContractAddress: xc_types.ContractAddress(masterAddr.String()),
			NativeAsset:     chain,
			Memo:            memo,
		},
	}

	dests := []*xc_types.LegacyTxInfoEndpoint{
		{
			// The destination uses the owner account already
			Address: xc_types.Address(jettonTfMaybe.Destination.String()),
			Amount:  amount,
			// Asset:           tf.Jetton.Symbol,
			ContractAddress: xc_types.ContractAddress(masterAddr.String()),
			NativeAsset:     chain,
			Memo:            memo,
		},
	}
	return sources, dests, true, nil
}

func (client *Client) LookupTransferForTokenWallet(ctx context.Context, tokenWallet *address.Address) (*_tonapi.JettonTransferAction, error) {
	// GetAccountJettonsHistory
	resp, err := client.Client.GetAccountEvents(ctx, _tonapi.GetAccountEventsParams{
		AccountID: tokenWallet.String(),
		Limit:     10,
	})

	if err != nil {
		return nil, fmt.Errorf("could not resolve token master address: %v", err)
	}

	if len(resp.Events) == 0 {
		return nil, fmt.Errorf("could not resolve token master address: no transfer history")
	}
	evt := resp.Events[0]

	if len(evt.Actions) == 0 {
		return nil, fmt.Errorf("could not resolve token master address: no transfer history")
	}

	act := evt.Actions[0]
	if !act.JettonTransfer.IsSet() {
		return nil, fmt.Errorf("could not resolve token master address: no transfer history")
	}

	fmt.Printf("jetton name: %v, %v\n", act.JettonTransfer.Value.Jetton.Name, act.JettonTransfer)
	return &act.JettonTransfer.Value, nil
}
