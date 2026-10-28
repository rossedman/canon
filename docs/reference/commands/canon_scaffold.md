## canon scaffold

Generate a project or component that already follows the canon

### Synopsis

Scaffold renders a template from the canon. Kinds: go-cli, go-service,
py-service, tf-module, dockerfile and gha-workflow. Without --write it lists
the files it would create. With --write it creates them (never overwriting)
and checks the result.

```
canon scaffold <kind> <name> [flags]
```

### Examples

```
  canon scaffold go-cli hello
  canon scaffold go-cli hello --var module=github.com/me/hello --write
  canon scaffold dockerfile api --dest services/api --write
```

### Options

```
      --dest string          directory to write into, relative to --root (default ".")
  -h, --help                 help for scaffold
      --json                 print the result as JSON
      --var stringToString   template variable as key=value (repeatable) (default [])
      --write                write the files instead of listing them
```

### Options inherited from parent commands

```
      --canon-dir string   load the canon from this directory instead of the embedded one
      --config string      config file (default: first of $XDG_CONFIG_HOME/canon/config.yaml, ~/.canon.yaml, ./.canon.yaml)
      --log-level string   log level: debug, info, warn or error (default "warn")
      --root string        repository root that paths are relative to (default ".")
```

### SEE ALSO

* [canon](canon.md)	 - Engineering standards as tools an agent can call

