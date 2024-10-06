package factory_test

import (
	"testing"

	"github.com/openweb3-io/crosschain/factory"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/suite"
)

type CrosschainTestSuite struct {
	suite.Suite
	Factory          *factory.Factory
	TestNativeAssets []xc.NativeAsset
	TestChainConfigs []*xc.ChainConfig
}

func (s *CrosschainTestSuite) SetupTest() {
	s.Factory = factory.NewDefaultFactory()
	// count := 0
	// s.Factory.AllAssets.Range(func(key, value any) bool {
	// 	count++
	// 	fmt.Printf("loaded asset %d: %s %v\n", count, key, value)
	// 	return true
	// })
	s.TestNativeAssets = []xc.NativeAsset{
		xc.ETH,
		xc.MATIC,
		xc.BNB,
		xc.SOL,
		// xc.ATOM,
	}
	for _, native := range s.TestNativeAssets {
		asset, _ := s.Factory.GetAssetConfig("", native)
		chainConfig := asset.(*xc.ChainConfig)
		s.TestChainConfigs = append(s.TestChainConfigs, chainConfig)
	}
}

func TestCrosschain(t *testing.T) {
	suite.Run(t, new(CrosschainTestSuite))
}

// NewObject functions

func (s *CrosschainTestSuite) TestNewDefaultFactory() {
	require := s.Require()
	require.NotNil(s.Factory)
}

func (s *CrosschainTestSuite) TestNewTxBuilder() {
	require := s.Require()
	for _, asset := range s.TestChainConfigs {
		builder, _ := s.Factory.NewTxBuilder(asset)
		require.NotNil(builder)
	}

	_, err := s.Factory.NewTxBuilder(&xc.ChainConfig{Chain: "TEST"})
	require.ErrorContains(err, "no tx-builder defined for")
}

func (s *CrosschainTestSuite) TestNewSigner() {
	require := s.Require()
	for _, chain := range s.TestChainConfigs {
		pri := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
		signer, err := s.Factory.NewSigner(chain, pri)
		require.NoError(err)
		require.NotNil(signer)
	}

	_, err := s.Factory.NewSigner(&xc.ChainConfig{Chain: "TEST"}, "")
	require.ErrorContains(err, "unsupported signing alg")
}

func (s *CrosschainTestSuite) TestNewAddressBuilder() {
	require := s.Require()
	for _, cfg := range s.TestChainConfigs {
		builder, err := s.Factory.NewAddressBuilder(cfg)
		require.NoError(err)
		require.NotNil(builder)
	}

	_, err := s.Factory.NewAddressBuilder(&xc.ChainConfig{Chain: "TEST"})
	require.ErrorContains(err, "no address builder defined for")
}

// GetObject functions (excluding config)

func (s *CrosschainTestSuite) TestGetAddressFromPublicKey() {
	require := s.Require()
	for _, asset := range s.TestChainConfigs {
		address, _ := s.Factory.GetAddressFromPublicKey(asset, []byte{})
		require.NotNil(address)
	}
}

func (s *CrosschainTestSuite) TestGetAllPossibleAddressesFromPublicKey() {
	require := s.Require()
	for _, asset := range s.TestChainConfigs {
		addresses, _ := s.Factory.GetAllPossibleAddressesFromPublicKey(asset, []byte{})
		require.NotNil(addresses)
	}
}

// MustObject functions

func (s *CrosschainTestSuite) TestMustAddress() {
	require := s.Require()
	for _, asset := range s.TestChainConfigs {
		asset := asset.GetChain()
		address := s.Factory.MustAddress(asset, "myaddress") // trivial impl
		require.Equal(xc.Address("myaddress"), address, "Error on: "+asset.Chain)
	}
}

// Convert functions

func (s *CrosschainTestSuite) TestGetAssetID() {
	require := s.Require()
	assetID := xc.GetAssetIDFromAsset("USDC", "SOL")
	require.Equal(xc.AssetID("USDC.SOL"), assetID)
}

func (s *CrosschainTestSuite) TestGetAssetConfig() {
	require := s.Require()
	task, err := s.Factory.GetAssetConfig("USDC", "SOL")
	token := task.(*xc.TokenAssetConfig)
	native := task.GetChain()
	require.NoError(err)
	require.NotNil(token)
	require.Equal("USDC", token.Asset)
	require.Equal(xc.SOL, native.Chain)
}

