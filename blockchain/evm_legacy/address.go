package evm_legacy

import (
	evmaddress "github.com/openweb3-io/crosschain/blockchain/evm/address"
	xc "github.com/openweb3-io/crosschain/types"
)

type AddressBuilder = evmaddress.AddressBuilder

var NewAddressBuilder = evmaddress.NewAddressBuilder

var _ xc.AddressBuilder = AddressBuilder{}
