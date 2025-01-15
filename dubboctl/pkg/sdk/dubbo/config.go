package dubbo

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

const (
	DubboYamlFile   = "dubbo.yaml"
	Dockerfile      = "Dockerfile"
	DataDir         = ".dubbo"
	DefaultTemplate = "common"
)

type DubboConfig struct {
	Root     string    `yaml:"-"`
	Name     string    `yaml:"name,omitempty" jsonschema:"pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`
	Runtime  string    `yaml:"runtime,omitempty"`
	Template string    `yaml:"template,omitempty"`
	Created  time.Time `yaml:"created,omitempty"`
}

func NewDubboConfig(path string) (*DubboConfig, error) {
	var err error
	f := &DubboConfig{}
	if path == "" {
		if path, err = os.Getwd(); err != nil {
			return f, err
		}
	}
	f.Root = path
	fd, err := os.Stat(path)
	if err != nil {
		return f, err
	}
	if !fd.IsDir() {
		return nil, fmt.Errorf("function path must be a directory")
	}
	filename := filepath.Join(path, DubboYamlFile)
	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return f, err
	}

	bb, err := os.ReadFile(filename)
	if err != nil {
		return f, err
	}
	err = yaml.Unmarshal(bb, f)
	if err != nil {
		return f, err
	}
	return f, nil
}

func NewDubboConfigWithTemplate(defaults *DubboConfig, initialized bool) *DubboConfig {
	if !initialized {
		if defaults.Template == "" {
			defaults.Template = DefaultTemplate
		}
	}
	return defaults
}

func (dc *DubboConfig) Initialized() bool {
	return !dc.Created.IsZero()
}

func (f *DubboConfig) EnsureDockerfile(cmd *cobra.Command) (err error) {
	dockerfilepath := filepath.Join(f.Root, Dockerfile)
	dockerfilebytes, ok := DockerfileByRuntime[f.Runtime]
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "The runtime of your current project is not one of Java or go. We cannot help you generate a Dockerfile template.")
		return
	}
	if err = os.WriteFile(dockerfilepath, []byte(dockerfilebytes), 0o644); err != nil {
		return
	}
	return
}

func (f *DubboConfig) Write() (err error) {
	dubboyamlpath := filepath.Join(f.Root, DubboYamlFile)
	var dubbobytes []byte
	if dubbobytes, err = yaml.Marshal(f); err != nil {
		return
	}
	if err = os.WriteFile(dubboyamlpath, dubbobytes, 0o644); err != nil {
		return
	}
	return
}
