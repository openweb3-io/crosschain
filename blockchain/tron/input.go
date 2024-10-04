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

func (input *TxInput) GetBlockchain() types.Blockchain {
	return types.BlockchainTron
}

func (input *TxInput) SetGasFeePriority(other types.GasFeePriority) error {
	multiplier, err := other.GetDefault()
	if err != nil {
		return err
	}
	// tron doesn't do prioritization
	_ = multiplier
	return nil
}
