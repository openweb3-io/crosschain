package liteserver_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

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

	// addr, err := address.ParseAddr("UQAAPs-fYmdebzSwCd76x4oL8g80O8pubf0FiiO6EdX6MX3Z")
	addr, err := address.ParseAddr("UQAoD2KBwmnfVJwBbm3-xLg67TYG5Qv5sn5x9JddUWzzGRZy")
	// addr, err := address.ParseAddr("EQB-U9ZcM16Sc2p-xcSyhTCU7YGK8UH5Qvq4CFnM2ejNgU_x")
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

func (suite *ClientTestSuite) Test_GetBalanceForAsset() {
	rounds := 5
	roundRates := make([]float64, 0)
	errMap := make(map[string]int)
	minCost := int64(math.MaxInt64)
	maxCost := int64(0)
	for r := 0; r < rounds; r++ {
		failedCnt := 0
		Total := 0
		ctx := context.Background()
		for i := 0; i < 1000; i++ {
			st := time.Now()
			balance, err := suite.client.FetchBalanceForAsset(ctx, "UQAoD2KBwmnfVJwBbm3-xLg67TYG5Qv5sn5x9JddUWzzGRZy", "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs")
			if err != nil {
				failedCnt++
			}
			Total++

			cost := time.Since(st).Milliseconds()
			if cost > maxCost {
				maxCost = cost
			}
			if cost < minCost {
				minCost = cost
			}
			fmt.Println(
				"i", i+1,
				"balance", balance,
				"err", err,
				"time", cost,
				"round", r+1,
				"failed", failedCnt,
				"total", Total,
				"failed rate", float64(failedCnt)/float64(Total),
			)

			if err != nil {
				errMap[err.Error()]++
			}
		}
		roundRates = append(roundRates, float64(failedCnt)/float64(Total))
	}

	for r, rate := range roundRates {
		fmt.Println("round", r+1, "rate", rate)
	}

	fmt.Println("max cost", maxCost, "min cost", minCost)

	for err, cnt := range errMap {
		fmt.Println("err", err, "cnt", cnt)
	}
}
