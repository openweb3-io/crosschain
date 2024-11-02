package address_test

import (
	"encoding/hex"
	"log"
	"testing"

	"github.com/openweb3-io/crosschain/blockchain/ton/address"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/require"
)

func TestNewAddressBuilder(t *testing.T) {
	builder, err := address.NewAddressBuilder(&xc_types.ChainConfig{})
	require.NoError(t, err)
	require.NotNil(t, builder)
}

func TestGetAddressFromPublicKey(t *testing.T) {
	builder, _ := address.NewAddressBuilder(&xc_types.ChainConfig{})
	bytes, _ := hex.DecodeString("c1172b7926116d2a396bd7d69b9880cc0657e8ba2db9f62b4c210c518321c8b1")
	address, err := builder.GetAddressFromPublicKey(bytes)
	require.NoError(t, err)
	require.Equal(t, xc_types.Address("EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2"), address)
}

func TestParseAddress(t *testing.T) {
	builder, _ := address.NewAddressBuilder(&xc_types.ChainConfig{})
	bytes, _ := hex.DecodeString("c1172b7926116d2a396bd7d69b9880cc0657e8ba2db9f62b4c210c518321c8b1")
	derivedAddr, err := builder.GetAddressFromPublicKey(bytes)
	require.NoError(t, err)
	require.Equal(t, xc_types.Address("EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2"), derivedAddr)

	addr, err := address.ParseAddress("EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2", "testnet")
	require.NoError(t, err)
	require.Equal(t, "EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2", addr.String())

	addr, err = address.ParseAddress("EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2", "testnet")
	require.NoError(t, err)
	require.Equal(t, "EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2", addr.String())

	addr, err = address.ParseAddress("0QAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSvD5", "testnet")
	require.NoError(t, err)
	require.Equal(t, "0QAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSvD5", addr.String())

	// use alternative address format
	addr, err = address.ParseAddress("0:237E5119FFA2A028CC4F95C9CA37566852F1DD4D3EA15704D6F791065507DE4A", "mainnet")
	require.NoError(t, err)
	require.Equal(t, "EQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSha2", addr.String())
	// addr, err = address.ParseAddress("0:337339704B339026E9485A854FED6D412E4EA0508758F92FEC9730593DAE32E7", "testnet")
	// require.NoError(t, err)
	// require.Equal(t, "UQAzczlwSzOQJulIWoVP7W1BLk6gUIdY-S_slzBZPa4y5wn-", addr.String())
}

func TestParseTestnetAddress(t *testing.T) {
	builder, _ := address.NewAddressBuilder(&xc_types.ChainConfig{Network: "testnet"})
	bytes, _ := hex.DecodeString("c1172b7926116d2a396bd7d69b9880cc0657e8ba2db9f62b4c210c518321c8b1")
	derivedAddr, err := builder.GetAddressFromPublicKey(bytes)
	require.NoError(t, err)
	require.Equal(t, xc_types.Address("kQAjflEZ_6KgKMxPlcnKN1ZoUvHdTT6hVwTW95EGVQfeSq08"), derivedAddr)
}

// func TestParseAddressMetadata(t *testing.T) {
// 	addr1, err := tonutil.ParseAddr("EQAiboDEv_qRrcEdrYdwbVLNOXBHwShFbtKGbQVJ2OKxY0to")
// 	require.NoError(t, err)

// 	addr2, err := tonutil.ParseAddr("kQAiboDEv_qRrcEdrYdwbVLNOXBHwShFbtKGbQVJ2OKxY_Di")
// 	require.NoError(t, err)

// 	require.Equal(t, addr1, addr2)
// }

func TestGetAddressShard(t *testing.T) {
	addr, err := address.ParseAddress("EQAiboDEv_qRrcEdrYdwbVLNOXBHwShFbtKGbQVJ2OKxY0to", "mainnet")
	log.Printf("0x20 addr: %v, shard:0x%02x", addr.String(), addr.Data()[0])
	require.NoError(t, err)

	addr, err = address.ParseAddress("UQCYqk93_LQf4sDuTQk0yfmTpJARwvEv9eD2lHa5rYNmNclA", "mainnet")
	log.Printf("addr: %v, shard:0x%02x", addr.String(), addr.Data()[0])
	require.NoError(t, err)

	addr, err = address.ParseAddress("Uf_mlXHnufWO3-vvopflR_NpIFMiidvp_xt20Qf8usMBBPEE", "mainnet")
	log.Printf("0x80 addr: %v, shard:0x%02x", addr.String(), addr.Data()[0])
	require.NoError(t, err)
}
