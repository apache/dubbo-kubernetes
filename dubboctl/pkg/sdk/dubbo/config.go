package dubbo

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

const (
	DubboFile = "dubbo.yaml"
	DataDir   = ".dubbo"
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

	fd, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fd.IsDir() {
		return nil, nil
	}
	filename := filepath.Join(path, DubboFile)
	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}

	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(bb, nil)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func NewDubboConfigWithTemplate(defaults *DubboConfig, initialized bool) *DubboConfig {
	if !initialized {
		if defaults.Template == "" {
			defaults.Template = "sample"
		}
	}
	return defaults
}

func (dc *DubboConfig) Initialized() bool {
	return !dc.Created.IsZero()
}
