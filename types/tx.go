package types

// TxStatus is the status of a tx on chain, currently success or failure.
type TxStatus uint8

// TxStatus values
const (
	TxStatusSuccess TxStatus = 0
	TxStatusFailure TxStatus = 1
)

type TxHash string

type TxSignature []byte

type TxDataToSign []byte

// NewTxSignatures creates a new array of TxSignature, useful to cast [][]byte into []TxSignature
func NewTxSignatures(data [][]byte) []TxSignature {
	ret := make([]TxSignature, len(data))
	for i, sig := range data {
		ret[i] = TxSignature(sig)
	}
	return ret
}

type Tx interface {
	Serialize() ([]byte, error)
	Hash() TxHash
	Sighashes() ([]TxDataToSign, error)
	AddSignatures(...TxSignature) error
	GetSignatures() []TxSignature
}

type TxVariantInput interface {
	TxInput
	GetVariant() TxVariantInputType
}

// Markers for each type of Variant Tx
type StakeTxInput interface {
	TxVariantInput
	Staking()
}
type UnstakeTxInput interface {
	TxVariantInput
	Unstaking()
}
type WithdrawTxInput interface {
	TxVariantInput
	Withdrawing()
}
