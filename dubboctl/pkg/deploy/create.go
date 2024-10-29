// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/common"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

import (
	"github.com/AlecAivazis/survey/v2"

	"github.com/ory/viper"

	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/util"
)

// ErrNoRuntime indicates that the language runtime flag was not passed.
type ErrNoRuntime error

// ErrInvalidRuntime indicates that the passed language runtime was invalid.
type ErrInvalidRuntime error

// ErrInvalidTemplate indicates that the passed template was invalid.
type ErrInvalidTemplate error

// NewCreateCmd creates a create command using the given client creator.
func AddCreate(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an application",
		Long: `
NAME
	{{.Name}} create - Create an application

SYNOPSIS
	{{.Name}} create [-l|--language] [-t|--template] [-r|--repository]
	            [-c|--confirm] [path]

DESCRIPTION
	Creates a new application.

	  $ {{.Name}} create -l go

	Creates a function in the current directory '.' which is written in the
	language/runtime 'go' common .

	If [path] is provided, the function is initialized at that path, creating
	the path if necessary.

	To complete this command interactively, use --confirm (-c):
	  $ {{.Name}} create -c

	Initialize the current project directly into a dubbo project without using a template
      $ dubboctl create --init

	Available Language Runtimes and Templates:
{{ .Options | indent 2 " " | indent 1 "\t" }}

	To install more language runtimes and their templates see '{{.Name}} repository'.


EXAMPLES
	o Create a Node.js function in the current directory (the default path) which
	  handles http events (the default template).
	  $ {{.Name}} create -l java

	o Create a java common in the directory 'mydubbo'.
	  $ {{.Name}} create -l java mydubbo

	o Create a Main common in ./mydubbo.
	  $ {{.Name}} create -l go -t common mydubbo
		`,
		SuggestFor: []string{"vreate", "creaet", "craete", "new"},
		PreRunE:    common.BindEnv("language", "template", "repository", "confirm", "init"),
		Aliases:    []string{"init"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args, newClient)
		},
	}

	// Flags
	cmd.Flags().StringP("language", "l", "", "Language Runtime (see help text for list) ($DUBBO_LANGUAGE)")
	cmd.Flags().StringP("template", "t", "", "Application template. (see help text for list) ($DUBBO_TEMPLATE)")
	cmd.Flags().StringP("repository", "r", "", "URI to a Git repository containing the specified template ($DUBBO_REPOSITORY)")
	cmd.Flags().BoolP("init", "i", false,
		"Initialize the current project directly into a dubbo project without using a template")

	common.AddConfirmFlag(cmd, false)

	// Help Action
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) { runCreateHelp(cmd, args, newClient) })

	// Tab completion
	if err := cmd.RegisterFlagCompletionFunc("language", newRuntimeCompletionFunc(newClient)); err != nil {
		fmt.Fprintf(os.Stderr, "unable to provide language runtime suggestions: %v\n", err)
	}
	if err := cmd.RegisterFlagCompletionFunc("template", newTemplateCompletionFunc(newClient)); err != nil {
		fmt.Fprintf(os.Stderr, "unable to provide template suggestions: %v\n", err)
	}

	baseCmd.AddCommand(cmd)
}

// Run Create
func runCreate(cmd *cobra.Command, args []string, newClient ClientFactory) (err error) {
	// Config
	// Create a config based on args.  Also uses the newClient to create a
	// temporary client for completing options such as available runtimes.
	cfg, err := newCreateConfig(cmd, args, newClient)
	if err != nil {
		return
	}

	// Client
	// From environment variables, flags, arguments, and user prompts if --confirm
	// (in increasing levels of precedence)
	client, done := newClient(
		dubbo.WithRepository(cfg.Repository))
	defer done()

	// Validate - a deeper validation than that which is performed when
	// instantiating the client with the raw config above.
	if err = cfg.Validate(client); err != nil {
		return
	}

	// Create
	_, err = client.Init(&dubbo.Dubbo{
		Name:     cfg.Name,
		Root:     cfg.Path,
		Runtime:  cfg.Runtime,
		Template: cfg.Template,
		Build: dubbo.BuildSpec{
			CnMirror: dubbo.BooleanWithComment{
				Comment: "Specify `cnMirror: true` to use the mirror in mainland China",
			},
		},
	}, cfg.Init, cmd)
	if err != nil {
		return err
	}

	// Confirm
	fmt.Fprintf(cmd.OutOrStderr(), "Created %v dubbo application in %v\n", cfg.Runtime, cfg.Path)
	return nil
}

