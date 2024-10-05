package ton

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"math/big"
	"time"

	tonaddress "github.com/openweb3-io/crosschain/blockchain/ton/address"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	"github.com/pkg/errors"

	"github.com/openweb3-io/crosschain/blockchain/ton/tx"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	"github.com/openweb3-io/crosschain/builder"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	_ton "github.com/xssnick/tonutils-go/ton"
	"go.uber.org/zap"
)

type Client struct {
	client  *tonapi.Client
	lclient ton.APIClientWrapped
}

func NewClient(cfg xc_types.IAsset) (*Client, error) {
	client := liteclient.NewConnectionPool()

	// from cfg
	// url := "https://ton-blockchain.github.io/testnet-global.config.json"
	url := cfg.GetChain().URL
	if url == "" {
		url = "https://api.tontech.io/ton/wallet-mainnet.autoconf.json"
	}
	err := client.AddConnectionsFromConfigUrl(context.Background(), url)
	if err != nil {
		return nil, err
	}
	liteApiClient := _ton.NewAPIClient(client)

	tonApi, err := tonapi.New(tonapi.WithToken("AEXRCJJGQBXCFWQAAAAD3RYTVUWCXT5JW6YN2QU7LHXMKPMOXHFB75P4JSD52AVOVQWPGNY"))
	if err != nil {
		return nil, err
	}

	return &Client{tonApi, liteApiClient}, nil
}

func (client *Client) FetchTransferInput(ctx context.Context, args *builder.TransferArgs) (xc_types.TxInput, error) {
	acc, err := client.client.GetAccount(ctx, tonapi.GetAccountParams{
		AccountID: string(args.From),
	})
	if err != nil {
		return nil, err
	}

	seq, err := client.client.GetAccountSeqno(ctx, tonapi.GetAccountSeqnoParams{
		AccountID: string(args.From),
	})
	if err != nil {
		return nil, err
	}

	input := &TxInput{
		Timestamp:       333,
		AccountStatus:   acc.Status,
		TonBalance:      xc_types.NewBigIntFromInt64(acc.GetBalance()),
		Seq:             uint32(seq.Seqno),
		EstimatedMaxFee: xc_types.NewBigIntFromInt64(0), // TODO
		From:            args.From,
		To:              args.To,
		TokenDecimals:   args.TokenDecimals,
		Amount:          args.Amount,
		Memo:            args.Memo,
		ContractAddress: args.ContractAddress,
	}

	if input.ContractAddress != nil {
		input.TokenWallet, err = client.GetJettonWallet(ctx, args.From, xc_types.ContractAddress(*args.ContractAddress))
		if err != nil {
			return input, err
		}

		maxFee, err := client.EstimateMaxFee(ctx, args.From, args.To, input.TokenWallet, args.TokenDecimals, args.Memo, input.Seq)
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

	result, err := client.client.GetAccountJettonBalance(ctx, tonapi.GetAccountJettonBalanceParams{
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

func (client *Client) EstimateMaxFee(ctx context.Context, from xc_types.Address, to xc_types.Address, jettonWalletAddress xc_types.Address, tokenDecimals int32, memo string, seq uint32) (*types.BigInt, error) {
	fromAddr, _ := address.ParseAddr(string(from))
	toAddr, _ := address.ParseAddr(string(to))
	jettonWalletAddr, err := tonaddress.ParseAddress(jettonWalletAddress, "")
	if err != nil {
		return nil, err
	}
	amountTlb, _ := tlb.FromNano(big.NewInt(1), int(tokenDecimals))

	example, err := BuildJettonTransfer(
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

	tx := tx.NewTx(fromAddr, cellBuilder, nil)
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

	res, err := client.client.EmulateMessageToWallet(ctx, &tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(b),
		/*Params: []tonapi.EmulateMessageToWalletReqParamsItem{
			{
				Address: string(from),
			},
		},*/
	}, tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "could not estimate fee")
	}

	gas := types.NewBigIntFromInt64(res.Event.Extra * -1)

	return &gas, nil
}

func (a *Client) GetBalanceForAsset(ctx context.Context, ownerAddress types.Address, assetAddr types.Address) (*types.BigInt, error) {
	jettonBalance, err := a.client.GetAccountJettonBalance(ctx, tonapi.GetAccountJettonBalanceParams{
		AccountID: string(ownerAddress),
		JettonID:  string(assetAddr),
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetJettonWallet failed")
	}

	amount := types.NewBigIntFromStr(jettonBalance.Balance)
	return &amount, nil
}

func (a *Client) FetchBalance(ctx context.Context, address types.Address) (*types.BigInt, error) {
	account, err := a.client.GetAccount(ctx, tonapi.GetAccountParams{
		AccountID: string(address),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get balance failed")
	}

	balance := types.NewBigIntFromInt64(account.Balance)
	return &balance, nil
}

func (a *Client) EstimateGas(ctx context.Context, _tx types.Tx) (*types.BigInt, error) {
	tx := _tx.(*tx.Tx)

	boc, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	res, err := a.client.EmulateMessageToWallet(ctx, &tonapi.EmulateMessageToWalletReq{
		Boc: base64.StdEncoding.EncodeToString(boc),
	}, tonapi.EmulateMessageToWalletParams{})
	if err != nil {
		return nil, errors.Wrap(err, "EmulateMessageToWallet failed")
	}

	gas := types.NewBigIntFromInt64(res.Event.Extra * -1)
	return &gas, nil
}

func (a *Client) SubmitTx(ctx context.Context, _tx types.Tx) error {
	tx := _tx.(*tx.Tx)

	st := time.Now()
	defer func() {
		zap.S().Info("broadcast transaction", zap.Duration("cost", time.Since(st)))
	}()

	// route all requests to the same node
	ctx = a.lclient.Client().StickyContext(ctx)

	payload, err := tx.Serialize()
	if err != nil {
		return err
	}

	_, err = a.client.SendMessage(ctx, payload)
	if err != nil {
		return errors.Wrap(err, "SendMessage failed")
	}

	return nil
}

func (client *Client) FetchLegacyTxInput(ctx context.Context, from types.Address, to types.Address) (types.TxInput, error) {
	// No way to pass the amount in the input using legacy interface, so we estimate using min amount.
	args, _ := xcbuilder.NewTransferArgs(from, to, types.NewBigIntFromUint64(1))
	return client.FetchTransferInput(ctx, args)
}
