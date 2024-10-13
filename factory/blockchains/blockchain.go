package blockchains

import (
	"errors"
	"fmt"

	"github.com/openweb3-io/crosschain/blockchain/btc"
	btcclient "github.com/openweb3-io/crosschain/blockchain/btc/client"
	"github.com/openweb3-io/crosschain/blockchain/btc_cash"
	cosmosbuilder "github.com/openweb3-io/crosschain/blockchain/cosmos/builder"
	cosmosclient "github.com/openweb3-io/crosschain/blockchain/cosmos/client"

	evm_legacy "github.com/openweb3-io/crosschain/blockchain/evm_legacy"
	solanabuilder "github.com/openweb3-io/crosschain/blockchain/solana/builder"

	evmbuilder "github.com/openweb3-io/crosschain/blockchain/evm/builder"
	evmclient "github.com/openweb3-io/crosschain/blockchain/evm/client"
	solanaclient "github.com/openweb3-io/crosschain/blockchain/solana/client"
	tonclient "github.com/openweb3-io/crosschain/blockchain/ton/client"
	tronclient "github.com/openweb3-io/crosschain/blockchain/tron/client"
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
	xc "github.com/openweb3-io/crosschain/types"
)

type ClientCreator func(cfg *xc.ChainConfig) (xc_client.IClient, error)

var (
	creatorMap = make(map[xc.Blockchain]ClientCreator)
)

func RegisterClient(cfg xc.Blockchain, creator ClientCreator) {
	creatorMap[cfg] = creator
}

func init() {
	RegisterClient(xc.BlockchainBtc, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return btcclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainBtcLegacy, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return btcclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainBtcCash, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return btc_cash.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainCosmos, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return cosmosclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainCosmosEvmos, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return cosmosclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainTon, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return tonclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainTron, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return tronclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainSolana, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return solanaclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainEVM, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return evmclient.NewClient(cfg)
	})

	RegisterClient(xc.BlockchainEVMLegacy, func(cfg *xc.ChainConfig) (xc_client.IClient, error) {
		return evm_legacy.NewClient(cfg)
	})
}

func NewClient(cfg *xc.ChainConfig, blockchain xc.Blockchain) (xc_client.IClient, error) {
	creator, ok := creatorMap[blockchain]
	if !ok {
		return nil, fmt.Errorf("creator %s not found", cfg.Blockchain)
	}

	return creator(cfg)
}

func NewAddressBuilder(cfg *xc.ChainConfig) (xc.AddressBuilder, error) {
	switch xc.Blockchain(cfg.Blockchain) {
	case xc.BlockchainEVM:
		return evmaddress.NewAddressBuilder(cfg)
	//case types.BlockchainEVMLegacy:
	//	return evm_legacy.NewAddressBuilder(cfg)
	case xc.BlockchainCosmos, xc.BlockchainCosmosEvmos:
		return cosmosaddress.NewAddressBuilder(cfg)
	case xc.BlockchainSolana:
		return solanaaddress.NewAddressBuilder(cfg)
	//case types.BlockchainAptos:
	//	return aptos.NewAddressBuilder(cfg)
	case xc.BlockchainBtc, xc.BlockchainBtcLegacy:
		return btcaddress.NewAddressBuilder(cfg)
	case xc.BlockchainBtcCash:
		return btc_cash.NewAddressBuilder(cfg)
	// case types.BlockchainSui:
	// 	return sui.NewAddressBuilder(cfg)
	//case types.BlockchainSubstrate:
	//	return substrate.NewAddressBuilder(cfg)
	case xc.BlockchainTron:
		return tron.NewAddressBuilder(cfg)
	case xc.BlockchainTon:
		return tonaddress.NewAddressBuilder(cfg)
	}
	return nil, errors.New("no address builder defined for: " + string(cfg.ID()))
}

func NewSigner(cfg *xc.ChainConfig, secret string) (*signer.Signer, error) {
	return signer.New(cfg.Blockchain, secret, cfg)
}

func NewTxBuilder(cfg *xc.ChainConfig) (xcbuilder.TxBuilder, error) {
	switch xc.Blockchain(cfg.Blockchain) {
	case xc.BlockchainEVM:
		return evmbuilder.NewTxBuilder(cfg)
	//case BlockchainEVMLegacy:
	//	return evm_legacy.NewTxBuilder(cfg)
	case xc.BlockchainCosmos, xc.BlockchainCosmosEvmos:
		return cosmosbuilder.NewTxBuilder(cfg)
	case xc.BlockchainSolana:
		return solanabuilder.NewTxBuilder(cfg)
	//case BlockchainAptos:
	//	return aptos.NewTxBuilder(cfg)
	//case BlockchainSui:
	//	return sui.NewTxBuilder(cfg)
	case xc.BlockchainBtc, xc.BlockchainBtcLegacy:
		return btc.NewTxBuilder(cfg)
	case xc.BlockchainBtcCash:
		return btc_cash.NewTxBuilder(cfg)
	// case BlockchainSubstrate:
	//	return substrate.NewTxBuilder(cfg)
	case xc.BlockchainTron:
		return tron.NewTxBuilder(cfg)
	case xc.BlockchainTon:
		return ton.NewTxBuilder(cfg)
	}
	return nil, errors.New("no tx-builder defined for: " + string(cfg.ID()))
}
