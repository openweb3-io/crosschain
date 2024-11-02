package client_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/openweb3-io/crosschain/blockchain/evm"
	"github.com/openweb3-io/crosschain/blockchain/evm/builder"
	"github.com/openweb3-io/crosschain/blockchain/evm/client"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/signer"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/suite"
)

var (
	// // ethereum testnet
	// endpoint        = "https://sepolia.infura.io/v3/79f87ec10fba42d7a85aaa69406c6f96"
	// chainId         = 11155111
	// contractAddress = "0x779877A7B0D9E8603169DdbD7836e478b4624789"

	// arb testnet sepolia
	endpoint        = "https://sepolia-rollup.arbitrum.io/rpc"
	chainId         = 421614
	contractAddress = "0x7AC8519283B1bba6d683FF555A12318Ec9265229"

	// // bsc testnet
	// endpoint        = "https://data-seed-prebsc-1-s1.bnbchain.org:8545"
	// chainId         = 97
	// contractAddress = "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd"

	fromAddress = "0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8"
	pkStrHex    = "8e812436a0e3323166e1f0e8ba79e19e217b2c4a53c970d4cca0cfb1078979df"
)

type ClientTestSuite struct {
	suite.Suite
	signer signer.Signer
	client *client.Client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	pkBytes, err := hex.DecodeString(pkStrHex)
	suite.Require().NoError(err)

	priv := crypto.ToECDSAUnsafe(pkBytes)
	suite.signer = evm.NewLocalSigner(priv)

	suite.client, err = client.NewClient(&xc_types.ChainConfig{
		ChainID: int64(chainId),
		Client: &xc_types.ClientConfig{
			URL: endpoint,
		},
	})
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) Test_Tranfser() {
	ctx := context.Background()

	// gas := xc_types.NewBigIntFromInt64(21000)
	args, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(3000),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	sig, err := suite.signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(sig)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) Test_TranfserERC20() {
	ctx := context.Background()
	contractAddress := xc_types.ContractAddress(contractAddress)

	// gas := xc_types.NewBigIntFromInt64(43000)
	args, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(6000),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	args2, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(6000),
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract: contractAddress,
			Decimals: 18,
		}),
	)
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args2, input)
	suite.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	sig, err := suite.signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(sig)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestFetchBalance() {
	ctx := context.Background()

	addr := xc_types.Address("0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8")
	contractAddress := xc_types.ContractAddress(contractAddress)

	balance, err := suite.client.FetchBalance(ctx, addr)
	suite.Require().NoError(err)

	fmt.Printf("[EVM]] address: %s, balance %v\n", addr, balance)

	balance, err = suite.client.FetchBalanceForAsset(ctx, addr, contractAddress)
	suite.Require().NoError(err)

	fmt.Printf("[EVM] contract %v, address: %v, balance: %v\n", contractAddress, addr, balance)

}

func (suite *ClientTestSuite) TestEstimateGas() {
	ctx := context.Background()
	args, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(3000),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)

	sig, err := suite.signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(sig)
	suite.Require().NoError(err)

	gasFee, err := suite.client.EstimateGasFee(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("gas fee: %v\n", gasFee)
}

func (suite *ClientTestSuite) TestFetchLegacyTxInfo() {
	ctx := context.Background()
	txHash := "0xf927c37306875204c516138ce239916e4eb9987254869342d53163975a96854d"
	txInfo, err := suite.client.FetchLegacyTxInfo(ctx, xc_types.TxHash(txHash))
	suite.Require().NoError(err)

	buf, _ := json.MarshalIndent(txInfo, "", "  ")
	fmt.Printf("tx: %s\n", string(buf))
}

func (suite *ClientTestSuite) TestFetchTxInfo() {
	ctx := context.Background()
	txHash := "0xf927c37306875204c516138ce239916e4eb9987254869342d53163975a96854d"
	txInfo, err := suite.client.FetchTxInfo(ctx, xc_types.TxHash(txHash))
	suite.Require().NoError(err)

	buf, _ := json.MarshalIndent(txInfo, "", "  ")
	fmt.Printf("tx: %s\n", string(buf))
}
