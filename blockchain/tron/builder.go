package tron

import (
	"errors"
	"fmt"
	"time"

	"github.com/openweb3-io/crosschain/blockchain/tron/tx_input"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/anypb"
)

type TxBuilder struct {
	Chain *types.ChainConfig
}

func NewTxBuilder(chain *types.ChainConfig) (*TxBuilder, error) {
	return &TxBuilder{
		Chain: chain,
	}, nil
}

func (b *TxBuilder) NewTransfer(args *xcbuilder.TransferArgs, input types.TxInput) (types.Tx, error) {
	asset, _ := args.GetAsset()
	if asset == nil || asset.GetContract() == "" {
		return b.NewNativeTransfer(args, input)
	} else {
		return b.NewTokenTransfer(args, input)
	}
}

func (b *TxBuilder) NewNativeTransfer(args *xcbuilder.TransferArgs, input types.TxInput) (types.Tx, error) {
	txInput := input.(*tx_input.TxInput)

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
		TronTx: tx,
		Args:   args,
	}, nil
}

func (b *TxBuilder) NewTokenTransfer(args *xcbuilder.TransferArgs, input types.TxInput) (types.Tx, error) {
	txInput := input.(*tx_input.TxInput)
	asset, _ := args.GetAsset()
	if asset == nil {
		return nil, errors.New("asset needed")
	}

	from_bytes, err := common.DecodeCheck(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	to_bytes, err := common.DecodeCheck(string(args.GetTo()))
	if err != nil {
		return nil, err
	}

	contract_bytes, err := common.DecodeCheck(string(asset.GetContract()))
	if err != nil {
		return nil, err
	}

	addrType, err := eABI.NewType("address", "", nil)
	if err != nil {
		return nil, fmt.Errorf("internal type construction error: %v", err)
	}
	amountType, err := eABI.NewType("uint256", "", nil)
	if err != nil {
		return nil, fmt.Errorf("internal type construction error: %v", err)
	}
	txArgs := eABI.Arguments{
		{
			Type: addrType,
		},
		{
			Type: amountType,
		},
	}

	paramBz, err := txArgs.PackValues([]interface{}{
		eth_common.BytesToAddress(to_bytes),
		args.GetAmount().Int(),
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
		TronTx: tx,
		Args:   args,
	}, nil
}

func Signature(method string) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(method))
	b := hasher.Sum(nil)
	return b[:4]
}

func (b *TxBuilder) Stake(args xcbuilder.StakeArgs, input types.StakeTxInput) (types.Tx, error) {
	stakeInput, ok := input.(*tx_input.StakingInput)
	if !ok {
		return nil, fmt.Errorf("invalid input %T, expected %T", input, stakeInput)
	}

	addressBytes, err := common.DecodeCheck(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	resource := core.ResourceCode_BANDWIDTH
	if stakeInput.Resource == tx_input.ResourceEnergy {
		resource = core.ResourceCode_ENERGY
	}

	params := &core.FreezeBalanceV2Contract{
		OwnerAddress:  addressBytes,
		FrozenBalance: args.GetAmount().Int().Int64(),
		Resource:      resource,
	}

	contract := &core.Transaction_Contract{}
	contract.Type = core.Transaction_Contract_FreezeBalanceV2Contract
	param, err := anypb.New(params)
	if err != nil {
		return nil, err
	}
	contract.Parameter = param

	tx := &core.Transaction{}
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: stakeInput.RefBlockBytes,
		RefBlockHash:  stakeInput.RefBlockHash,
		// tron wants milliseconds
		Expiration: time.Unix(stakeInput.Expiration, 0).UnixMilli(),
		Timestamp:  time.Unix(stakeInput.Timestamp, 0).UnixMilli(),
	}

	return &Tx{
		TronTx: tx,
		Args:   nil,
	}, nil
}

func (b *TxBuilder) Unstake(args xcbuilder.StakeArgs, input types.UnstakeTxInput) (types.Tx, error) {
	unstakeInput, ok := input.(*tx_input.UnstakingInput)
	if !ok {
		return nil, fmt.Errorf("invalid input %T, expected %T", input, unstakeInput)
	}

	addressBytes, err := common.DecodeCheck(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	resource := core.ResourceCode_BANDWIDTH
	if unstakeInput.Resource == tx_input.ResourceEnergy {
		resource = core.ResourceCode_ENERGY
	}

	params := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    addressBytes,
		UnfreezeBalance: args.GetAmount().Int().Int64(),
		Resource:        resource,
	}

	contract := &core.Transaction_Contract{}
	contract.Type = core.Transaction_Contract_UnfreezeBalanceV2Contract
	param, err := anypb.New(params)
	if err != nil {
		return nil, err
	}
	contract.Parameter = param

	tx := &core.Transaction{}
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: unstakeInput.RefBlockBytes,
		RefBlockHash:  unstakeInput.RefBlockHash,
		// tron wants milliseconds
		Expiration: time.Unix(unstakeInput.Expiration, 0).UnixMilli(),
		Timestamp:  time.Unix(unstakeInput.Timestamp, 0).UnixMilli(),
	}

	return &Tx{
		TronTx: tx,
		Args:   nil,
	}, nil
}

func (b *TxBuilder) Withdraw(args xcbuilder.StakeArgs, input types.WithdrawTxInput) (types.Tx, error) {
	withdrawInput, ok := input.(*tx_input.WithdrawInput)
	if !ok {
		return nil, fmt.Errorf("invalid input %T, expected %T", input, withdrawInput)
	}

	addressBytes, err := common.DecodeCheck(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	params := &core.WithdrawExpireUnfreezeContract{
		OwnerAddress: addressBytes,
	}

	contract := &core.Transaction_Contract{}
	contract.Type = core.Transaction_Contract_WithdrawExpireUnfreezeContract
	param, err := anypb.New(params)
	if err != nil {
		return nil, err
	}
	contract.Parameter = param

	tx := &core.Transaction{}
	tx.RawData = &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		RefBlockBytes: withdrawInput.RefBlockBytes,
		RefBlockHash:  withdrawInput.RefBlockHash,
		// tron wants milliseconds
		Expiration: time.Unix(withdrawInput.Expiration, 0).UnixMilli(),
		Timestamp:  time.Unix(withdrawInput.Timestamp, 0).UnixMilli(),
	}

	return &Tx{
		TronTx: tx,
		Args:   nil,
	}, nil
}
