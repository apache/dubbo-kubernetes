package dubbo

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

const (
	DubboConfigFile = "dubbo.yaml"
	DataDir         = ".dubbo"
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
	filename := filepath.Join(path, DubboConfigFile)
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
			defaults.Template = "common"
		}
	}
	return defaults
}

func (dc *DubboConfig) Initialized() bool {
	return !dc.Created.IsZero()
}
