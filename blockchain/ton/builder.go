package ton

import (
	"context"
	"fmt"

	tonaddress "github.com/openweb3-io/crosschain/blockchain/ton/address"
	"github.com/openweb3-io/crosschain/blockchain/ton/tx"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var Zero = xc_types.NewBigIntFromInt64(0)

type TxBuilder struct {
	chain *xc_types.ChainConfig
}

func NewTxBuilder(chain *xc_types.ChainConfig) (*TxBuilder, error) {
	return &TxBuilder{
		chain: chain,
	}, nil
}

func (b *TxBuilder) NewTransfer(args *xcbuilder.TransferArgs, input xc_types.TxInput) (xc_types.Tx, error) {
	ctx := context.Background()

	txInput := input.(*TxInput)

	toAddr, err := address.ParseAddr(string(args.GetTo()))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid TON to address: %s", args.GetTo())
	}
	// TODO 应该在外部传入地址的时候决定 bounce
	toAddr = toAddr.Bounce(false)
	fromAddr, err := address.ParseAddr(string(args.GetFrom()))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid TON address %s", args.GetFrom())
	}

	var message *wallet.Message

	asset, _ := args.GetAsset()
	memo, _ := args.GetMemo()

	if asset != nil && asset.GetContract() != "" {
		tokenAddr, err := tonaddress.ParseAddress(txInput.TokenWallet, "")
		if err != nil {
			return nil, fmt.Errorf("invalid TON token address %s: %v", txInput.TokenWallet, err)
		}

		amountTlb, err := tlb.FromNano(args.GetAmount().Int(), int(asset.GetDecimals()))
		if err != nil {
			return nil, err
		}

		// Spend max 0.2 TON per Jetton transfer.  If we don't have 0.05 TON, we should
		// lower the max to our balance less max-fees.
		maxJettonFee := xc_types.NewBigIntFromInt64(50000000)
		remainingTonBal := txInput.TonBalance.Sub(&txInput.EstimatedMaxFee)
		if maxJettonFee.Cmp(&remainingTonBal) > 0 && remainingTonBal.Cmp(&Zero) > 0 {
			maxJettonFee = remainingTonBal
		}

		message, err = BuildJettonTransfer(
			uint64(txInput.Timestamp),
			fromAddr,
			tokenAddr,
			toAddr,
			amountTlb,
			tlb.FromNanoTON(maxJettonFee.Int()),
			memo,
		)
		if err != nil {
			return nil, err
		}
	} else {
		message, err = BuildTransfer(toAddr, tlb.FromNanoTON(args.GetAmount().Int()), memo)
		if err != nil {
			return nil, errors.Wrap(err, "BuildTransfer failed")
		}
	}

	seqnoFetcher := func(ctx context.Context, subWallet uint32) (uint32, error) {
		return txInput.Seq, nil
	}

	w, err := wallet.FromAddress(seqnoFetcher, fromAddr, wallet.V4R2)
	if err != nil {
		return nil, err
	}

	// initialized := acc.IsActive && acc.State.Status == tlb.AccountStatusActive
	cellBuilder, err := w.BuildMessages(ctx, false, []*wallet.Message{message})
	if err != nil {
		return nil, err
	}

	return tx.NewTx(fromAddr, cellBuilder, nil), nil
}

func BuildTransfer(
	to *address.Address,
	amount tlb.Coins,
	comment string,
) (_ *wallet.Message, err error) {
	var body *cell.Cell
	if comment != "" {
		body, err = wallet.CreateCommentCell(comment)
		if err != nil {
			return nil, err
		}
	}

	return wallet.SimpleMessageAutoBounce(to, amount, body), nil
}

func BuildJettonTransfer(
	randomInt uint64,
	from *address.Address,
	jettonWalletAddress *address.Address,
	to *address.Address,
	amount tlb.Coins,
	maxFee tlb.Coins,
	comment string,
) (_ *wallet.Message, err error) {
	var body *cell.Cell
	if comment != "" {
		body, err = wallet.CreateCommentCell(comment)
		if err != nil {
			return nil, err
		}
	}

	amountForwardTON := tlb.MustFromTON("0.01")

	tokenBody, err := tlb.ToCell(jetton.TransferPayload{
		QueryID:             randomInt,
		Amount:              amount,
		Destination:         to,
		ResponseDestination: from,
		CustomPayload:       nil,
		ForwardTONAmount:    amountForwardTON, // tlb.ZeroCoins,
		ForwardPayload:      body,
	})
	if err != nil {
		return nil, nil
	}

	return wallet.SimpleMessage(jettonWalletAddress, maxFee, tokenBody), nil
}

func ParseComment(body *cell.Cell) (string, bool) {
	if body != nil {
		l := body.BeginParse()
		if val, err := l.LoadUInt(32); err == nil && val == 0 {
			str, _ := l.LoadStringSnake()
			return str, true
		}
	}
	return "", false
}
