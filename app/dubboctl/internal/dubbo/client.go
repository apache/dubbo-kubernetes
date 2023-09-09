/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubbo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultTemplate is the default function signature / environmental context
	// of the resultant function.  All runtimes are expected to have at least
	// one implementation of each supported function signature.  Currently that
	// includes common
	DefaultTemplate = "common"
)

type Client struct {
	repositoriesPath string        // path to repositories
	repositoriesURI  string        // repo URI (overrides repositories path)
	templates        *Templates    // Templates management
	repositories     *Repositories // Repositories management
	builder          Builder       // Builds a runnable image source
	pusher           Pusher        // Pushes function image to a remote
	deployer         Deployer      // Deploys or Updates a function}
}

// Builder of function source to runnable image.
type Builder interface {
	// Build a function project with source located at path.
	Build(context.Context, *Dubbo) error
}

// Pusher of function image to a registry.
type Pusher interface {
	// Push the image of the function.
	// Returns Image Digest - SHA256 hash of the produced image
	Push(ctx context.Context, f *Dubbo) error
}

// Deployer of function source to running status.
type Deployer interface {
	// Deploy a function of given name, using given backing image.
	Deploy(context.Context, *Dubbo, ...DeployOption) (DeploymentResult, error)
}

type DeploymentResult struct {
	Status    Status
	Namespace string
}

type Status int

const (
	Failed Status = iota
	Deployed
)

// Repositories accessor
func (c *Client) Repositories() *Repositories {
	return c.repositories
}

// Templates accessor
func (c *Client) Templates() *Templates {
	return c.templates
}

// Runtimes available in totality.
// Not all repository/template combinations necessarily exist,
// and further validation is performed when a template+runtime is chosen.
// from a given repository.  This is the global list of all available.
// Returned list is unique and sorted.
func (c *Client) Runtimes() ([]string, error) {
	runtimes := util.NewSortedSet()

	// Gather all runtimes from all repositories
	// into a uniqueness map
	repositories, err := c.Repositories().All()
	if err != nil {
		return []string{}, err
	}
	for _, repo := range repositories {
		for _, runtime := range repo.Runtimes {
			runtimes.Add(runtime.Name)
		}
	}

	// Return a unique, sorted list of runtimes
	return runtimes.Items(), nil
}

// Option defines a function which when passed to the Client constructor
// optionally mutates private members at time of instantiation.
type Option func(*Client)

func WithPusher(pusher Pusher) Option {
	return func(c *Client) {
		c.pusher = pusher
	}
}

// WithRepository sets a specific URL to a Git repository from which to pull
// templates.  This setting's existence precldes the use of either the inbuilt
// templates or any repositories from the extensible repositories path.
func WithRepository(uri string) Option {
	return func(c *Client) {
		c.repositoriesURI = uri
	}
}

// WithBuilder provides the concrete implementation of a builder.
func WithBuilder(d Builder) Option {
	return func(c *Client) {
		c.builder = d
	}
}

func WithDeployer(d Deployer) Option {
	return func(c *Client) {
		c.deployer = d
	}
}

// WithRepositoriesPath sets the location on disk to use for extensible template
// repositories.  Extensible template repositories are additional templates
// that exist on disk and are not built into the binary.
func WithRepositoriesPath(path string) Option {
	return func(c *Client) {
		c.repositoriesPath = path
	}
}

// Init Initialize a new function with the given function struct defaults.
// Does not build/push/deploy. Local FS changes only.  For higher-level
// control see New or Apply.
//
// <path> will default to the absolute path of the current working directory.
// <name> will default to the current working directory.
// When <name> is provided but <path> is not, a directory <name> is created
// in the current working directory and used for <path>.
func (c *Client) Init(cfg *Dubbo) (*Dubbo, error) {
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

	// The path for the new function should not have any contentious files
	// (hidden files OK, unless it's one used by dubbo)
	if err := assertEmptyRoot(cfg.Root); err != nil {
		return cfg, err
	}

	// Create a new application (in memory)
	f := NewDubboWith(cfg)

	// Create a .dubbo directory which is also added to a .gitignore
	if err = EnsureRunDataDir(f.Root); err != nil {
		return f, err
	}

	// Write out the new function's Template files.
	// Templates contain values which may result in the function being mutated
	// (default builders, etc), so a new (potentially mutated) function is
	// returned from Templates.Write
	err = c.Templates().Write(f)
	if err != nil {
		return f, err
	}

	f.Created = time.Now()
	err = f.Write()
	if err != nil {
		return f, err
	}

	// Load the now-initialized application.
	return NewDubbo(oldRoot)
}

// EnsureRunDataDir creates a .dubbo directory at the given path, and
// registers it as ignored in a .gitignore file.
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
		fmt.Fprintf(os.Stderr, "warning: error when syncing .gitignore. %s", err)
	}
	return nil
}

// returns true if the given path contains an initialized function.
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

// New client for function management.
func New(options ...Option) *Client {
	// Instantiate client with static defaults.
	c := &Client{}
	for _, o := range options {
		o(c)
	}

	// Initialize sub-managers using now-fully-initialized client.
	c.repositories = newRepositories(c)
	c.templates = newTemplates(c)

	return c
}

type BuildOptions struct{}

type BuildOption func(c *BuildOptions)

