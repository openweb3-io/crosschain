package types

import (
	"fmt"
	"slices"
	"strings"
)

type SignatureType string

const (
	K256Keccak = SignatureType("k256-keccak")
	K256Sha256 = SignatureType("k256-sha256")
	Ed255      = SignatureType("ed255")
	Schnorr    = SignatureType("schnorr")
)

// Blockchain is the type of a chain
type Blockchain string

// List of supported Blockchain
const (
	BlockchainAptos         = Blockchain("aptos")
	BlockchainBitcoin       = Blockchain("bitcoin")
	BlockchainBitcoinCash   = Blockchain("bitcoin-cash")
	BlockchainBitcoinLegacy = Blockchain("bitcoin-legacy")
	BlockchainCosmos        = Blockchain("cosmos")
	BlockchainCosmosEvmos   = Blockchain("evmos")
	BlockchainEVM           = Blockchain("evm")
	BlockchainEVMLegacy     = Blockchain("evm-legacy")
	BlockchainSubstrate     = Blockchain("substrate")
	BlockchainSolana        = Blockchain("solana")
	BlockchainSui           = Blockchain("sui")
	BlockchainTron          = Blockchain("tron")
	BlockchainTon           = Blockchain("ton")
	// Crosschain is a client-only blockchain
	BlockchainCrosschain = Blockchain("crosschain")
)

var SupportedBlockchains = []Blockchain{
	BlockchainAptos,
	BlockchainBitcoin,
	BlockchainBitcoinCash,
	BlockchainBitcoinLegacy,
	BlockchainCosmos,
	BlockchainCosmosEvmos,
	BlockchainEVM,
	BlockchainEVMLegacy,
	BlockchainSubstrate,
	BlockchainSolana,
	BlockchainSui,
	BlockchainTron,
	BlockchainTon,
}

type StakingProvider string

const Kiln StakingProvider = "kiln"
const Figment StakingProvider = "figment"
const Twinstake StakingProvider = "twinstake"
const Native StakingProvider = "native"

var SupportedStakingProviders = []StakingProvider{
	Native,
	Kiln,
	Figment,
	Twinstake,
}

func (stakingProvider StakingProvider) Valid() bool {
	return slices.Contains(SupportedStakingProviders, stakingProvider)
}

type TxVariantInputType string

func NewStakingInputType(blockchain Blockchain, variant string) TxVariantInputType {
	return TxVariantInputType(fmt.Sprintf("blockchains/%s/staking/%s", blockchain, variant))
}

func NewUnstakingInputType(blockchain Blockchain, variant string) TxVariantInputType {
	return TxVariantInputType(fmt.Sprintf("blockchains/%s/unstaking/%s", blockchain, variant))
}

func NewWithdrawingInputType(blockchain Blockchain, variant string) TxVariantInputType {
	return TxVariantInputType(fmt.Sprintf("blockchains/%s/withdrawing/%s", blockchain, variant))
}

func (variant TxVariantInputType) Blockchain() Blockchain {
	return Blockchain(strings.Split(string(variant), "/")[1])
}
func (variant TxVariantInputType) Variant() string {
	return (strings.Split(string(variant), "/")[3])
}

func (variant TxVariantInputType) Validate() error {
	if len(strings.Split(string(variant), "/")) != 4 {
		return fmt.Errorf("invalid input variant type: %s", variant)
	}
	return nil
}

func (native NativeAsset) IsValid() bool {
	return NativeAsset(native).Blockchain() != ""
}

func (native NativeAsset) Blockchain() Blockchain {
	switch native {
	case BTC:
		return BlockchainBitcoin
	case BCH:
		return BlockchainBitcoinCash
	case DOGE, LTC:
		return BlockchainBitcoinLegacy
	case AVAX, CELO, ETH, ETHW, MATIC, OptETH, ArbETH, BERA:
		return BlockchainEVM
	case BNB, FTM, ETC, EmROSE, AurETH, ACA, KAR, KLAY, OAS, CHZ, XDC, CHZ2:
		return BlockchainEVMLegacy
	case APTOS:
		return BlockchainAptos
	case ATOM, XPLA, INJ, HASH, LUNC, LUNA, SEI, TIA:
		return BlockchainCosmos
	case SUI:
		return BlockchainSui
	case SOL:
		return BlockchainSolana
	case DOT, TAO, KSM:
		return BlockchainSubstrate
	case TRX:
		return BlockchainTron
	case TON:
		return BlockchainTon
	}
	return ""
}

func (blockchain Blockchain) SignatureAlgorithm() SignatureType {
	switch blockchain {
	case BlockchainBitcoin, BlockchainBitcoinCash, BlockchainBitcoinLegacy:
		return K256Sha256
	case BlockchainEVM, BlockchainEVMLegacy, BlockchainCosmos, BlockchainCosmosEvmos, BlockchainTron:
		return K256Keccak
	case BlockchainAptos, BlockchainSolana, BlockchainSui, BlockchainTon, BlockchainSubstrate:
		return Ed255
	}
	return ""
}

type PublicKeyFormat string

var Raw PublicKeyFormat = "raw"
var Compressed PublicKeyFormat = "compressed"
var Uncompressed PublicKeyFormat = "uncompressed"

func (blockchain Blockchain) PublicKeyFormat() PublicKeyFormat {
	switch blockchain {
	case BlockchainBitcoin, BlockchainBitcoinCash, BlockchainBitcoinLegacy:
		return Compressed
	case BlockchainCosmos, BlockchainCosmosEvmos:
		return Compressed
	case BlockchainEVM, BlockchainEVMLegacy, BlockchainTron:
		return Uncompressed
	case BlockchainAptos, BlockchainSolana, BlockchainSui, BlockchainTon, BlockchainSubstrate:
		return Raw
	}
	return ""
}
