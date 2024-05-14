## dubboctl profile list

List all existing profiles specification

### Synopsis

Displays which profiles are in the profiles directory and displays the content of the specified profile.
Typical use cases are:

```sh
dubboctl profile list

dubboctl profile list default
```

| parameter  | shorthand | describe                                        | Example                                           | required |
|------------|-----------|-------------------------------------------------|---------------------------------------------------|----------|
| --profiles |           | Specify the directory where profiles are stored | dubboctl profile list --profiles path/to/profiles | No       |

### SEE ALSO

* [dubboctl profile](dubboctl_profile.md) - Commands help user to list and describe profiles
