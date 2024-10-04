package driver_test

import (
	"testing"

	"github.com/openweb3-io/crosschain/factory/driver"
	"github.com/stretchr/testify/suite"
)

type BlockchainTestSuite struct {
	suite.Suite
}

func TestBlockchainTestSuite(t *testing.T) {
	suite.Run(t, new(BlockchainTestSuite))
}

func (suite *BlockchainTestSuite) SetupTest(t *testing.T) {
}

func (suite *BlockchainTestSuite) TestNewClient() {
	require := suite.Require()

	client, err := driver.NewClient("ton")
	require.NoError(err)
	require.NotNil(client)
}
