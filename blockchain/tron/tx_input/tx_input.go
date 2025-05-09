package tx_input

import (
	xc_types "github.com/openweb3-io/crosschain/types"
)

type Resource string

const (
	ResourceBandwidth = Resource("BANDWIDTH")
	ResourceEnergy    = Resource("ENERGY")
)

type TxInput struct {
	RefBlockBytes []byte
	RefBlockHash  []byte
	Expiration    int64
	Timestamp     int64
}

func (input *TxInput) GetBlockchain() xc_types.Blockchain {
	return xc_types.BlockchainTron
}

func (input *TxInput) SetGasFeePriority(other xc_types.GasFeePriority) error {
	multiplier, err := other.GetDefault()
	if err != nil {
		return err
	}
	// tron doesn't do prioritization
	_ = multiplier
	return nil
}

func (input *TxInput) IndependentOf(other xc_types.TxInput) (independent bool) {
	// tron uses recent-block-hash like mechanism like solana, but with explicit timestamps
	return true
}
func (input *TxInput) SafeFromDoubleSend(others ...xc_types.TxInput) (safe bool) {
	for _, other := range others {
		oldInput, ok := other.(*TxInput)
		if ok {
			if input.Timestamp <= oldInput.Expiration {
				return false
			}
		} else {
			// can't tell (this shouldn't happen) - default false
			return false
		}
	}
	// all others timed out - we're safe
	return true
}
