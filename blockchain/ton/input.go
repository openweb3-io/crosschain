package ton

import (
	"github.com/openweb3-io/crosschain/types"
	"github.com/shopspring/decimal"
	"github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
)

type TxInput struct {
	Timestamp       int64
	AccountStatus   tonapi.AccountStatus
	Seq             uint32
	EstimatedMaxFee types.BigInt
	TonBalance      types.BigInt
	From            types.Address
	To              types.Address
	// Token                string
	TokenWallet     *address.Address
	TokenDecimals   int32
	Network         string
	Amount          types.BigInt
	Memo            string
	ContractAddress *types.Address
}

func NewTxInput() *TxInput {
	return &TxInput{}
}

func (input *TxInput) GetBlockchain() types.Blockchain {
	return types.BlockchainTon
}

func (input *TxInput) SetGasFeePriority(other types.GasFeePriority) error {
	multiplier, err := other.GetDefault()
	if err != nil {
		return err
	}
	// TON doesn't have prioritization fees but we can map it to update the max fee reservation
	multipliedFee := multiplier.Mul(decimal.NewFromBigInt(input.EstimatedMaxFee.Int(), 0)).BigInt()
	input.EstimatedMaxFee = types.BigInt(*multipliedFee)
	return nil
}
