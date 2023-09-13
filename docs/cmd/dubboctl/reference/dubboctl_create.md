## dubboctl create

Create an application

### Synopsis

	dubboctl create [-l|--language] [-t|--template] [-r|--repository]
	            [-c|--confirm] [path]

### DESCRIPTION

    Creates a new function project.

	  $ dubboctl create -l go

	Creates a function in the current directory '.' which is written in the
	language/runtime 'go' common .

	If [path] is provided, the function is initialized at that path, creating
	the path if necessary.

	To complete this command interactively, use --confirm (-c):
	  $ dubboctl create -c

	To install more language runtimes and their templates see 'dubboctl repository'.

### EXAMPLES

    o Create a Node.js function in the current directory (the default path) which
     handles http events (the default template).
      $ dubboctl create -l java

	o Create a java common in the directory 'myfunc'.
	  $ dubboctl create -l java myfunc

	o Create a Main common in ./myfunc.
	  $ dubboctl create -l go -t common myfunc

-t is followed by the warehouse name + application name. If it is the template provided by dubboctl by default, then there is only the application name, which is common. Otherwise, for example, if the user's warehouse is named ruiyi and the application is named mesh, -t should be followed by ruiyi/mesh.

### Usage:

    dubboctl create [flags]

### Aliases:

    create, init

### Flags:

    -c, --confirm             Prompt to confirm options interactively ($DUBBO_CONFIRM)
    -h, --help                help for create
    -i, --init                Initialize the current project directly into a dubbo project without using a template
    -l, --language string     Language Runtime (see help text for list) ($DUBBO_LANGUAGE)
    -r, --repository string   URI to a Git repository containing the specified template ($DUBBO_REPOSITORY) 
    -t, --template string     Application template. (see help text for list) ($DUBBO_TEMPLATE)

### SEE ALSO

* [dubboctl](dubboctl.md) - Management tool for dubbo-kubernetes