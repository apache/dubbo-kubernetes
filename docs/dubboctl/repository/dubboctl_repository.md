## dubboctl repository

Manage set of installed repositories.

### Synopsis

	dubboctl repo [-c|--confirm]
	dubboctl repo list [-r|--repositories] [-c|--confirm]
	dubboctl repo add <name> <url>[-r|--repositories] [-c|--confirm]
	dubboctl repo rename <old> <new> [-r|--repositories] [-c|--confirm]
	dubboctl repo remove <name> [-r|--repositories] [-c|--confirm]

### Description

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

### COMMANDS

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

### EXAMPLES

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

Flags:
-c, --confirm Prompt to confirm options interactively ($DUBBO_CONFIRM)
-h, --help help for repository

### SEE ALSO

* [dubboctl](../reference/dubboctl.md) - Management tool for dubbo-kubernetes
* [dubboctl repository add](dubboctl_repository_add.md)
* [dubboctl repository list](dubboctl_repository_add.md)
* [dubboctl repository remove](dubboctl_repository_remove.md)
* [dubboctl repository rename](dubboctl_repository_rename.md)
