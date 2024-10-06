package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/address"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/tx"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/tx_input"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/tx_input/gas"
	localcodectypes "github.com/openweb3-io/crosschain/blockchain/cosmos/types"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
)

var DefaultMaxTotalFeeHuman, _ = xc.NewAmountHumanReadableFromStr("2")

// TxBuilder for Cosmos
type TxBuilder struct {
	xcbuilder.TxBuilder
	Chain           *xc.ChainConfig
	CosmosTxConfig  client.TxConfig
	CosmosTxBuilder client.TxBuilder
}

var _ xcbuilder.FullBuilder = &TxBuilder{}

// NewTxBuilder creates a new Cosmos TxBuilder
func NewTxBuilder(chain *xc.ChainConfig) (TxBuilder, error) {
	cosmosCfg := localcodectypes.MakeCosmosConfig()

	return TxBuilder{
		Chain:           chain,
		CosmosTxConfig:  cosmosCfg.TxConfig,
		CosmosTxBuilder: cosmosCfg.TxConfig.NewTxBuilder(),
	}, nil
}

func DefaultMaxGasPrice(nativeAsset *xc.ChainConfig) float64 {
	// Don't spend more than e.g. 2 LUNA on a transaction
	maxFee := DefaultMaxTotalFeeHuman.ToBlockchain(nativeAsset.Decimals)
	return gas.TotalFeeToFeePerGas(maxFee.String(), gas.NativeTransferGasLimit)
}

// Old transfer interface
func (txBuilder TxBuilder) NewTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	txInput := input.(*tx_input.TxInput)
	native := txBuilder.Chain
	max := native.ChainMaxGasPrice
	if max <= 0 {
		max = DefaultMaxGasPrice(native)
	}
	// enforce a maximum gas price
	if txInput.GasPrice > max {
		txInput.GasPrice = max
	}

	// cosmos is unique in that:
	// - the native asset is in one of the native modules, x/bank
	// - x/bank can have multiple assets, all of which can typically pay for gas
	//   - this means cosmos has "multiple" native assets and can add more later, similar to tokens.
	// - there can be other modules with tokens, like cw20 in x/wasm.
	// To abstract this, we detect the module for the asset and rely on that for the transfer types.
	// A native transfer can be a token transfer and vice versa.
	// Right now gas is always paid in the "default" gas coin, set by config.

	// because cosmos assets can be transferred via a number of different modules, we have to rely on txInput
	// to determine which cosmos module we should
	switch txInput.AssetType {
	case tx_input.BANK:
		return txBuilder.NewBankTransfer(args, input)
	case tx_input.CW20:
		return txBuilder.NewCW20Transfer(args, input)
	default:
		return nil, errors.New("unknown cosmos asset type: " + string(txInput.AssetType))
	}
}

// See NewTransfer
func (txBuilder TxBuilder) NewNativeTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	return txBuilder.NewTransfer(args, input)
}

// See NewTransfer
func (txBuilder TxBuilder) NewTokenTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	return txBuilder.NewTransfer(args, input)
}

// x/bank MsgSend transfer
func (txBuilder TxBuilder) NewBankTransfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	txInput := input.(*tx_input.TxInput)
	amountInt := big.Int(args.GetAmount())

	asset, _ := args.GetAsset()
	if asset == nil {
		asset = txBuilder.Chain
	}

	if txInput.GasLimit == 0 {
		txInput.GasLimit = gas.NativeTransferGasLimit
	}

	denom := txBuilder.GetDenom(asset)
	msgSend := &banktypes.MsgSend{
		FromAddress: string(args.GetFrom()),
		ToAddress:   string(args.GetTo()),
		Amount: types.Coins{
			{
				Denom:  denom,
				Amount: math.NewIntFromBigInt(&amountInt),
			},
		},
	}

	fees := txBuilder.calculateFees(asset, args.GetAmount(), txInput, true)
	return txBuilder.createTxWithMsg(txInput, msgSend, txArgs{
		Memo:          txInput.LegacyMemo,
		FromPublicKey: txInput.LegacyFromPublicKey,
	}, fees)
}

func (txBuilder TxBuilder) NewCW20Transfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	txInput := input.(*tx_input.TxInput)

	asset, _ := args.GetAsset()
	if asset == nil {
		asset = txBuilder.Chain
	}

	if txInput.GasLimit == 0 {
		txInput.GasLimit = gas.TokenTransferGasLimit
	}
	contract := asset.GetContract()
	contractTransferMsg := fmt.Sprintf(`{"transfer": {"amount": "%s", "recipient": "%s"}}`, args.GetAmount().String(), args.GetTo())
	msgSend := &wasmtypes.MsgExecuteContract{
		Sender:   string(args.GetFrom()),
		Contract: string(contract),
		Msg:      wasmtypes.RawContractMessage(json.RawMessage(contractTransferMsg)),
	}

	fees := txBuilder.calculateFees(asset, args.GetAmount(), txInput, false)

	return txBuilder.createTxWithMsg(txInput, msgSend, txArgs{
		Memo:          txInput.LegacyMemo,
		FromPublicKey: txInput.LegacyFromPublicKey,
	}, fees)
}

