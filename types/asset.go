package types

type ChainConfig struct {
	ChainGasMultiplier float64 `yaml:"chain_gas_multiplier,omitempty"`
	ChainMaxGasPrice   float64 `yaml:"chain_max_gas_price,omitempty"`
	ChainMinGasPrice   float64 `yaml:"chain_min_gas_price,omitempty"`
}
