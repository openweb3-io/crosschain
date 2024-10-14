package httpclient_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	httpclient "github.com/openweb3-io/crosschain/blockchain/tron/http_client"
	"github.com/stretchr/testify/suite"
)

type HttpClientTestSuite struct {
	suite.Suite
	client *httpclient.Client
}

func TestHttpClientTestSuite(t *testing.T) {
	suite.Run(t, new(HttpClientTestSuite))
}

func (suite *HttpClientTestSuite) SetupTest() {
	// endpoint := "https://go.getblock.io/41513af034b3452cb27c8e5ca67b6e68"
	// for testnet: https://nile.tronscan.org
	endpoint := "https://go.getblock.io/4e19dacf44974a3d8e40031ef8aca8b8"

	var err error
	suite.client, err = httpclient.NewHttpClient(endpoint)
	suite.Require().NoError(err)
}

func (suite *HttpClientTestSuite) TestGetChainParameters() {
	ctx := context.Background()

	resp, err := suite.client.GetChainParameters(ctx)
	suite.Require().NoError(err)
	fmt.Printf("resp: %v\n", resp)
}

func (suite *HttpClientTestSuite) TestInvokeContract() {
	require := suite.Require()

	contractAddress := "TF17BgPaZYbz8oxbjhriubPDsA7ArKoLX3"
	from := "THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F"
	to := "TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA"

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
		fmt.Printf("a: %s\n", string([]byte(r)))
	}
}

func (suite *HttpClientTestSuite) Test_Estimateenergy() {
	require := suite.Require()

	contractAddress := "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj" // "TF17BgPaZYbz8oxbjhriubPDsA7ArKoLX3"
	from := "THjVQt6hpwZyWnkDm1bHfPvdgysQFoN8AL"            // "THKrowiEfCe8evdbaBzDDvQjM5DGeB3s3F"
	to := "TVjsyZ7fYF3qLF6BQgPmTEZy1xrNNyVAAA"

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

	fmt.Printf("EnergyRequired: %v\n", estimate.EnergyRequired)
}
