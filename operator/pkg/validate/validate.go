package validate

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
)

func NewValidateCommand(ctx cli.Context) *cobra.Command {
	vc := &cobra.Command{
		Use:   "validate -f FILENAME [options]",
		Short: "Validate Dubbo rules files",
		Long:  "The validate command is used to validate the dubbo related rule file",
		Example: `  # Validate current deployments under 'default' namespace with in the cluster
  kubectl get deployments -o yaml | dubboctl validate -f -

  # Validate current services under 'default' namespace with in the cluster
  kubectl get services -o yaml | dubboctl validate -f -
`,
		Args:    cobra.NoArgs,
		Aliases: []string{"v"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return nil
		},
	}
	return vc
}
