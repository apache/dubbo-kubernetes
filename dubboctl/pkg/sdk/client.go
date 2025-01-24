package sdk

import (
	"bufio"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	templates        *Templates
	repositories     *Repositories
	repositoriesPath string
}

type Option func(client *Client)

func New(options ...Option) *Client {
	c := &Client{}
	for _, o := range options {
		o(c)
	}
	c.repositories = newRepositories(c)
	c.templates = newTemplates(c)

	return c
}

func (c *Client) Templates() *Templates {
	return c.templates
}

func (c *Client) Runtimes() ([]string, error) {
	runtimes := util.NewSortedSet()
	return runtimes.Items(), nil
}

func (c *Client) Repositories() *Repositories {
	return c.repositories
}

func (c *Client) Initialize(dcfg *dubbo.DubboConfig, initialized bool, cmd *cobra.Command) (*dubbo.DubboConfig, error) {
	var err error
	oldRoot := dcfg.Root

	dcfg.Root, err = filepath.Abs(dcfg.Root)
	if err != nil {
		return dcfg, err
	}
	if err = os.MkdirAll(dcfg.Root, 0o755); err != nil {
		return dcfg, err
	}

	has, err := hasInitialized(dcfg.Root)
	if err != nil {
		return dcfg, err
	}
	if has {
		return dcfg, fmt.Errorf("%v already initialized", dcfg.Root)
	}

	if dcfg.Root == "" {
		if dcfg.Root, err = os.Getwd(); err != nil {
			return dcfg, err
		}
	}
	if dcfg.Name == "" {
		dcfg.Name = nameFromPath(dcfg.Root)
	}

	if !initialized {
		if err := assertEmptyRoot(dcfg.Root); err != nil {
			return dcfg, err
		}
	}
	// TODO remove initiallized
	f := dubbo.NewDubboConfigWithTemplate(dcfg, initialized)

	if err = runDataDir(f.Root); err != nil {
		return f, err
	}

	if !initialized {
		err = c.Templates().Write(f)
		if err != nil {
			return f, err
		}
	}

	f.Created = time.Now()
	err = f.WriteYamlFile()
	if err != nil {
		return f, err
	}
	err = f.WriteDockerfile(cmd)
	if err != nil {
		return f, err
	}

	return dubbo.NewDubboConfig(oldRoot)
}

func hasInitialized(path string) (bool, error) {
	var err error
	filename := filepath.Join(path, dubbo.DubboYamlFile)

	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	bb, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}
	f := dubbo.DubboConfig{}
	if err = yaml.Unmarshal(bb, &f); err != nil {
		return false, err
	}

	return f.Initialized(), nil
}

func nameFromPath(path string) string {
	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
	return pathParts[len(pathParts)-1]
}

func assertEmptyRoot(path string) (err error) {
	files, err := contentiousFilesIn(path)
	if err != nil {
		return
	} else if len(files) > 0 {
		return fmt.Errorf("the chosen directory '%v' contains contentious files: %v.  Has the Service function already been created?  Try either using a different directory, deleting the function if it exists, or manually removing the files", path, files)
	}
	empty, err := isEffectivelyEmpty(path)
	if err != nil {
		return
	} else if !empty {
		err = errors.New("the directory must be empty of visible files and recognized config files before it can be initialized")
		return
	}
	return
}

var contentiousFiles = []string{
	dubbo.DubboYamlFile,
	".gitignore",
}

func contentiousFilesIn(dir string) (contentious []string, err error) {
	files, err := os.ReadDir(dir)
	for _, file := range files {
		for _, name := range contentiousFiles {
			if file.Name() == name {
				contentious = append(contentious, name)
			}
		}
	}
	return
}

func isEffectivelyEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return false, nil
		}
	}
	return true, nil
}

func runDataDir(root string) error {
	if err := os.MkdirAll(filepath.Join(root, dubbo.DataDir), os.ModePerm); err != nil {
		return err
	}
	filePath := filepath.Join(root, ".gitignore")
	roFile, err := os.Open(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer roFile.Close()
	if !os.IsNotExist(err) {
		s := bufio.NewScanner(roFile)
		for s.Scan() {
			if strings.HasPrefix(s.Text(), "# /"+dubbo.DataDir) { // if it was commented
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "#/"+dubbo.DataDir) {
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "/"+dubbo.DataDir) { // if it is there
				return nil // we're done
			}
		}
	}
	roFile.Close()
	rwFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer rwFile.Close()
	if _, err = rwFile.WriteString(`
# Applications use the .dubbo directory for local runtime data which should
# generally not be tracked in source control. To instruct the system to track
# .dubbo in source control, comment the following line (prefix it with '# ').
/.dubbo
`); err != nil {
		return err
	}
	if err = rwFile.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: error when syncing .gitignore. %s\n", err)
	}
	return nil
}

func WithRepositoriesPath(path string) Option {
	return func(c *Client) {
		c.repositoriesPath = path
	}
}
