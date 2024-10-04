package signer

import "context"

type SignerProvider interface {
	Register(network string, creator SignerCreator)
	Provide(ctx context.Context, appId, network, key string) (Signer, error)
}
