package http_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/openweb3-io/crosschain/blockchain/tron"
	tron_http "github.com/openweb3-io/crosschain/blockchain/tron/client/http"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/suite"
)

var (
	// endpoint = "grpc.nile.trongrid.io:50051"
	endpoint = "https://go.getblock.io/4e19dacf44974a3d8e40031ef8aca8b8"
	chainId  = big.NewInt(1001)
	// endpoint = "https://methodical-greatest-choice.tron-mainnet.quiknode.pro/265ecbce554ed6512e0c7af5d55e202e1c07374a"
	// chainId  = big.NewInt(728126428)

	// senderPubk  = "THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F"
	senderPrivk = "8e812436a0e3323166e1f0e8ba79e19e217b2c4a53c970d4cca0cfb1078979df"
)

type ClientTestSuite struct {
	suite.Suite
}

func (suite *ClientTestSuite) SetupTest() {
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) TestTransfer() {
	ctx := context.Background()

	//testnet
	client, err := tron_http.NewClient(&xc_types.ChainConfig{
		Client: &xc_types.ClientConfig{
			URL: endpoint,
		},
		ChainID: chainId.Int64(),
	})
	suite.Require().NoError(err)

	amount := xc_types.NewBigIntFromInt64(3)

	args, err := xcbuilder.NewTransferArgs(
		"THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F",
		"TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA",
		amount,
	)
	suite.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := tron.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	gas, err := client.EstimateGasFee(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("gas: %v\n", gas)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	pkBytes, err := hex.DecodeString(senderPrivk)
	suite.Require().NoError(err)
	priv := crypto.ToECDSAUnsafe(pkBytes)

	signer := tron.NewLocalSigner(priv)
	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %v\n", tx.Hash())
}

func (suite *ClientTestSuite) TestTranfserTRC20() {
	ctx := context.Background()

	//testnet
	client, err := tron_http.NewClient(&xc_types.ChainConfig{
		Client: &xc_types.ClientConfig{
			URL: endpoint,
		},
		ChainID: chainId.Int64(),
	})
	suite.Require().NoError(err)

	contractAddress := xc_types.ContractAddress("TF17BgPaZYbz8oxbjhriubPDsA7ArKoLX3")

	args, err := xcbuilder.NewTransferArgs(
		"THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F",
		"TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA",
		xc_types.NewBigIntFromInt64(1),
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract: contractAddress,
			Decimals: 18,
		}),
	)
	suite.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := tron.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	calculatedGas, err := client.EstimateGasFee(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("gas: %v\n", calculatedGas)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	pkBytes, err := hex.DecodeString(senderPrivk)
	suite.Require().NoError(err)
	priv := crypto.ToECDSAUnsafe(pkBytes)

	signer := tron.NewLocalSigner(priv)
	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("trx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) Test_FetchBalance() {
	ctx := context.Background()

	senderPubk := "THjVQt6hpwZyWnkDm1bHfPvdgysQFoN8AL"
	client, err := tron_http.NewClient(&xc_types.ChainConfig{
		Client: &xc_types.ClientConfig{
			URL: endpoint,
		},
		ChainID: chainId.Int64(),
	})
	suite.Require().NoError(err)

	out, err := client.FetchBalance(ctx, xc_types.Address(senderPubk))
	suite.Require().NoError(err)
	fmt.Printf("sender: %s TRX balance: %v\n", senderPubk, out)

	// contractAddr := xc_types.ContractAddress("TNuoKL1ni8aoshfFL1ASca1Gou9RXwAzfn")
	contractAddr := xc_types.ContractAddress("TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj")
	out, err = client.FetchBalanceForAsset(ctx, xc_types.Address(senderPubk), contractAddr)
	suite.Require().NoError(err)

	fmt.Printf("sender: %s token balance: %v\n", senderPubk, out)
}

func (suite *ClientTestSuite) Test_EstimateGasTransfer() {
	ctx := context.Background()

	client, err := tron_http.NewClient(&xc_types.ChainConfig{
		Client: &xc_types.ClientConfig{
			URL: endpoint,
		},
		ChainID: chainId.Int64(),
	})
	suite.Require().NoError(err)

	amount := xc_types.NewBigIntFromInt64(3)

	args, err := xcbuilder.NewTransferArgs(
		"THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F",
		"TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA",
		amount,
	)
	suite.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := tron.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	gas, err := client.EstimateGasFee(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("gas: %v\n", gas)
}

func (suite *ClientTestSuite) TestEstimateGasTranfserTRC20() {
	ctx := context.Background()

	client, err := tron_http.NewClient(&xc_types.ChainConfig{
		Client: &xc_types.ClientConfig{
			URL: endpoint,
		},
		ChainID: chainId.Int64(),
	})
	suite.Require().NoError(err)

	contractAddress := xc_types.ContractAddress("TF17BgPaZYbz8oxbjhriubPDsA7ArKoLX3")
	args, err := xcbuilder.NewTransferArgs(
		"THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F",
		"TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA",
		xc_types.NewBigIntFromInt64(1),
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract: contractAddress,
			Decimals: 18,
		}),
	)
	suite.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := tron.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	calculatedGas, err := client.EstimateGasFee(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("gas: %v\n", calculatedGas)
}
