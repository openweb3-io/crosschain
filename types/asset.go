package types

import "fmt"

type ChainConfig struct {
	URL                string
	ChainGasMultiplier float64 `yaml:"chain_gas_multiplier,omitempty"`
	ChainMaxGasPrice   float64 `yaml:"chain_max_gas_price,omitempty"`
	ChainMinGasPrice   float64 `yaml:"chain_min_gas_price,omitempty"`
	Decimals           int32   `yaml:"decimals,omitempty"`
}

func (asset *ChainConfig) GetDecimals() int32 {
	return asset.Decimals
}

func (asset *ChainConfig) GetChain() *ChainConfig {
	return asset
}

func (native *ChainConfig) GetContract() string {
	return ""
}

type AssetID string

type IAsset interface {
	// ID() AssetID
	GetContract() string
	GetDecimals() int32
	GetChain() *ChainConfig
}

type TokenAssetConfig struct {
	Asset       string       `yaml:"asset,omitempty"`
	Decimals    int32        `yaml:"decimals,omitempty"`
	Contract    string       `yaml:"contract,omitempty"`
	ChainConfig *ChainConfig `yaml:"-"`
}

func (c *TokenAssetConfig) String() string {
	return fmt.Sprintf(
		"TokenAssetConfig(asset=%s chain=%v decimals=%d contract=%s)",
		// c.ID(),
		c.Asset,
		c.ChainConfig,
		c.Decimals,
		c.Contract,
	)
}

/*
func (asset *TokenAssetConfig) ID() AssetID {
	return AssetID("not impl")
}
*/

func (asset *TokenAssetConfig) GetDecimals() int32 {
	return asset.Decimals
}

func (asset *TokenAssetConfig) GetContract() string {
	return asset.Contract
}

func (asset *TokenAssetConfig) GetChain() *ChainConfig {
	return asset.ChainConfig
}