// Build the function at path. Errors if the function is either unloadable or does
// not contain a populated Image.
func (c *Client) Build(ctx context.Context, f *Dubbo, options ...BuildOption) (*Dubbo, error) {
	fmt.Fprintf(os.Stderr, "Building...\n")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If not logging verbosely, the ongoing progress of the build will not
	// be streaming to stdout, and the lack of activity has been seen to cause
	// users to prematurely exit due to the sluggishness of pulling large images
	//if !c.verbose {
	//	c.printBuildActivity(ctx) // print friendly messages until context is canceled
	//}

	// Options for the build task
	oo := BuildOptions{}
	for _, o := range options {
		o(&oo)
	}

	var err error

	if err = c.builder.Build(ctx, f); err != nil {
		return f, err
	}

	if err = f.Stamp(); err != nil {
		return f, err
	}

	// use by the cli for user echo (rather than rely on verbose mode here)
	message := fmt.Sprintf("🙌 Application built: %v", f.Image)
	if runtime.GOOS == "windows" {
		message = fmt.Sprintf("Application built: %v", f.Image)
	}
	fmt.Fprintf(os.Stderr, "%s\n", message)

	return f, err
}

type DeployParams struct {
	skipBuiltCheck bool
}
type DeployOption func(f *DeployParams)

func (c *Client) Deploy(ctx context.Context, d *Dubbo, opts ...DeployOption) (*Dubbo, error) {
	deployParams := &DeployParams{skipBuiltCheck: false}
	for _, opt := range opts {
		opt(deployParams)
	}

	go func() {
		<-ctx.Done()
	}()

	// Application must be built (have an associated image) before being deployed.
	// Note that externally built images may be specified in the dubbo.yaml
	// if !deployParams.skipBuiltCheck && !d.Built() {
	//	 return d, ErrNotBuilt
	//}

	// Application must have a name to be deployed (a path on the network at which
	// it should take up residence.
	if d.Name == "" {
		return d, ErrNameRequired
	}

	fmt.Fprintf(os.Stderr, "⬆️  Deploying function to the cluster or generate manifest\n")
	result, err := c.deployer.Deploy(ctx, d)
	if err != nil {
		fmt.Printf("deploy error: %v\n", err)
		return d, err
	}

	// Update the function with the namespace into which the function was
	// deployed
	d.Deploy.Namespace = result.Namespace

	if result.Status == Deployed {
		fmt.Fprintf(os.Stderr, "✅ Application deployed in namespace %q or manifest had been generated\n", result.Namespace)
	}

	return d, nil
}

// BuildStamp accesses the current (last) build stamp for the function.
// Inbuilt functions return empty string.
func (f Dubbo) BuildStamp() string {
	path := filepath.Join(f.Root, RunDataDir, "built")
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// Built returns true if the application is considered built.
// Note that this only considers the application as it exists on-disk at
// f.Root.
func (f Dubbo) Built() bool {
	// If there is no build stamp, it is not built.
	stamp := f.BuildStamp()
	if stamp == "" {
		return false
	}

	// Missing an image name always means !Built (but does not satisfy staleness
	// checks).
	// NOTE: This will be updated in the future such that a build does not
	// automatically update the function's serialized, source-controlled state,
	// because merely building does not indicate the function has changed, but
	// rather that field should be populated on deploy.  I.e. the Image name
	// and image stamp should reside as transient data in .func until such time
	// as the given image has been deployed.
	// An example of how this bug manifests is that every rebuild of a function
	// registers the func.yaml as being dirty for source-control purposes, when
	// this should only happen on deploy.
	if f.Image == "" {
		return false
	}

	// Calculate the current filesystem hash and see if it has changed.
	//
	// If this comparison returns true, the Function has a populated image,
	// existing builds-tamp, and the calculated fingerprint has not changed.
	//
	// It's a pretty good chance the thing doesn't need to be rebuilt, though
	// of course filesystem racing conditions do exist, including both direct
	// source code modifications or changes to the image cache.
	hash, _, err := Fingerprint(f.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error calculating function's fingerprint: %v\n", err)
		return false
	}
	return stamp == hash
}

// Push the image for the named service to the configured registry
func (c *Client) Push(ctx context.Context, f *Dubbo) (*Dubbo, error) {
	if !f.Built() {
		return f, ErrNotBuilt
	}
	var err error
	if err = c.pusher.Push(ctx, f); err != nil {
		return f, err
	}

	return f, nil
}

// DEFAULTS
// ---------

// Manual implementations (noobs) of required interfaces.
// In practice, the user of this client package (for example the CLI) will
// provide a concrete implementation for only the interfaces necessary to
// complete the given command.  Integrators importing the package would
// provide a concrete implementation for all interfaces to be used. To
// enable partial definition (in particular used for testing) they
// are defaulted to noop implementations such that they can be provided
// only when necessary.  Unit tests for the concrete implementations
// serve to keep the core logic here separate from the imperative, and
// with a minimum of external dependencies.
// -----------------------------------------------------

// Builder
type noopBuilder struct{ output io.Writer }

func (n *noopBuilder) Build(ctx context.Context, _ Dubbo) error { return nil }

// Pusher
type noopPusher struct{ output io.Writer }

func (n *noopPusher) Push(ctx context.Context, f *Dubbo) error { return nil }

// Deployer
type noopDeployer struct{ output io.Writer }

func (n *noopDeployer) Deploy(ctx context.Context, _ *Dubbo, option ...DeployOption) (DeploymentResult, error) {
	return DeploymentResult{}, nil
}
