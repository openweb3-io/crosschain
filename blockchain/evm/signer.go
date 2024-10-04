package evm

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/openweb3-io/crosschain/signer"
)

type LocalSigner struct {
	key *ecdsa.PrivateKey
}

func NewLocalSigner(key *ecdsa.PrivateKey) signer.Signer {
	return &LocalSigner{key}
}

func (s *LocalSigner) PublicKey(ctx context.Context) ([]byte, error) {
	pubkey := s.key.Public().(*ecdsa.PublicKey)
	return crypto.FromECDSAPub(pubkey), nil
}

func (s *LocalSigner) SharedKey(theirKey []byte) ([]byte, error) {
	return nil, nil
}

func (s *LocalSigner) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	return crypto.Sign(payload, s.key)
}
