package util

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	Repositories = "repositories"
)

func Dir() (path string) {
	if home, err := os.UserHomeDir(); err == nil {
		path = filepath.Join(home, ".config", "dubbo")
	}
	return
}

func GetCreatePath() (err error) {
	if err = os.MkdirAll(Dir(), os.ModePerm); err != nil {
		return fmt.Errorf("error creating global config path: %v", err)
	}
	if err = os.MkdirAll(RepositoriesPath(), os.ModePerm); err != nil {
		return fmt.Errorf("error creating global config repositories path: %v", err)
	}
	return
}

func RepositoriesPath() string {
	path := filepath.Join(Dir(), Repositories)
	if e := os.Getenv("DUBBO_REPOSITORIES_PATH"); e != "" {
		path = e
	}
	return path
}
