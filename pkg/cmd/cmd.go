package cmd

import (
	"flag"
	"github.com/spf13/cobra"
)

func AddFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}
