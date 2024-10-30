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
)

import (
	"github.com/AlecAivazis/survey/v2"

	"github.com/ory/viper"

	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/operator/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/operator/util"
)

// command constructors
// --------------------
func AddRepository(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Short:   "Manage installed template repositories",
		Use:     "repository",
		Aliases: []string{"repo", "repositories"},
		Long: `
NAME
	dubboctl - Manage set of installed repositories.

SYNOPSIS
	dubboctl repo [-c|--confirm]
	dubboctl repo list [-r|--repositories] [-c|--confirm]
	dubboctl repo add <name> <url>[-r|--repositories] [-c|--confirm]
	dubboctl repo rename <old> <new> [-r|--repositories] [-c|--confirm]
	dubboctl repo remove <name> [-r|--repositories] [-c|--confirm]

DESCRIPTION
	Manage template repositories installed on disk at either the default location
	(~/.config/dubbo/repositories) or the location specified by the --repository
	flag.  Once added, a template from the repository can be used when creating
	a new Dubbo.

	Interactive Prompts:
	To complete these commands interactively, pass the --confirm (-c) flag to
	the 'repository' command, or any of the inidivual subcommands.

	The Default Repository:
	The default repository is not stored on disk, but embedded in the binary and
	can be used without explicitly specifying the name.  The default repository
	is always listed first, and is assumed when creating a new function without
	specifying a repository name prefix.
	For example, to create a new one using the 'common' template from the
	default repository.
		$ dubboctl create -l go -t common

	The Repository Flag:
	Installing repositories locally is optional.  To use a template from a remote
	repository directly, it is possible to use the --repository flag on create.
	This leaves the local disk untouched.  For example, To create a scaffold using
	the dubboctl-samples http template without installing the template
	repository locally, use the --repository (-r) flag on create:
		$ dubboctl create -l go \
			--template http \
			--repository https://github.com/sjmshsh/dubboctl-samples

	Alternative Repositories Location:
	Repositories are stored on disk in ~/.config/dubbo/repositories by default.
	This location can be altered by setting the DUBBO_REPOSITORIES_PATH
	environment variable.


COMMANDS

	With no arguments, this help text is shown.  To manage repositories with
	an interactive prompt, use the use the --confirm (-c) flag.
	  $ dubboctl repository -c

	add
	  Add a new repository to the installed set.
	    $ dubboctl repository add <name> <URL>

	  For Example, to add the ruiyi Project repository:
	    $ dubboctl repository add ruiyi https://github.com/sjmshsh/dubboctl-samples

	  Once added, a function can be created with templates from the new repository
	  by prefixing the template name with the repository.  For example, to create
	  a new function using the dubbogo template:
	    $ dubboctl create -l go -t ruiyi/dubbogo

	list
	  List all available repositories, including the installed default
	  repository.  Repositories available are listed by name. 

	rename
	  Rename a previously installed repository from <old> to <new>. Only installed
	  repositories can be renamed.
	    $ dubboctl repository rename <name> <new name>

	remove
	  Remove a repository by name.  Removes the repository from local storage
	  entirely.  When in confirm mode (--confirm) it will confirm before
	  deletion, but in regular mode this is done immediately, so please use
	  caution, especially when using an altered repositories location
	  (via the DUBBO_REPOSITORIES_PATH environment variable).
	    $ dubboctl repository remove <name>

EXAMPLES
	o Run in confirmation mode (interactive prompts) using the --confirm flag
	  $ dubboctl repository -c

	o Add a repository and create a new function using a template from it:
	  $ dubboctl repository add ruiyi https://github.com/sjmshsh/dubboctl-samples
	  $ dubboctl repository list
	  default
	  functastic
	  $ dubboctl create -l go -t ruiyi/dubbogo
	  ...

	o Add a repository specifying the branch to use (dubboctl):
	  $ dubboctl repository add ruiyi https://github.com/sjmshsh/dubboctl-samples#dubboctl
	  $ dubboctl repository list
	  default
	  ruiyi
	  $ dubboctl create -l go -t http
	  ...

	o List all repositories including the URL from which remotes were installed
	  $ dubboctl repository list -v
	  default
	  master	https://github.com/sjmshsh/dubboctl-samples#master

	o Rename an installed repository
	  $ dubboctl repository list
	  default
	  ruiyi
	  $ dubboctl repository rename ruiyi dubboTest
	  $ dubboctl repository list
	  default
	  dubboTest

	o Remove an installed repository
	  $ dubboctl repository list
	  default
	  dubboTest
	  $ dubboctl repository remove dubboTest
	  $ dubboctl repository list
	  default
`,
		SuggestFor: []string{"repositories", "repos", "template", "templates", "pack", "packs"},
		PreRunE:    common.BindEnv("confirm"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepository(cmd, args, newClient)
		},
	}

	common.AddConfirmFlag(cmd, false)

	cmd.AddCommand(NewRepositoryListCmd(newClient))
	cmd.AddCommand(NewRepositoryAddCmd(newClient))
	cmd.AddCommand(NewRepositoryRenameCmd(newClient))
	cmd.AddCommand(NewRepositoryRemoveCmd(newClient))

	baseCmd.AddCommand(cmd)
}

