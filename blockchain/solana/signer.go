package solana

import (
	"context"
	"crypto"
	"crypto/ed25519"
	crypto_rand "crypto/rand"
	"errors"

	"github.com/openweb3-io/crosschain/signer"
	"github.com/openweb3-io/crosschain/types"
)

type Signature [64]byte

type LocalSigner struct {
	key ed25519.PrivateKey
}

func NewLocalSigner(key ed25519.PrivateKey) signer.Signer {
	return &LocalSigner{
		key: key,
	}
}

func (s *LocalSigner) PublicKey(ctx context.Context) ([]byte, error) {
	return s.key.Public().(ed25519.PublicKey), nil
}

func (s *LocalSigner) SharedKey(theirKey []byte) ([]byte, error) {
	return nil, errors.New("shared key is not supported in Solana")
}

func (s *LocalSigner) Sign(payload types.TxDataToSign) (types.TxSignature, error) {
	p := ed25519.PrivateKey(s.key)
	signData, err := p.Sign(crypto_rand.Reader, payload, crypto.Hash(0))
	if err != nil {
		return nil, err
	}

	var signature Signature
	copy(signature[:], signData)

	return signature[:], nil
}
