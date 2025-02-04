package dubbo

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DubboLogFile    = ".dubbo/dubbo.log"
	Dockerfile      = "Dockerfile"
	DataDir         = ".dubbo"
	DefaultTemplate = "common"
)

type DubboConfig struct {
	Root        string     `yaml:"-"`
	Name        string     `yaml:"name,omitempty" jsonschema:"pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`
	Image       string     `yaml:"image,omitempty"`
	ImageDigest string     `yaml:"-"`
	Runtime     string     `yaml:"runtime,omitempty"`
	Template    string     `yaml:"template,omitempty"`
	Created     time.Time  `yaml:"created,omitempty"`
	Build       BuildSpec  `yaml:"build,omitempty"`
	Deploy      DeploySpec `yaml:"deploy,omitempty"`
}

type BuildSpec struct {
	BuilderImages map[string]string `yaml:"builderImages,omitempty"`
	Buildpacks    []string          `yaml:"buildpacks,omitempty"`
}

type DeploySpec struct {
	Namespace     string `yaml:"namespace,omitempty"`
	Output        string `yaml:"output,omitempty"`
	ContainerPort int    `yaml:"containerPort,omitempty"`
	TargetPort    int    `yaml:"targetPort,omitempty"`
	NodePort      int    `yaml:"nodePort,omitempty"`
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

	filename := filepath.Join(path, DubboLogFile)
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

func NewDubboConfigWithTemplate(dc *DubboConfig, initialized bool) *DubboConfig {
	if !initialized {
		if dc.Template == "" {
			dc.Template = DefaultTemplate
		}
		if dc.Template == "" {
			dc.Template = "initialzed"
		}
	}
	return dc
}

func (dc *DubboConfig) WriteFile() (err error) {
	file := filepath.Join(dc.Root, DubboLogFile)
	var bytes []byte
	if bytes, err = yaml.Marshal(dc); err != nil {
		return
	}
	if err = os.WriteFile(file, bytes, 0o644); err != nil {
		return
	}
	return
}

func (dc *DubboConfig) WriteDockerfile(cmd *cobra.Command) (err error) {
	path := filepath.Join(dc.Root, Dockerfile)
	bytes, ok := DockerfileByRuntime[dc.Runtime]
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "The runtime of your current project is not one of Java or go. We cannot help you generate a Dockerfile template.")
		return
	}
	if err = os.WriteFile(path, []byte(bytes), 0o644); err != nil {
		return
	}
	return
}

func (dc *DubboConfig) Validate() error {
	if dc.Root == "" {
		return errors.New("dubbo root path is required")
	}

	var ctr int
	errs := [][]string{
		validateOptions(),
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("'%v' contains errors:", DubboLogFile))

	for _, ee := range errs {
		if len(ee) > 0 {
			b.WriteString("\n") // Precede each group of errors with a linebreak
		}
		for _, e := range ee {
			ctr++
			b.WriteString("\t" + e)
		}
	}

	if ctr == 0 {
		return nil // Return nil if there were no validation errors.
	}

	return errors.New(b.String())
}

func (dc *DubboConfig) BuildStamp() string {
	path := filepath.Join(dc.Root, DataDir, "built")
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func (dc *DubboConfig) Built() bool {
	stamp := dc.BuildStamp()
	if stamp == "" {
		return false
	}

	if dc.Image == "" {
		return false
	}

	hash, _, err := Fingerprint(dc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error calculating function's fingerprint: %v\n", err)
		return false
	}
	return stamp == hash
}

func (dc *DubboConfig) Initialized() bool {
	return !dc.Created.IsZero()
}

func Fingerprint(dc *DubboConfig) (hash, log string, err error) {
	h := sha256.New()   // Hash builder
	l := bytes.Buffer{} // Log buffer

	root := dc.Root
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	// TODO
	output := ""

	err = filepath.Walk(abs, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if info.IsDir() && (info.Name() == DataDir || info.Name() == ".git" || info.Name() == ".idea") {
			return filepath.SkipDir
		}
		if info.Name() == DubboLogFile || info.Name() == Dockerfile || info.Name() == output {
			return nil
		}
		fmt.Fprintf(h, "%v:%v:", path, info.ModTime().UnixNano())   // Write to the Hashed
		fmt.Fprintf(&l, "%v:%v\n", path, info.ModTime().UnixNano()) // Write to the Log
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil)), l.String(), err
}

func validateOptions() []string {
	return nil
}
