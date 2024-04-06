package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
)

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

var cfg = config{}

type config struct {
	name          string
	skipValidator bool
	force         bool
	basePath      string
	gomodule      string
	version       string
}

func (c config) policyPath() string {
	return path.Join(c.basePath, c.lowercase())
}

func (c config) lowercase() string {
	return strings.ToLower(c.name)
}

var rootCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap a new policy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.name == "" {
			return errors.New("-name is required")
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Bootstraping policy: %s at path %s\n", cfg.name, cfg.policyPath())
		if !cfg.force {
			_, err := os.Stat(cfg.policyPath())
			if err == nil {
				return fmt.Errorf("path %s already exists use -force to overwrite it", cfg.policyPath())
			}
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleting old policy code\n")
			if err := os.RemoveAll(cfg.policyPath()); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Generating proto file\n")
		if err := generateType(cfg); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Generating plugin file\n")
		if err := generatePlugin(cfg); err != nil {
			return err
		}
		path := fmt.Sprintf("generate/policy/%s", cfg.lowercase())
		if err := exec.Command("make", path).Run(); err != nil {
			return err
		}
		_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf(`
Successfully bootstrapped policy
regenerate auto generated files with: make generate/policy/%s

Useful files:
  - %s the proto definition
  - %s the validator
  - %s the plugin implementation
`,
			cfg.lowercase(),
			fmt.Sprintf("%s/api/%s/%s.proto", cfg.policyPath(), cfg.version, cfg.lowercase()),
			fmt.Sprintf("%s/api/%s/validator.go", cfg.policyPath(), cfg.version),
			fmt.Sprintf("%s/plugin/%s/plugin.go", cfg.policyPath(), cfg.version),
		)))
		return nil
	},
}

func generateType(c config) error {
	apiPath := path.Join(c.policyPath(), "api", c.version)
	if err := os.MkdirAll(apiPath, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path.Join(apiPath, c.lowercase()+".go"))
	if err != nil {
		return err
	}
	err = typeTemplate.Execute(f, map[string]interface{}{
		"name":      c.name,
		"nameLower": c.lowercase(),
		"module":    path.Join(c.gomodule, c.basePath),
		"version":   c.version,
	})
	if err != nil {
		return err
	}
	if c.skipValidator {
		return nil
	}
	f, err = os.Create(path.Join(apiPath, "validator.go"))
	if err != nil {
		return err
	}
	return validatorTemplate.Execute(f, map[string]interface{}{
		"name":    c.name,
		"version": c.version,
	})
}

func generatePlugin(c config) error {
	dir := path.Join(c.policyPath(), "plugin", c.version)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path.Join(dir, "plugin.go"))
	if err != nil {
		return err
	}
	return pluginTemplate.Execute(f, map[string]interface{}{
		"name":    c.name,
		"version": c.version,
		"package": fmt.Sprintf("%s/%s/%s/api/%s", c.gomodule, c.basePath, c.lowercase(), c.version),
	})
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&cfg.name, "name", "", "The name of the policy (UpperCamlCase)")
	rootCmd.Flags().StringVar(&cfg.basePath, "path", "pkg/plugins/policies", "Where to put the generated code")
	rootCmd.Flags().StringVar(&cfg.gomodule, "gomodule", "github.com/apache/dubbo-kubernetes", "Where to put the generated code")
	rootCmd.Flags().StringVar(&cfg.version, "version", "v1alpha1", "The version to use")
	rootCmd.Flags().BoolVar(&cfg.skipValidator, "skip-validator", false, "don't generator a validator empty file")
	rootCmd.Flags().BoolVar(&cfg.force, "force", false, "Overwrite any existing code")
}

var typeTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`// +kubebuilder:object:generate=true
package {{ .version }}

// {{ .name }}
// +dubbo:policy:skip_registration=true
type {{ .name }} struct {
}

`))

var pluginTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`package {{ .version }}

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/matchers"
	api "{{ .package }}"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

var _ core_plugins.PolicyPlugin = &plugin{}
var log = core.Log.WithName("{{.name}}")

type plugin struct {
}

func NewPlugin() core_plugins.Plugin {
	return &plugin{}
}

func (p plugin) MatchedPolicies(dataplane *core_mesh.DataplaneResource, resources xds_context.Resources) (core_xds.TypedMatchingPolicies, error) {
	return matchers.MatchedPolicies(api.{{ .name }}Type, dataplane, resources)
}

func (p plugin) Apply(rs *core_xds.ResourceSet, ctx xds_context.Context, proxy *core_xds.Proxy) error {
	log.Info("apply is not implemented")
	return nil
}
`))

var validatorTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`package {{.version}}

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

func (r *{{.name}}Resource) validate() error {
	var verr validators.ValidationError

	return verr.OrNil()
}
`))
