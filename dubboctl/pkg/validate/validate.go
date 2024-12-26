package validate

import (
	"errors"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
	"io"
)

var (
	errFiles = errors.New(`error: you must specify resources by --filename.
Example resource specifications include:
   '-f default.yaml'
   '--filename=default.json'`)
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
			dn := ctx.DubboNamespace()
			return validateFiles(&dn, nil, nil)
		},
	}
	return vc
}

func validateFiles(dubboNamespace *string, files []string, writer io.Writer) error {
	if len(files) == 0 {
		return errFiles
	}

	return nil
}
