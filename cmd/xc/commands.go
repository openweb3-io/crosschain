package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openweb3-io/crosschain/cmd/xc/setup"
	"github.com/spf13/cobra"
)

func CmdChains() *cobra.Command {
	return &cobra.Command{
		Use:   "chains",
		Short: "List information on all supported chains.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// xcFactory := setup.UnwrapXc(cmd.Context())
			// chain := setup.UnwrapChain(cmd.Context())

			/*
				client, err := xcFactory.NewClient(assetConfig(chain, "", 0))
				if err != nil {
					return err
				}
			*/

			// fetch from server

			return nil
		},
	}
}

func CmdTxInput() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tx-input <address>",
		Aliases: []string{"input"},
		Short:   "Check inputs for a new transaction.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			xcFactory := setup.UnwrapXc(cmd.Context())
			chain := setup.UnwrapChain(cmd.Context())
			addressRaw := args[0]

			addressTo, _ := cmd.Flags().GetString("to")
			contract, _ := cmd.Flags().GetString("contract")
			client, err := xcFactory.NewClient(assetConfig(chain, contract, 0))
			if err != nil {
				return err
			}

			from := xcFactory.MustAddress(chain, addressRaw)
			to := xcFactory.MustAddress(chain, addressTo)
			input, err := client.FetchLegacyTxInput(context.Background(), from, to)
			if err != nil {
				return fmt.Errorf("could not fetch transaction inputs: %v", err)
			}

			bz, _ := json.MarshalIndent(input, "", "  ")
			fmt.Println(string(bz))
			return nil
		},
	}
	cmd.Flags().String("contract", "", "Optional contract of token asset")
	cmd.Flags().String("to", "", "Optional destination address")
	return cmd
}
