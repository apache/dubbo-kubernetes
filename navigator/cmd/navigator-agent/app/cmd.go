package app

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/navigator/cmd/navigator-agent/options"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/spf13/cobra"
)

var (
	proxyArgs options.ProxyArgs
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "navi-agent",
		Short:        "Dubbo Navi agent.",
		Long:         "Dubbo Navi agent runs in the sidecar or gateway container and bootstraps Envoy.",
		SilenceUsage: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
	}
	cmd.AddFlags(rootCmd)
	proxyCmd := newProxyCommand()
	addFlags(proxyCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(waitCmd)

	return rootCmd
}

func newProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "XDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
		RunE: func(c *cobra.Command, args []string) error {
			err := initProxy(args)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func initProxy(args []string) error {
	proxyArgs.Type = model.SidecarProxy
	if len(args) > 0 {
		proxyArgs.Type = model.NodeType(args[0])
		if !model.IsApplicationNodeType(proxyArgs.Type) {
			return fmt.Errorf("invalid proxy Type: %s", string(proxyArgs.Type))
		}
	}
	return nil
}

func addFlags(proxyCmd *cobra.Command) {
	proxyArgs = options.NewProxyArgs()
}
