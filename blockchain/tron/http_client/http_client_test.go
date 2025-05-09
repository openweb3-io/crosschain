package httpclient_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	httpclient "github.com/openweb3-io/crosschain/blockchain/tron/http_client"
	"github.com/stretchr/testify/suite"
)

const (
	// EndpointMainNet = "https://go.getblock.io/41513af034b3452cb27c8e5ca67b6e68"
	EndpointMainNet = "https://go.getblock.io/4e19dacf44974a3d8e40031ef8aca8b8"
	// EndpointTestNet = "https://nile.tronscan.org"
	EndpointTestNet = "https://nile.trongrid.io"

	MainNetAccountTop = "TTeT5eGRpDumftCCjKrWUvvUKuLZsxuZf9"

	TestNetContractAddressJst  = "TF17BgPaZYbz8oxbjhriubPDsA7ArKoLX3"
	TestNetContractAddressUsdt = "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj"

	TestNetAccountFoo = "THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F"
	TestNetAccountBar = "TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA"
	TestNetAccountBaz = "THjVQt6hpwZyWnkDm1bHfPvdgysQFoN8AL"
	TestNetAccountTop = "TDGSR64oU4QDpViKfdwawSiqwyqpUB6JUD"
)

type HttpClientTestSuite struct {
	suite.Suite
	client *httpclient.Client
}

func TestHttpClientTestSuite(t *testing.T) {
	suite.Run(t, new(HttpClientTestSuite))
}

func (suite *HttpClientTestSuite) SetupTest() {
	network := os.Getenv("network")

	endpoint := EndpointTestNet
	if network == "main" {
		endpoint = EndpointMainNet
	}

	var err error
	suite.client, err = httpclient.NewHttpClient(endpoint)
	suite.Require().NoError(err)
}

func (suite *HttpClientTestSuite) TestGetChainParameters() {
	ctx := context.Background()

	resp, err := suite.client.GetChainParameters(ctx)
	suite.Require().NoError(err)
	suite.T().Logf("resp: %v\n", resp)
}

func (suite *HttpClientTestSuite) TestGetAccount() {
	ctx := context.Background()
	nileTopAccount := TestNetAccountTop

	resp, err := suite.client.GetAccount(ctx, nileTopAccount)
	suite.Require().NoError(err)
	suite.T().Logf("resp: %v\n", resp)
}

func (suite *HttpClientTestSuite) TestGetAccountResource() {
	ctx := context.Background()
	nileTopAccount := TestNetAccountTop

	resp, err := suite.client.GetAccountResource(ctx, nileTopAccount)
	suite.Require().NoError(err)
	suite.T().Logf("resp: %v\n", resp)
}

func (suite *HttpClientTestSuite) TestInvokeContract() {
	require := suite.Require()

	contractAddress := TestNetContractAddressJst
	from := TestNetAccountFoo
	to := TestNetAccountBar

	toAddrB, _ := common.DecodeCheck(to)
	toAddrHex := hex.EncodeToString(toAddrB)

	param := map[string]any{
		"address": toAddrHex,
		"uint256": big.NewInt(1).String(),
	}

	p, _ := json.Marshal(param)

	estimate, err := suite.client.TriggerConstantContracts(
		context.Background(),
		from,
		contractAddress,
		"transfer(address,uint256)",
		string(p),
	)
	require.NoError(err)

	for _, r := range estimate.ConstantResult {
		suite.T().Logf("a: %s\n", string(r))
	}
}

func (suite *HttpClientTestSuite) TestEstimateEnergy() {
	require := suite.Require()

	contractAddress := TestNetContractAddressUsdt
	from := TestNetAccountBaz
	to := TestNetAccountBar

	amount := big.NewInt(1)
	params := []map[string]any{
		{
			"address": to,
		},
		{
			"uint256": amount.String(),
		},
	}
	b, _ := json.Marshal(params)

	estimate, err := suite.client.EstimateEnergy(
		context.Background(),
		from,
		contractAddress,
		"transfer(address,uint256)",
		string(b),
		0,
	)
	require.NoError(err)

	suite.T().Logf("EnergyRequired: %v\n", estimate.EnergyRequired)
}

