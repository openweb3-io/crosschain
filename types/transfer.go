package types

type TransferArgs struct {
	From Address
	To   Address
	// Token                string
	TokenDecimals        int32
	Network              string
	Amount               BigInt
	Memo                 string
	ContractAddress      *Address
	Extra                string
	Gas                  *BigInt // 固定设置的Gas
	GasPrice             *BigInt
	FeePayer             *string
	MaxPriorityFeePerGas *BigInt // for ethereum
	MaxFeePerGas         *BigInt // for ethereum
}

func NewTransferArgs() *TransferArgs {
	return &TransferArgs{}
}
