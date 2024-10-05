package tron

import (
	"errors"
	"fmt"
	"time"

	"github.com/decred/base58"
	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/openweb3-io/crosschain/types"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/anypb"
)

type TxBuilder struct {
	Chain *types.ChainConfig
}

func NewTxBuilder(chain *types.ChainConfig) *TxBuilder {
	return &TxBuilder{
		Chain: chain,
	}
}

func (b *TxBuilder) BuildTransfer(input types.TxInput) (types.Tx, error) {
	txInput := input.(*TxInput)

	asset, _ := txInput.Args.GetAsset()
	if asset == nil || asset.GetContract() == "" {
		return b.BuildNativeTransfer(input)
	} else {
		return b.BuildTokenTransfer(input)
	}
}

func (b *TxBuilder) BuildNativeTransfer(input types.TxInput) (types.Tx, error) {
	txInput := input.(*TxInput)
	args := txInput.Args

	from_bytes, err := common.DecodeCheck(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}
	to_bytes, err := common.DecodeCheck(string(args.GetTo()))
	if err != nil {
		return nil, err
	}

	params := &core.TransferContract{}
	params.Amount = args.GetAmount().Int().Int64()
	params.OwnerAddress = from_bytes
	params.ToAddress = to_bytes

	parameter, err := anypb.New(params)
	if err != nil {
		return nil, err
	}

	contract := &core.Transaction_Contract{
		Type:      core.Transaction_Contract_TransferContract,
		Parameter: parameter,
	}

	tx := new(core.Transaction)
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: txInput.RefBlockBytes,
		RefBlockHash:  txInput.RefBlockHash,
		// tron wants milliseconds
		Expiration: time.Unix(txInput.Expiration, 0).UnixMilli(),
		Timestamp:  time.Unix(txInput.Timestamp, 0).UnixMilli(),

		// unused ?
		RefBlockNum: 0,
	}

	return &Tx{
		tronTx: tx,
		input:  txInput,
	}, nil
}

func (b *TxBuilder) BuildTokenTransfer(input types.TxInput) (types.Tx, error) {
	txInput := input.(*TxInput)
	txArgs := txInput.Args
	asset, _ := txArgs.GetAsset()
	if asset == nil {
		return nil, errors.New("asset needed")
	}

	from_bytes, _, err := base58.CheckDecode(string(txArgs.GetFrom()))
	if err != nil {
		return nil, err
	}

	to_bytes, _, err := base58.CheckDecode(string(txArgs.GetTo()))
	if err != nil {
		return nil, err
	}

	contract_bytes, _, err := base58.CheckDecode(string(asset.GetContract()))
	if err != nil {
		return nil, err
	}

	addrType, err := eABI.NewType("address", "", nil)
	if err != nil {
		return nil, fmt.Errorf("internal type construction error: %v", err)
	}
	amountType, err := eABI.NewType("address", "", nil)
	if err != nil {
		return nil, fmt.Errorf("internal type construction error: %v", err)
	}
	args := eABI.Arguments{
		{
			Type: addrType,
		},
		{
			Type: amountType,
		},
	}

	paramBz, err := args.PackValues([]interface{}{
		eth_common.BytesToAddress(to_bytes),
		txArgs.GetAmount().Int(),
	})
	methodSig := Signature("transfer(address,uint256)")
	data := append(methodSig, paramBz...)

	if err != nil {
		return nil, err
	}

	params := &core.TriggerSmartContract{}
	params.ContractAddress = contract_bytes
	params.Data = data
	params.OwnerAddress = from_bytes
	params.CallValue = 0

	contract := &core.Transaction_Contract{}
	contract.Type = core.Transaction_Contract_TriggerSmartContract
	param, err := anypb.New(params)
	if err != nil {
		return nil, err
	}
	contract.Parameter = param

	tx := &core.Transaction{}
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: txInput.RefBlockBytes,
		RefBlockHash:  txInput.RefBlockHash,
		// tron wants milliseconds
		Expiration: time.Unix(txInput.Expiration, 0).UnixMilli(),
		Timestamp:  time.Unix(txInput.Timestamp, 0).UnixMilli(),

		// unused ?
		RefBlockNum: 0,
	}

	// set limit for token contracts
	// maxPrice := int64(b.Asset.GetChain().ChainMaxGasPrice)
	maxPrice := int64(b.Chain.ChainMaxGasPrice)
	tx.RawData.FeeLimit = maxPrice
	if tx.RawData.FeeLimit == 0 {
		// 2k tron sanity limit
		tx.RawData.FeeLimit = 2000000000
	}

	return &Tx{
		tronTx: tx,
		input:  txInput,
	}, nil
}

func Signature(method string) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(method))
	b := hasher.Sum(nil)
	return b[:4]
}
