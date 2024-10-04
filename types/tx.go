package types

type TxHash []byte

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
