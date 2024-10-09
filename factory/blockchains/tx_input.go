package blockchains

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openweb3-io/crosschain/factory/blockchains/registry"
	xc "github.com/openweb3-io/crosschain/types"
)

const SerializedInputTypeKey = "type"

func MarshalTxInput(methodInput xc.TxInput) ([]byte, error) {
	data := map[string]interface{}{}
	methodBz, err := json.Marshal(methodInput)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(methodBz, &data)
	// force union with method type envelope
	if variant, ok := methodInput.(xc.TxVariantInput); ok {
		data[SerializedInputTypeKey] = variant.GetVariant()
	} else {
		data[SerializedInputTypeKey] = methodInput.GetBlockchain()
	}

	bz, _ := json.Marshal(data)
	return bz, nil
}

// Create a copy of a interface object, to avoid modifying the original
// in the tx-input registry.
func makeCopy[T any](input T) T {
	srcVal := reflect.ValueOf(input)
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}

	newVal := reflect.New(srcVal.Type())

	return newVal.Interface().(T)
}

func NewTxInput(blockchain xc.Blockchain) (xc.TxInput, error) {
	for _, txInput := range registry.GetSupportedBaseTxInputs() {
		if txInput.GetBlockchain() == blockchain {
			return makeCopy(txInput), nil
		}
		// aliases for fork chains
		switch blockchain {
		case xc.BlockchainBtc, xc.BlockchainBtcCash, xc.BlockchainBtcLegacy:
			if txInput.GetBlockchain() == xc.BlockchainBtc {
				return makeCopy(txInput), nil
			}
		case xc.BlockchainCosmos, xc.BlockchainCosmosEvmos:
			if txInput.GetBlockchain() == xc.BlockchainCosmos {
				return makeCopy(txInput), nil
			}
		}
	}

	return nil, fmt.Errorf("no tx-input mapped for driver %s", blockchain)
}

func UnmarshalTxInput(data []byte) (xc.TxInput, error) {
	var env xc.TxInputEnvelope
	buf := []byte(data)
	err := json.Unmarshal(buf, &env)
	if err != nil {
		return nil, err
	}
	input, err := NewTxInput(env.Type)
	if err != nil {
		input2, err2 := NewVariantInput(xc.TxVariantInputType(env.Type))
		if err2 != nil {
			return nil, err
		}
		input = input2
	}
	err = json.Unmarshal(env.TxInput, input)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func MarshalVariantInput(methodInput xc.TxVariantInput) ([]byte, error) {
	data := map[string]interface{}{}
	methodBz, err := json.Marshal(methodInput)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(methodBz, &data)
	// force union with method type envelope
	data[SerializedInputTypeKey] = methodInput.GetVariant()

	bz, _ := json.Marshal(data)
	return bz, nil
}

func NewVariantInput(variantType xc.TxVariantInputType) (xc.TxVariantInput, error) {
	if err := variantType.Validate(); err != nil {
		return nil, err
	}

	for _, variant := range registry.GetSupportedTxVariants() {
		if variant.GetVariant() == variantType {
			return makeCopy(variant), nil
		}
	}

	return nil, fmt.Errorf("no staking-input mapped for %s", variantType)
}

func UnmarshalVariantInput(data []byte) (xc.TxVariantInput, error) {
	type variantInputEnvelope struct {
		Type xc.TxVariantInputType `json:"type"`
	}
	var env variantInputEnvelope
	buf := []byte(data)
	err := json.Unmarshal(buf, &env)
	if err != nil {
		return nil, err
	}
	input, err := NewVariantInput(env.Type)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, input)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func UnmarshalStakingInput(data []byte) (xc.StakeTxInput, error) {
	inp, err := UnmarshalVariantInput(data)
	if err != nil {
		return nil, err
	}
	staking, ok := inp.(xc.StakeTxInput)
	if !ok {
		return staking, fmt.Errorf("not a staking input: %T", inp)
	}
	return staking, nil
}

func UnmarshalUnstakingInput(data []byte) (xc.UnstakeTxInput, error) {
	inp, err := UnmarshalVariantInput(data)
	if err != nil {
		return nil, err
	}
	staking, ok := inp.(xc.UnstakeTxInput)
	if !ok {
		return staking, fmt.Errorf("not an unstaking input: %T", inp)
	}
	return staking, nil
}

func UnmarshalWithdrawingInput(data []byte) (xc.WithdrawTxInput, error) {
	inp, err := UnmarshalVariantInput(data)
	if err != nil {
		return nil, err
	}
	staking, ok := inp.(xc.WithdrawTxInput)
	if !ok {
		return staking, fmt.Errorf("not an unstaking input: %T", inp)
	}
	return staking, nil
}
