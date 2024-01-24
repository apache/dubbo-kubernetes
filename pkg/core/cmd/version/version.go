package version

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

import (
	dubbo_version "github.com/apache/dubbo-kubernetes/pkg/version"
)

var (
	gitVersion   = "dubbo-kube-%s"
	gitCommit    = "$Format:%H$"
	gitTreeState = "" // state of git tree, either "clean" or "dirty"
	gitTag       = ""
	buildDate    = "1970-01-01T00:00:00Z"
)

type Version struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

func GetVersion() Version {
	version := Version{
		GitVersion:   fmt.Sprintf(gitVersion, gitTag),
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	return version
}

func GetVersionInfo() string {
	version := GetVersion()
	result, _ := json.Marshal(version)
	return string(result)
}

func NewVersionCmd() *cobra.Command {
	args := struct {
		detailed bool
	}{}
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Long:  `Print version.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if args.detailed {
				cmd.Println(dubbo_version.Build.FormatDetailedProductInfo())
			} else {
				cmd.Printf("%s: %s\n", dubbo_version.Product, dubbo_version.Build.Version)
			}

			return nil
		},
	}
	// flags
	cmd.PersistentFlags().BoolVarP(&args.detailed, "detailed", "a", false, "Print detailed version")
	return cmd
}
