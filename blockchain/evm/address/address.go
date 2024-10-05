package address

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	xc_types "github.com/openweb3-io/crosschain/types"
)

// AddressBuilder for EVM
type AddressBuilder struct {
}

// NewAddressBuilder creates a new EVM AddressBuilder
func NewAddressBuilder(asset xc_types.IAsset) (xc_types.AddressBuilder, error) {
	return AddressBuilder{}, nil
}

// GetAddressFromPublicKey returns an Address given a public key
func (ab AddressBuilder) GetAddressFromPublicKey(publicKeyBytes []byte) (xc_types.Address, error) {
	var publicKey *ecdsa.PublicKey
	var err error
	if len(publicKeyBytes) == 33 {
		publicKey, err = crypto.DecompressPubkey(publicKeyBytes)
		if err != nil {
			return xc_types.Address(""), errors.New("invalid k256 public key")
		}
	} else {
		publicKey, err = crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return xc_types.Address(""), err
		}
	}

	address := crypto.PubkeyToAddress(*publicKey).Hex()
	return xc_types.Address(address), nil
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

// FromHex returns a go-ethereum Address decoded Crosschain address (hex string).
func FromHex(address xc_types.Address) (common.Address, error) {
	str := TrimPrefixes(string(address))

	// HexToAddress from go-ethereum doesn't handle any error case
	// We wrap it just in case we need to handle some errors in the future
	return common.HexToAddress(str), nil
}

func TrimPrefixes(addressOrTxHash string) string {
	str := strings.TrimPrefix(addressOrTxHash, "0x")
	str = strings.TrimPrefix(str, "xdc")
	return str
}
func DecodeHex(hexS string) ([]byte, error) {
	return hex.DecodeString(TrimPrefixes(hexS))
}

func Ensure0x(val string) string {
	if !strings.HasPrefix(val, "0x") {
		return "0x" + val
	}
	return val
}