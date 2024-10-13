package main

import (
	"github.com/openweb3-io/crosschain/cmd/xc/setup"
	"github.com/openweb3-io/crosschain/types"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:          "xc",
		Short:        "Manually interact with blockchains",
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			args, err := setup.RpcArgsFromCmd(cmd)
			if err != nil {
				return err
			}

			xcFactory, err := setup.LoadFactory(args)
			if err != nil {
				return err
			}

			chainConfig, err := setup.LoadChain(xcFactory, args.Chain)
			if err != nil {
				return err
			}

			ctx := setup.CreateContext(xcFactory, chainConfig)
			logrus.WithFields(logrus.Fields{
				"rpc": chainConfig.Client.URL,
				// "network": chainConfig.Network,
				"chain": chainConfig.Chain,
			}).Info("chain")

			cmd.SetContext(ctx)
			return nil
		},
	}

	setup.AddRpcArgs(cmd)

	cmd.AddCommand(CmdTxInput())
	cmd.AddCommand(CmdChains())

	_ = cmd.Execute()
}

func assetConfig(chain *xc.ChainConfig, contractMaybe xc.ContractAddress, decimals int32) types.IAsset {
	if contractMaybe != "" {
		token := xc.TokenAssetConfig{
			Contract: contractMaybe,
			// Chain:       chain.Chain,
			ChainConfig: chain,
			Decimals:    decimals,
		}
		return &token
	} else {
		return chain
	}
}
