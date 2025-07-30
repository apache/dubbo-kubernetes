package app

import (
	"github.com/apache/dubbo-kubernetes/navigator/pkg/cmd"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "navi-agent",
		SilenceUsage: true,
	}
	cmd.AddFlags(rootCmd)
	return rootCmd
}

func newProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "XDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
}
