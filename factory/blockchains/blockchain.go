package blockchains

import (
	"errors"
	"fmt"

	"github.com/openweb3-io/crosschain/blockchain/btc"
	"github.com/openweb3-io/crosschain/blockchain/btc_cash"
	cosmosbuilder "github.com/openweb3-io/crosschain/blockchain/cosmos/builder"
	solanabuilder "github.com/openweb3-io/crosschain/blockchain/solana/builder"

	evmbuilder "github.com/openweb3-io/crosschain/blockchain/evm/builder"
	xcbuilder "github.com/openweb3-io/crosschain/builder"

	btcaddress "github.com/openweb3-io/crosschain/blockchain/btc/address"
	cosmosaddress "github.com/openweb3-io/crosschain/blockchain/cosmos/address"
	evmaddress "github.com/openweb3-io/crosschain/blockchain/evm/address"
	solanaaddress "github.com/openweb3-io/crosschain/blockchain/solana/address"
	tonaddress "github.com/openweb3-io/crosschain/blockchain/ton/address"
	"github.com/openweb3-io/crosschain/factory/signer"

	// "github.com/openweb3-io/crosschain/blockchain/aptos"
	// "github.com/openweb3-io/crosschain/chain/evm_legacy"

	// "github.com/openweb-io/crosschain/blockchain/evm_legacy"
	// "github.com/openweb-io/crosschain/blockchain/substrate"
	// "github.com/openweb3-io/crosschain/blockchain/sui"
	"github.com/openweb3-io/crosschain/blockchain/ton"
	"github.com/openweb3-io/crosschain/blockchain/tron"
	xc_client "github.com/openweb3-io/crosschain/client"
	"github.com/openweb3-io/crosschain/types"
)

type ClientCreator func(cfg *types.ChainConfig) (xc_client.IClient, error)

var (
	creatorMap = make(map[types.Blockchain]ClientCreator)
)

func RegisterClient(cfg types.Blockchain, creator ClientCreator) {
	creatorMap[cfg] = creator
}

func init() {
	RegisterClient("ton", func(cfg *types.ChainConfig) (xc_client.IClient, error) {
		return ton.NewClient(cfg)
	})
}

func NewClient(cfg *types.ChainConfig) (xc_client.IClient, error) {
	creator, ok := creatorMap[cfg.Blockchain]
	if !ok {
		return nil, fmt.Errorf("creator %s not found", cfg.Blockchain)
	}

	return creator(cfg)
}

func NewAddressBuilder(cfg *types.ChainConfig) (types.AddressBuilder, error) {
	switch types.Blockchain(cfg.Blockchain) {
	case types.BlockchainEVM:
		return evmaddress.NewAddressBuilder(cfg)
	//case types.BlockchainEVMLegacy:
	//	return evm_legacy.NewAddressBuilder(cfg)
	case types.BlockchainCosmos, types.BlockchainCosmosEvmos:
		return cosmosaddress.NewAddressBuilder(cfg)
	case types.BlockchainSolana:
		return solanaaddress.NewAddressBuilder(cfg)
	//case types.BlockchainAptos:
	//	return aptos.NewAddressBuilder(cfg)
	case types.BlockchainBitcoin, types.BlockchainBitcoinLegacy:
		return btcaddress.NewAddressBuilder(cfg)
	case types.BlockchainBitcoinCash:
		return btc_cash.NewAddressBuilder(cfg)
	// case types.BlockchainSui:
	// 	return sui.NewAddressBuilder(cfg)
	//case types.BlockchainSubstrate:
	//	return substrate.NewAddressBuilder(cfg)
	case types.BlockchainTron:
		return tron.NewAddressBuilder(cfg)
	case types.BlockchainTon:
		return tonaddress.NewAddressBuilder(cfg)
	}
	return nil, errors.New("no address builder defined for: " + string(cfg.ID()))
}

func NewSigner(cfg *types.ChainConfig, secret string) (*signer.Signer, error) {
	return signer.New(cfg.Blockchain, secret, cfg)
}

func NewTxBuilder(cfg *types.ChainConfig) (xcbuilder.TxBuilder, error) {
	switch types.Blockchain(cfg.Blockchain) {
	case types.BlockchainEVM:
		return evmbuilder.NewTxBuilder(cfg)
	//case BlockchainEVMLegacy:
	//	return evm_legacy.NewTxBuilder(cfg)
	case types.BlockchainCosmos, types.BlockchainCosmosEvmos:
		return cosmosbuilder.NewTxBuilder(cfg)
	case types.BlockchainSolana:
		return solanabuilder.NewTxBuilder(cfg)
	//case BlockchainAptos:
	//	return aptos.NewTxBuilder(cfg)
	//case BlockchainSui:
	//	return sui.NewTxBuilder(cfg)
	case types.BlockchainBitcoin, types.BlockchainBitcoinLegacy:
		return btc.NewTxBuilder(cfg)
	case types.BlockchainBitcoinCash:
		return btc_cash.NewTxBuilder(cfg)
	// case BlockchainSubstrate:
	//	return substrate.NewTxBuilder(cfg)
	case types.BlockchainTron:
		return tron.NewTxBuilder(cfg)
	case types.BlockchainTon:
		return ton.NewTxBuilder(cfg)
	}
	return nil, errors.New("no tx-builder defined for: " + string(cfg.ID()))
}
