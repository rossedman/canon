## canon decide

Answer a "how do we do X" question from decision records

```
canon decide <question...> [flags]
```

### Examples

```
  canon decide how do we handle errors in Go
  canon decide "which base image for a Go service?" --scope containers
```

### Options

```
  -h, --help            help for decide
      --json            print the result as JSON
      --limit int       most records to return (default 3)
      --scope strings   limit to scopes
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

