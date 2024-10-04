package solana

import "github.com/openweb3-io/crosschain/types"

type TxInput struct {
	From            types.Address
	To              types.Address
	ContractAddress *types.Address
}
