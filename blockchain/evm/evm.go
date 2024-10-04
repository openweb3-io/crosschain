package evm

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/openweb3-io/crosschain/blockchain/evm/erc20"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	_types "github.com/openweb3-io/crosschain/types"
)

type EvmApi struct {
	endpoint string
	chainId  *big.Int
	client   *ethclient.Client
}

func New(
	endpoint string,
	chainId *big.Int,
) *EvmApi {
	if endpoint == "" {
		endpoint = "https://eth-mainnet.public.blastapi.io"
	}

	client, err := ethclient.Dial(endpoint)
	if err != nil {
		log.Fatalf("error dial rpc, err %v", err)
	}

	return &EvmApi{
		endpoint: endpoint,
		chainId:  chainId,
		client:   client,
	}
}

func (a *EvmApi) isDynamicFeeTx(input *_types.TransferArgs) bool {
	return input.MaxFeePerGas != nil && input.MaxPriorityFeePerGas != nil
}

func (a *EvmApi) EstimateGas(ctx context.Context, input *_types.TransferArgs) (*big.Int, error) {
	var msg ethereum.CallMsg

	// validate address
	mixedFromAddress, err := common.NewMixedcaseAddressFromString(input.From)
	if err != nil {
		return big.NewInt(0), err
	}
	fromAddress := mixedFromAddress.Address()

	mixedToAddress, err := common.NewMixedcaseAddressFromString(input.To)
	if err != nil {
		return big.NewInt(0), err
	}
	toAddress := mixedToAddress.Address()

	var contractAddress *common.Address
	if input.ContractAddress != nil {
		var err error
		mixedContractAddress, err := common.NewMixedcaseAddressFromString(*input.ContractAddress)
		if err != nil {
			return nil, err
		}

		_contractAddress := mixedContractAddress.Address()
		contractAddress = &_contractAddress
	}

	if contractAddress == nil {
		msg = ethereum.CallMsg{
			From:  fromAddress,
			To:    &toAddress,
			Value: big.NewInt(input.Amount.Int().Int64()), // wei
			Data:  nil,
		}
	} else {
		data, err := a.buildContractTransferData(ctx, toAddress, input.Amount)
		if err != nil {
			return nil, err
		}

		msg = ethereum.CallMsg{
			From:  fromAddress,
			To:    contractAddress,
			Value: big.NewInt(0), // wei
			Data:  data,
		}
	}

	gas, err := a.client.EstimateGas(ctx, msg)
	if err != nil {
		return big.NewInt(0), err
	}

	return big.NewInt(int64(gas)), nil
}

func (a *EvmApi) BuildTransaction(ctx context.Context, input *_types.TransferArgs) (_types.Tx, error) {
	// validate address
	mixedFromAddress, err := common.NewMixedcaseAddressFromString(input.From)
	if err != nil {
		return nil, _types.WrapErr(_types.ErrInvalidAddress, fmt.Errorf("%s is not a valid address", input.From))
	}
	fromAddress := mixedFromAddress.Address()

	mixedToAddress, err := common.NewMixedcaseAddressFromString(input.To)
	if err != nil {
		return nil, _types.WrapErr(_types.ErrInvalidAddress, fmt.Errorf("%s is not a valid address", input.To))
	}
	toAddress := mixedToAddress.Address()

	var contractAddress *common.Address
	if input.ContractAddress != nil {
		mixedContractAddress, err := common.NewMixedcaseAddressFromString(*input.ContractAddress)
		if err != nil {
			return nil, err
		}
		_contractAddress := mixedContractAddress.Address()
		contractAddress = &_contractAddress
	}

	var to *common.Address
	if input.ContractAddress != nil {
		to = contractAddress
	} else {
		to = &toAddress
	}

	// end input

	// get nonce
	nonce, err := a.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Printf("Failed to get nonce: %v", err)
		return nil, err
	}

	balance, err := a.client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %v", err)
	}

	// compare transfer amount
	if input.Amount.Int().Cmp(balance) > 0 {
		// insufficient transfer amount
		return nil, fmt.Errorf("insuffiecent amount, balance: %v, amount: %v", balance.String(), input.Amount.String())
	}

	gasPrice := input.GasPrice.Int()
	// GasPrice should be estimated only for LegacyTx
	if !a.isDynamicFeeTx(input) && gasPrice == nil {
		gasPrice, err = a.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to suggest gas price")
		}
	}

	// 构造 value & data
	var data []byte
	var value *big.Int
	if contractAddress != nil {
		var err error
		data, err = a.buildContractTransferData(ctx, toAddress, input.Amount)
		if err != nil {
			return nil, err
		}
	} else {
		value = input.Amount.Int()
	}

	var gas uint64
	if input.Gas != nil {
		gas = input.Gas.Uint64()
	} else {
		var err error

		// Gas estimation cannot succeed without code for method invocations
		if input.ContractAddress != nil {
			if code, err := a.client.PendingCodeAt(ctx, fromAddress); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, fmt.Errorf("error no code")
			}
		}

		if a.isDynamicFeeTx(input) {
			gasFeeCap := input.MaxFeePerGas.Int()
			gasTipCap := input.MaxPriorityFeePerGas.Int()
			gas, err = a.client.EstimateGas(ctx, ethereum.CallMsg{
				From:      fromAddress,
				To:        to,
				GasFeeCap: gasFeeCap,
				GasTipCap: gasTipCap,
				Value:     value,
				Data:      data,
			})
		} else {
			gas, err = a.client.EstimateGas(ctx, ethereum.CallMsg{
				From:     fromAddress,
				To:       to,
				GasPrice: gasPrice,
				Value:    value,
				Data:     data,
			})
		}
		if err != nil {
			return nil, err
		}
	}

	tx := a.buildTransaction(
		nonce,
		value,
		gas,
		gasPrice,
		fromAddress,
		to,
		input.MaxFeePerGas,
		input.MaxPriorityFeePerGas,
		data,
	)

	// 计算hash
	eSigner := types.NewLondonSigner(a.chainId) // TODO extract from input.Network
	hash := eSigner.Hash(tx)

	// sign
	sig, err := input.Signer.Sign(ctx, hash.Bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to remote sign transaction")
	}

	signedTx, err := tx.WithSignature(eSigner, sig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to sign WithSignature transaction")
	}

	return &Tx{
		Transaction: signedTx,
		nonce:       nonce,
	}, nil
}

