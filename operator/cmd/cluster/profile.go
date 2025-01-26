package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/helm"
	"github.com/apache/dubbo-kubernetes/operator/pkg/render"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"sort"
	"strings"
)

const (
	jsonOutput           = "json"
	yamlOutput           = "yaml"
	flagsOutput          = "flags"
	pathSeparator        = "."
	escapedPathSeparator = "\\" + pathSeparator
)

const (
	dubboOperatorTreeString = `
apiVersion: install.dubbo.io/v1alpha1
kind: DubboOperator
`
)

type profileListArgs struct {
	manifestsPath string
}

type profileDumpArgs struct {
	files         []string
	configPath    string
	outputFormat  string
	manifestsPath string
}

func addProfileListFlags(cmd *cobra.Command, args *profileListArgs) {
	cmd.PersistentFlags().StringVarP(&args.manifestsPath, "manifests", "d", "", "Specify a path to a directory of charts and profiles")
}

func addProfileDumpFlags(cmd *cobra.Command, args *profileDumpArgs) {
	cmd.PersistentFlags().StringSliceVarP(&args.files, "files", "f", nil, "")
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-path", "p", "", "")
	cmd.PersistentFlags().StringVarP(&args.outputFormat, "output", "o", yamlOutput, "Output format: one of json|yaml|flags")
	cmd.PersistentFlags().StringVarP(&args.manifestsPath, "manifests", "d", "", "")
}

func ProfileCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	plArgs := &profileListArgs{}
	plc := profileListCmd(rootArgs, plArgs)
	pc := &cobra.Command{
		Use:   "profile",
		Short: "Commands related to Dubbo configuration profiles",
		Long:  "The profile command lists, dumps or diffs Dubbo configuration profiles.",
		Example: "dubboctl profile list\n" +
			"dubboctl install --set profile=demo  # Use a profile from the list",
	}

	pc.AddCommand(plc)
	addProfileListFlags(plc, plArgs)

	AddFlags(pc, rootArgs)
	return pc
}

func profileListCmd(rootArgs *RootArgs, plArgs *profileListArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lists available Dubbo configuration profiles",
		Long:  "The list subcommand lists the available Dubbo configuration profiles.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return profileList(cmd, rootArgs, plArgs)
		},
	}
}

func profileDumpCmd(rootArgs *RootArgs, pdArgs *profileDumpArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "dump [PROFILE]",
		Short: "Dumps an Dubbo configuration profile",
		Long:  "The dump subcommand dumps the values in an Dubbo configuration profile.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("too many positional arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			l := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr(), InstallerScope)
			return profileDump(args, rootArgs, pdArgs, l)
		},
	}
}

func profileList(cmd *cobra.Command, args *RootArgs, plArgs *profileListArgs) error {
	profiles, err := helm.ListProfiles(plArgs.manifestsPath)
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		cmd.Println("No profiles available.")
	} else {
		cmd.Println("Dubbo configuration profiles:")
		sort.Strings(profiles)
		for _, profile := range profiles {
			cmd.Printf("    %s\n", profile)
		}
	}
	return nil
}

func profileDump(args []string, rootArgs *RootArgs, pdArgs *profileDumpArgs, l clog.Logger) error {
	if len(args) == 1 && pdArgs.files != nil {
		return fmt.Errorf("cannot specify both profile name and filename flag")
	}

	switch pdArgs.outputFormat {
	case jsonOutput, yamlOutput, flagsOutput:
	default:
		return fmt.Errorf("unknown output format: %v", pdArgs.outputFormat)
	}

	setFlags := applyFlagAliases(make([]string, 0), pdArgs.manifestsPath)
	if len(args) == 1 {
		setFlags = append(setFlags, "profile="+args[0])
	}

	y, _, err := render.GenerateManifest(pdArgs.files, setFlags, l, nil)
	if err != nil {
		return err
	}

	y, err = tpath.GetConfigSubtree(y, "spec")
	if err != nil {
		return err
	}

	if pdArgs.configPath == "" {
		if y, err = prependHeader(y); err != nil {
			return err
		}
	} else {
		if y, err = tpath.GetConfigSubtree(y, pdArgs.configPath); err != nil {
			return err
		}
	}

	switch pdArgs.outputFormat {
	case jsonOutput:
		j, err := yamlToPrettyJSON(y)
		if err != nil {
			return err
		}
		l.Print(j + "\n")
	case yamlOutput:
		l.Print(y + "\n")
	case flagsOutput:
		f, err := yamlToFlags(y)
		if err != nil {
			return err
		}
		l.Print(strings.Join(f, "\n") + "\n")
	}

	return nil
}

func yamlToFlags(yml string) ([]string, error) {
	uglyJSON, err := yaml.YAMLToJSON([]byte(yml))
	if err != nil {
		return []string{}, err
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(uglyJSON, &decoded); err != nil {
		return []string{}, err
	}
	spec, ok := decoded["spec"]
	if !ok {
		spec = decoded
	}
	setflags, err := walk("", "", spec)
	if err != nil {
		return []string{}, err
	}
	sort.Strings(setflags)
	return setflags, nil
}

func walk(path, separator string, obj interface{}) ([]string, error) {
	switch v := obj.(type) {
	case map[string]interface{}:
		accum := make([]string, 0)
		for key, vv := range v {
			childwalk, err := walk(fmt.Sprintf("%s%s%s", path, separator, pathComponent(key)), ".", vv)
			if err != nil {
				return accum, err
			}
			accum = append(accum, childwalk...)
		}
		return accum, nil
	case []interface{}:
		accum := make([]string, 0)
		for idx, vv := range v {
			indexwalk, err := walk(fmt.Sprintf("%s[%d]", path, idx), ".", vv)
			if err != nil {
				return accum, err
			}
			accum = append(accum, indexwalk...)
		}
		return accum, nil
	case string:
		return []string{fmt.Sprintf("%s=%q", path, v)}, nil
	default:
		return []string{fmt.Sprintf("%s=%v", path, v)}, nil
	}
}

func prependHeader(yml string) (string, error) {
	out, err := tpath.AddSpecRoot(yml)
	if err != nil {
		return "", err
	}
	out2, err := util.OverlayYAML(dubboOperatorTreeString, out)
	if err != nil {
		return "", err
	}
	return out2, nil
}

// Convert the generated YAML to pretty JSON.
func yamlToPrettyJSON(yml string) (string, error) {
	uglyJSON, err := yaml.YAMLToJSON([]byte(yml))
	if err != nil {
		return "", err
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(uglyJSON, &decoded); err != nil {
		return "", err
	}
	prettyJSON, err := json.MarshalIndent(decoded, "", "    ")
	if err != nil {
		return "", err
	}
	return string(prettyJSON), nil
}

func pathComponent(component string) string {
	if !strings.Contains(component, pathSeparator) {
		return component
	}
	return strings.ReplaceAll(component, pathSeparator, escapedPathSeparator)
}
