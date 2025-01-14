package sdk

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type Client struct {
}

func (c *Client) Create(cfg *Dubbo, init bool, cmd *cobra.Command) (*Dubbo, error) {
	// convert Root path to absolute
	var err error
	oldRoot := cfg.Root
	cfg.Root, err = filepath.Abs(cfg.Root)
	if err != nil {
		return cfg, err
	}

	// Create project root directory, if it doesn't already exist
	if err = os.MkdirAll(cfg.Root, 0o755); err != nil {
		return cfg, err
	}

	// Create should never clobber a pre-existing function
	hasApp, err := hasInitializedApplication(cfg.Root)
	if err != nil {
		return cfg, err
	}
	if hasApp {
		return cfg, fmt.Errorf("application at '%v' already initialized", cfg.Root)
	}

	// Path is defaulted to the current working directory
	if cfg.Root == "" {
		if cfg.Root, err = os.Getwd(); err != nil {
			return cfg, err
		}
	}

	// Name is defaulted to the directory of the given path.
	if cfg.Name == "" {
		cfg.Name = nameFromPath(cfg.Root)
	}

	if !init {
		// The path for the new function should not have any contentious files
		// (hidden files OK, unless it's one used by dubbo)
		if err := assertEmptyRoot(cfg.Root); err != nil {
			return cfg, err
		}
	}

	// Create a new application (in memory)
	f := NewDubboWith(cfg, init)

	// Create a .dubbo directory which is also added to a .gitignore
	if err = EnsureRunDataDir(f.Root); err != nil {
		return f, err
	}

	if !init {
		// Write out the new function's Template files.
		// Templates contain values which may result in the function being mutated
		// (default builders, etc), so a new (potentially mutated) function is
		// returned from Templates.Write
		err = c.Templates().Write(f)
		if err != nil {
			return f, err
		}
	}
	f.Created = time.Now()
	err = f.Write()
	if err != nil {
		return f, err
	}
	err = f.EnsureDockerfile(cmd)
	if err != nil {
		return f, err
	}

	// Load the now-initialized application.
	return NewDubbo(oldRoot)
}

func hasInitializedApplication(path string) (bool, error) {
	var err error
	filename := filepath.Join(path, DubboFile)

	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err // invalid path or access error
	}
	bb, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}
	f := Dubbo{}
	if err = yaml.Unmarshal(bb, &f); err != nil {
		return false, err
	}

	return f.Initialized(), nil
}
