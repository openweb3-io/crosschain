package types

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

const FLOAT_PRECISION = 6

// BigInt is a big integer amount as blockchain expects it for tx.
type BigInt big.Int

// AmountHumanReadable is a decimal amount as a human expects it for readability.
type AmountHumanReadable decimal.Decimal

func (amount BigInt) String() string {
	bigInt := big.Int(amount)
	return bigInt.String()
}

// Int converts an BigInt into *bit.Int
func (amount BigInt) Int() *big.Int {
	bigInt := big.Int(amount)
	return &bigInt
}

func (amount BigInt) Sign() int {
	bigInt := big.Int(amount)
	return bigInt.Sign()
}

// Uint64 converts an BigInt into uint64
func (amount BigInt) Uint64() uint64 {
	bigInt := big.Int(amount)
	return bigInt.Uint64()
}

// UnmaskFloat64 converts an BigInt into float64 given the number of decimals
func (amount BigInt) UnmaskFloat64() float64 {
	bigInt := big.Int(amount)
	bigFloat := new(big.Float).SetInt(&bigInt)
	exponent := new(big.Float).SetFloat64(math.Pow10(FLOAT_PRECISION))
	bigFloat = bigFloat.Quo(bigFloat, exponent)
	f64, _ := bigFloat.Float64()
	return f64
}

// Use the underlying big.Int.Cmp()
func (amount *BigInt) Cmp(other *BigInt) int {
	return amount.Int().Cmp(other.Int())
}

// Use the underlying big.Int.Add()
func (amount *BigInt) Add(x *BigInt) BigInt {
	sum := *amount
	return BigInt(*sum.Int().Add(sum.Int(), x.Int()))
}

// Use the underlying big.Int.Sub()
func (amount *BigInt) Sub(x *BigInt) BigInt {
	diff := *amount
	return BigInt(*diff.Int().Sub(diff.Int(), x.Int()))
}

// Use the underlying big.Int.Mul()
func (amount *BigInt) Mul(x *BigInt) BigInt {
	prod := *amount
	return BigInt(*prod.Int().Mul(prod.Int(), x.Int()))
}

// Use the underlying big.Int.Div()
func (amount *BigInt) Div(x *BigInt) BigInt {
	quot := *amount
	return BigInt(*quot.Int().Div(quot.Int(), x.Int()))
}

func (amount *BigInt) Abs() BigInt {
	abs := *amount
	return BigInt(*abs.Int().Abs(abs.Int()))
}

var zero = big.NewInt(0)

func (amount *BigInt) IsZero() bool {
	return amount.Int().Cmp(zero) == 0
}

func (amount *BigInt) ToHuman(decimals int32) AmountHumanReadable {
	dec := decimal.NewFromBigInt(amount.Int(), -decimals)
	return AmountHumanReadable(dec)
}

func (amount BigInt) ApplyGasPriceMultiplier(chain *ChainConfig) BigInt {
	if chain.ChainGasMultiplier > 0.01 {
		return MultiplyByFloat(amount, chain.ChainGasMultiplier)
	}
	// no multiplier configured, return same
	return amount
}

func MultiplyByFloat(amount BigInt, multiplier float64) BigInt {
	if amount.Uint64() == 0 {
		return amount
	}
	// We are computing (100000 * multiplier * amount) / 100000
	precision := uint64(1000000)
	multBig := NewBigIntFromUint64(uint64(float64(precision) * multiplier))
	divBig := NewBigIntFromUint64(precision)
	product := multBig.Mul(&amount)
	result := product.Div(&divBig)
	return result
}

// NewBigIntFromUint64 creates a new BigInt from a uint64
func NewBigIntFromUint64(u64 uint64) BigInt {
	bigInt := new(big.Int).SetUint64(u64)
	return BigInt(*bigInt)
}

// NewBigIntFromInt64 creates a new BigInt from a uint64
func NewBigIntFromInt64(u64 int64) BigInt {
	bigInt := new(big.Int).SetInt64(u64)
	return BigInt(*bigInt)
}

// NewBigIntToMaskFloat64 creates a new BigInt as a float64 times 10^FLOAT_PRECISION
func NewBigIntToMaskFloat64(f64 float64) BigInt {
	bigFloat := new(big.Float).SetFloat64(f64)
	exponent := new(big.Float).SetFloat64(math.Pow10(FLOAT_PRECISION))
	bigFloat = bigFloat.Mul(bigFloat, exponent)
	var bigInt big.Int
	bigFloat.Int(&bigInt)
	return BigInt(bigInt)
}

// NewBigIntFromStr creates a new BigInt from a string
func NewBigIntFromStr(str string) BigInt {
	var ok bool
	var bigInt *big.Int
	bigInt, ok = new(big.Int).SetString(str, 0)
	if !ok {
		return NewBigIntFromUint64(0)
	}
	return BigInt(*bigInt)
}

// NewAmountHumanReadableFromStr creates a new AmountHumanReadable from a string
func NewAmountHumanReadableFromStr(str string) (AmountHumanReadable, error) {
	decimal, err := decimal.NewFromString(str)
	return AmountHumanReadable(decimal), err
}

func (amount AmountHumanReadable) ToBlockchain(decimals int32) BigInt {
	factor := decimal.NewFromInt32(10).Pow(decimal.NewFromInt32(decimals))
	raised := ((decimal.Decimal)(amount)).Mul(factor)
	return BigInt(*raised.BigInt())
}

func (amount AmountHumanReadable) String() string {
	return decimal.Decimal(amount).String()
}

func (amount AmountHumanReadable) Div(x AmountHumanReadable) AmountHumanReadable {
	return AmountHumanReadable(decimal.Decimal(amount).Div(decimal.Decimal(x)))
}

func (b AmountHumanReadable) MarshalJSON() ([]byte, error) {
	return []byte("\"" + b.String() + "\""), nil
}

func (b *AmountHumanReadable) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	str := strings.Trim(string(p), "\"")
	decimal, err := decimal.NewFromString(str)
	if err != nil {
		return err
	}
	*b = AmountHumanReadable(decimal)
	return nil
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	return []byte("\"" + b.String() + "\""), nil
}

func (b *BigInt) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	str := strings.Trim(string(p), "\"")
	var z big.Int
	_, ok := z.SetString(str, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", p)
	}
	*b = BigInt(z)
	return nil
}
