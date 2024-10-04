package solana

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	solana_sdk "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/openweb3-io/crosschain/types"
)

type Tx struct {
	SolTx            *solana_sdk.Transaction
	ParsedSolTx      *rpc.ParsedTransaction
	inputSignatures  []types.TxSignature
	transientSigners []solana.PrivateKey
}

func (tx *Tx) Hash() types.TxHash {
	if tx.SolTx != nil && len(tx.SolTx.Signatures) > 0 {
		sig := tx.SolTx.Signatures[0]
		return types.TxHash(sig.String())
	}
	return types.TxHash("")
}

// Sighashes returns the tx payload to sign, aka sighashes
func (tx Tx) Sighashes() ([]types.TxDataToSign, error) {
	if tx.SolTx == nil {
		return nil, errors.New("transaction not initialized")
	}
	messageContent, err := tx.SolTx.Message.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("unable to encode message for signing: %w", err)
	}
	return []types.TxDataToSign{messageContent}, nil
}

// Some instructions on solana require new accounts to sign the transaction
// in addition to the funding account.  These are transient signers are not
// sensitive and the key material only needs to live long enough to sign the transaction.
func (tx *Tx) AddTransientSigner(transientSigner solana.PrivateKey) {
	tx.transientSigners = append(tx.transientSigners, transientSigner)
}

// AddSignatures adds a signature to Tx
func (tx *Tx) AddSignatures(signatures ...types.TxSignature) error {
	if tx.SolTx == nil {
		return errors.New("transaction not initialized")
	}
	solSignatures := make([]solana.Signature, len(signatures))
	for i, signature := range signatures {
		if len(signature) != solana.SignatureLength {
			return fmt.Errorf("invalid signature (%d): %x", len(signature), signature)
		}
		copy(solSignatures[i][:], signature)
	}
	tx.SolTx.Signatures = solSignatures
	tx.inputSignatures = signatures

	// add transient signers
	for _, transient := range tx.transientSigners {
		bz, _ := tx.SolTx.Message.MarshalBinary()
		sig, err := transient.Sign(bz)
		if err != nil {
			return fmt.Errorf("unable to sign with transient signer: %v", err)
		}
		tx.SolTx.Signatures = append(tx.SolTx.Signatures, sig)
		tx.inputSignatures = append(tx.inputSignatures, sig[:])
	}
	return nil
}

func (tx Tx) GetSignatures() []types.TxSignature {
	return tx.inputSignatures
}

// Serialize returns the serialized tx
func (tx Tx) Serialize() ([]byte, error) {
	if tx.SolTx == nil {
		return []byte{}, errors.New("transaction not initialized")
	}
	return tx.SolTx.MarshalBinary()
}

func NewTxFrom(solTx *solana.Transaction) *Tx {
	tx := &Tx{
		SolTx: solTx,
	}
	return tx
}
