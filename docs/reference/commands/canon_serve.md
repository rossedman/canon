## canon serve

Run the MCP server over stdio, or streamable HTTP with --http

```
canon serve [flags]
```

### Examples

```
  claude mcp add canon -- canon serve
  canon serve --http :8080
```

### Options

```
  -h, --help          help for serve
      --http string   listen address for streamable HTTP (e.g. :8080); stdio when empty
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

