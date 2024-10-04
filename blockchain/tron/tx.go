package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/openweb3-io/crosschain/types"
	"google.golang.org/protobuf/proto"
)

type Tx struct {
	tronTx *core.Transaction
	input  *TxInput
}

func (tx *Tx) Serialize() ([]byte, error) {
	return proto.Marshal(tx.tronTx)
}

func (tx *Tx) Hash() types.TxHash {
	hashBase, _ := proto.Marshal(tx.tronTx.RawData)
	digest := sha256.Sum256(hashBase)
	return types.TxHash(hex.EncodeToString(digest[:]))
}

func (tx Tx) Sighashes() ([]types.TxDataToSign, error) {
	rawData, err := proto.Marshal(tx.tronTx.GetRawData())
	if err != nil {
		return nil, errors.New("unable to get raw data")
	}
	hasher := sha256.New()
	hasher.Write(rawData)

	return []types.TxDataToSign{hasher.Sum(nil)}, nil
}

func (tx *Tx) AddSignatures(sigs ...types.TxSignature) error {
	for _, sig := range sigs {
		tx.tronTx.Signature = append(tx.tronTx.Signature, sig)
	}
	return nil
}

func (tx *Tx) GetSignatures() []types.TxSignature {
	sigs := []types.TxSignature{}
	if tx.tronTx != nil {
		for _, sig := range tx.tronTx.Signature {
			sigs = append(sigs, sig)
		}
	}
	return sigs
}
