package cluster

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

type templateArgs struct {
	template string
}

func addTemplateFlags(cmd *cobra.Command, args *templateArgs) {
	cmd.PersistentFlags().StringVarP(&args.template, "template", "t", "", "java or go sdk template")
}

func CreateCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	tArgs := &templateArgs{}
	sc := sdkCmd(ctx, rootArgs, tArgs)
	cc := &cobra.Command{
		Use:   "create",
		Short: "Create a custom sdk",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	addFlags(cc, rootArgs)
	addFlags(sc, rootArgs)
	addTemplateFlags(cc, tArgs)
	cc.AddCommand(sc)
	return cc
}

func sdkCmd(ctx cli.Context, _ *RootArgs, tArgs *templateArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "sdk",
		Short: "Generates dubbo sdk language templates",
		Long:  "",
		Example: `
    Create a java common in the directory 'mydubbo'.
    dubboctl create sdk java -t common mydubbo

	Create a go common in the directory ./mydubbo.
	dubboctl create sdk go -t common mydubbogo
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if args[0] == "java" {
					// TODO
					fmt.Println("This is java sdk.")
				}
				if args[0] == "go" {
					// TODO
					fmt.Println("This is go sdk.")
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}

type createArgs struct {
	dirname string
	path    string
	create  string
}

func create(kc kube.CLIClient, tArgs *templateArgs, cl clog.Logger) {
	return
}

func newCreate(kc kube.CLIClient, tArgs *templateArgs, cl clog.Logger) (createArgs, error) {
	var (
		path         string
		dirName      string
		absolutePath string
	)
	dirName, absolutePath = deriveNameAndAbsolutePathFromPath(path)

	_ = createArgs{
		dirname: dirName,
		path:    absolutePath,
	}
	return createArgs{}, nil
}

func cwd() (cwd string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Unable to determine current working directory: %v", err))
	}
	return cwd
}

func deriveNameAndAbsolutePathFromPath(path string) (string, string) {
	var absPath string

	if path == "" {
		path = cwd()
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", ""
	}

	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
	return pathParts[len(pathParts)-1], absPath
}