type createConfig struct {
	Path       string // Absolute path to function source
	Runtime    string // Language Runtime
	Repository string // Repository URI (overrides builtin and installed)
	Confirm    bool   // Confirm values via an interactive prompt

	// like common is a template
	Template string

	// Name of the function
	Name string

	Init bool
}

// newCreateConfig returns a config populated from the current execution context
// (args, flags and environment variables)
// The client constructor function is used to create a transient client for
// accessing things like the current valid templates list, and uses the
// current value of the config at time of prompting.
func newCreateConfig(cmd *cobra.Command, args []string, newClient ClientFactory) (cfg createConfig, err error) {
	var (
		path         string
		dirName      string
		absolutePath string
	)

	if len(args) >= 1 {
		path = args[0]
	}

	dirName, absolutePath = common.DeriveNameAndAbsolutePathFromPath(path)

	// Config is the final default values based off the execution context.
	// When prompting, these become the defaults presented.
	cfg = createConfig{
		Name:       dirName,
		Path:       absolutePath,
		Repository: viper.GetString("repository"),
		Runtime:    viper.GetString("language"), // users refer to it is language
		Template:   viper.GetString("template"),
		Confirm:    viper.GetBool("confirm"),
		Init:       viper.GetBool("init"),
	}
	// If not in confirm/prompting mode, this cfg structure is complete.
	if !cfg.Confirm {
		return
	}

	// Create a temporary client for use by the following prompts to complete
	// runtime/template suggestions etc
	client, done := newClient()
	defer done()

	// IN confirm mode.  If also in an interactive terminal, run prompts.
	if util.InteractiveTerminal() {
		createdCfg, err := cfg.prompt(client)
		if err != nil {
			return createdCfg, err
		}
		fmt.Println("Command:")
		fmt.Println(singleCommand(cmd, args, createdCfg))
		return createdCfg, nil
	}

	// Confirming, but noninteractive
	// Print out the final values as a confirmation.  Only show Repository or
	// Repositories, not both (repository takes precedence) in order to avoid
	// likely confusion if both are displayed and one is empty.
	// be removed and both displayed.
	fmt.Printf("Path:         %v\n", cfg.Path)
	fmt.Printf("Language:     %v\n", cfg.Runtime) // users refer to it as language
	if cfg.Repository != "" {                     // if an override was provided
		fmt.Printf("Repository:   %v\n", cfg.Repository) // show only the override
	}
	fmt.Printf("Template:     %v\n", cfg.Template)
	return
}

// singleCommand that could be used by the current user to minimally recreate the current state.
func singleCommand(cmd *cobra.Command, args []string, cfg createConfig) string {
	var b strings.Builder
	b.WriteString(cmd.Root().Name())    // process executable
	b.WriteString(" -l " + cfg.Runtime) // language runtime is required
	if cmd.Flags().Lookup("template").Changed {
		b.WriteString(" -t " + cfg.Template)
	}
	if cmd.Flags().Lookup("repository").Changed {
		b.WriteString(" -r " + cfg.Repository)
	}
	if len(args) > 0 {
		b.WriteString(" " + cfg.Path) // optional trailing <path> argument
	}
	return b.String()
}

// Validate the current state of the config, returning any errors.
// Note this is a deeper validation using a client already configured with a
// preliminary config object from flags/config, such that the client instance
// can be used to determine possible values for runtime, templates, etc.  a
// pre-client validation should not be required, as the Client does its own
// validation.
func (c createConfig) Validate(client *dubbo.Client) (err error) {
	dirName, _ := common.DeriveNameAndAbsolutePathFromPath(c.Path)
	if err = util.ValidateApplicationName(dirName); err != nil {
		return
	}

	if c.Runtime == "" {
		return noRuntimeError(client)
	}
	if c.Runtime != "" && c.Repository == "" &&
		!isValidRuntime(client, c.Runtime) {
		return newInvalidRuntimeError(client, c.Runtime)
	}

	if c.Template != "" && c.Repository == "" &&
		!isValidTemplate(client, c.Runtime, c.Template) {
		return newInvalidTemplateError(client, c.Runtime, c.Template)
	}

	return
}

