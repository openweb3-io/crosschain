package ton

import (
	"context"

	"github.com/openweb3-io/crosschain/blockchain/ton/tx"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	"github.com/openweb3-io/crosschain/types"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var Zero = types.NewBigIntFromInt64(0)

type TxBuilder struct {
}

func NewTxBuilder() *TxBuilder {
	return &TxBuilder{}
}

func (b *TxBuilder) BuildTransaction(ctx context.Context, input types.TxInput) (types.Tx, error) {
	txInput := input.(*TxInput)

	toAddr, err := address.ParseAddr(string(txInput.To))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid TON to address: %s", txInput.To)
	}
	// TODO 应该在外部传入地址的时候决定 bounce
	toAddr = toAddr.Bounce(false)
	fromAddr, err := address.ParseAddr(string(txInput.From))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid TON address %s", txInput.From)
	}

	var message *wallet.Message
	if txInput.ContractAddress != nil {
		amountTlb, err := tlb.FromNano(txInput.Amount.Int(), int(txInput.TokenDecimals))
		if err != nil {
			return nil, err
		}

		// Spend max 0.2 TON per Jetton transfer.  If we don't have 0.05 TON, we should
		// lower the max to our balance less max-fees.
		maxJettonFee := types.NewBigIntFromInt64(50000000)
		remainingTonBal := txInput.TonBalance.Sub(&txInput.EstimatedMaxFee)
		if maxJettonFee.Cmp(&remainingTonBal) > 0 && remainingTonBal.Cmp(&Zero) > 0 {
			maxJettonFee = remainingTonBal
		}

		message, err = BuildJettonTransfer(
			uint64(txInput.Timestamp),
			fromAddr,
			txInput.TokenWallet,
			toAddr,
			amountTlb,
			tlb.FromNanoTON(maxJettonFee.Int()),
			txInput.Memo,
		)
		if err != nil {
			return nil, err
		}
	} else {
		message, err = BuildTransfer(toAddr, tlb.FromNanoTON(txInput.Amount.Int()), txInput.Memo)
		if err != nil {
			return nil, errors.Wrap(err, "BuildTransfer failed")
		}
	}

	seqnoFetcher := func(ctx context.Context, subWallet uint32) (uint32, error) {
		return txInput.Seq, nil
	}

	w, err := wallet.FromAddress(ctx, seqnoFetcher, fromAddr, wallet.V4R2)
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
	tokenWallet *address.Address,
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

	return wallet.SimpleMessage(tokenWallet, maxFee, tokenBody), nil
}
