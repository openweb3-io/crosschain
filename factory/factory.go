package factory

import (
	"fmt"
	"sync"

	remoteclient "github.com/openweb3-io/crosschain/blockchain/crosschain"
	"github.com/openweb3-io/crosschain/builder"
	xc_client "github.com/openweb3-io/crosschain/client"
	"github.com/openweb3-io/crosschain/factory/blockchains"
	"github.com/openweb3-io/crosschain/factory/signer"
	"github.com/openweb3-io/crosschain/types"
	xc "github.com/openweb3-io/crosschain/types"
)

type IFactory interface {
	NewClient(cfg *types.ChainConfig) (xc_client.IClient, error)
	NewTxBuilder(cfg *types.ChainConfig) (builder.TxBuilder, error)
	NewSigner(cfg *types.ChainConfig, secret string) (*signer.Signer, error)
}

type Factory struct {
	AllAssets                        *sync.Map
	callbackGetAssetConfig           func(assetID types.AssetID) (types.IAsset, error)
	callbackGetAssetConfigByContract func(contract string, nativeAsset types.NativeAsset) (types.IAsset, error)
}

var _ IFactory = &Factory{}

func NewDefaultFactory() *Factory {
	return &Factory{
		AllAssets: &sync.Map{},
	}
}

func (f *Factory) NewClient(cfg *types.ChainConfig) (xc_client.IClient, error) {
	client := cfg.Client

	switch xc.Blockchain(client.Blockchain) {
	case xc.BlockchainCrosschain:
		return remoteclient.NewClient(cfg, client.Auth)
	default:
		return blockchains.NewClient(cfg, xc.Blockchain(client.Blockchain))
	}
}

func (f *Factory) GetAssetConfig(asset string, nativeAsset types.NativeAsset) (types.IAsset, error) {
	assetID := types.GetAssetIDFromAsset(asset, nativeAsset)
	return f.cfgFromAsset(assetID)
}

func (f *Factory) cfgFromAsset(assetID types.AssetID) (types.IAsset, error) {
	cfgI, found := f.AllAssets.Load(assetID)
	if !found {
		if f.callbackGetAssetConfig != nil {
			return f.callbackGetAssetConfig(assetID)
		}
		return &types.ChainConfig{}, fmt.Errorf("could not lookup asset: '%s'", assetID)
	}
	if cfg, ok := cfgI.(*types.ChainConfig); ok {
		// native asset
		// cfg.Type = AssetTypeNative
		// cfg.Chain = cfg.Asset
		// cfg.NativeAsset = NativeAsset(cfg.Asset)
		return cfg, nil
	}
	if cfg, ok := cfgI.(*types.TokenAssetConfig); ok {
		// token
		cfg, _ = f.cfgEnrichToken(cfg)
		return cfg, nil
	}
	return &types.ChainConfig{}, fmt.Errorf("invalid asset: '%s'", assetID)
}

func (f *Factory) cfgEnrichToken(partialCfg *types.TokenAssetConfig) (*types.TokenAssetConfig, error) {
	cfg := partialCfg
	if cfg.Chain != "" {
		chainI, found := f.AllAssets.Load(types.AssetID(cfg.Chain))
		if !found {
			return cfg, fmt.Errorf("unsupported native asset: %s", cfg.Chain)
		}
		// make copy so edits do not persist to local store
		native := *chainI.(*types.ChainConfig)
		cfg.ChainConfig = &native
	} else {
		return cfg, fmt.Errorf("unsupported native asset: (empty)")
	}
	return cfg, nil
}

// PutAssetConfig adds an AssetConfig to the current Config cache
func (f *Factory) PutAssetConfig(cfgI types.IAsset) (types.IAsset, error) {
	f.AllAssets.Store(cfgI.ID(), cfgI)
	return f.cfgFromAsset(cfgI.ID())
}

func (f *Factory) RegisterGetAssetConfigByContractCallback(callback func(contract string, nativeAsset types.NativeAsset) (types.IAsset, error)) {
	f.callbackGetAssetConfigByContract = callback
}

func (f *Factory) UnregisterGetAssetConfigByContractCallback() {
	f.callbackGetAssetConfigByContract = nil
}

func (f *Factory) RegisterGetAssetConfigCallback(callback func(assetID types.AssetID) (types.IAsset, error)) {
	f.callbackGetAssetConfig = callback
}

func (f *Factory) UnregisterGetAssetConfigCallback() {
	f.callbackGetAssetConfig = nil
}

// GetAllPossibleAddressesFromPublicKey returns all PossibleAddress(es) given a public key
func (f *Factory) GetAllPossibleAddressesFromPublicKey(cfg *types.ChainConfig, publicKey []byte) ([]types.PossibleAddress, error) {
	builder, err := blockchains.NewAddressBuilder(cfg)
	if err != nil {
		return []types.PossibleAddress{}, err
	}
	return builder.GetAllPossibleAddressesFromPublicKey(publicKey)
}

// GetAddressFromPublicKey returns an Address given a public key
func (f *Factory) GetAddressFromPublicKey(cfg *types.ChainConfig, publicKey []byte) (types.Address, error) {
	return getAddressFromPublicKey(cfg, publicKey)
}

func getAddressFromPublicKey(cfg *types.ChainConfig, publicKey []byte) (types.Address, error) {
	builder, err := blockchains.NewAddressBuilder(cfg)
	if err != nil {
		return "", err
	}
	return builder.GetAddressFromPublicKey(publicKey)
}

// NewAddressBuilder creates a new AddressBuilder
func (f *Factory) NewAddressBuilder(cfg *types.ChainConfig) (types.AddressBuilder, error) {
	return blockchains.NewAddressBuilder(cfg)
}

// NewTxBuilder creates a new TxBuilder
func (f *Factory) NewTxBuilder(cfg *types.ChainConfig) (builder.TxBuilder, error) {
	return blockchains.NewTxBuilder(cfg)
}

// NewSigner creates a new Signer
func (f *Factory) NewSigner(cfg *types.ChainConfig, secret string) (*signer.Signer, error) {
	return blockchains.NewSigner(cfg, secret)
}
