package validation

import (
	"fmt"

	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/shopspring/decimal"
)

func Count32EthChunks(amount xc_types.BigInt) (uint64, error) {
	ethInc, _ := xc_types.NewAmountHumanReadableFromStr("32")
	decimals := int32(18)

	weiInc := ethInc.ToBlockchain(decimals)

	if amount.Cmp(&weiInc) < 0 {
		return 0, fmt.Errorf("must stake at least 32 ether")
	}
	amountHuman := amount.ToHuman(decimals)

	quot := amountHuman.Div(ethInc)
	rounded := (decimal.Decimal)(quot).Round(0)
	if quot.String() != rounded.String() {
		return 0, fmt.Errorf("must stake an increment of 32 ether")
	}
	return quot.ToBlockchain(0).Uint64(), nil
}
