## How to generate a new Kuma policy

Use the tool:

```shell
go run ./tools/policy-gen/bootstrap/... --name CaseNameOfPolicy
```

The output of the tool will tell you where the important files are!

## Add plugin name to the configuration

To enable policy you need to adjust configuration of two places:
* Remove `+dubbo:policy:skip_registration=true` from your policy schema.
* Add import in `pkg/plugins/policies/imports.go`
* `pkg/plugins/policies/core/ordered/ordered.go`. Plugins name is equals to `DubboctlArg` in file `zz_generated.resource.go`. It's important to place the plugin in the correct place because the order of executions is important.