// isValidRuntime determines if the given language runtime is a valid choice.
func isValidRuntime(client *dubbo.Client, runtime string) bool {
	runtimes, err := client.Runtimes()
	if err != nil {
		return false
	}
	for _, v := range runtimes {
		if v == runtime {
			return true
		}
	}
	return false
}

// isValidTemplate determines if the given template is valid for the given
// runtime.
func isValidTemplate(client *dubbo.Client, runtime, template string) bool {
	if !isValidRuntime(client, runtime) {
		return false
	}
	templates, err := client.Templates().List(runtime)
	if err != nil {
		return false
	}
	for _, v := range templates {
		if v == template {
			return true
		}
	}
	return false
}

func noRuntimeError(client *dubbo.Client) error {
	b := strings.Builder{}
	fmt.Fprintln(&b, "Required flag \"language\" not set.")
	fmt.Fprintln(&b, "Available language runtimes are:")
	runtimes, err := client.Runtimes()
	if err != nil {
		return err
	}
	for _, v := range runtimes {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrNoRuntime(errors.New(b.String()))
}

func newInvalidRuntimeError(client *dubbo.Client, runtime string) error {
	b := strings.Builder{}
	fmt.Fprintf(&b, "The language runtime '%v' is not recognized.\n", runtime)
	fmt.Fprintln(&b, "Available language runtimes are:")
	runtimes, err := client.Runtimes()
	if err != nil {
		return err
	}
	for _, v := range runtimes {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrInvalidRuntime(errors.New(b.String()))
}

func newInvalidTemplateError(client *dubbo.Client, runtime, template string) error {
	b := strings.Builder{}
	fmt.Fprintf(&b, "The template '%v' was not found for language runtime '%v'.\n", template, runtime)
	fmt.Fprintln(&b, "Available templates for this language runtime are:")
	templates, err := client.Templates().List(runtime)
	if err != nil {
		return err
	}
	for _, v := range templates {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrInvalidTemplate(errors.New(b.String()))
}

// prompt the user with value of config members, allowing for interactively
// mutating the values. The provided clientFn is used to construct a transient
// client for use during prompt autocompletion/suggestions (such as suggesting
// valid templates)
func (c createConfig) prompt(client *dubbo.Client) (createConfig, error) {
	var qs []*survey.Question

	runtimes, err := client.Runtimes()
	if err != nil {
		return createConfig{}, err
	}

	init := false

	// ask for init
	qs = []*survey.Question{
		{
			Name: "Init",
			Prompt: &survey.Confirm{
				Message: "Create a new dubbo project or directly initialize it into a dubbo project",
				Default: init,
			},
		},
	}
	if err = survey.Ask(qs, &init); err != nil {
		return createConfig{}, err
	}

	// ask for path...
	qs = []*survey.Question{
		{
			Name: "Path",
			Prompt: &survey.Input{
				Message: "Scaffold Path:",
				Default: c.Path,
			},
			Validate: func(val interface{}) error {
				derivedName, _ := common.DeriveNameAndAbsolutePathFromPath(val.(string))
				return util.ValidateApplicationName(derivedName)
			},
			Transform: func(ans interface{}) interface{} {
				_, absolutePath := common.DeriveNameAndAbsolutePathFromPath(ans.(string))
				return absolutePath
			},
		}, {
			Name: "Runtime",
			Prompt: &survey.Select{
				Message: "Language Runtime:",
				Options: runtimes,
				Default: common.SurveySelectDefault(c.Runtime, runtimes),
			},
		},
	}
	if err := survey.Ask(qs, &c); err != nil {
		return c, err
	}

	if !init {
		// Second loop: choose template with autocompletion filtered by chosen runtime
		qs = []*survey.Question{
			{
				Name: "Template",
				Prompt: &survey.Input{
					Message: "Template:",
					Default: c.Template,
					Suggest: func(prefix string) []string {
						suggestions, err := templatesWithPrefix(prefix, c.Runtime, client)
						if err != nil {
							fmt.Fprintf(os.Stderr, "unable to suggest: %v\n", err)
						}
						return suggestions
					},
				},
			},
		}
		if err := survey.Ask(qs, &c); err != nil {
			return c, err
		}
	}
	return c, nil
}

// Tab Completion and Prompt Suggestions Helpers
// ---------------------------------------------

type flagCompletionFunc func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

func newRuntimeCompletionFunc(newClient ClientFactory) flagCompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		_, err := newCreateConfig(cmd, args, newClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating client config for flag completion: %v\n", err)
		}
		client, done := newClient()
		defer done()
		return CompleteRuntimeList(cmd, args, toComplete, client)
	}
}

func newTemplateCompletionFunc(newClient ClientFactory) flagCompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		_, err := newCreateConfig(cmd, args, newClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating client config for flag completion: %v\n", err)
		}
		client, done := newClient()
		defer done()
		return CompleteTemplateList(cmd, args, toComplete, client)
	}
}

