package client

import (
	"context"

	"github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	xc_types "github.com/openweb3-io/crosschain/types"
)

type IClient interface {
	// Fetch the basic transaction input for any new transaction
	FetchLegacyTxInput(ctx context.Context, from xc_types.Address, to xc_types.Address) (xc_types.TxInput, error)

	// Fetching transaction info - legacy endpoint
	FetchLegacyTxInfo(ctx context.Context, txHash xc_types.TxHash) (*xc_types.LegacyTxInfo, error)

	/**
	 * get balance
	 */
	FetchBalance(ctx context.Context, address xc_types.Address) (*xc_types.BigInt, error)

	FetchBalanceForAsset(ctx context.Context, address xc_types.Address, contractAddress xc_types.ContractAddress) (*xc_types.BigInt, error)

	/**
	 * estimate gas
	 */
	EstimateGas(ctx context.Context, input xc_types.Tx) (*xc_types.BigInt, error)

	/**
	 * send signed tx
	 */
	BroadcastTx(ctx context.Context, tx xc_types.Tx) error

	FetchTransferInput(ctx context.Context, args *builder.TransferArgs) (xc_types.TxInput, error)
}

type StakingClient interface {
	// Fetch staked balances accross different possible states
	FetchStakeBalance(ctx context.Context, args StakedBalanceArgs) ([]*StakedBalance, error)

	// Fetch inputs required for a staking transaction
	FetchStakingInput(ctx context.Context, args builder.StakeArgs) (xc_types.StakeTxInput, error)

	// Fetch inputs required for a unstaking transaction
	FetchUnstakingInput(ctx context.Context, args builder.StakeArgs) (xc_types.UnstakeTxInput, error)

	// Fetch input for a withdraw transaction -- not all chains use this as they combine it with unstake
	FetchWithdrawInput(ctx context.Context, args builder.StakeArgs) (xc_types.WithdrawTxInput, error)
}

// Special 3rd-party interface for Ethereum as ethereum doesn't understand delegated staking
type ManualUnstakingClient interface {
	CompleteManualUnstaking(ctx context.Context, unstake *Unstake) error
}

type ClientError string

// A transaction terminally failed due to no balance
const NoBalance ClientError = "NoBalance"

// A transaction terminally failed due to no balance after accounting for gas cost
const NoBalanceForGas ClientError = "NoBalanceForGas"

// A transaction terminally failed due to another reason
const TransactionFailure ClientError = "TransactionFailure"

// A transaction failed to submit because it already exists
const TransactionExists ClientError = "TransactionExists"

// deadline exceeded and transaction can no longer be accepted
const TransactionTimedOut ClientError = "TransactionTimedOut"

// A network error occured -- there may be nothing wrong with the transaction
const NetworkError ClientError = "NetworkError"

// No outcome for this error known
const UnknownError ClientError = "UnknownError"

type State string

var Activating State = "activating"
var Active State = "active"
var Deactivating State = "deactivating"
var Inactive State = "inactive"

type StakedBalanceState struct {
	Active       types.BigInt `json:"active,omitempty"`
	Activating   types.BigInt `json:"activating,omitempty"`
	Deactivating types.BigInt `json:"deactivating,omitempty"`
	Inactive     types.BigInt `json:"inactive,omitempty"`
}

type StakedBalance struct {
	// the validator that the stake is delegated to
	Validator string `json:"validator"`
	// Optional; the account that the stake is associated with
	Account string `json:"account,omitempty"`
	// The states balance of the balance in the validator [+account]
	Balance StakedBalanceState `json:"balance"`
}

func NewStakedBalances(balances StakedBalanceState, validator, account string) *StakedBalance {
	return &StakedBalance{
		Validator: validator,
		Account:   account,
		Balance:   balances,
	}
}

func NewStakedBalance(balance types.BigInt, state State, validator, account string) *StakedBalance {
	balances := StakedBalanceState{}
	switch state {
	case Activating:
		balances.Activating = balance
	case Active:
		balances.Active = balance
	case Deactivating:
		balances.Deactivating = balance
	case Inactive:
		balances.Inactive = balance
	}
	return &StakedBalance{
		Validator: validator,
		Account:   account,
		Balance:   balances,
	}
}
