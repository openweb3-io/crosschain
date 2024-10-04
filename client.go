package crosschain

import (
	"context"

	"github.com/openweb3-io/crosschain/types"
)

type IClient interface {
	/**
	 * get balance
	 */
	GetBalance(ctx context.Context, address types.Address) (*types.BigInt, error)

	/**
	 * estimate gas
	 */
	EstimateGas(ctx context.Context, input types.Tx) (*types.BigInt, error)

	/**
	 * send signed tx
	 */
	BroadcastSignedTx(ctx context.Context, tx types.Tx) error
}
