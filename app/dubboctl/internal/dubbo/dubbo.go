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
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	"gopkg.in/yaml.v2"
)

const (
	// DubboFile is the file used for the serialized form of a function.
	DubboFile = "dubbo.yaml"

	Dockerfile = "Dockerfile"

	// RunDataDir holds transient runtime metadata
	// By default it is excluded from source control.
	RunDataDir = ".dubbo"

	// buildstamp is the name of the file within the run data directory whose
	// existence indicates the function has been built, and whose content is
	// a fingerprint of the filesystem at the time of the build.
	buildstamp = "built"
)

type Dubbo struct {
	// Root on disk at which to find/create source and config files.
	Root string `yaml:"-"`

	// Name of the application.
	Name string `yaml:"name,omitempty" jsonschema:"pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`

	// Runtime is the language plus context.  java|go etc.
	Runtime string `yaml:"runtime,omitempty"`

	// Template for the application.
	Template string `yaml:"template,omitempty"`

	// Optional full OCI image tag in form:
	//   [registry]/[namespace]/[name]:[tag]
	// example:
	//   quay.io/alice/my.function.name
	// Registry is optional and is defaulted to DefaultRegistry
	// example:
	//   alice/my.function.name
	// If Image is provided, it overrides the default of concatenating
	// "Registry+Name:latest" to derive the Image.
	Image string `yaml:"image,omitempty"`

	// SHA256 hash of the latest image that has been built
	ImageDigest string `yaml:"-"`

	// Created time is the moment that creation was successfully completed
	// according to the client which is in charge of what constitutes being
	// fully "Created" (aka initialized)
	Created time.Time `yaml:"created,omitempty"`

	// BuildSpec define the build properties for a function
	Build BuildSpec `yaml:"build,omitempty"`

	// DeploySpec define the deployment properties for a function
	Deploy DeploySpec `yaml:"deploy,omitempty"`
}

type Env struct {
	Name  *string `yaml:"name,omitempty" jsonschema:"pattern=^[-._a-zA-Z][-._a-zA-Z0-9]*$"`
	Value *string `yaml:"value,omitempty"`
}

type Label struct {
	// Key consist of optional prefix part (ended by '/') and name part
	// Prefix part validation pattern: [a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*
	// Name part validation pattern: ([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]
	Key   *string `yaml:"key" jsonschema:"pattern=^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\\/)?([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$"`
	Value *string `yaml:"value,omitempty" jsonschema:"pattern=^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$"`
}

type BuildSpec struct {
	// BuilderImages define optional explicit builder images to use by
	// builder implementations in lea of the in-code defaults.  They key
	// is the builder's short name.  For example:
	// builderImages:
	//   pack: example.com/user/my-pack-node-builder
	BuilderImages map[string]string `yaml:"builderImages,omitempty"`

	// Optional list of build-packs to use when building the application
	Buildpacks []string `yaml:"buildpacks,omitempty"`

	BuildEnvs []Env `json:"buildEnvs,omitempty"`
}

type DeploySpec struct {
	Namespace     string `yaml:"namespace,omitempty"`
	Output        string `yaml:"output,omitempty"`
	ContainerPort int    `yaml:"containerPort,omitempty"`
	TargetPort    int    `yaml:"targetPort,omitempty"`
	NodePort      int    `yaml:"nodePort,omitempty"`
	Registry      string `yaml:"registry,omitempty"`
	UseProm       bool   `yaml:"-"`
}

