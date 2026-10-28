## canon explain

Show the record behind a rule or finding, and what enforces it

```
canon explain <rule-or-record> [flags]
```

### Examples

```
  canon explain canon-final-stage-base
  canon explain go-errors-wrap
```

### Options

```
  -h, --help   help for explain
      --json   print the result as JSON
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

