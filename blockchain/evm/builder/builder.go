package builder

import (
	"errors"
	"fmt"
	"math/big"

	xcbuilder "github.com/openweb3-io/crosschain/builder"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/openweb3-io/crosschain/blockchain/evm/abi/exit_request"
	"github.com/openweb3-io/crosschain/blockchain/evm/abi/stake_batch_deposit"
	"github.com/openweb3-io/crosschain/blockchain/evm/address"
	"github.com/openweb3-io/crosschain/blockchain/evm/tx"
	"github.com/openweb3-io/crosschain/blockchain/evm/tx_input"
	"github.com/openweb3-io/crosschain/builder/validation"
	xc "github.com/openweb3-io/crosschain/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
)

var DefaultMaxTipCapGwei uint64 = 5

type GethTxBuilder interface {
	BuildTxWithPayload(chain *xc.ChainConfig, to xc.Address, value xc.BigInt, data []byte, input xc.TxInput) (xc.Tx, error)
}

// supports evm after london merge
type EvmTxBuilder struct {
}

var _ GethTxBuilder = &EvmTxBuilder{}

// TxBuilder for EVM
type TxBuilder struct {
	Chain         *xc.ChainConfig
	gethTxBuilder GethTxBuilder
	// Legacy bool
}

var _ xcbuilder.TxBuilder = &TxBuilder{}
var _ xcbuilder.FullBuilder = &TxBuilder{}
var _ xcbuilder.Staking = &TxBuilder{}

func NewEvmTxBuilder() *EvmTxBuilder {
	return &EvmTxBuilder{}
}

// NewTxBuilder creates a new EVM TxBuilder
func NewTxBuilder(chain *xc.ChainConfig) (TxBuilder, error) {
	return TxBuilder{
		Chain:         chain,
		gethTxBuilder: &EvmTxBuilder{},
	}, nil
}

func (txBuilder TxBuilder) WithTxBuilder(buider GethTxBuilder) TxBuilder {
	txBuilder.gethTxBuilder = buider
	return txBuilder
}

// NewTxBuilder creates a new EVM TxBuilder for legacy tx
// func NewLegacyTxBuilder(asset xc.ITask) (xc.TxBuilder, error) {
// 	return TxBuilder{
// 		Asset: asset,
// 		// Legacy: true,
// 	}, nil
// }

// NewTransfer creates a new transfer for an Asset, either native or token
func (txBuilder TxBuilder) Transfer(args *xcbuilder.TransferArgs, input xc.TxInput) (xc.Tx, error) {
	return txBuilder.NewTransfer(args.GetFrom(), args.GetTo(), args.GetAmount(), input)
}
func (txBuilder TxBuilder) NewTransfer(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	txInput := input.(*tx_input.TxInput)
	asset, _ := txInput.Args.GetAsset()
	if asset == nil {
		asset = txBuilder.Chain
	}

	switch asset := asset.(type) {
	/* TODO
	case *xc.TaskConfig:
		return txBuilder.NewTask(from, to, amount, input)
	*/

	case *xc.ChainConfig:
		return txBuilder.NewNativeTransfer(from, to, amount, input)

	case *xc.TokenAssetConfig:
		return txBuilder.NewTokenTransfer(from, to, amount, input)

	default:
		// TODO this should return error
		contract := asset.GetContract()
		zap.S().Warn("new transfer for unknown asset type",
			zap.String("chain", string(asset.GetChain().Chain)),
			zap.String("contract", string(contract)),
			zap.String("asset_type", fmt.Sprintf("%T", asset)),
		)
		if contract != "" {
			return txBuilder.NewTokenTransfer(from, to, amount, input)
		} else {
			return txBuilder.NewNativeTransfer(from, to, amount, input)
		}
	}
}

// NewNativeTransfer creates a new transfer for a native asset
func (txBuilder TxBuilder) NewNativeTransfer(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	return txBuilder.gethTxBuilder.BuildTxWithPayload(txBuilder.Chain, to, amount, []byte{}, input)
}

// NewTokenTransfer creates a new transfer for a token asset
func (txBuilder TxBuilder) NewTokenTransfer(from xc.Address, to xc.Address, amount xc.BigInt, input xc.TxInput) (xc.Tx, error) {
	txInput := input.(*tx_input.TxInput)
	asset, ok := txInput.Args.GetAsset()
	if !ok {
		return nil, errors.New("asset needed")
	}

	zero := xc.NewBigIntFromUint64(0)
	contract := asset.GetContract()
	payload, err := BuildERC20Payload(to, amount)
	if err != nil {
		return nil, err
	}
	return txBuilder.gethTxBuilder.BuildTxWithPayload(asset.GetChain(), xc.Address(contract), zero, payload, input)
}

func BuildERC20Payload(to xc.Address, amount xc.BigInt) ([]byte, error) {
	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]
	// fmt.Println(hexutil.Encode(methodID)) // 0xa9059cbb

	toAddress, err := address.FromHex(to)
	if err != nil {
		return nil, err
	}
	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	// fmt.Println(hexutil.Encode(paddedAddress)) // 0x0000000000000000000000004592d8f8d7b001e72cb26a73e4fa1806a51ac79d

	paddedAmount := common.LeftPadBytes(amount.Int().Bytes(), 32)
	// fmt.Println(hexutil.Encode(paddedAmount)) // 0x00000000000000000000000000000000000000000000003635c9adc5dea00000

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	return data, nil
}

