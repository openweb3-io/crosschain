package builder

import (
	"github.com/openweb3-io/crosschain/types"
)

type TransferArgs struct {
	options builderOptions
	from    types.Address
	to      types.Address
	amount  types.BigInt

	// ContractAddress *types.Address
	// Network         string
	// Extra           string
	// Gas             *types.BigInt // 固定设置的Gas
	// GasPrice             *types.BigInt
	// FeePayer             *string
	// MaxPriorityFeePerGas *types.BigInt // for ethereum
	// MaxFeePerGas         *types.BigInt // for ethereum
}

func NewTransferArgs(from types.Address, to types.Address, amount types.BigInt, options ...BuilderOption) (*TransferArgs, error) {
	builderOptions := builderOptions{}
	args := &TransferArgs{
		options: builderOptions,
		from:    from,
		to:      to,
		amount:  amount,
	}
	for _, opt := range options {
		err := opt(&args.options)
		if err != nil {
			return args, err
		}
	}
	return args, nil
}

func (args *TransferArgs) SetFrom(v types.Address) {
	args.from = v
}

func (args *TransferArgs) SetTo(v types.Address) {
	args.to = v
}

func (args *TransferArgs) SetAmount(v types.BigInt) { args.amount = v }

func (args *TransferArgs) SetMemo(v string) {
	WithMemo(v)(&args.options)
}

func (args *TransferArgs) SetAsset(asset types.IAsset) {
	WithAsset(asset)(&args.options)
}

func (args *TransferArgs) GetFrom() types.Address {
	return args.from
}

func (args *TransferArgs) GetTo() types.Address {
	return args.to
}

func (args *TransferArgs) GetAmount() types.BigInt { return args.amount }

// Exposed options
func (args *TransferArgs) GetMemo() (string, bool) {
	return args.options.GetMemo()
}

func (args *TransferArgs) GetAsset() (types.IAsset, bool) {
	return args.options.GetAsset()
}

func (args *TransferArgs) GetPublicKey() ([]byte, bool) {
	return args.options.GetPublicKey()
}

func (args *TransferArgs) GetExtra() (map[string]any, bool) {
	return args.options.GetExtra()
}
