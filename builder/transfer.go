package builder

import "github.com/openweb3-io/crosschain/types"

type TransferArgs struct {
	options builderOptions
	From    types.Address
	To      types.Address
	// Token                string
	TokenDecimals   int32
	Network         string
	Amount          types.BigInt
	Memo            string
	ContractAddress *types.Address
	Extra           string
	Gas             *types.BigInt // 固定设置的Gas
	// GasPrice             *types.BigInt
	FeePayer             *string
	MaxPriorityFeePerGas *types.BigInt // for ethereum
	MaxFeePerGas         *types.BigInt // for ethereum
}

func NewTransferArgs(from types.Address, to types.Address, amount types.BigInt, options ...BuilderOption) (*TransferArgs, error) {
	builderOptions := builderOptions{}
	args := &TransferArgs{
		options: builderOptions,
		From:    from,
		To:      to,
		Amount:  amount,
	}
	for _, opt := range options {
		err := opt(&args.options)
		if err != nil {
			return args, err
		}
	}
	return args, nil
}
