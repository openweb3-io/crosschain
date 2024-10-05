package tx_input

import (
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/openweb3-io/crosschain/types"
	"github.com/shopspring/decimal"
)

type TxInput struct {
	From   types.Address
	To     types.Address
	Amount types.BigInt

	RecentBlockHash     solana.Hash      `json:"recent_block_hash,omitempty"`
	ToIsATA             bool             `json:"to_is_ata,omitempty"`
	TokenProgram        solana.PublicKey `json:"token_program"`
	ShouldCreateATA     bool             `json:"should_create_ata,omitempty"`
	SourceTokenAccounts []*TokenAccount  `json:"source_token_accounts,omitempty"`
	PrioritizationFee   types.BigInt     `json:"prioritization_fee,omitempty"`
	Timestamp           int64            `json:"timestamp,omitempty"`
}

func (input *TxInput) GetBlockchain() types.Blockchain {
	return types.BlockchainSolana
}

func (input *TxInput) SetGasFeePriority(other types.GasFeePriority) error {
	multiplier, err := other.GetDefault()
	if err != nil {
		return err
	}
	multipliedFee := multiplier.Mul(decimal.NewFromBigInt(input.PrioritizationFee.Int(), 0)).BigInt()
	input.PrioritizationFee = types.BigInt(*multipliedFee)
	return nil
}

type TokenAccount struct {
	Account solana.PublicKey `json:"account,omitempty"`
	Balance types.BigInt     `json:"balance,omitempty"`
}

// Solana recent-block-hash timeout margin
const SafetyTimeoutMargin = (5 * time.Minute)

// Returns the microlamports to set the compute budget unit price.
// It will not go about the max price amount for safety concerns.
func (input *TxInput) GetLimitedPrioritizationFee(chain *types.ChainConfig) uint64 {
	fee := input.PrioritizationFee.Uint64()
	max := uint64(chain.ChainMaxGasPrice)
	if max == 0 {
		// set default max price to spend max 1 SOL on a transaction
		// 1 SOL = (1 * 10 ** 9) * 10 ** 6 microlamports
		// /200_000 compute units
		max = 5_000_000_000
	}
	if fee > max {
		fee = max
	}
	return fee
}

// NewTxInput returns a new Solana TxInput
func NewTxInput() *TxInput {
	return &TxInput{}
}
