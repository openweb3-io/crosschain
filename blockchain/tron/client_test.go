package tron_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/openweb3-io/crosschain/blockchain/tron"
	"github.com/openweb3-io/crosschain/builder"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/test-go/testify/suite"
)

var (
	endpoint = "grpc.nile.trongrid.io:50051"
	chainId  = big.NewInt(1001)

	senderPubk  = "THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F"
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
	client := tron.NewClient(endpoint, chainId)

	amount := xc_types.NewBigIntFromInt64(3)

	args, err := builder.NewTransferArgs(
		"THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F",
		"TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA",
		amount,
	)
	suite.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder := tron.NewTxBuilder(&xc_types.ChainConfig{})
	tx, err := builder.BuildTransfer(input)
	suite.Require().NoError(err)

	gas, err := client.EstimateGas(ctx, tx)
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

	err = client.SubmitTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %x\n", tx.Hash())
}

func (suite *ClientTestSuite) TestTranfserTRC20() {
	ctx := context.Background()

	//testnet
	client := tron.NewClient(endpoint, chainId)

	contractAddress := types.ContractAddress("TNuoKL1ni8aoshfFL1ASca1Gou9RXwAzfn")
	// gas := types.NewBigIntFromInt64(1)

	args, err := builder.NewTransferArgs(
		"THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F",
		"TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA",
		xc_types.NewBigIntFromInt64(3),
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract: contractAddress,
			Decimals: 18,
		}),
	)
	suite.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder := tron.NewTxBuilder(&xc_types.ChainConfig{})
	tx, err := builder.BuildTransfer(input)
	suite.Require().NoError(err)

	calculatedGas, err := client.EstimateGas(ctx, tx)
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

	client.SubmitTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("trx hash: %x\n", tx.Hash())
}

func (suite *ClientTestSuite) TestFetchBalance() {
	ctx := context.Background()

	senderPubk := "THjVQt6hpwZyWnkDm1bHfPvdgysQFoN8AL"
	client := tron.NewClient(endpoint, chainId)
	out, err := client.FetchBalance(ctx, types.Address(senderPubk))
	suite.Require().NoError(err)
	fmt.Printf("\n %s TRX balance: %v", senderPubk, out)

	contractAddress := "TNuoKL1ni8aoshfFL1ASca1Gou9RXwAzfn"
	contractAddr := types.Address(contractAddress)
	out, err = client.FetchBalanceForAsset(ctx, types.Address(senderPubk), &contractAddr)
	suite.Require().NoError(err)

	fmt.Printf("\n %s token balance: %v", senderPubk, out)
}