func (txBuilder TxBuilder) GetDenom(asset xc.IAsset) string {
	denom := txBuilder.Chain.ChainCoin
	if asset.GetContract() != "" {
		denom = string(asset.GetContract())
	}
	if token, ok := asset.(*xc.TokenAssetConfig); ok {
		if token.Contract != "" {
			denom = string(token.Contract)
		}
	}

	return denom
}

// Returns the amount in blockchain that is percentage of amount.
// E.g. amount = 100, tax = 0.05, returns 5.
func GetTaxFrom(amount xc.BigInt, tax float64) xc.BigInt {
	if tax > 0.00001 {
		precisionInt := uint64(10000000)
		taxBig := xc.NewBigIntFromUint64(uint64(float64(precisionInt) * tax))
		// some chains may implement a tax (terra classic)
		product := amount.Mul(&taxBig).Int()
		quotiant := product.Div(product, big.NewInt(int64(precisionInt)))
		return xc.NewBigIntFromStr(quotiant.String())
	}
	return xc.NewBigIntFromUint64(0)
}

func (txBuilder TxBuilder) calculateFees(asset xc.IAsset, amount xc.BigInt, input *tx_input.TxInput, includeTax bool) types.Coins {
	gasDenom := txBuilder.Chain.GasCoin
	if gasDenom == "" {
		gasDenom = txBuilder.Chain.ChainCoin
	}
	feeCoins := types.Coins{
		{
			Denom:  gasDenom,
			Amount: math.NewIntFromUint64(uint64(input.GasPrice * float64(input.GasLimit))),
		},
	}
	if includeTax {
		taxRate := txBuilder.Chain.ChainTransferTax
		tax := GetTaxFrom(amount, taxRate)
		if tax.Uint64() > 0 {
			taxDenom := txBuilder.Chain.ChainCoin
			if token, ok := asset.(*xc.TokenAssetConfig); ok && token.Contract != "" {
				taxDenom = string(token.Contract)
			}
			taxStr, _ := math.NewIntFromString(tax.String())
			// cannot add two coins that are the same so must check
			if feeCoins[0].Denom == taxDenom {
				// add to existing
				feeCoins[0].Amount = feeCoins[0].Amount.Add(taxStr)
			} else {
				// add new
				feeCoins = append(feeCoins, types.Coin{
					Denom:  taxDenom,
					Amount: taxStr,
				})
			}
		}
	}
	// Must be sorted or cosmos client panics
	sort.Slice(feeCoins, func(i, j int) bool {
		return feeCoins[i].Denom < feeCoins[j].Denom
	})
	return feeCoins
}

type txArgs struct {
	Memo          string
	FromPublicKey []byte
}

// createTxWithMsg creates a new Tx given Cosmos Msg
func (txBuilder TxBuilder) createTxWithMsg(input *tx_input.TxInput, msg types.Msg, args txArgs, fees types.Coins) (xc.Tx, error) {
	cosmosTxConfig := txBuilder.CosmosTxConfig
	cosmosBuilder := txBuilder.CosmosTxBuilder

	err := cosmosBuilder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}

	cosmosBuilder.SetMemo(args.Memo)
	cosmosBuilder.SetGasLimit(input.GasLimit)

	cosmosBuilder.SetFeeAmount(fees)

	sigMode := signingtypes.SignMode_SIGN_MODE_DIRECT
	sigsV2 := []signingtypes.SignatureV2{
		{
			PubKey: address.GetPublicKey(txBuilder.Chain, args.FromPublicKey),
			Data: &signingtypes.SingleSignatureData{
				SignMode:  sigMode,
				Signature: nil,
			},
			Sequence: input.Sequence,
		},
	}
	err = cosmosBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	chainId := input.ChainId
	if chainId == "" {
		chainId = txBuilder.Chain.ChainIDStr
	}

	signerData := authsigning.SignerData{
		AccountNumber: input.AccountNumber,
		ChainID:       chainId,
		Sequence:      input.Sequence,
	}

	sighashData, err := authsigning.GetSignBytesAdapter(context.Background(), cosmosTxConfig.SignModeHandler(), sigMode, signerData, cosmosBuilder.GetTx())
	// sighashData, err := cosmosTxConfig.SignModeHandler().GetSignBytes(context.Background(), sigMode, signerData, cosmosBuilder.GetTx())
	if err != nil {
		return nil, err
	}
	sighash := tx.GetSighash(txBuilder.Chain, sighashData)
	return &tx.Tx{
		CosmosTx:        cosmosBuilder.GetTx(),
		ParsedTransfers: []types.Msg{msg},
		CosmosTxBuilder: cosmosBuilder,
		CosmosTxEncoder: cosmosTxConfig.TxEncoder(),
		SigsV2:          sigsV2,
		TxDataToSign:    sighash,
	}, nil
}
