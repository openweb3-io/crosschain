package types

import (
	"fmt"
	"strings"
)

// NativeAsset is an asset on a blockchain used to pay gas fees.
// In Crosschain, for simplicity, a NativeAsset represents a chain.
type NativeAsset string

// List of supported NativeAsset
const (
	ACA    = NativeAsset("ACA")    // Acala
	APTOS  = NativeAsset("APTOS")  // APTOS
	ArbETH = NativeAsset("ArbETH") // Arbitrum
	ATOM   = NativeAsset("ATOM")   // Cosmos
	AurETH = NativeAsset("AurETH") // Aurora
	AVAX   = NativeAsset("AVAX")   // Avalanche
	BERA   = NativeAsset("BERA")   // Berachain
	BCH    = NativeAsset("BCH")    // Bitcoin Cash
	BNB    = NativeAsset("BNB")    // Binance Coin
	BTC    = NativeAsset("BTC")    // Bitcoin
	CELO   = NativeAsset("CELO")   // Celo
	CHZ    = NativeAsset("CHZ")    // Chiliz
	CHZ2   = NativeAsset("CHZ2")   // Chiliz 2.0
	DOGE   = NativeAsset("DOGE")   // Dogecoin
	DOT    = NativeAsset("DOT")    // Polkadot
	ETC    = NativeAsset("ETC")    // Ethereum Classic
	ETH    = NativeAsset("ETH")    // Ethereum
	ETHW   = NativeAsset("ETHW")   // Ethereum PoW
	FTM    = NativeAsset("FTM")    // Fantom
	HASH   = NativeAsset("HASH")   // Provenance
	INJ    = NativeAsset("INJ")    // Injective
	LTC    = NativeAsset("LTC")    // Litecoin
	LUNA   = NativeAsset("LUNA")   // Terra V2
	LUNC   = NativeAsset("LUNC")   // Terra Classic
	KAR    = NativeAsset("KAR")    // Karura
	KLAY   = NativeAsset("KLAY")   // Klaytn
	KSM    = NativeAsset("KSM")    // Kusama
	XDC    = NativeAsset("XDC")    // XinFin
	MATIC  = NativeAsset("MATIC")  // Polygon
	OAS    = NativeAsset("OAS")    // Oasys (not Oasis!)
	OptETH = NativeAsset("OptETH") // Optimism
	EmROSE = NativeAsset("EmROSE") // Rose (Oasis EVM-compat "Emerald" parachain)
	SOL    = NativeAsset("SOL")    // Solana
	SUI    = NativeAsset("SUI")    // SUI
	XPLA   = NativeAsset("XPLA")   // XPLA
	TAO    = NativeAsset("TAO")    // Bittensor
	TIA    = NativeAsset("TIA")    // celestia
	TON    = NativeAsset("TON")    // TON
	TRX    = NativeAsset("TRX")    // TRON
	SEI    = NativeAsset("SEI")    // Sei
)

var NativeAssetList []NativeAsset = []NativeAsset{
	BCH,
	BTC,
	DOGE,
	LTC,
	ACA,
	APTOS,
	ArbETH,
	ATOM,
	AurETH,
	AVAX,
	BERA,
	BNB,
	CELO,
	CHZ,
	CHZ2,
	DOT,
	ETC,
	ETH,
	ETHW,
	FTM,
	INJ,
	HASH,
	LUNA,
	LUNC,
	KAR,
	KLAY,
	KSM,
	XDC,
	MATIC,
	OAS,
	OptETH,
	EmROSE,
	SOL,
	SUI,
	XPLA,
	TAO,
	TIA,
	TON,
	TRX,
	SEI,
}

type StakingConfig struct {
	// the contract used for staking, if relevant
	StakeContract string `yaml:"stake_contract,omitempty"`
	// the contract used for unstaking, if relevant
	UnstakeContract string `yaml:"unstake_contract,omitempty"`
	// Compatible providers for staking
	Providers []StakingProvider `yaml:"providers,omitempty"`
}

func (staking *StakingConfig) Enabled() bool {
	return len(staking.Providers) > 0
}

