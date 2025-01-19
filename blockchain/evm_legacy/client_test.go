package evm_legacy_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/openweb3-io/crosschain/blockchain/evm_legacy"
	testtypes "github.com/openweb3-io/crosschain/testutil/types"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := evm_legacy.NewClient(&xc.ChainConfig{})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestFetchTxInput(t *testing.T) {

	vectors := []struct {
		name       string
		resp       interface{}
		val        *evm_legacy.TxInput
		err        string
		multiplier float64
	}{
		// Send ether normal tx
		{
			name: "fetchTxInput legacy",
			resp: []string{
				// eth_getTransactionCount
				`"0x6"`,
				// eth_gasPrice
				`"0xba43b7400"`,
				// eth_estimateGas
				`"0x52e4"`,
			},
			val: &evm_legacy.TxInput{
				Nonce:    6,
				GasLimit: 21220,
				GasPrice: xc.NewBigIntFromUint64(50000000000),
			},
			err:        "",
			multiplier: 1.0,
		},
		{
			name: "fetchTxInput legacy",
			resp: []string{
				// eth_getTransactionCount
				`"0x6"`,
				// eth_gasPrice
				`"0xba43b7400"`,
				// eth_estimateGas
				`"0x52e4"`,
			},
			val: &evm_legacy.TxInput{
				Nonce:    6,
				GasLimit: 21220,
				GasPrice: xc.NewBigIntFromUint64(100000000000),
			},
			err:        "",
			multiplier: 2.0,
		},
	}
	for _, v := range vectors {
		fmt.Println("testing ", v.name)
		server, close := testtypes.MockJSONRPC(t, v.resp)
		defer close()
		cfg := &xc.ChainConfig{
			Client: &xc.ClientConfig{
				URL: server.URL,
			},
			Chain:              xc.ETH,
			Blockchain:         xc.BlockchainEVMLegacy,
			ChainGasMultiplier: v.multiplier,
		}
		client, err := evm_legacy.NewClient(cfg)
		require.NoError(t, err)
		input, err := client.FetchLegacyTxInput(context.Background(), xc.Address(""), xc.Address(""), nil)
		require.NoError(t, err)
		if v.err != "" {
			require.Equal(t, evm_legacy.TxInput{}, input)
			require.ErrorContains(t, err, v.err)
		} else {
			require.Nil(t, err)
			require.NotNil(t, input)
			require.Equal(t, v.val, input)
		}
	}
}

func TestFetchBalance_ArbETH(t *testing.T) {
	cfg := &xc.ChainConfig{
		Client: &xc.ClientConfig{
			URL: "https://arbitrum.rpc.subquery.network/public",
		},
		Chain:      xc.ArbETH,
		Blockchain: xc.BlockchainEVM,
	}

	client, err := evm_legacy.NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	balance, err := client.FetchBalance(context.Background(), xc.Address("0xAAB961723E77c77AF13968F89bEB80EE8c815cf4"))
	require.NoError(t, err)
	require.NotNil(t, balance)

	log.Printf("balance: %v\n", balance)

	balance, err = client.FetchBalanceForAsset(context.Background(), xc.Address("0xAAB961723E77c77AF13968F89bEB80EE8c815cf4"), xc.ContractAddress("0x58CB98A966F62aA6F2190eB3AA03132A0c3de3D5"))
	require.NoError(t, err)
	require.NotNil(t, balance)
	log.Printf("OpenWorld balance: %v\n", balance)
}
