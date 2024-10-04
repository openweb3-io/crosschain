package ton

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"math/big"
	"time"

	"github.com/pkg/errors"

	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	"github.com/openweb3-io/crosschain/types"
	"github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	_ton "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"go.uber.org/zap"
)

type Client struct {
	client  *tonapi.Client
	lclient ton.APIClientWrapped
}

func NewClient(cfg string) (*Client, error) {
	client := liteclient.NewConnectionPool()

	// from cfg
	// url := "https://ton-blockchain.github.io/testnet-global.config.json"
	url := "https://api.tontech.io/ton/wallet-mainnet.autoconf.json"
	err := client.AddConnectionsFromConfigUrl(context.Background(), url)
	if err != nil {
		return nil, err
	}
	liteApiClient := _ton.NewAPIClient(client)

	tonApi, err := tonapi.New(tonapi.WithToken("AEXRCJJGQBXCFWQAAAAD3RYTVUWCXT5JW6YN2QU7LHXMKPMOXHFB75P4JSD52AVOVQWPGNY"))

	return &Client{tonApi, liteApiClient}, nil
}

func (a *Client) FetchTransferInput(ctx context.Context, args *types.TransferArgs) (types.TxInput, error) {
	acc, err := a.client.GetAccount(ctx, tonapi.GetAccountParams{
		AccountID: string(args.From),
	})
	if err != nil {
		return nil, err
	}

	seq, err := a.client.GetAccountSeqno(ctx, tonapi.GetAccountSeqnoParams{
		AccountID: string(args.From),
	})
	if err != nil {
		return nil, err
	}

	input := &TxInput{
		Timestamp:       333,
		AccountStatus:   acc.Status,
		TonBalance:      types.NewBigIntFromInt64(acc.GetBalance()),
		Seq:             uint32(seq.Seqno),
		EstimatedMaxFee: types.NewBigIntFromInt64(0), // TODO
		From:            args.From,
		To:              args.To,
		TokenDecimals:   args.TokenDecimals,
		Amount:          args.Amount,
		Memo:            args.Memo,
		ContractAddress: args.ContractAddress,
	}

	if input.ContractAddress != nil {
		fromAddr, _ := address.ParseAddr(string(args.From))
		toAddr, _ := address.ParseAddr(string(args.To))
		contractAddr, _ := address.ParseAddr(string(*args.ContractAddress))
		amountTlb, _ := tlb.FromNano(big.NewInt(1), int(args.TokenDecimals))

		token := jetton.NewJettonMasterClient(a.lclient, contractAddr)
		tokenWallet, err := token.GetJettonWallet(ctx, fromAddr)
		if err != nil {
			return nil, err
		}

		input.TokenWallet = tokenWallet.Address()

		example, err := BuildJettonTransfer(
			10,
			fromAddr,
			input.TokenWallet,
			toAddr,
			amountTlb,
			tlb.MustFromTON("1.0"),
			args.Memo,
		)
		if err != nil {
			return nil, err
		}

		seqnoFetcher := func(ctx context.Context, subWallet uint32) (uint32, error) {
			return input.Seq, nil
		}

		w, err := wallet.FromAddress(ctx, seqnoFetcher, fromAddr, wallet.V4R2)
		if err != nil {
			return nil, err
		}

		cellBuilder, err := w.BuildMessages(ctx, false, []*wallet.Message{example})
		if err != nil {
			return nil, err
		}

		tx := NewTx(fromAddr, cellBuilder, nil)
		sighashes, err := tx.Sighashes()
		if err != nil {
			return nil, err
		}
		if len(sighashes) != 1 {
			return nil, errors.New("invalid sighashes")
		}

		/*
			_, privKey, err := ed25519.GenerateKey(nil)
			if err != nil {
				return nil, err
			}
		*/
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

		res, err := a.client.EmulateMessageToWallet(ctx, &tonapi.EmulateMessageToWalletReq{
			Boc: base64.StdEncoding.EncodeToString(b),
			Params: []tonapi.EmulateMessageToWalletReqParamsItem{
				{
					Address: string(args.From),
				},
			},
		}, tonapi.EmulateMessageToWalletParams{})
		if err != nil {
			return nil, errors.Wrap(err, "could not estimate fee")
		}

		input.EstimatedMaxFee = types.NewBigIntFromInt64(res.Event.Extra * -1)
	}

	return input, nil
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

func (a *Client) GetBalance(ctx context.Context, address types.Address) (*types.BigInt, error) {
	account, err := a.client.GetAccount(ctx, tonapi.GetAccountParams{
		AccountID: string(address),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get balance failed")
	}

	balance := types.NewBigIntFromInt64(account.Balance)
	return &balance, nil
}

func (a *Client) EstimateGas(ctx context.Context, tx types.Tx) (*types.BigInt, error) {
	_tx := tx.(*Tx)

	boc, err := _tx.Serialize()
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

func (a *Client) BroadcastSignedTx(ctx context.Context, _tx types.Tx) error {
	tx := _tx.(*Tx)

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