func NewRepositoryListCmd(newClient ClientFactory) *cobra.Command {
	cmds := &cobra.Command{
		Short:   "List repositories",
		Use:     "list",
		Aliases: []string{"ls"},
		PreRunE: common.BindEnv("confirm"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepositoryList(cmd, args, newClient)
		},
	}

	common.AddConfirmFlag(cmds, false)
	return cmds
}

func NewRepositoryAddCmd(newClient ClientFactory) *cobra.Command {
	cmds := &cobra.Command{
		Short:      "Add a repository",
		Use:        "add <name> <url>",
		SuggestFor: []string{"ad", "install"},
		PreRunE:    common.BindEnv("confirm"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepositoryAdd(cmd, args, newClient)
		},
	}

	common.AddConfirmFlag(cmds, false)
	return cmds
}

func NewRepositoryRenameCmd(newClient ClientFactory) *cobra.Command {
	cmds := &cobra.Command{
		Short:   "Rename a repository",
		Use:     "rename <old> <new>",
		Aliases: []string{"mv"},
		PreRunE: common.BindEnv("confirm"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepositoryRename(cmd, args, newClient)
		},
	}

	common.AddConfirmFlag(cmds, false)
	return cmds
}

func NewRepositoryRemoveCmd(newClient ClientFactory) *cobra.Command {
	cmds := &cobra.Command{
		Short:      "Remove a repository",
		Use:        "remove <name>",
		Aliases:    []string{"rm"},
		SuggestFor: []string{"delete", "del"},
		PreRunE:    common.BindEnv("confirm"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepositoryRemove(cmd, args, newClient)
		},
	}

	common.AddConfirmFlag(cmds, false)
	return cmds
}

// command implementations
// -----------------------

// Run
// (list by default or interactive with -c|--confirm)
func runRepository(cmd *cobra.Command, args []string, newClient ClientFactory) (err error) {
	cfg, err := newRepositoryConfig(args)
	if err != nil {
		return
	}

	// If in noninteractive, normal mode the help text is shown
	if !cfg.Confirm {
		return cmd.Help()
	}

	// If in interactive mode, the user chan choose which subcommand to invoke
	// Prompt for action to perform
	question := &survey.Question{
		Name: "Action",
		Prompt: &survey.Select{
			Message: "Operation to perform:",
			Options: []string{"list", "add", "rename", "remove"},
			Default: "list",
		},
	}
	answer := struct{ Action string }{}
	if err = survey.Ask([]*survey.Question{question}, &answer); err != nil {
		return
	}

	// Run the command indicated
	switch answer.Action {
	case "list":
		return runRepositoryList(cmd, args, newClient)
	case "add":
		return runRepositoryAdd(cmd, args, newClient)
	case "rename":
		return runRepositoryRename(cmd, args, newClient)
	case "remove":
		return runRepositoryRemove(cmd, args, newClient)
	}
	return fmt.Errorf("invalid action '%v'", answer.Action) // Unreachable
}

// List
func runRepositoryList(_ *cobra.Command, args []string, newClient ClientFactory) (err error) {
	_, err = newRepositoryConfig(args)
	if err != nil {
		return
	}

	client, done := newClient()
	defer done()

	// List all repositories given a client instantiated about config.
	rr, err := client.Repositories().All()
	if err != nil {
		return
	}

	// Print repository names, or name plus url
	// This follows the format of `git remote`, as it is likely familiar.
	for _, r := range rr {
		fmt.Fprintln(os.Stdout, r.Name+"\t"+r.URL())
	}
	return
}

