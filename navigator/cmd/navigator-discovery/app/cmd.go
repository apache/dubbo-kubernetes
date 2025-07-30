package app

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "navi-discovery",
		SilenceUsage: true,
	}
	return rootCmd
}
