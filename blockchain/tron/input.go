package tron

import "github.com/openweb3-io/crosschain/types"

type TxInput struct {
	From            types.Address
	To              types.Address
	Amount          *types.BigInt
	ContractAddress *types.Address

	RefBlockBytes []byte
	RefBlockHash  []byte
	Expiration    int64
	Timestamp     int64
}
