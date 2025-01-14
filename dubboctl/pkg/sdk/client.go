package sdk

import (
	"bufio"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DubboFile       = "dubbo.yaml"
	RunDataDir      = ".dubbo"
	DefaultTemplate = "common"
)

type Client struct {
	templates *Templates
}
type Option func(*Client)

func New(options ...Option) *Client {
	c := &Client{}
	for _, o := range options {
		o(c)
	}

	c.templates = newTemplates(c)

	return c
}

func newTemplates(client *Client) *Templates {
	return &Templates{client: client}
}

type Templates struct {
	client *Client
}

type DubboConfig struct {
	Root     string    `yaml:"-"`
	Name     string    `yaml:"name,omitempty" jsonschema:"pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`
	Runtime  string    `yaml:"runtime,omitempty"`
	Template string    `yaml:"template,omitempty"`
	Created  time.Time `yaml:"created,omitempty"`
}

func (f *DubboConfig) Validate() error {
	if f.Root == "" {
		return errors.New("dubbo root path is required")
	}

	var ctr int
	errs := [][]string{
		validateOptions(),
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("'%v' contains errors:", DubboFile))

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

func validateOptions() []string {
	return nil
}

func (c *Client) Templates() *Templates {
	return c.templates
}

func (f *DubboConfig) Write() (err error) {
	if err = f.Validate(); err != nil {
		return
	}
	dubboyamlpath := filepath.Join(f.Root, DubboFile)
	var dubbobytes []byte
	if dubbobytes, err = yaml.Marshal(f); err != nil {
		return
	}
	if err = os.WriteFile(dubboyamlpath, dubbobytes, 0644); err != nil {
		return
	}
	return
}

type Template interface {
	Name() string
	Runtime() string
	Repository() string
	Fullname() string
	Write(ctx context.Context, f *DubboConfig) error
}

type template struct {
	name    string
	runtime string
}

func (t template) Name() string {
	return t.name
}

func (t template) Runtime() string {
	return t.runtime
}

func (c *Client) Create(dcfg *DubboConfig, initial bool, cmd *cobra.Command) (*DubboConfig, error) {
	var err error
	oldRoot := dcfg.Root
	dcfg.Root, err = filepath.Abs(dcfg.Root)
	if err != nil {
		return dcfg, err
	}

	if err = os.MkdirAll(dcfg.Root, 0o755); err != nil {
		return dcfg, err
	}

	has, err := hasInitializedApplication(dcfg.Root)
	if err != nil {
		return dcfg, err
	}
	if has {
		return dcfg, fmt.Errorf("application at '%v' already initialized", dcfg.Root)
	}

	if dcfg.Root == "" {
		if dcfg.Root, err = os.Getwd(); err != nil {
			return dcfg, err
		}
	}

	if dcfg.Name == "" {
		dcfg.Name = nameFromPath(dcfg.Root)
	}

	if !initial {
		if err := assertEmptyRoot(dcfg.Root); err != nil {
			return dcfg, err
		}
	}

	f := NewDubboWithTemplate(dcfg, initial)

	if err = EnsureRunDataDir(f.Root); err != nil {
		return f, err
	}

	f.Created = time.Now()
	err = f.Write()
	if err != nil {
		return f, err
	}

	return NewDubbo(oldRoot)
}

func NewDubbo(path string) (*DubboConfig, error) {
	var err error
	f := &DubboConfig{}
	// Path defaults to current working directory, and if provided explicitly
	// Path must exist and be a directory
	if path == "" {
		if path, err = os.Getwd(); err != nil {
			return f, err
		}
	}
	f.Root = path // path is not persisted, as this is the purview of the FS

	// Path must exist and be a directory
	fd, err := os.Stat(path)
	if err != nil {
		return f, err
	}
	if !fd.IsDir() {
		return f, fmt.Errorf("function path must be a directory")
	}

	// If no dubbo.yaml in directory, return the default function which will
	// have f.Initialized() == false
	filename := filepath.Join(path, DubboFile)
	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return f, err
	}

	// Path is valid and dubbo.yaml exists: load it
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

func hasInitializedApplication(path string) (bool, error) {
	var err error
	filename := filepath.Join(path, DubboFile)

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
	f := DubboConfig{}
	if err = yaml.Unmarshal(bb, &f); err != nil {
		return false, err
	}

	return f.Initialized(), nil
}

func (f *DubboConfig) Initialized() bool {
	return !f.Created.IsZero()
}

func nameFromPath(path string) string {
	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
	return pathParts[len(pathParts)-1]
	/* the above may have edge conditions as it assumes the trailing value
	 * is a directory name.  If errors are encountered, we _may_ need to use the
	 * inbuilt logic in the std lib and either check if the path indicated is a
	 * directory (appending slash) and then run:
					 base := filepath.Base(filepath.Dir(path))
					 if base == string(os.PathSeparator) || base == "." {
									 return "" // Consider it underivable: string zero value
					 }
					 return base
	*/
}

func assertEmptyRoot(path string) (err error) {
	// If there exists contentious files (config files for instance), this function may have already been initialized.
	files, err := contentiousFilesIn(path)
	if err != nil {
		return
	} else if len(files) > 0 {
		return fmt.Errorf("the chosen directory '%v' contains contentious files: %v.  Has the Service function already been created?  Try either using a different directory, deleting the function if it exists, or manually removing the files", path, files)
	}

	// Ensure there are no non-hidden files, and again none of the aforementioned contentious files.
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
	DubboFile,
	".gitignore",
}

// contentiousFilesIn the given directory
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
	// Check for any non-hidden files
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

func EnsureRunDataDir(root string) error {
	// Ensure the runtime directory exists
	if err := os.MkdirAll(filepath.Join(root, RunDataDir), os.ModePerm); err != nil {
		return err
	}

	// Update .gitignore
	//
	// Ensure .dubbo is added to .gitignore unless the user explicitly
	// commented out the ignore line for some awful reason.
	// Also creates the .gitignore in the application's root directory if it does
	// not already exist (note that this may not be in the root of the repo
	// if the function is at a subpart of a monorepo)
	filePath := filepath.Join(root, ".gitignore")
	roFile, err := os.Open(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer roFile.Close()
	if !os.IsNotExist(err) { // if no error openeing it
		s := bufio.NewScanner(roFile) // create a scanner
		for s.Scan() {                // scan each line
			if strings.HasPrefix(s.Text(), "# /"+RunDataDir) { // if it was commented
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "#/"+RunDataDir) {
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "/"+RunDataDir) { // if it is there
				return nil // we're done
			}
		}
	}
	// Either .gitignore does not exist or it does not have the ignore
	// directive for .func yet.
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

	// Flush to disk immediately since this may affect subsequent calculations
	// of the build stamp
	if err = rwFile.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: error when syncing .gitignore. %s\n", err)
	}
	return nil
}

func NewDubboWithTemplate(defaults *DubboConfig, init bool) *DubboConfig {
	if !init {
		if defaults.Template == "" {
			defaults.Template = DefaultTemplate
		}
	} else {
		defaults.Template = "init"
	}
	return defaults
}