// Add
func runRepositoryAdd(_ *cobra.Command, args []string, newClient ClientFactory) (err error) {
	// Supports both composable, discrete CLI commands or prompt-based "config"
	// by setting the argument values (name and ulr) to value of positional args,
	// but only requires them if not prompting.  If prompting, those values
	// become the prompt defaults.

	cfg, err := newRepositoryConfig(args)
	if err != nil {
		return
	}

	// Adding a repository requires there be a config path structure on disk
	if err = util.CreatePaths(); err != nil {
		return
	}

	// Create a client instance which utilizes the given repositories path.
	// Note that this MAY not be in the config structure if the environment
	// variable to override said path was provided explicitly.
	// be created in XDG_CONFIG_HOME/dubbo even if the repo path environment
	// was set to some other location on disk.
	client, done := newClient()
	defer done()

	// Preconditions
	// If not confirming/prompting, assert the args were both provided.
	if len(args) != 2 && !cfg.Confirm {
		return fmt.Errorf("usage: dubbo repository add <name> <url>")
	}

	// Extract Params
	// Populate a struct with the arguments (if provided)
	params := struct {
		Name string
		URL  string
	}{}
	if len(args) > 0 {
		params.Name = args[0]
	}
	if len(args) > 1 {
		params.URL = args[1]
	}

	// Prompt/Confirm
	// If confirming/prompting, interactively populate the params from the user
	// (using the current values as defaults)
	//
	// If terminal not interactive, effective values are echoed.
	//
	// Note that empty values can be passed to the final client's Add method if:
	//   Argument(s) not provided
	//   Confirming (-c|--confirm)
	//   Is a noninteractive terminal
	// This is an expected case.  The empty value will be echoed to stdout, the
	// API will be invoked, and a helpful error message will indicate that the
	// request is missing required parameters.
	if cfg.Confirm && util.InteractiveTerminal() {
		questions := []*survey.Question{
			{
				Name:     "Name",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Name for the new repository:",
					Default: params.Name,
				},
			}, {
				Name:     "URL",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "URL of the new repository:",
					Default: params.URL,
				},
			},
		}
		if err = survey.Ask(questions, &params); err != nil {
			return
			// not checking for terminal.InterruptError because failure to complete,
			// for whatever reason, should exit the program non-zero.
		}
	} else if cfg.Confirm {
		fmt.Fprintf(os.Stdout, "Name: %v\n", params.Name)
		fmt.Fprintf(os.Stdout, "URL:  %v\n", params.URL)
	}

	// Add repository
	var n string
	if n, err = client.Repositories().Add(params.Name, params.URL); err != nil {
		return
	}
	fmt.Fprintf(os.Stdout, "Repository added: %s\n", n)
	return
}

// Rename
func runRepositoryRename(_ *cobra.Command, args []string, newClient ClientFactory) (err error) {
	cfg, err := newRepositoryConfig(args)
	if err != nil {
		return
	}
	client, done := newClient()
	defer done()

	// Preconditions
	if len(args) != 2 && !cfg.Confirm {
		return fmt.Errorf("usage: dubbo repository rename <old> <new>")
	}

	// Extract Params
	params := struct {
		Old string
		New string
	}{}
	if len(args) > 0 {
		params.Old = args[0]
	}
	if len(args) > 1 {
		params.New = args[1]
	}

	// Repositories installed according to the client
	// (does not include the builtin default)
	repositories, err := installedRepositories(client)
	if err != nil {
		return
	}

	// Confirm (interactive prompt mode)
	if cfg.Confirm && util.InteractiveTerminal() {
		questions := []*survey.Question{
			{
				Name:     "Old",
				Validate: survey.Required,
				Prompt: &survey.Select{
					Message: "Repository to rename:",
					Options: repositories,
				},
			}, {
				Name:     "New",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "New name:",
					Default: params.New,
				},
			},
		}
		if err = survey.Ask(questions, &params); err != nil {
			return // for any reason, including interrupt, is an nonzero exit
		}
	} else if cfg.Confirm {
		fmt.Fprintf(os.Stdout, "Old: %v\n", params.Old)
		fmt.Fprintf(os.Stdout, "New: %v\n", params.New)
	}

	// Rename the repository
	if err = client.Repositories().Rename(params.Old, params.New); err != nil {
		return
	}
	fmt.Fprintln(os.Stdout, "Repository renamed")
	return
}