func (*EvmTxBuilder) BuildTxWithPayload(chain *xc.ChainConfig, to xc.Address, value xc.BigInt, data []byte, inputRaw xc.TxInput) (xc.Tx, error) {
	address, err := address.FromHex(to)
	if err != nil {
		return nil, err
	}

	input := inputRaw.(*tx_input.TxInput)
	var chainId *big.Int = input.ChainId.Int()
	if input.ChainId.Uint64() == 0 {
		chainId = new(big.Int).SetInt64(chain.ChainID)
	}

	// Protection from setting very high gas tip
	maxTipGwei := uint64(chain.ChainMaxGasPrice)
	if maxTipGwei == 0 {
		maxTipGwei = DefaultMaxTipCapGwei
	}
	maxTipWei := GweiToWei(maxTipGwei)
	gasTipCap := input.GasTipCap

	if gasTipCap.Cmp(&maxTipWei) > 0 {
		// limit to max
		gasTipCap = maxTipWei
	}

	return &tx.Tx{
		EthTx: types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainId,
			Nonce:     input.Nonce,
			GasTipCap: gasTipCap.Int(),
			GasFeeCap: input.GasFeeCap.Int(),
			Gas:       input.GasLimit,
			To:        &address,
			Value:     value.Int(),
			Data:      data,
		}),
		Signer: types.LatestSignerForChainID(chainId),
	}, nil
}

func GweiToWei(gwei uint64) xc.BigInt {
	bigGwei := big.NewInt(int64(gwei))

	ten := big.NewInt(10)
	nine := big.NewInt(9)
	factor := big.NewInt(0).Exp(ten, nine, nil)

	bigGwei.Mul(bigGwei, factor)
	return xc.BigInt(*bigGwei)
}

func (txBuilder TxBuilder) Stake(stakeArgs xcbuilder.StakeArgs, input xc.StakeTxInput) (xc.Tx, error) {
	batchDepositInput := input.(*tx_input.BatchDepositInput)
	asset, ok := batchDepositInput.Args.GetAsset()
	if !ok {
		return nil, errors.New("asset needed")
	}

	switch input := input.(type) {
	case *tx_input.BatchDepositInput:
		evmBuilder := NewEvmTxBuilder()

		owner, ok := stakeArgs.GetStakeOwner()
		if !ok {
			owner = stakeArgs.GetFrom()
		}
		ownerAddr, err := address.FromHex(owner)
		if err != nil {
			return nil, err
		}
		ownerBz := ownerAddr.Bytes()
		withdrawCred := [32]byte{}
		copy(withdrawCred[32-len(ownerBz):], ownerBz)
		// set the credential type
		withdrawCred[0] = 1
		credentials := make([][]byte, len(input.PublicKeys))
		for i := range credentials {
			credentials[i] = withdrawCred[:]
		}
		data, err := stake_batch_deposit.Serialize(asset.GetChain(), input.PublicKeys, credentials, input.Signatures)
		if err != nil {
			return nil, fmt.Errorf("invalid input for %T: %v", input, err)
		}
		contract := asset.GetChain().Staking.StakeContract
		tx, err := evmBuilder.BuildTxWithPayload(asset.GetChain(), xc.Address(contract), stakeArgs.GetAmount(), data, &input.TxInput)
		if err != nil {
			return nil, fmt.Errorf("could not build tx for %T: %v", input, err)
		}
		return tx, nil
	default:
		return nil, fmt.Errorf("unsupported staking type %T", input)
	}
}
func (txBuilder TxBuilder) Unstake(stakeArgs xcbuilder.StakeArgs, input xc.UnstakeTxInput) (xc.Tx, error) {
	batchDepositInput := input.(*tx_input.ExitRequestInput)
	asset, ok := batchDepositInput.Args.GetAsset()
	if !ok {
		return nil, errors.New("asset needed")
	}

	switch input := input.(type) {
	case *tx_input.ExitRequestInput:
		evmBuilder := NewEvmTxBuilder()

		count, err := validation.Count32EthChunks(stakeArgs.GetAmount())
		if err != nil {
			return nil, err
		}
		if int(count) > len(input.PublicKeys) {
			return nil, fmt.Errorf("need at least %d validators to unstake target amount, but there are only %d in eligible state", count, len(input.PublicKeys))
		}

		data, err := exit_request.Serialize(input.PublicKeys[:count])
		if err != nil {
			return nil, fmt.Errorf("invalid input for %T: %v", input, err)
		}
		contract := asset.GetChain().Staking.UnstakeContract
		zero := xc.NewBigIntFromUint64(0)
		tx, err := evmBuilder.BuildTxWithPayload(asset.GetChain(), xc.Address(contract), zero, data, &input.TxInput)
		if err != nil {
			return nil, fmt.Errorf("could not build tx for %T: %v", input, err)
		}
		return tx, nil
	default:
		return nil, fmt.Errorf("unsupported unstaking type %T", input)
	}
}

func (txBuilder TxBuilder) Withdraw(stakeArgs xcbuilder.StakeArgs, input xc.WithdrawTxInput) (xc.Tx, error) {
	return nil, fmt.Errorf("ethereum stakes are claimed automatically")
}
