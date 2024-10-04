package builder

import (
	"github.com/openweb3-io/crosschain/types"
)

// TxBuilder is a Builder that can transfer assets
type TxBuilder interface {
	NewTransfer(from types.Address, to types.Address, amount types.BigInt, input types.TxInput) (types.Tx, error)
}

// TxTokenBuilder is a Builder that can transfer token assets, in addition to native assets
// This interface is soon being removed.
type TxTokenBuilder interface {
	TxBuilder
	NewNativeTransfer(from types.Address, to types.Address, amount types.BigInt, input types.TxInput) (types.Tx, error)
	NewTokenTransfer(from types.Address, to types.Address, amount types.BigInt, input types.TxInput) (types.Tx, error)
}

// TxXTransferBuilder is a Builder that can mutate an asset into another asset
// This interface is soon being removed.
type TxXTransferBuilder interface {
	TxBuilder
	NewTask(from types.Address, to types.Address, amount types.BigInt, input types.TxInput) (types.Tx, error)
}

type FullTransferBuilder interface {
	Transfer
	TxBuilder
}
type FullBuilder interface {
	FullTransferBuilder
	Staking
}

type Transfer interface {
	Transfer(args *TransferArgs, input types.TxInput) (types.Tx, error)
}

type Staking interface {
	Stake(stakingArgs StakeArgs, input types.StakeTxInput) (types.Tx, error)
	Unstake(stakingArgs StakeArgs, input types.UnstakeTxInput) (types.Tx, error)
	Withdraw(stakingArgs StakeArgs, input types.WithdrawTxInput) (types.Tx, error)
}
