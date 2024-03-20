package cmd

import (
	"os"
)

import (
	"github.com/spf13/cobra"
)

type args struct {
	pluginDir string
	version   string
	goModule  string
}

func newRootCmd() *cobra.Command {
	rootArgs := &args{}

	cmd := &cobra.Command{
		Use:   "policy-gen",
		Short: "Tool to generate plugin-based policies for Dubbo",
		Long:  "Tool to generate plugin-based policies for Dubbo.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// once command line flags have been parsed,
			// avoid printing usage instructions
			cmd.SilenceUsage = true
			return nil
		},
	}

	cmd.AddCommand(newCoreResource(rootArgs))
	cmd.AddCommand(newK8sResource(rootArgs))
	cmd.AddCommand(newOpenAPI(rootArgs))
	cmd.AddCommand(newPluginFile(rootArgs))

	cmd.PersistentFlags().StringVar(&rootArgs.pluginDir, "plugin-dir", "", "path to the policy plugin director")
	cmd.PersistentFlags().StringVar(&rootArgs.version, "version", "v1alpha1", "policy version")
	cmd.PersistentFlags().StringVar(&rootArgs.goModule, "gomodule", "github.com/apache/dubbo-kubernetes", "Where to put the generated code")

	return cmd
}

func DefaultRootCmd() *cobra.Command {
	return newRootCmd()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := DefaultRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