func (f *Dubbo) Validate() error {
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

// Initialized returns if the function has been initialized.
// Any errors are considered failure (invalid or inaccessible root, config file, etc).
func (f *Dubbo) Initialized() bool {
	return !f.Created.IsZero()
}

// nameFromPath returns the default name for a function derived from a path.
// This consists of the last directory in the given path, if derivable (empty
// paths, paths consisting of all slashes, etc. return the zero value "")
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

// assertEmptyRoot ensures that the directory is empty enough to be used for
// initializing a new function.
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

// contentiousFiles are files which, if extant, preclude the creation of a
// function rooted in the given directory.
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

// effectivelyEmpty directories are those which have no visible files
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

// NewDubboWith defaults as provided.
func NewDubboWith(defaults *Dubbo, init bool) *Dubbo {
	if !init {
		if defaults.Template == "" {
			defaults.Template = DefaultTemplate
		}
	} else {
		defaults.Template = "init"
	}
	if defaults.Build.BuilderImages == nil {
		defaults.Build.BuilderImages = make(map[string]string)
	}
	return defaults
}

// NewDubbo from a given path.
// Invalid paths, or no function at path are errors.
// Syntactic errors are returned immediately (yaml unmarshal errors).
// Dubbo which are syntactically valid are also then logically validated.
// Dubbo from earlier versions are brought up to current using migrations.
// Migrations are run prior to validators such that validation can omit
// concerning itself with backwards compatibility. Migrators must therefore
// selectively consider the minimal validation necessary to enable migration.
func NewDubbo(path string) (*Dubbo, error) {
	var err error
	f := &Dubbo{}
	f.Build.BuilderImages = make(map[string]string)
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

// buildStamp returns the current (last) build stamp for the function
// at the given path, if it can be found.
func (f *Dubbo) buildStamp() string {
	buildstampPath := filepath.Join(f.Root, RunDataDir, buildstamp)
	if _, err := os.Stat(buildstampPath); err != nil {
		return ""
	}
	b, err := os.ReadFile(buildstampPath)
	if err != nil {
		return ""
	}
	return string(b)
}

func (f *Dubbo) CheckLabels(ns string, client *Client) error {
	key := client2.ObjectKey{
		Namespace: metav1.NamespaceSystem,
		Name:      ns,
	}

	var err error

	err = client.KubeCtl.Get(context.Background(), key, &corev1.Namespace{})
	if err != nil {
		if errors2.IsNotFound(err) {
			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: metav1.NamespaceSystem,
					Name:      ns,
				},
			}
			if err := client.KubeCtl.Create(context.Background(), nsObj); err != nil {
				return err
			}
			return nil
		} else {
			return fmt.Errorf("failed to check if namespace %v exists: %v", ns, err)
		}
	}

	namespaceSelector := client2.MatchingLabels{
		"dubbo-deploy": "enabled",
	}
	nsList := &corev1.NamespaceList{}
	if err = client.KubeCtl.List(context.Background(), nsList, namespaceSelector); err != nil {
		if errors2.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	var namespace string
	if len(nsList.Items) > 0 {
		namespace = nsList.Items[0].Name
	}

	env := os.Getenv("DUBBO_DEPLOY_NS")
	if env != "" {
		namespace = env
	}

	if namespace != "" {
		var name string
		var dns string

		nacosSelector := client2.MatchingLabels{
			"dubbo.apache.org/nacos": "true",
		}

		nacosList := &corev1.ServiceList{}
		if err := client.KubeCtl.List(context.Background(), nacosList, nacosSelector, client2.InNamespace(namespace)); err != nil {
			if errors2.IsNotFound(err) {
				return nil
			} else {
				return err
			}
		}
		if len(nacosList.Items) > 0 {
			name = nacosList.Items[0].Name
			dns = fmt.Sprintf("nacos://%s.%s.svc", name, namespace)
			f.Deploy.Registry = dns
		}

		zkSelector := client2.MatchingLabels{
			"dubbo.apache.org/zookeeper": "true",
		}

		zkList := &corev1.ServiceList{}
		if err := client.KubeCtl.List(context.Background(), zkList, zkSelector, client2.InNamespace(namespace)); err != nil {
			if errors2.IsNotFound(err) {
				return nil
			} else {
				return err
			}
		}

		if len(zkList.Items) > 0 {
			name = zkList.Items[0].Name
			dns = fmt.Sprintf("zookeeper://%s.%s.svc", name, namespace)
			f.Deploy.Registry = dns
		}

		promSelector := client2.MatchingLabels{
			"dubbo.apache.org/prometheus": "true",
		}

		promList := &corev1.ServiceList{}
		if err := client.KubeCtl.List(context.Background(), promList, promSelector, client2.InNamespace(namespace)); err != nil {
			if errors2.IsNotFound(err) {
				return nil
			} else {
				return err
			}
		}
		if len(promList.Items) > 0 {
			f.Deploy.UseProm = true
		}
	}

	return nil
}

// Write aka (save, serialize, marshal) the function to disk at its path.
// Only valid functions can be written.
// In order to retain built status (staleness checks), the file is only
// modified if the structure actually changes.
func (f *Dubbo) Write() (err error) {
	if err = f.Validate(); err != nil {
		return
	}
	dubboyamlpath := filepath.Join(f.Root, DubboFile)
	var dubbobytes []byte
	if dubbobytes, err = yaml.Marshal(f); err != nil {
		return
	}
	if err = os.WriteFile(dubboyamlpath, dubbobytes, 0o644); err != nil {
		return
	}
	return
}

type (
	stampOptions struct{ journal bool }
	stampOption  func(o *stampOptions)
)

// Stamp a function as being built.
//
// This is a performance optimization used when updates to the
// function are known to have no effect on its built container.  This
// stamp is checked before certain operations, and if it has been updated,
// the build can be skipped.  If in doubt, just use .Write only.
//
// Updates the build stamp at ./dubbo/built (and the log
// at .dubbo/built.log) to reflect the current state of the filesystem.
// Note that the caller should call .Write first to flush any changes to the
// application in-memory to the filesystem prior to calling stamp.
//
// The runtime data directory .dubbo is created in the application root if
// necessary.
func (f *Dubbo) Stamp(oo ...stampOption) (err error) {
	options := &stampOptions{}
	for _, o := range oo {
		o(options)
	}
	if err = EnsureRunDataDir(f.Root); err != nil {
		return
	}

	// Calculate the hash and a logfile of what comprised it
	var hash, log string
	if hash, log, err = Fingerprint(f); err != nil {
		return
	}

	// Write out the hash
	if err = os.WriteFile(filepath.Join(f.Root, RunDataDir, "built"), []byte(hash), os.ModePerm); err != nil {
		return
	}

	// Write out the logfile, optionally timestamped for retention.
	logfileName := "built.log"
	if options.journal {
		logfileName = timestamp(logfileName)
	}
	logfile, err := os.Create(filepath.Join(f.Root, RunDataDir, logfileName))
	if err != nil {
		return
	}
	defer logfile.Close()
	_, err = fmt.Fprintln(logfile, log)
	return
}

func (f *Dubbo) EnsureDockerfile(cmd *cobra.Command) (err error) {
	dockerfilepath := filepath.Join(f.Root, Dockerfile)
	dockerfilebytes, ok := DockerfileByRuntime[f.Runtime]
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "The runtime of your current project is not one of Java or go. We cannot help you generate a Dockerfile template.")
		return
	}
	if err = os.WriteFile(dockerfilepath, []byte(dockerfilebytes), 0o644); err != nil {
		return
	}
	return
}

