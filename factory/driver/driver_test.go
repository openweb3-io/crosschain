package driver_test

import (
	"testing"

	"github.com/openweb3-io/crosschain/factory/driver"
	"github.com/stretchr/testify/suite"
)

type DriverTestSuite struct {
	suite.Suite
}

func TestDriverTestSuite(t *testing.T) {
	suite.Run(t, new(DriverTestSuite))
}

func (suite *DriverTestSuite) SetupTest(t *testing.T) {
}

func (suite *DriverTestSuite) TestNewClient() {
	require := suite.Require()

	client, err := driver.NewClient("ton")
	require.NoError(err)
	require.NotNil(client)
}
