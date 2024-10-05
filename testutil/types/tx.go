package testutil

import (
	xc_types "github.com/openweb3-io/crosschain/types"
)

// An object that only supports .Serialize for SubmitTx()
type MockXcTx struct {
	SerializedSignedTx []byte
	Signatures         []xc_types.TxSignature
}

var _ xc_types.Tx = &MockXcTx{}

func (tx *MockXcTx) Hash() xc_types.TxHash {
	panic("not supported")
}
func (tx *MockXcTx) Sighashes() ([]xc_types.TxDataToSign, error) {
	panic("not supported")
}
func (tx *MockXcTx) AddSignatures(...xc_types.TxSignature) error {
	panic("not supported")
}
func (tx *MockXcTx) GetSignatures() []xc_types.TxSignature {
	return tx.Signatures
}
func (tx *MockXcTx) Serialize() ([]byte, error) {
	return tx.SerializedSignedTx, nil
}
