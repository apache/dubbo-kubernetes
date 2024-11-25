package cluster

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

type InstallArgs struct {
	Files            []string
	Sets             []string
	Revision         string
	ManifestPath     string
	SkipConfirmation bool
	ReadinessTimeout time.Duration
}

func (i *InstallArgs) String() string {
	var b strings.Builder
	b.WriteString("Files:    " + (fmt.Sprint(i.Files) + "\n"))
	b.WriteString("Sets:    " + (fmt.Sprint(i.Sets) + "\n"))
	b.WriteString("Revision:    " + (fmt.Sprint(i.Revision) + "\n"))
	b.WriteString("ManifestPath:    " + (fmt.Sprint(i.ManifestPath) + "\n"))
	return b.String()
}

func InstallCmd(ctx cli.Context) *cobra.Command {
	return nil
}
