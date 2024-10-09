package liteserver

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/liteclient"

	"github.com/openweb3-io/crosschain/blockchain/ton"
	tonaddress "github.com/openweb3-io/crosschain/blockchain/ton/address"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	tontx "github.com/openweb3-io/crosschain/blockchain/ton/tx"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	xcclient "github.com/openweb3-io/crosschain/client"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	_ton "github.com/xssnick/tonutils-go/ton"
	"go.uber.org/zap"
)

type Client struct {
	cfg    *xc_types.ChainConfig
	Client *_ton.APIClient
}

var _ xcclient.IClient = &Client{}

func NewClient(cfg *xc_types.ChainConfig) (*Client, error) {
	var url = cfg.URL
	if url == "" {
		url = "https://api.tontech.io/ton/wallet-mainnet.autoconf.json"
	}

	c := liteclient.NewConnectionPool()
	err := c.AddConnectionsFromConfigUrl(context.Background(), url)
	if err != nil {
		return nil, err
	}
	client := _ton.NewAPIClient(c)

	return &Client{cfg, client}, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *xcbuilder.TransferArgs) (xc_types.TxInput, error) {
	b, err := client.Client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	fromAddr, err := address.ParseAddr(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	wrappedClient := client.Client.WaitForBlock(b.SeqNo)
	acc, err := wrappedClient.GetAccount(ctx, b, fromAddr)
	if err != nil {
		return nil, err
	}

	seqResp, err := wrappedClient.RunGetMethod(ctx, b, fromAddr, "seqno")
	if err != nil {
		return nil, err
	}

	seq, err := seqResp.Int(0)
	if err != nil {
		return nil, err
	}

	balance := acc.State.Balance.Nano()

	input := &ton.TxInput{
		Timestamp:       time.Now().Unix(),
		AccountStatus:   ton.AccountStatus(acc.State.Status),
		TonBalance:      xc_types.BigInt(*balance),
		Seq:             uint32(seq.Uint64()),
		EstimatedMaxFee: xc_types.NewBigIntFromInt64(0), // TODO
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

	addr, err := address.ParseAddr(string(from))
	if err != nil {
		return "", err
	}

	jettonAddr, err := address.ParseAddr(string(contract))
	if err != nil {
		return "", err
	}

	jettonCli := jetton.NewJettonMasterClient(client.Client, jettonAddr)
	if err != nil {
		return "", err
	}

	result, err := jettonCli.GetJettonWallet(ctx, addr)

	if err != nil {
		return "", err
	}

	return xc_types.Address(result.Address().String()), nil
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

	_, err = tx.Serialize()
	if err != nil {
		return nil, err
	}

	/* TODO emulate method not founded for liteclient
	res, err := client.Client.EmulateMessageToWallet(ctx, &tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(b),
	}, tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "could not estimate fee")
	}

	gas := xc_types.NewBigIntFromInt64(res.Event.Extra * -1)
	*/
	gas := xc_types.NewBigIntFromInt64(0)

	return &gas, nil
}

func (client *Client) FetchBalanceForAsset(ctx context.Context, ownerAddress xc_types.Address, asset xc_types.IAsset) (*xc_types.BigInt, error) {
	ownerAddr, err := address.ParseAddr(string(ownerAddress))
	if err != nil {
		return nil, err
	}

	jettonAddr, err := address.ParseAddr(string(asset.GetContract()))
	if err != nil {
		return nil, err
	}

	b, err := client.Client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	jettonCli := jetton.NewJettonMasterClient(client.Client, jettonAddr)
	if err != nil {
		return nil, err
	}

	jettonWallet, err := jettonCli.GetJettonWalletAtBlock(ctx, ownerAddr, b)
	if err != nil {
		return nil, errors.Wrap(err, "GetJettonWallet failed")
	}

	w, err := jettonWallet.GetBalanceAtBlock(ctx, b)
	if err != nil {
		return nil, err
	}

	return (*xc_types.BigInt)(w), nil
}

func (client *Client) FetchBalance(ctx context.Context, ownerAddress xc_types.Address) (*xc_types.BigInt, error) {
	addr, err := address.ParseAddr(string(ownerAddress))
	if err != nil {
		return nil, err
	}

	b, err := client.Client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	account, err := client.Client.GetAccount(ctx, b, addr)
	if err != nil {
		return nil, errors.Wrap(err, "get balance failed")
	}

	balance := account.State.Balance.Nano()

	return (*xc_types.BigInt)(balance), nil
}

func (a *Client) EstimateGas(ctx context.Context, tx xc_types.Tx) (*xc_types.BigInt, error) {
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

	_, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	/* TODO
	res, err := a.Client.EmulateMessageToWallet(ctx, &tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(boc),
	}, tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "EmulateMessageToWallet failed")
	}

	gas := xc_types.NewBigIntFromInt64(res.Event.Extra * -1)
	*/

	gas := xc_types.NewBigIntFromInt64(0)

	return &gas, nil
}

func (a *Client) BroadcastTx(ctx context.Context, _tx xc_types.Tx) error {
	tx := _tx.(*tontx.Tx)

	st := time.Now()
	defer func() {
		zap.S().Info("broadcast transaction", zap.Duration("cost", time.Since(st)))
	}()

	/*
		payload, err := tx.Serialize()
		if err != nil {
			return err
		}*/

	err := a.Client.SendExternalMessage(ctx, tx.ExternalMessage)
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
	chainInfo, err := client.Client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	/*
		chainInfo, err := client.Client.GetRawMasterchainInfo(ctx)
		if err != nil {
			return nil, err
		}*/

	tx, err := client.FetchTonTxByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	sources := []*xc_types.LegacyTxInfoEndpoint{}
	dests := []*xc_types.LegacyTxInfoEndpoint{}
	chain := client.cfg.Chain

	totalFee := xc_types.BigInt(*tx.TotalFees.Coins.Nano())

	outMsgs, err := tx.IO.Out.ToSlice()
	if err != nil {
		return nil, err
	}

	for _, msg := range outMsgs {
		intMsg := msg.AsInternal()
		if intMsg.Bounced {
			// if the message bounced, do no add endpoints
		} else {
			memo := ""

			memo = intMsg.Comment()

			if intMsg.DstAddr != nil && intMsg.DstAddr.String() != "" && intMsg.Amount.Nano().Int64() != 0 {
				// addr, err := client.substituteOrParse(addrBook, *)
				addr, err := tonaddress.ParseAddress(xc_types.Address(intMsg.DstAddr.String()), "")
				if err != nil {
					return nil, fmt.Errorf("invalid address %s: %v", intMsg.DstAddr.String(), err)
				}
				value := xc_types.BigInt(*intMsg.Amount.Nano())
				dests = append(dests, &xc_types.LegacyTxInfoEndpoint{
					Address:         xc_types.Address(addr.String()),
					ContractAddress: "",
					Amount:          value,
					NativeAsset:     chain,
					Memo:            memo,
				})
			}
			if intMsg.SrcAddr != nil && intMsg.SrcAddr.String() != "" && intMsg.Amount.Nano().Int64() != 0 {
				addr, err := tonaddress.ParseAddress(xc_types.Address(intMsg.SrcAddr.String()), "")
				if err != nil {
					return nil, fmt.Errorf("invalid address %v: %v", intMsg.SrcAddr, err)
				}
				value := xc_types.BigInt(*intMsg.Amount.Nano())
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

	switch tx.IO.In.MsgType {
	case tlb.MsgTypeInternal:
		{
			intMsg := tx.IO.In.AsInternal()

			b, _ := json.MarshalIndent(tx, "", "\t")
			fmt.Printf("tx.InMsg.Value.DecodedBody: %v\n", string(b))

			sources = append(sources, &xc_types.LegacyTxInfoEndpoint{
				// this is the token wallet of the sender/owner
				Address: xc_types.Address(intMsg.SrcAddr.String()),
				Amount:  xc_types.BigInt(*intMsg.Amount.Nano()),
				// Asset:           tf.Jetton.Symbol,
				ContractAddress: "",
				NativeAsset:     chain,
			})

			dests = append(dests, &xc_types.LegacyTxInfoEndpoint{
				// this is the token wallet of the sender/owner
				Address: xc_types.Address(intMsg.DstAddr.String()),
				Amount:  xc_types.BigInt(*intMsg.Amount.Nano()),
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

	// TODO
	block, err := client.Client.GetBlockData(ctx, nil)
	if err != nil {
		return nil, err
	}

	sources = append(sources, jettonSources...)
	dests = append(dests, jettonDests...)
	info := &xc_types.LegacyTxInfo{
		BlockHash:     strconv.FormatInt(int64(block.BlockInfo.Shard.GetShardID()), 10),
		BlockIndex:    int64(block.BlockInfo.SeqNo),
		BlockTime:     int64(tx.Now), // block.GenUtime
		Confirmations: int64(chainInfo.SeqNo - block.BlockInfo.SeqNo),
		// Use the InMsg hash as this can be determined offline,
		// whereas the tx.Hash is determined by the chain after submitting.
		TxID:        tontx.Normalize(string(tx.Hash)),
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
func (client *Client) detectJettonMovements(ctx context.Context, tx *tlb.Transaction) ([]*xc_types.LegacyTxInfoEndpoint, []*xc_types.LegacyTxInfoEndpoint, error) {
	internalMsg := tx.IO.In.AsExternalIn()

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
func (client *Client) FetchTonTxByHash(ctx context.Context, txHash xc_types.TxHash) (_ *tlb.Transaction, err error) {
	hashBytes, err := hex.DecodeString(string(txHash))
	if err != nil {
		return nil, err
	}

	transaction, _ := client.Client.FindLastTransactionByInMsgHash(ctx, nil, hashBytes)

	if transaction == nil {
		transaction, err = client.Client.FindLastTransactionByOutMsgHash(ctx, nil, hashBytes)

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
	/*
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
	*/
	return nil, nil, true, nil
}

func (client *Client) LookupTransferForTokenWallet(ctx context.Context, tokenWallet *address.Address) (*jetton.TransferPayload, error) {
	/*
		// GetAccountJettonsHistory
		resp, err := client.Client.GetAccountEvents(ctx, tonapi.GetAccountEventsParams{
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
	*/
	return nil, nil
}
