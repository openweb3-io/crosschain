package types_test

import (
	"testing"

	. "github.com/openweb3-io/crosschain/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type CrosschainTestSuite struct {
	suite.Suite
}

func TestCrosschain(t *testing.T) {
	suite.Run(t, new(CrosschainTestSuite))
}

func (s *CrosschainTestSuite) TestNewBigIntFromUint64() {
	require := s.Require()
	amount := NewBigIntFromUint64(123)
	require.NotNil(amount)
	require.Equal(amount.Uint64(), uint64(123))
	require.Equal(amount.String(), "123")
}

func (s *CrosschainTestSuite) TestNewBigIntFromFloat64() {
	require := s.Require()
	amount := NewBigIntToMaskFloat64(1.23)
	require.NotNil(amount)
	require.Equal(amount.Uint64(), uint64(1230000))
	require.Equal(amount.String(), "1230000")

	amountFloat := amount.UnmaskFloat64()
	require.Equal(amountFloat, 1.23)
}

func (s *CrosschainTestSuite) TestAmountHumanReadable() {
	require := s.Require()
	amountDec, _ := decimal.NewFromString("10.3")
	amount := AmountHumanReadable(amountDec)
	require.NotNil(amount)
	require.Equal(amount.String(), "10.3")
}

func (s *CrosschainTestSuite) TestNewAmountHumanReadableFromStr() {
	require := s.Require()
	amount, err := NewAmountHumanReadableFromStr("10.3")
	require.NoError(err)
	require.NotNil(amount)
	require.Equal(amount.String(), "10.3")

	amount, err = NewAmountHumanReadableFromStr("0")
	require.NoError(err)
	require.NotNil(amount)
	require.Equal(amount.String(), "0")

	amount, err = NewAmountHumanReadableFromStr("")
	require.Error(err)
	require.NotNil(amount)
	require.Equal(amount.String(), "0")

	amount, err = NewAmountHumanReadableFromStr("invalid")
	require.Error(err)
	require.NotNil(amount)
	require.Equal(amount.String(), "0")
}

func (s *CrosschainTestSuite) TestNewBlockchainAmountStr() {
	require := s.Require()
	amount := NewBigIntFromStr("10")
	require.EqualValues(amount.Uint64(), 10)

	amount = NewBigIntFromStr("10.1")
	require.EqualValues(amount.Uint64(), 0)

	amount = NewBigIntFromStr("0x10")
	require.EqualValues(amount.Uint64(), 16)
}

func (s *CrosschainTestSuite) TestLegacyGasCalculation() {
	require := s.Require()

	// Multiplier should default to 1
	require.EqualValues(
		1000,
		NewBigIntFromUint64(1000).ApplyGasPriceMultiplier(&ChainConfig{}).Uint64(),
	)
	require.EqualValues(
		1200,
		NewBigIntFromUint64(1000).ApplyGasPriceMultiplier(&ChainConfig{ChainGasMultiplier: 1.2}).Uint64(),
	)
	require.EqualValues(
		500,
		NewBigIntFromUint64(1000).ApplyGasPriceMultiplier(&ChainConfig{ChainGasMultiplier: .5}).Uint64(),
	)
	require.EqualValues(
		1500,
		NewBigIntFromUint64(1000).ApplyGasPriceMultiplier(&ChainConfig{ChainGasMultiplier: 1.5}).Uint64(),
	)
}