// return templates for language runtime whose full name (including repository)
// have the given prefix.
func templatesWithPrefix(prefix, runtime string, client *dubbo.Client) ([]string, error) {
	var (
		suggestions    []string
		templates, err = client.Templates().List(runtime)
	)
	if err != nil {
		return suggestions, err
	}
	for _, template := range templates {
		if strings.HasPrefix(template, prefix) {
			suggestions = append(suggestions, template)
		}
	}
	return suggestions, nil
}

// runCreateHelp prints help for the create command using a template
// and options.
func runCreateHelp(cmd *cobra.Command, args []string, newClient ClientFactory) {
	failSoft := func(err error) {
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "error: help text may be partial: %v\n", err)
		}
	}

	tpl := newHelpTemplate(cmd)

	cfg, err := newCreateConfig(cmd, args, newClient)
	failSoft(err)

	client, done := newClient(
		dubbo.WithRepository(cfg.Repository))
	defer done()

	options, err := RuntimeTemplateOptions(client) // human-friendly
	failSoft(err)

	data := struct {
		Options string
		Name    string
	}{
		Options: options,
		Name:    cmd.Root().Use,
	}

	if err := tpl.Execute(cmd.OutOrStdout(), data); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "unable to display help text: %v\n", err)
	}
}

// newHelpTemplate returns a template for the create command's help text
func newHelpTemplate(cmd *cobra.Command) *template.Template {
	body := cmd.Long + "\n\n" + cmd.UsageString()
	t := template.New("help")
	fm := template.FuncMap{
		"indent": func(i int, c string, v string) string {
			indentation := strings.Repeat(c, i)
			return indentation + strings.Replace(v, "\n", "\n"+indentation, -1)
		},
	}
	t.Funcs(fm)
	return template.Must(t.Parse(body))
}

// RuntimeTemplateOptions is a human-friendly table of valid Language Runtime
// to Template combinations.
// Exported for use in docs.
func RuntimeTemplateOptions(client *dubbo.Client) (string, error) {
	runtimes, err := client.Runtimes()
	if err != nil {
		return "", err
	}
	builder := strings.Builder{}
	writer := tabwriter.NewWriter(&builder, 0, 0, 3, ' ', 0)

	fmt.Fprint(writer, "Language\tTemplate\n")
	fmt.Fprint(writer, "--------\t--------\n")
	for _, r := range runtimes {
		templates, err := client.Templates().List(r)
		// Not all language packs will have templates for
		// all available runtimes. Without this check
		if err != nil && !errors.Is(err, dubbo.ErrTemplateNotFound) {
			return "", err
		}
		for _, t := range templates {
			fmt.Fprintf(writer, "%v\t%v\n", r, t) // write tabbed
		}
	}
	writer.Flush()
	return builder.String(), nil
}
