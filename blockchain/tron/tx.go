package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	"google.golang.org/protobuf/proto"
)

type Tx struct {
	TronTx *core.Transaction
	Args   *xcbuilder.TransferArgs
}

func (tx *Tx) Serialize() ([]byte, error) {
	return proto.Marshal(tx.TronTx)
}

func (tx *Tx) Hash() types.TxHash {
	hashBase, _ := proto.Marshal(tx.TronTx.RawData)
	digest := sha256.Sum256(hashBase)
	return types.TxHash(hex.EncodeToString(digest[:]))
}

func (tx Tx) Sighashes() ([]types.TxDataToSign, error) {
	rawData, err := proto.Marshal(tx.TronTx.GetRawData())
	if err != nil {
		return nil, errors.New("unable to get raw data")
	}
	hasher := sha256.New()
	hasher.Write(rawData)

	return []types.TxDataToSign{hasher.Sum(nil)}, nil
}

func (tx *Tx) AddSignatures(sigs ...types.TxSignature) error {
	for _, sig := range sigs {
		tx.TronTx.Signature = append(tx.TronTx.Signature, sig)
	}
	return nil
}

func (tx *Tx) GetSignatures() []types.TxSignature {
	sigs := []types.TxSignature{}
	if tx.TronTx != nil {
		for _, sig := range tx.TronTx.Signature {
			sigs = append(sigs, sig)
		}
	}
	return sigs
}