// timestamp returns the given string prefixed with a microsecond-precision
// timestamp followed by a dot.
// YYYYMMDDHHMMSS.$nanosecond.$s
func timestamp(s string) string {
	t := time.Now()
	return fmt.Sprintf("%s.%09d.%s", t.Format("20060102150405"), t.Nanosecond(), s)
}

// Fingerprint the files at a given path.  Returns a hash calculated from the
// filenames and modification timestamps of the files within the given root.
// Also returns a logfile consisting of the filenames and modification times
// which contributed to the hash.
// Intended to determine if there were appreciable changes to a function's
// source code, certain directories and files are ignored, such as
// .git and .dubbo.
func Fingerprint(f *Dubbo) (hash, log string, err error) {
	h := sha256.New()   // Hash builder
	l := bytes.Buffer{} // Log buffer

	root := f.Root
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	output := f.Deploy.Output

	err = filepath.Walk(abs, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		// Always ignore .dubbo, .git etc
		if info.IsDir() && (info.Name() == RunDataDir || info.Name() == ".git" || info.Name() == ".idea") {
			return filepath.SkipDir
		}
		if info.Name() == DubboFile || info.Name() == Dockerfile || info.Name() == output {
			return nil
		}
		fmt.Fprintf(h, "%v:%v:", path, info.ModTime().UnixNano())   // Write to the Hashed
		fmt.Fprintf(&l, "%v:%v\n", path, info.ModTime().UnixNano()) // Write to the Log
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil)), l.String(), err
}
