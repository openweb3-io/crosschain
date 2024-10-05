package tx_input

import (
	"github.com/gagliardetto/solana-go"
	xc_types "github.com/openweb3-io/crosschain/types"
)

type StakingInput struct {
	TxInput
	ValidatorVoteAccount solana.PublicKey `json:"validator_vote_account"`
	// The new staking account to create
	StakingKey solana.PrivateKey `json:"staking_key"`
}

var _ xc_types.TxVariantInput = &StakingInput{}
var _ xc_types.StakeTxInput = &StakingInput{}

func (*StakingInput) Staking() {}

func (*StakingInput) GetVariant() xc_types.TxVariantInputType {
	return xc_types.NewStakingInputType(xc_types.BlockchainSolana, string(xc_types.Native))
}

type ExistingStake struct {
	ActivationEpoch   xc_types.BigInt `json:"activation_epoch"`
	DeactivationEpoch xc_types.BigInt `json:"deactivation_epoch"`
	// The total activating-or-activated amount
	AmountActive xc_types.BigInt `json:"amount_active"`
	// unlocked/inactive amount
	AmountInactive xc_types.BigInt `json:"amount_inactive"`
	// ValidatorVoteAccount solana.PublicKey    `json:"validator_vote_account"`
	StakeAccount solana.PublicKey `json:"stake_account"`
}
type UnstakingInput struct {
	TxInput

	// The new staking account to create in the event of a split occuring
	StakingKey     solana.PrivateKey `json:"staking_key"`
	EligibleStakes []*ExistingStake  `json:"eligible_stakes"`
}

var _ xc_types.TxVariantInput = &UnstakingInput{}
var _ xc_types.UnstakeTxInput = &UnstakingInput{}

func (*UnstakingInput) Unstaking() {}

func (*UnstakingInput) GetVariant() xc_types.TxVariantInputType {
	return xc_types.NewUnstakingInputType(xc_types.BlockchainSolana, string(xc_types.Native))
}

type WithdrawInput struct {
	TxInput
	EligibleStakes []*ExistingStake `json:"eligible_stakes"`
}

var _ xc_types.TxVariantInput = &WithdrawInput{}
var _ xc_types.WithdrawTxInput = &WithdrawInput{}

func (*WithdrawInput) Withdrawing() {}

func (*WithdrawInput) GetVariant() xc_types.TxVariantInputType {
	return xc_types.NewWithdrawingInputType(xc_types.BlockchainSolana, string(xc_types.Native))
}