// Remove
func runRepositoryRemove(_ *cobra.Command, args []string, newClient ClientFactory) (err error) {
	cfg, err := newRepositoryConfig(args)
	if err != nil {
		return
	}
	client, done := newClient()
	defer done()

	// Preconditions
	if len(args) != 1 && !cfg.Confirm {
		return fmt.Errorf("usage: dubbo repository remove <name>")
	}

	// Extract param(s)
	params := struct {
		Name string
		Sure bool
	}{}
	if len(args) > 0 {
		params.Name = args[0]
	}
	// "Are you sure" confirmation flag
	// (not using name 'Confirm' to avoid confusion with cfg.Confirm)
	// defaults to Yes.  This is debatable, but I don't want to choose the repo
	// to remove and then have to see a prompt and then have to hit 'y'.  Just
	// prompting once to make sure, which requires another press of enter, seems
	// sufficient.
	params.Sure = true

	// Repositories installed according to the client
	// (does not include the builtin default)
	repositories, err := installedRepositories(client)
	if err != nil {
		return
	}

	if len(repositories) == 0 {
		return errors.New("no repositories installed. use 'add' to install")
	}

	// Confirm (interactive prompt mode)
	if cfg.Confirm && util.InteractiveTerminal() {
		questions := []*survey.Question{
			{
				Name:     "Name",
				Validate: survey.Required,
				Prompt: &survey.Select{
					Message: "Repository to remove:",
					Options: repositories,
				},
			}, {
				Name: "Sure",
				Prompt: &survey.Confirm{
					Message: "This will remove the repository from local disk. Are you sure?",
					Default: params.Sure,
				},
			},
		}
		if err = survey.Ask(questions, &params); err != nil {
			return // for any reason, including interrupt, is a nonzero exit
		}
	} else if cfg.Confirm {
		fmt.Fprintf(os.Stdout, "Repository: %v\n", params.Name)
	}

	// Cancel if they got cold feet.
	if !params.Sure {
		// While an argument could be made to the contrary, I believe it is
		// important than an abort by the user, either by answering no to the
		// confirmation or by an os interrupt such as ^C be considered an error,
		// and thus a non-zero program exit.  This is because a user may have
		// chained the command, and an abort (for whatever reason) should cancel
		// the whole chain.  For example, given the command:
		//    dubbo repo rm -cv && doSomethingOnSuccess
		// The trailing command 'doSomethingOnSuccess' should not be evaluated if
		// the first, `dubbo repo rm`, does not exit 0.
		fmt.Fprintln(os.Stdout, "Repository remove canceled")
		return fmt.Errorf("repository removal canceled")
	}

	// Remove the repository
	if err = client.Repositories().Remove(params.Name); err != nil {
		return
	}
	fmt.Fprintln(os.Stdout, "Repository removed")
	return
}

// Installed repositories
// All repositories which have been installed (does not include builtin)
func installedRepositories(client *dubbo.Client) ([]string, error) {
	// Client API contract stipulates the list always lists the defeault builtin
	// repo, and always lists it at index 0
	repositories, err := client.Repositories().List()
	if err != nil {
		return []string{}, err
	}
	return repositories[1:], nil
}

// client config
// -------------

// repositoryConfig used for instantiating a fn.Client
type repositoryConfig struct {
	Confirm bool // Enables interactive confirmation/prompting mode
}

// newRepositoryConfig creates a configuration suitable for use instantiating the
// fn Client. Note that parameters for the individual commands (add, remove etc)
// are collected separately in their requisite run functions.
func newRepositoryConfig(args []string) (cfg repositoryConfig, err error) {
	// initial config is populated based on flags, which are themselves
	// first populated by static defaults, then environment variables,
	// finally command flags.
	cfg = repositoryConfig{
		Confirm: viper.GetBool("confirm"),
	}
	return
}
