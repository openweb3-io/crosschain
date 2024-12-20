package ton

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/shopspring/decimal"
)

type AccountStatus string

const (
	AccountStatusNonexist AccountStatus = "nonexist"
	AccountStatusUninit   AccountStatus = "uninit"
	AccountStatusActive   AccountStatus = "active"
	AccountStatusFrozen   AccountStatus = "frozen"
)

type TxInput struct {
	AccountStatus   AccountStatus
	Seq             uint32
	PublicKey       []byte `json:"public_key,omitempty"`
	Timestamp       int64
	TokenWallet     xc_types.Address
	EstimatedMaxFee xc_types.BigInt
	TonBalance      xc_types.BigInt
}

func NewTxInput() *TxInput {
	return &TxInput{}
}

func (input *TxInput) GetBlockchain() xc_types.Blockchain {
	return xc_types.BlockchainTon
}

func (input *TxInput) SetPublicKey(pk []byte) error {
	if len(pk) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid ed25519 public key size: %d", len(pk))
	}
	input.PublicKey = pk
	return nil
}

func (input *TxInput) SetPublicKeyFromStr(pkStr string) error {
	pkStr = strings.TrimPrefix(pkStr, "0x")
	pk, err := hex.DecodeString(pkStr)
	if err != nil {
		return fmt.Errorf("invalid hex: %v", err)
	}
	return input.SetPublicKey(pk)
}

func (input *TxInput) SetGasFeePriority(other xc_types.GasFeePriority) error {
	multiplier, err := other.GetDefault()
	if err != nil {
		return err
	}
	// TON doesn't have prioritization fees but we can map it to update the max fee reservation
	multipliedFee := multiplier.Mul(decimal.NewFromBigInt(input.EstimatedMaxFee.Int(), 0)).BigInt()
	input.EstimatedMaxFee = xc_types.BigInt(*multipliedFee)
	return nil
}

func (input *TxInput) IndependentOf(other xc_types.TxInput) (independent bool) {
	// different sequence means independence
	if evmOther, ok := other.(*TxInput); ok {
		return evmOther.Seq != input.Seq
	}
	return
}
func (input *TxInput) SafeFromDoubleSend(others ...xc_types.TxInput) (safe bool) {
	if !xc_types.SameTxInputTypes(input, others...) {
		return false
	}
	// all same sequence means no double send
	for _, other := range others {
		if input.IndependentOf(other) {
			return false
		}
	}
	// sequence all same - we're safe
	return true
}
