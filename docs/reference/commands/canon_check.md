## canon check

Check files against the canon and report findings

### Synopsis

Check runs every rule in force for this repo's profile against the given
files or directories (default: the whole repo). Each finding names its rule,
the record it enforces, and why. Exits 1 when a finding reaches --fail-on.

```
canon check [paths...] [flags]
```

### Examples

```
  canon check
  canon check --changed-only --sarif > canon.sarif
  canon check cmd/ --scope go --min-severity suggestion
```

### Options

```
      --base string           base branch for --changed-only (default "origin/main")
      --changed-only          check only files changed since --base
      --fail-on string        exit 1 when a finding is at least this severe: error, warning, suggestion or none (default "error")
  -h, --help                  help for check
      --json                  print the result as JSON
      --min-severity string   lowest severity to report: error, warning or suggestion (default "warning")
      --profile string        use this profile instead of detecting one
      --sarif                 print findings as SARIF 2.1.0
      --scope strings         limit to scopes: go, python, containers, terraform, yaml, ci, oss
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

