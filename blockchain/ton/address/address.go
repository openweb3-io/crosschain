package address

import (
	"github.com/xssnick/tonutils-go/address"
)

func Normalize(addressS string) (string, error) {
	addr, err := address.ParseAddr(addressS)
	if err != nil {
		return addressS, err
	}
	return addr.String(), nil
}
