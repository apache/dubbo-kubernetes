package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

type Version struct {
	ClientVersion *BuildInfo `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
}

var (
	Info BuildInfo
)

func CobraCommandWithOptions() *cobra.Command {
	var (
		short   bool
		output  string
		version Version
	)
	ves := &cobra.Command{
		Use:   "version",
		Short: "Display the information of the currently built version",
		Long:  "The version command displays the information of the currently built version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" && output != "yaml" && output != "json" {
				return errors.New(`--output must be 'yaml' or 'json'`)
			}
			version.ClientVersion = &Info
			switch output {
			case "":
				if short {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "client version: %s\n", version.ClientVersion.Version)
				}
			case "yaml":
				if marshaled, err := yaml.Marshal(&version); err == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(marshaled))
				}
			case "json":
				if marshaled, err := json.MarshalIndent(&version, "", "  "); err == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(marshaled))
				}
			}

			return nil
		},
	}

	ves.Flags().BoolVarP(&short, "short", "s", false, "Use --short=false to generate full version information")
	ves.Flags().StringVarP(&output, "output", "o", "", "One of 'yaml' or 'json'.")
	return ves
}
