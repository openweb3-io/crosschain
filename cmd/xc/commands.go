package main

import "github.com/spf13/cobra"

func CmdChains() *cobra.Command {
	return &cobra.Command{
		Use:   "chains",
		Short: "List information on all supported chains.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
