package registry

import (
	"fmt"

	xc "github.com/openweb3-io/crosschain/types"
)

var supportedBaseInputTx = []xc.TxInput{}
var supportedVariantTx = []xc.TxVariantInput{}

func RegisterTxBaseInput(txInput xc.TxInput) {
	for _, existing := range supportedBaseInputTx {
		if existing.GetBlockchain() == txInput.GetBlockchain() {
			panic(fmt.Sprintf("base input %T blockchain %s duplicates %T", txInput, txInput.GetBlockchain(), existing))
		}
	}
	supportedBaseInputTx = append(supportedBaseInputTx, txInput)
}

func GetSupportedBaseTxInputs() []xc.TxInput {
	return supportedBaseInputTx
}
func RegisterTxVariantInput(variant xc.TxVariantInput) {
	for _, existing := range supportedVariantTx {
		if existing.GetVariant() == variant.GetVariant() {
			panic(fmt.Sprintf("staking input %T blockchain %s duplicates %T", variant, variant.GetVariant(), existing))
		}
	}
	i1, ok1 := variant.(xc.StakeTxInput)
	i2, ok2 := variant.(xc.UnstakeTxInput)
	i3, ok3 := variant.(xc.WithdrawTxInput)
	if !ok1 && !ok2 && !ok3 {
		panic(fmt.Sprintf("staking input %T must implement one of %T, %T, %T", variant, i1, i2, i3))
	}

	supportedVariantTx = append(supportedVariantTx, variant)
}

func GetSupportedTxVariants() []xc.TxVariantInput {
	return supportedVariantTx
}