package builder

import (
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/tx"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
)

var _ xcbuilder.TxXTransferBuilder = &TxBuilder{}

func (txBuilder TxBuilder) NewTask(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	txInput := input.(*tx_input.TxInput)
	asset := txInput.Asset.(*xc.TaskConfig)
	amountInt := big.Int(amount)
	amountCoin := types.Coin{
		Denom:  txBuilder.GetDenom(asset),
		Amount: math.NewIntFromBigInt(&amountInt),
	}

	if strings.HasPrefix(asset.Code, "CosmosUndelegateOperator") {
		validatorAddress, ok := asset.DefaultParams["validator_address"]
		if !ok {
			return &tx.Tx{}, fmt.Errorf("must provide validator_address in task '%s'", asset.ID())
		}
		msgUndelegate := &stakingtypes.MsgUndelegate{
			DelegatorAddress: string(from),
			Amount:           amountCoin,
			ValidatorAddress: fmt.Sprintf("%s", validatorAddress),
		}

		fees := txBuilder.calculateFees(amount, txInput, false)
		return txBuilder.createTxWithMsg(txInput, msgUndelegate, txArgs{
			Memo:          txInput.LegacyMemo,
			FromPublicKey: txInput.LegacyFromPublicKey,
		}, fees)
	}

	return &tx.Tx{}, fmt.Errorf("not implemented task: '%s'", asset.ID())
}