type ChainConfig struct {
	Chain NativeAsset `yaml:"chain,omitempty"`

	URL                string
	ChainGasMultiplier float64 `yaml:"chain_gas_multiplier,omitempty"`
	ChainMaxGasPrice   float64 `yaml:"chain_max_gas_price,omitempty"`
	ChainMinGasPrice   float64 `yaml:"chain_min_gas_price,omitempty"`
	Decimals           int32   `yaml:"decimals,omitempty"`
	Provider           string  `yaml:"provider,omitempty"`
	ChainID            int64   `yaml:"chain_id,omitempty"`

	ExplorerURL string `yaml:"explorer_url,omitempty"`
	NoGasFees   bool   `yaml:"no_gas_fees,omitempty"`

	Staking StakingConfig `yaml:"staking,omitempty"`

	// Internal
	// dereferenced api token if used
	AuthSecret string `yaml:"-"`
}

func (asset *ChainConfig) ID() AssetID {
	return GetAssetIDFromAsset("", asset.Chain)
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
	ID() AssetID
	GetContract() string
	GetDecimals() int32
	GetChain() *ChainConfig
}

type TokenAssetConfig struct {
	Asset       string       `yaml:"asset,omitempty"`
	Chain       NativeAsset  `yaml:"chain,omitempty"`
	Decimals    int32        `yaml:"decimals,omitempty"`
	Contract    string       `yaml:"contract,omitempty"`
	ChainConfig *ChainConfig `yaml:"-"`
}

func (c *TokenAssetConfig) String() string {
	return fmt.Sprintf(
		"TokenAssetConfig(id=%s, asset=%s chain=%v decimals=%d contract=%s)",
		c.ID(),
		c.Asset,
		c.ChainConfig,
		c.Decimals,
		c.Contract,
	)
}

func (asset *TokenAssetConfig) ID() AssetID {
	return AssetID("not impl")
}

func (asset *TokenAssetConfig) GetDecimals() int32 {
	return asset.Decimals
}

func (asset *TokenAssetConfig) GetContract() string {
	return asset.Contract
}

func (asset *TokenAssetConfig) GetChain() *ChainConfig {
	return asset.ChainConfig
}

func LegacyParseAssetAndNativeAsset(asset string, nativeAsset string) (string, NativeAsset) {
	if asset == "" && nativeAsset == "" {
		return "", ""
	}
	if asset == "" && nativeAsset != "" {
		asset = nativeAsset
	}

	assetSplit := strings.Split(asset, ".")
	if len(assetSplit) == 2 && NativeAsset(assetSplit[1]).IsValid() {
		asset = assetSplit[0]
		if nativeAsset == "" {
			nativeAsset = assetSplit[1]
		}
	}
	validNative := NativeAsset(asset).IsValid()

	if nativeAsset == "" {
		if validNative {
			nativeAsset = asset
		} else {
			nativeAsset = "ETH"
		}
	}

	return asset, NativeAsset(nativeAsset)
}

// GetAssetIDFromAsset return the canonical AssetID given two input strings asset, nativeAsset.
// Input can come from user input.
// Examples:
// - GetAssetIDFromAsset("USDC", "") -> "USDC.ETH"
// - GetAssetIDFromAsset("USDC", "ETH") -> "USDC.ETH"
// - GetAssetIDFromAsset("USDC", "SOL") -> "USDC.SOL"
// - GetAssetIDFromAsset("USDC.SOL", "") -> "USDC.SOL"
// See tests for more examples.
func GetAssetIDFromAsset(asset string, nativeAsset NativeAsset) AssetID {
	// id is SYMBOL for ERC20 and SYMBOL.CHAIN for others
	// e.g. BTC, ETH, USDC, SOL, USDC.SOL
	asset, nativeAsset = LegacyParseAssetAndNativeAsset(asset, string(nativeAsset))
	validNative := NativeAsset(asset).IsValid()

	// native asset, e.g. BTC, ETH, SOL
	if asset == string(nativeAsset) {
		return AssetID(asset)
	}
	if nativeAsset == "ETH" && !validNative {
		return AssetID(asset + ".ETH")
	}
	// token, e.g. USDC, USDC.SOL
	return AssetID(asset + "." + string(nativeAsset))
}
