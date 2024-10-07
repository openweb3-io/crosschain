package client_test

import (
	"context"
	"encoding/hex"
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
	endpoint = "https://sepolia.infura.io/v3/4538f2b2d74c4f48b1a74de742293c51"
	chainId  = 11155111
	pkStrHex = "8e812436a0e3323166e1f0e8ba79e19e217b2c4a53c970d4cca0cfb1078979df"
)

type ClientTestSuite struct {
	suite.Suite
	signer signer.Signer
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) SetupTest() {
	pkBytes, err := hex.DecodeString(pkStrHex)
	s.Require().NoError(err)

	priv := crypto.ToECDSAUnsafe(pkBytes)
	s.signer = evm.NewLocalSigner(priv)

}

func (s *ClientTestSuite) TestTranfser() {
	ctx := context.Background()

	//testnet Holesky 17000
	//testnet sepolia 11155111
	client, err := client.NewClient(&xc_types.ChainConfig{
		// URL: "https://eth-mainnet.public.blastapi.io",
		ChainID: int64(chainId),
		URL:     endpoint,
		// URL: "http://chainproto-admin.chainproto.dev/rpc/ethereum/11155111/testnet",
	})
	s.Require().NoError(err)

	// gas := types.NewBigIntFromInt64(21000)
	args, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(3000),
	)
	s.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	s.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	s.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	s.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	s.Require().NoError(err)
	s.Require().Equal(len(sighashes), 1)

	sig, err := s.signer.Sign(sighashes[0])
	s.Require().NoError(err)

	err = tx.AddSignatures(sig)
	s.Require().NoError(err)

	err = client.BroadcastTx(ctx, tx)
	s.Require().NoError(err)

	fmt.Printf("tx hash: %v\n", tx.Hash())
}

func (s *ClientTestSuite) TestTranfserERC20() {
	ctx := context.Background()
	contractAddress := xc_types.ContractAddress("0x779877A7B0D9E8603169DdbD7836e478b4624789")

	//testnet Holesky 17000
	//testnet sepolia 11155111
	client, err := client.NewClient(&xc_types.ChainConfig{
		ChainID: int64(chainId),
		URL:     "https://sepolia.infura.io/v3/4538f2b2d74c4f48b1a74de742293c51",
	})
	s.Require().NoError(err)

	// gas := xc_types.NewBigIntFromInt64(43000)
	args, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(6000),
	)
	s.Require().NoError(err)

	input, err := client.FetchTransferInput(ctx, args)
	s.Require().NoError(err)

	args2, err := xcbuilder.NewTransferArgs(
		"0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		"0x388C818CA8B9251b393131C08a736A67ccB19297",
		xc_types.NewBigIntFromInt64(6000),
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Contract: contractAddress,
			Decimals: 18,
		}),
	)
	s.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	s.Require().NoError(err)

	tx, err := builder.NewTransfer(args2, input)
	s.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	s.Require().NoError(err)
	s.Require().Equal(len(sighashes), 1)

	sig, err := s.signer.Sign(sighashes[0])
	s.Require().NoError(err)

	err = tx.AddSignatures(sig)
	s.Require().NoError(err)

	err = client.BroadcastTx(ctx, tx)
	s.Require().NoError(err)

	fmt.Printf("tx hash: %v\n", tx.Hash())
}

func (s *ClientTestSuite) TestFetchBalance() {
	ctx := context.Background()

	client, err := client.NewClient(&xc_types.ChainConfig{
		ChainID: int64(chainId),
		URL:     endpoint,
	})
	s.Require().NoError(err)

	addr := xc_types.Address("0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8")
	contractAddress := xc_types.ContractAddress("0x779877A7B0D9E8603169DdbD7836e478b4624789")

	balance, err := client.FetchBalance(ctx, addr)
	s.Require().NoError(err)

	fmt.Printf("[EVM]] address: %s, balance %v\n", addr, balance)

	balance, err = client.FetchBalanceForAsset(ctx, addr, contractAddress)
	s.Require().NoError(err)

	fmt.Printf("[EVM] contract %v, address: %v, balance: %v\n", contractAddress, addr, balance)

}
