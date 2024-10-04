package ton

import (
	"github.com/openweb3-io/crosschain/types"
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
