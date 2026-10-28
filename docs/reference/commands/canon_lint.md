## canon lint

Validate the canon: records, rules, profiles and templates

### Synopsis

Lint checks every record has its required sections and valid frontmatter,
every rule ID resolves in both directions, supersedes has no cycles, profiles
name real scopes and records, and every template renders.

```
canon lint [flags]
```

### Examples

```
  canon lint
  canon lint --canon-dir ./canon
```

### Options

```
  -h, --help   help for lint
      --json   print problems as JSON
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

