package tx_input

import (
	xc "github.com/openweb3-io/crosschain/types"
)

type ExitRequestInput struct {
	TxInput
	PublicKeys [][]byte `json:"public_keys"`
}

var _ xc.TxVariantInput = &ExitRequestInput{}
var _ xc.UnstakeTxInput = &ExitRequestInput{}

func NewExitRequestInput() *ExitRequestInput {
	return &ExitRequestInput{}
}

func (*ExitRequestInput) GetVariant() xc.TxVariantInputType {
	return xc.NewUnstakingInputType(xc.BlockchainEVM, "exit-request")
}

// Mark as valid for un-staking transactions
func (*ExitRequestInput) Unstaking() {}
