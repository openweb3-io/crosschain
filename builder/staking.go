package builder

import (
	"fmt"

	"github.com/openweb3-io/crosschain/builder/validation"
	xc_types "github.com/openweb3-io/crosschain/types"
)

type StakeArgs struct {
	options builderOptions
	from    xc_types.Address
	amount  xc_types.BigInt
}

var _ TransactionOptions = &StakeArgs{}

// Staking arguments
func (args *StakeArgs) GetFrom() xc_types.Address  { return args.from }
func (args *StakeArgs) GetAmount() xc_types.BigInt { return args.amount }

// Exposed options
func (args *StakeArgs) GetMemo() (string, bool)     { return args.options.GetMemo() }
func (args *StakeArgs) GetTimestamp() (int64, bool) { return args.options.GetTimestamp() }
func (args *StakeArgs) GetPriority() (xc_types.GasFeePriority, bool) {
	return args.options.GetPriority()
}
func (args *StakeArgs) GetPublicKey() ([]byte, bool) { return args.options.GetPublicKey() }

// Staking options
func (args *StakeArgs) GetValidator() (string, bool)            { return args.options.GetValidator() }
func (args *StakeArgs) GetStakeOwner() (xc_types.Address, bool) { return args.options.GetStakeOwner() }
func (args *StakeArgs) GetStakeAccount() (string, bool)         { return args.options.GetStakeAccount() }

func NewStakeArgs(chain xc_types.NativeAsset, from xc_types.Address, amount xc_types.BigInt, options ...BuilderOption) (StakeArgs, error) {
	builderOptions := builderOptions{}
	args := StakeArgs{
		builderOptions,
		from,
		amount,
	}
	for _, opt := range options {
		err := opt(&args.options)
		if err != nil {
			return args, err
		}
	}

	// Chain specific validation of arguments
	switch chain.Blockchain() {
	case xc_types.BlockchainEVM:
		// Eth must stake or unstake in increments of 32
		_, err := validation.Count32EthChunks(args.GetAmount())
		if err != nil {
			return args, err
		}
	case xc_types.BlockchainCosmos, xc_types.BlockchainSolana:
		if _, ok := args.GetValidator(); !ok {
			return args, fmt.Errorf("validator to be delegated to is required for %s chain", chain)
		}
	}

	return args, nil
}
