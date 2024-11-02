package liteserver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/openweb3-io/crosschain/blockchain/ton/client/liteserver"
	"github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/suite"
	"github.com/xssnick/tonutils-go/address"
)

type ClientTestSuite struct {
	suite.Suite
	client *liteserver.Client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	var err error
	suite.client, err = liteserver.NewClient(&types.ChainConfig{})
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) Test_GetPublicKey() {
	ctx := context.Background()
	require := suite.Require()

	addr, err := address.ParseAddr("EQB-U9ZcM16Sc2p-xcSyhTCU7YGK8UH5Qvq4CFnM2ejNgU_x")
	// addr, err := address.ParseAddr("EQAAlNYul6D4UrJpv7nYmYZ2beusTT-687rI0joN9O4TdMNm") // not inited
	require.NoError(err)

	masterInfo, err := suite.client.Client.CurrentMasterchainInfo(ctx)
	require.NoError(err)

	rsp, err := suite.client.Client.RunGetMethod(ctx, masterInfo, addr, "get_public_key", nil)
	require.NoError(err)

	tuple := rsp.AsTuple()
	fmt.Printf("tuple: %v\n", len(tuple))
	require.GreaterOrEqual(len(tuple), 1)

	pk, err := rsp.Int(1)
	require.NoError(err)

	fmt.Printf("rsp: %v\n", pk.String())
}
