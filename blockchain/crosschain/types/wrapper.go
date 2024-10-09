package types

import (
	xc_types "github.com/openweb3-io/crosschain/types"
)

type TxInputWrapper struct {
	xc_types.TxInputEnvelope
	xc_types.TxInput
}
