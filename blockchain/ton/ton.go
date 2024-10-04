package ton

import (
	"github.com/xssnick/tonutils-go/ton"
)

type TonApi struct {
	client ton.APIClientWrapped
}

func NewTonApi(client ton.APIClientWrapped) *TonApi {
	return &TonApi{client}
}

/*
func (a *TonApi) Transfer(ctx context.Context, input *types.TransferArgs) error {
	dstAddr, err := address.ParseAddr(input.To)
	if err != nil {
		return err
	}

	w, err := wallet.FromSigner(ctx, a.client, input.Signer, wallet.V4R2)
	if err != nil {
		return err
	}

	tx, _, inMsgHash, err := w.Transfer(ctx, dstAddr, tlb.FromNanoTON(input.Amount), input.Memo, true)
	if err != nil {
		return err
	}
	fmt.Printf("tx hash: %s\n", hex.EncodeToString(tx.Hash))
	fmt.Printf("tx inMsgHash: %s\n", hex.EncodeToString(tx.IO.In.AsExternalIn().Body.Hash()))
	fmt.Printf("inMsgHash: %s\n", hex.EncodeToString(inMsgHash))

	return nil
}

*/
