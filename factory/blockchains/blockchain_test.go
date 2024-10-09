package blockchains_test

import (
	"testing"

	"github.com/openweb3-io/crosschain/factory/blockchains"
	xc "github.com/openweb3-io/crosschain/types"
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

func (s *BlockchainTestSuite) TestAllNewClient() {
	require := s.Require()

	for _, blockchain := range xc.SupportedBlockchains {
		// TODO: these require custom params for NewClient
		if blockchain == xc.BlockchainAptos || blockchain == xc.BlockchainSubstrate {
			continue
		}

		res, err := blockchains.NewClient(createChainFor(blockchain), blockchain)
		require.NoError(err, "Missing blockchain for NewClient: "+blockchain)
		require.NotNil(res)
	}
}

func createChainFor(blockchain xc.Blockchain) *xc.ChainConfig {
	fakeAsset := &xc.ChainConfig{
		// URL:         server.URL,
		Blockchain: blockchain,
	}
	if blockchain == xc.BlockchainBtc {
		fakeAsset.Chain = "BTC"
		fakeAsset.AuthSecret = "1234"
	}
	if blockchain == xc.BlockchainBtcLegacy {
		fakeAsset.Chain = "DOGE"
		fakeAsset.AuthSecret = "1234"
	}
	if blockchain == xc.BlockchainBtcCash {
		fakeAsset.Chain = "BCH"
		fakeAsset.AuthSecret = "1234"
	}
	if blockchain == xc.BlockchainSubstrate {
		fakeAsset.ChainPrefix = "0"
	}
	return fakeAsset
}
