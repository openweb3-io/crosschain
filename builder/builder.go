package builder

import (
	"github.com/openweb3-io/crosschain/types"
)

// TxBuilder is a Builder that can transfer assets
type TxBuilder interface {
	NewTransfer(args *TransferArgs, input types.TxInput) (types.Tx, error)
}

// TxTokenBuilder is a Builder that can transfer token assets, in addition to native assets
// This interface is soon being removed.
type TxTokenBuilder interface {
	TxBuilder
	NewNativeTransfer(args *TransferArgs, input types.TxInput) (types.Tx, error)
	NewTokenTransfer(args *TransferArgs, input types.TxInput) (types.Tx, error)
}

// TxXTransferBuilder is a Builder that can mutate an asset into another asset
// This interface is soon being removed.
type TxXTransferBuilder interface {
	TxBuilder
	NewTask(args *TransferArgs, input types.TxInput) (types.Tx, error)
}

type FullBuilder interface {
	TxBuilder
	Staking
}

type Staking interface {
	Stake(stakingArgs StakeArgs, input types.StakeTxInput) (types.Tx, error)
	Unstake(stakingArgs StakeArgs, input types.UnstakeTxInput) (types.Tx, error)
	Withdraw(stakingArgs StakeArgs, input types.WithdrawTxInput) (types.Tx, error)
}
