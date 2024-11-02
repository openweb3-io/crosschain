package liteserver

import (
	"fmt"
	"testing"

	"github.com/cordialsys/crosschain/chain/ton/api"
	"github.com/openweb3-io/crosschain/blockchain/ton/client/liteserver"
	"github.com/test-go/testify/suite"
	_ton "github.com/xssnick/tonutils-go/ton"
)

type ClientTestSuite struct {
	suite.Suite
	client *_ton.APIClient
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	var err error
	suite.client, err = liteserver.NewClient()
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) Test_GetPublicKey() {
	err = suite.client.RunGetMethod(&api.GetMethodRequest{
		Address: string(args.GetFrom()),
		Method:  api.GetPublicKeyMethod,
		Stack:   []api.StackItem{},
	}, getAddrResponse)
	if err != nil {
		return nil, fmt.Errorf("could not get address public-key: %v", err)
	}
}
