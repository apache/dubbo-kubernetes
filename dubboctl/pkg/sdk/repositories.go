package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Repositories struct {
	client *Client
	path   string
}

func newRepositories(client *Client) *Repositories {
	return &Repositories{
		client: client,
		path:   client.repositoriesPath,
	}
}

func (r *Repositories) All() (repos []Repository, err error) {
	var repo Repository

	if repo, err = NewRepository("", ""); err != nil {
		return
	}
	repos = append(repos, repo)

	if r.path == "" {
		return
	}

	if _, err = os.Stat(r.path); os.IsNotExist(err) {
		return repos, nil
	}

	ff, err := os.ReadDir(r.path)
	if err != nil {
		return
	}
	for _, f := range ff {
		if !f.IsDir() || strings.HasPrefix(f.Name(), ".") {
			continue
		}
		var abspath string
		abspath, err = filepath.Abs(r.path)
		if err != nil {
			return
		}
		if repo, err = NewRepository("", "file://"+filepath.ToSlash(abspath)+"/"+f.Name()); err != nil {
			return
		}
		repos = append(repos, repo)
	}
	return
}

func (r *Repositories) Get(name string) (repo Repository, err error) {
	all, err := r.All()
	if err != nil {
		return
	}
	if len(all) == 0 {
		err = errors.New("internal error: no repositories loaded")
		return
	}

	if name == DefaultRepositoryName {
		repo = all[0]
		return
	}

	for _, v := range all {
		if v.Name == name {
			repo = v
			return
		}
	}
	return repo, fmt.Errorf("repository not found")
}
