package exit_request_test

import (
	"encoding/hex"
	"testing"

	"github.com/openweb3-io/crosschain/blockchain/evm/abi/exit_request"
	"github.com/test-go/testify/require"
)

func mustHex(s string) []byte {
	bz, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return bz
}

func TestSerializeBatchDeposit(t *testing.T) {
	data, err := exit_request.Serialize([][]byte{
		mustHex("850F24E0A4B2B5568340891FCAECC2D08A788F03F13D2295419E6860545499A24975F2E4154992EBC401925E93A80B3C"),
	})
	require.NoError(t, err)
	expected := "254209ba0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000030850f24e0a4b2b5568340891fcaecc2d08a788f03f13d2295419e6860545499a24975f2e4154992ebc401925e93a80b3c00000000000000000000000000000000"
	require.Equal(t, expected, hex.EncodeToString(data))

	data, err = exit_request.Serialize([][]byte{
		mustHex("97F65FD297DB657FEF6F2A5D2461A3178A0A928542A085F661D8039BCDEA8C4EA064119453220173694094A884560ABC"),
		mustHex("8C1B6A0AEE2CEAFD6E3FDABD7538ABCA0F3D0F94512E79FD6A992582C4E1609A1414A51777BC78D05E148201E894D299"),
	})
	require.NoError(t, err)
	expected = "254209ba00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000003097f65fd297db657fef6f2a5d2461a3178a0a928542a085f661d8039bcdea8c4ea064119453220173694094a884560abc0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000308c1b6a0aee2ceafd6e3fdabd7538abca0f3d0f94512e79fd6a992582c4e1609a1414a51777bc78d05e148201e894d29900000000000000000000000000000000"
	require.Equal(t, expected, hex.EncodeToString(data))
}