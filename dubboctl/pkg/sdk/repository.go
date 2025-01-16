package sdk

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/fs"
	"net/url"
	"os"
	"path"
	"strings"
)

type Repository struct {
	Name     string
	Runtimes []Runtime
	fs       fs.Filesystem
}

type repositoryConfig struct {
	DefaultName   string `yaml:"name,omitempty"`
	TemplatesPath string `yaml:"templates,omitempty"`
}

func NewRepository(name, uri string) (r Repository, err error) {
	r = Repository{}
	repoConfig := repositoryConfig{}
	if repoConfig.TemplatesPath != "" {
		if err = checkDir(r.fs, repoConfig.TemplatesPath); err != nil {
			err = fmt.Errorf("templates path '%v' does not exist in repo '%v'. %v",
				repoConfig.TemplatesPath, r.Name, err)
			return
		}
	} else {
		repoConfig.TemplatesPath = "."
	}

	r.Name, err = repositoryDefaultName(repoConfig.DefaultName, uri)
	if err != nil {
		return
	}
	if name != "" {
		r.Name = name
	}
	r.Runtimes, err = repositoryRuntimes(nil, r.Name, repoConfig)
	return
}

func repositoryDefaultName(name, uri string) (string, error) {
	if name != "" {
		return name, nil
	}
	if uri != "" {
		parsed, err := url.Parse(uri)
		if err != nil {
			return "", err
		}
		ss := strings.Split(parsed.Path, "/")
		if len(ss) > 0 {
			return strings.TrimSuffix(ss[len(ss)-1], ".git"), nil
		}
	}
	return DefaultRepositoryName, nil
}

func repositoryRuntimes(fs fs.Filesystem, repoName string, repoConfig repositoryConfig) (runtimes []Runtime, err error) {
	runtimes = []Runtime{}

	fis, err := fs.ReadDir(repoConfig.TemplatesPath)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		runtime := Runtime{
			Name: fi.Name(),
		}

		runtime.Templates, err = runtimeTemplates(fs, repoConfig.TemplatesPath, repoName, runtime.Name)
		if err != nil {
			return
		}
		runtimes = append(runtimes, runtime)
	}
	return
}

func runtimeTemplates(fs fs.Filesystem, templatesPath, repoName, runtimeName string) (templates []Template, err error) {
	runtimePath := path.Join(templatesPath, runtimeName)
	if err = checkDir(fs, runtimePath); err != nil {
		err = fmt.Errorf("runtime path '%v' not found. %v", runtimePath, err)
		return
	}

	fis, err := fs.ReadDir(runtimePath)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		t := template{
			name:    fi.Name(),
			runtime: runtimeName,
		}

		templates = append(templates, t)
	}
	return
}

func checkDir(fs fs.Filesystem, path string) error {
	fi, err := fs.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err = fmt.Errorf("path '%v` not found", path)
	} else if err == nil && !fi.IsDir() {
		err = fmt.Errorf("path '%v' is not a directory", path)
	}
	return err
}

type Runtime struct {
	Name      string
	Templates []Template
}

func (r *Repository) Template(runtimeName, name string) (t Template, err error) {
	runtime, err := r.Runtime(runtimeName)
	if err != nil {
		return
	}
	for _, t := range runtime.Templates {
		if t.Name() == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template not found")
}

func (r *Repository) Runtime(name string) (runtime Runtime, err error) {
	if name == "" {
		return Runtime{}, fmt.Errorf("language runtime required")
	}
	for _, runtime = range r.Runtimes {
		if runtime.Name == name {
			return runtime, err
		}
	}
	return Runtime{}, fmt.Errorf("language runtime not found")
}
