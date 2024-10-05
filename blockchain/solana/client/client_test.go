package client_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"testing"

	solana_sdk "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/suite"

	"github.com/openweb3-io/crosschain/blockchain/solana"
	"github.com/openweb3-io/crosschain/blockchain/solana/builder"
	"github.com/openweb3-io/crosschain/blockchain/solana/client"
	xcbuilder "github.com/openweb3-io/crosschain/builder"

	"github.com/openweb3-io/crosschain/types"
)

var (
	//DBomk9vPzgLWpDBvvQpJUAB1aFz8EHsPq6xEuA1cGMcV
	senderPrivkeyStr = "3PiA3WZuqGKv1E5aGWYfjsYVXZWLEJiUzGtFHZ8SNXUkLBbX9goGAHouEhTeFGUiBXVvRkfkHRga7XPENyJ7c3nq"
	senderPrivateKey solana_sdk.PrivateKey

	//8FLngQGnatEDQwNBV27yFxuWDhvQfriaCL56fx84TxoN
	recipientPrivk      = "2vLh8LUmwr9LVbFrJXKLcYcgMXAy52X6PHqZ9yhLvVfW1Fz3k1uJjheLcpUvum5oLYv8xZX5AnEXoMAEZMUMLyja"
	recipientPrivateKey solana_sdk.PrivateKey
)

type ClientTestSuite struct {
	suite.Suite
	client *client.Client
}

func (suite *ClientTestSuite) SetupTest() {
	//testnet
	client, err := client.NewClient(&types.ChainConfig{
		URL: rpc.TestNet_RPC,
	})
	suite.Require().NoError(err)
	suite.client = client

	senderPrivateKey = solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr)
	fmt.Printf("sender address: %s \nprivate: %s\n", senderPrivateKey.PublicKey(), senderPrivateKey)

	// recipientPrivateKey = solana_sdk.NewWallet().PrivateKey
	recipientPrivateKey = solana_sdk.MustPrivateKeyFromBase58(recipientPrivk)
	fmt.Printf("recipient address: %s \nprivate: %s\n", recipientPrivateKey.PublicKey(), recipientPrivateKey)

}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) TestTranfser() {
	ctx := context.Background()

	asset := &types.TokenAssetConfig{}

	builder, err := builder.NewTxBuilder(asset)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, &xcbuilder.TransferArgs{
		From:   types.Address(senderPrivateKey.PublicKey().String()),
		To:     types.Address(recipientPrivateKey.PublicKey().String()), // must exist
		Amount: types.NewBigIntFromInt64(35),
	})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(input)
	suite.Require().NoError(err)

	fee, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("estimate SOL gas: %v\n", fee)

	privateKey := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr))
	signer := solana.NewLocalSigner(privateKey)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.SubmitTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestSPLTranfser(t *testing.T) {
	ctx := context.Background()

	input, err := suite.client.FetchTransferInput(ctx, &xcbuilder.TransferArgs{
		From:   types.Address(senderPrivateKey.PublicKey().String()),          //这里填写sol的主地址，转账时程序自动找到合约的关联账户地址
		To:     types.Address("AyqkhCrb8gt3PqiVMCshSy4to8wQcHzXtfCKbJ42qJLp"), //这里写sol的主地址，自动会创建关联地址
		Amount: types.NewBigIntFromInt64(35),
	})
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&types.TokenAssetConfig{
		Contract: "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr",
		Decimals: 6,
	})
	suite.Require().NoError(err)

	tx, err := builder.NewTokenTransfer(input)
	suite.Require().NoError(err)

	fee, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("estimate SOL gas: %v\n", fee)

	privateKey := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr))
	signer := solana.NewLocalSigner(privateKey)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.SubmitTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestSPLTranfserSetFeePayer(t *testing.T) {
	ctx := context.Background()

	// feePayer := recipientPrivateKey.PublicKey().String()

	input, err := suite.client.FetchTransferInput(ctx, &xcbuilder.TransferArgs{
		From:   types.Address(senderPrivateKey.PublicKey().String()),          //这里填写sol的主地址，转账时程序自动找到合约的关联账户地址
		To:     types.Address("AyqkhCrb8gt3PqiVMCshSy4to8wQcHzXtfCKbJ42qJLp"), //这里写sol的主地址，自动会创建关联地址
		Amount: types.NewBigIntFromInt64(1),
	})
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&types.TokenAssetConfig{
		Contract: "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr",
		Decimals: 6,
	})
	suite.Require().NoError(err)

	tx, err := builder.NewTokenTransfer(input)
	suite.Require().NoError(err)

	fee, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("estimate SOL gas: %v\n", fee)

	privateKey := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr))
	signer := solana.NewLocalSigner(privateKey)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.SubmitTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestFetchBalance() {
	ctx := context.Background()

	contractAddress := "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"

	out, err := suite.client.FetchBalance(ctx, types.Address(senderPrivateKey.PublicKey().String()), nil)
	suite.Require().NoError(err)
	fmt.Printf("\n %s SOL balance: %v", senderPrivateKey.PublicKey().String(), out)

	out, err = suite.client.FetchBalance(ctx, types.Address(senderPrivateKey.PublicKey().String()), (*types.Address)(&contractAddress))
	suite.Require().NoError(err)

	fmt.Printf("\n %s SPL token balance: %v", senderPrivateKey.PublicKey().String(), out)
}
