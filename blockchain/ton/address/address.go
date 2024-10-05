package address

import (
	"strings"

	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// TON prescibes using this subwallet for importing compatibility
const DefaultSubwalletId = 698983191

// Most stable TON wallet version
const DefaultWalletVersion = wallet.V3

// AddressBuilder for Template
type AddressBuilder struct {
	Asset xc_types.IAsset
}

var _ xc_types.AddressBuilder = AddressBuilder{}

// NewAddressBuilder creates a new Template AddressBuilder
func NewAddressBuilder(cfgI xc_types.IAsset) (xc_types.AddressBuilder, error) {
	return AddressBuilder{cfgI}, nil
}

// GetAddressFromPublicKey returns an Address given a public key
func (ab AddressBuilder) GetAddressFromPublicKey(publicKeyBytes []byte) (xc_types.Address, error) {
	addr, err := wallet.AddressFromPubKey(publicKeyBytes, DefaultWalletVersion, DefaultSubwalletId)
	if err != nil {
		return "", err
	}
	if ab.Asset.GetChain().Network == "testnet" {
		addr.SetTestnetOnly(true)
	}
	return xc_types.Address(addr.String()), nil
}

// GetAllPossibleAddressesFromPublicKey returns all PossubleAddress(es) given a public key
func (ab AddressBuilder) GetAllPossibleAddressesFromPublicKey(publicKeyBytes []byte) ([]xc_types.PossibleAddress, error) {
	address, err := ab.GetAddressFromPublicKey(publicKeyBytes)
	return []xc_types.PossibleAddress{
		{
			Address: address,
			Type:    xc_types.AddressTypeDefault,
		},
	}, err
}

func ParseAddress(addr xc_types.Address, net string) (*address.Address, error) {
	addrS := string(addr)
	if len(strings.Split(addrS, ":")) == 2 {
		addr, err := address.ParseRawAddr(addrS)
		if err == nil {
			if net == "testnet" {
				addr.SetTestnetOnly(true)
			}
		}
		return addr, err
	}

	return address.ParseAddr(addrS)
}

func Normalize(addressS string) (string, error) {
	addr, err := address.ParseAddr(addressS)
	if err != nil {
		return addressS, err
	}
	return addr.String(), nil
}