func (suite *HttpClientTestSuite) TestFreezeBalanceV2() {
	ctx := context.Background()
	address := TestNetAccountBar
	invalidAmount := big.NewInt(1)
	validAmount := big.NewInt(10000000)

	// Bandwidth
	_, err := suite.client.FreezeBalanceV2(ctx, address, httpclient.ResourceBandwidth, invalidAmount)
	suite.Error(err)

	resp, err := suite.client.FreezeBalanceV2(ctx, address, httpclient.ResourceBandwidth, validAmount)
	suite.Require().NoError(err)
	suite.T().Logf("FreezeBalanceV2 Bandwidth: %v\n", resp.RawData.Contract[0].Parameter.Value.FrozenBalance)

	// Energy
	_, err = suite.client.FreezeBalanceV2(ctx, address, httpclient.ResourceEnergy, invalidAmount)
	suite.Error(err)

	resp, err = suite.client.FreezeBalanceV2(ctx, address, httpclient.ResourceEnergy, validAmount)
	suite.Require().NoError(err)
	suite.T().Logf("FreezeBalanceV2 Energy: %v\n", resp.RawData.Contract[0].Parameter.Value.FrozenBalance)
}

func (suite *HttpClientTestSuite) TestUnfreezeBalanceV2() {
	ctx := context.Background()
	address := TestNetAccountBar
	invalidAmount := big.NewInt(1)
	validAmount := big.NewInt(10000000)

	// Bandwidth
	_, err := suite.client.UnfreezeBalanceV2(ctx, address, httpclient.ResourceBandwidth, invalidAmount)
	suite.Error(err)

	resp, err := suite.client.UnfreezeBalanceV2(ctx, address, httpclient.ResourceBandwidth, validAmount)
	suite.Require().NoError(err)
	suite.T().Logf("UnfreezeBalanceV2 Bandwidth: %v\n", resp.RawData.Contract[0].Parameter.Value.FrozenBalance)

	// Energy
	_, err = suite.client.UnfreezeBalanceV2(ctx, address, httpclient.ResourceEnergy, invalidAmount)
	suite.Error(err)

	_, err = suite.client.UnfreezeBalanceV2(ctx, address, httpclient.ResourceEnergy, validAmount)
	suite.Require().NoError(err)
	suite.T().Logf("UnfreezeBalanceV2 Energy: %v\n", resp.RawData.Contract[0].Parameter.Value.FrozenBalance)
}

func (suite *HttpClientTestSuite) TestWithdrawExpireUnfreeze() {
	ctx := context.Background()
	address := TestNetAccountBar

	resp, err := suite.client.WithdrawExpireUnfreeze(ctx, address)
	suite.Require().NoError(err)

	suite.T().Logf("WithdrawExpireUnfreeze: %v\n", resp)
}

func (suite *HttpClientTestSuite) TestDelegateResource() {
	ctx := context.Background()
	ownerAddress := TestNetAccountBar
	receiverAddress := TestNetAccountBaz
	amount := big.NewInt(10000000)

	resp, err := suite.client.DelegateResource(ctx, ownerAddress, receiverAddress, httpclient.ResourceBandwidth, amount)
	suite.Require().NoError(err)

	suite.T().Logf("DelegateResource Bandwidth: %v\n", resp)

	resp, err = suite.client.DelegateResource(ctx, ownerAddress, receiverAddress, httpclient.ResourceEnergy, amount)
	suite.Require().NoError(err)

	suite.T().Logf("DelegateResource Energy: %v\n", resp)
}

func (suite *HttpClientTestSuite) TestUnDelegateResource() {
	ctx := context.Background()
	ownerAddress := TestNetAccountBar
	receiverAddress := TestNetAccountBaz
	amount := big.NewInt(10000000)

	resp, err := suite.client.UnDelegateResource(ctx, ownerAddress, receiverAddress, httpclient.ResourceBandwidth, amount)
	suite.Require().NoError(err)

	suite.T().Logf("UnDelegateResource Bandwidth: %v\n", resp)

	resp, err = suite.client.UnDelegateResource(ctx, ownerAddress, receiverAddress, httpclient.ResourceEnergy, amount)
	suite.Require().NoError(err)

	suite.T().Logf("UnDelegateResource Energy: %v\n", resp)
}
