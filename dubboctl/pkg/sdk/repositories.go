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

func (r *Repositories) Add(name, url string) (string, error) {
	if r.path == "" {
		return "", fmt.Errorf("repository %v not added.", name)
	}

	repo, err := NewRepository(name, url)
	if err != nil {
		return "", fmt.Errorf("failed to create new repository: %w", err)
	}

	dest := filepath.Join(r.path, repo.Name)
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return "", fmt.Errorf("repository '%v' already exists", repo.Name)
	}

	err = repo.Write(dest)
	if err != nil {
		return "", fmt.Errorf("failed to write repository: %w", err)
	}
	return repo.Name, nil
}

func (r *Repositories) Remove(name string) error {
	if r.path == "" {
		return fmt.Errorf("repository %v not removed.", name)
	}
	if name == "" {
		return errors.New("name is required")
	}
	path := filepath.Join(r.path, name)
	return os.RemoveAll(path)
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