func (a *EvmApi) BroadcastSignedTx(ctx context.Context, _tx _types.Tx) error {
	tx := _tx.(*Tx)

	return a.client.SendTransaction(ctx, tx.Transaction)
}

func (a *EvmApi) buildContractTransferData(ctx context.Context, to common.Address, value _types.BigInt) ([]byte, error) {
	abi, err := erc20.IERC20MetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	return abi.Pack("transfer", to, value)
}

func (t *EvmApi) buildTransaction(
	nonce uint64,
	value *big.Int,
	gas uint64,
	gasPrice *big.Int,
	from common.Address,
	to *common.Address,
	maxFeePerGas *_types.BigInt,
	maxPriorityFeePerGas *_types.BigInt,
	data []byte,
) *types.Transaction {
	var tx *types.Transaction

	if to != nil {
		var txData types.TxData

		if maxFeePerGas != nil && maxPriorityFeePerGas != nil {
			gasTipCap := (*big.Int)(maxPriorityFeePerGas)
			gasFeeCap := (*big.Int)(maxFeePerGas)

			txData = &types.DynamicFeeTx{
				Nonce:     nonce,
				Gas:       gas,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				To:        to,
				Value:     value,
				Data:      data,
			}
		} else {
			txData = &types.LegacyTx{
				Nonce:    nonce,
				GasPrice: gasPrice,
				Gas:      gas,
				To:       to,
				Value:    value,
				Data:     data,
			}
		}
		tx = types.NewTx(txData)
		zap.S().Info("New transaction",
			zap.String("From", from.String()),
			zap.String("To", to.String()),
			zap.Uint64("Gas", gas),
			zap.String("GasPrice", gasPrice.String()),
			zap.String("Value", value.String()),
		)
	} else {
		if maxFeePerGas != nil && maxPriorityFeePerGas != nil {
			gasTipCap := (*big.Int)(maxPriorityFeePerGas)
			gasFeeCap := (*big.Int)(maxFeePerGas)

			txData := &types.DynamicFeeTx{
				Nonce:     nonce,
				Value:     value,
				Gas:       gas,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Data:      data,
			}
			tx = types.NewTx(txData)
		} else {
			tx = types.NewContractCreation(nonce, value, gas, gasPrice, data)
		}
		zap.S().Info("New contract",
			zap.String("From", from.String()),
			zap.Uint64("Gas", gas),
			zap.String("GasPrice", gasPrice.String()),
			zap.String("Value", value.String()),
			zap.String("Contract address", crypto.CreateAddress(from, nonce).String()),
		)
	}

	return tx
}

func (a *EvmApi) GetBalance(ctx context.Context, address string, contractAddress *string) (*big.Int, error) {
	balance := big.NewInt(0)
	addr, err := common.NewMixedcaseAddressFromString(address)
	if err != nil {
		return balance, err
	}
	if contractAddress == nil {
		balanceAt, err := a.client.BalanceAt(ctx, addr.Address(), nil)
		if err != nil {
			return balance, err
		}
		balance = balanceAt
	} else {
		contract := common.HexToAddress(*contractAddress)
		abi, err := erc20.IERC20MetaData.GetAbi()
		if err != nil {
			return balance, err
		}

		data, err := abi.Pack("balanceOf", addr.Address())
		if err != nil {
			return balance, err
		}

		resp, err := a.client.CallContract(ctx, ethereum.CallMsg{
			To:   &contract,
			Data: data,
		}, nil)
		if err != nil {
			return balance, err
		}

		if err := abi.UnpackIntoInterface(&balance, "balanceOf", resp); err != nil {
			return balance, err
		}
	}

	return balance, nil
}
