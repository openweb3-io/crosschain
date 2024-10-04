package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/openweb3-io/crosschain/factory"
	"github.com/openweb3-io/crosschain/types"
	"github.com/spf13/cobra"
)

type ContextKey string

const (
	ContextXc    ContextKey = "xc"
	ContextChain ContextKey = "chain"
)

func WrapXc(ctx context.Context, xcFactory *factory.Factory) context.Context {
	ctx = context.WithValue(ctx, ContextXc, xcFactory)
	return ctx
}

func UnwrapXc(ctx context.Context) *factory.Factory {
	return ctx.Value(ContextXc).(*factory.Factory)
}

func WrapChain(ctx context.Context, chain *types.ChainConfig) context.Context {
	ctx = context.WithValue(ctx, ContextChain, chain)
	return ctx
}

func UnwrapChain(ctx context.Context) *types.ChainConfig {
	return ctx.Value(ContextChain).(*types.ChainConfig)
}

func CreateContext(xcFactory *factory.Factory, chain *types.ChainConfig) context.Context {
	ctx := context.Background()
	ctx = WrapXc(ctx, xcFactory)
	// ctx = WrapChain(ctx, chain)
	return ctx
}

type RpcArgs struct {
	Chain string
}

func RpcArgsFromCmd(cmd *cobra.Command) (*RpcArgs, error) {
	return &RpcArgs{}, nil
}

func LoadFactory(rcpArgs *RpcArgs) (*factory.Factory, error) {
	// if rcpArgs.ConfigPath != "" {
	// 	// currently only way to set config file is via env
	// 	_ = os.Setenv(constants.ConfigEnv, rcpArgs.ConfigPath)
	// }
	xcFactory := factory.NewDefaultFactory()
	/*
		if rcpArgs.NotMainnet {
			xcFactory = factory.NewNotMainnetsFactory(&factory.FactoryOptions{})
		}
	*/

	/*
		if rcpArgs.Rpc != "" {
			if existing, ok := rcpArgs.Overrides[strings.ToLower(rcpArgs.Chain)]; ok {
				existing.Rpc = rcpArgs.Rpc
			} else {
				rcpArgs.Overrides[strings.ToLower(rcpArgs.Chain)] = &ChainOverride{
					Rpc: rcpArgs.Rpc,
				}
			}
		}
	*/

	// OverwriteCrosschainSettings(rcpArgs.Overrides, xcFactory)
	return xcFactory, nil
}

func LoadChain(xcFactory *factory.Factory, chain string) (*types.ChainConfig, error) {
	var nativeAsset types.NativeAsset
	for _, chainOption := range types.NativeAssetList {
		if strings.EqualFold(string(chainOption), chain) {
			nativeAsset = chainOption
		}
	}
	if nativeAsset == "" {
		return nil, fmt.Errorf("invalid chain: %s\noptions: %v", chain, types.NativeAssetList)
	}

	return &types.ChainConfig{}, nil

	/* TODO
	chainConfig, err := xcFactory.GetAssetConfig("", nativeAsset)
	if err != nil {
		return nil, err
	}
	chainCfg := chainConfig.(*xc.ChainConfig)
	return chainCfg, nil
	*/
}
