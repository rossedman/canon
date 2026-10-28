## canon review

Review a change: findings on changed files plus concerns linters miss

### Synopsis

Review checks the files a change touches and classifies the change for
concerns no linter reports: new dependencies, new Dockerfiles, Terraform
provider bumps, exported API changes and new packages without tests. The diff
is computed against --base, or read from --diff-file (- for stdin).

```
canon review [flags]
```

### Examples

```
  canon review
  git diff main | canon review --diff-file -
```

### Options

```
      --base string           branch to compare the working tree against (default "origin/main")
      --diff-file string      read a unified diff from this file, or - for stdin
      --fail-on string        exit 1 when a finding is at least this severe: error, warning, suggestion or none (default "none")
  -h, --help                  help for review
      --json                  print the result as JSON
      --min-severity string   lowest severity to report (default "warning")
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

