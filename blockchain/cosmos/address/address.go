package address

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	xc "github.com/openweb3-io/crosschain/types"
)

// AddressBuilder for Cosmos
type AddressBuilder struct {
	cfg *xc.ChainConfig
}

// NewAddressBuilder creates a new Cosmos AddressBuilder
func NewAddressBuilder(cfg *xc.ChainConfig) (xc.AddressBuilder, error) {
	return AddressBuilder{
		cfg,
	}, nil
}

// GetAddressFromPublicKey returns an Address given a public key
func (ab AddressBuilder) GetAddressFromPublicKey(publicKeyBytes []byte) (xc.Address, error) {
	publicKey := GetPublicKey(ab.cfg, publicKeyBytes)
	rawAddress := publicKey.Address()

	err := sdk.VerifyAddressFormat(rawAddress)
	if err != nil {
		return xc.Address(""), err
	}
	bech32Addr, err := sdk.Bech32ifyAddressBytes(ab.cfg.ChainPrefix, rawAddress)
	return xc.Address(bech32Addr), err
}

// GetAllPossibleAddressesFromPublicKey returns all PossubleAddress(es) given a public key
func (ab AddressBuilder) GetAllPossibleAddressesFromPublicKey(publicKeyBytes []byte) ([]xc.PossibleAddress, error) {
	address, err := ab.GetAddressFromPublicKey(publicKeyBytes)
	return []xc.PossibleAddress{
		{
			Address: address,
			Type:    xc.AddressTypeDefault,
		},
	}, err
}
