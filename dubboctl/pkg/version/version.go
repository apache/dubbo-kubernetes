package version

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	dubboVersion "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/spf13/cobra"
)

func NewVersionCommand(_ cli.Context) *cobra.Command {
	var versionCmd *cobra.Command
	versionCmd = dubboVersion.CobraCommandWithOptions()
	return versionCmd
}