func (s *CrosschainTestSuite) TestGetAssetConfigEdgeCases() {
	require := s.Require()
	task, err := s.Factory.GetAssetConfig("", "")
	require.Error(err)
	asset := task.GetChain()
	require.NotNil(asset)
	require.Equal(xc.NativeAsset(""), asset.Chain)
	require.Equal(xc.NativeAsset(""), asset.Chain)
}

/*
func (s *CrosschainTestSuite) TestGetTaskConfig() {
	require := s.Require()
	asset, err := s.Factory.GetTaskConfig("sol-wrap", "SOL")
	require.Nil(err)
	require.NotNil(asset)
}
*/

/*
func (s *CrosschainTestSuite) TestGetTaskConfigEdgeCases() {
	require := s.Require()
	singleAsset, _ := s.Factory.GetAssetConfig("USDC", "SOL")
	asset, err := s.Factory.GetTaskConfig("", "USDC.SOL")
	require.Nil(err)
	require.NotNil(singleAsset)
	require.NotNil(asset)
	require.Equal(singleAsset, asset)
}
*/

/*
func (s *CrosschainTestSuite) TestGetMultiAssetConfig() {
	require := s.Require()
	asset, err := s.Factory.GetMultiAssetConfig("SOL", "WSOL.SOL")
	require.Nil(err)
	require.NotNil(asset)
}
*/

/*
func (s *CrosschainTestSuite) TestGetMultiAssetConfigEdgeCases() {
	require := s.Require()
	singleAsset, _ := s.Factory.GetAssetConfig("USDC", "SOL")
	tasks, err := s.Factory.GetMultiAssetConfig("USDC.SOL", "")
	require.Nil(err)
	require.NotNil(singleAsset)
	require.NotNil(tasks)
	require.NotNil(tasks[0])
	require.Equal(singleAsset, tasks[0])
}
*/

/*
func (s *CrosschainTestSuite) TestGetAssetConfigByContract() {
	require := s.Require()
	s.Factory.PutAssetConfig(&xc.TokenAssetConfig{
		Chain:    "ETH",
		Contract: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d0",
		Asset:    "WETH",
	})
	s.Factory.PutAssetConfig(&xc.TokenAssetConfig{
		Chain:    "SOL",
		Contract: "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDu",
		Asset:    "USDC",
	})

	assetI, err := s.Factory.GetAssetConfigByContract("0xB4FBF271143F4FBf7B91A5ded31805e42b2208d0", "ETH")
	asset := assetI.(*xc.TokenAssetConfig)
	require.Nil(err)
	require.NotNil(asset)
	require.Equal("WETH", asset.Asset)

	assetI, err = s.Factory.GetAssetConfigByContract("4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDu", "SOL")
	asset = assetI.(*xc.TokenAssetConfig)
	require.Nil(err)
	require.NotNil(asset)
	require.Equal("USDC", asset.Asset)

	assetI, err = s.Factory.GetAssetConfigByContract("0x123456", "ETH")
	asset = assetI.(*xc.TokenAssetConfig)
	require.EqualError(err, "unknown contract: '0x123456'")
	require.NotNil(asset)
	require.Equal("", asset.Asset)
}
*/

/*
func (s *CrosschainTestSuite) TestPutAssetConfig() {
	require := s.Require()
	assetName := "TEST"

	assetI, err := s.Factory.GetAssetConfig(assetName, "")
	require.EqualError(err, "could not lookup asset: 'TEST.ETH'")
	require.NotNil(assetI)

	assetI, err = s.Factory.PutAssetConfig(&xc.TokenAssetConfig{Asset: assetName, Chain: "ETH"})
	fmt.Println(assetI)
	asset := assetI.(*xc.TokenAssetConfig)
	require.Nil(err)
	require.Equal(assetName, asset.Asset)

	assetI, err = s.Factory.GetAssetConfig("TEST", "")
	asset = assetI.(*xc.TokenAssetConfig)
	require.Nil(err)
	require.Equal(assetName, asset.Asset)
}
*/
