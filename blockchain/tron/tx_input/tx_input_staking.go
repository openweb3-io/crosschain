package tx_input

import (
	xc_types "github.com/openweb3-io/crosschain/types"
)

type StakingInput struct {
	TxInput
	Resource Resource         `json:"resource"`
	Amount   *xc_types.BigInt `json:"amount"`
}

var _ xc_types.TxVariantInput = &StakingInput{}
var _ xc_types.StakeTxInput = &StakingInput{}

func (*StakingInput) Staking() {}

func (*StakingInput) GetVariant() xc_types.TxVariantInputType {
	return xc_types.NewStakingInputType(xc_types.BlockchainTron, string(xc_types.Native))
}

type UnstakingInput struct {
	TxInput
	Resource Resource         `json:"resource"`
	Amount   *xc_types.BigInt `json:"amount"`
}

var _ xc_types.TxVariantInput = &UnstakingInput{}
var _ xc_types.UnstakeTxInput = &UnstakingInput{}

func (*UnstakingInput) Unstaking() {}

func (*UnstakingInput) GetVariant() xc_types.TxVariantInputType {
	return xc_types.NewUnstakingInputType(xc_types.BlockchainTron, string(xc_types.Native))
}

type WithdrawInput struct {
	TxInput
}

var _ xc_types.TxVariantInput = &WithdrawInput{}
var _ xc_types.WithdrawTxInput = &WithdrawInput{}

func (*WithdrawInput) Withdrawing() {}

func (*WithdrawInput) GetVariant() xc_types.TxVariantInputType {
	return xc_types.NewWithdrawingInputType(xc_types.BlockchainTron, string(xc_types.Native))
}
