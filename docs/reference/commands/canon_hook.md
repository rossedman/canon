## canon hook

Run as a Claude Code or git hook on the files just changed

### Synopsis

As a Claude Code PostToolUse hook, canon hook reads the hook payload on
stdin, checks the file the agent edited, and exits 2 with the findings on
stderr so the agent fixes them. Given paths instead, it checks those, which
suits git pre-commit hooks.

```
canon hook [paths...] [flags]
```

### Examples

```
  # .claude/settings.json
  {"hooks": {"PostToolUse": [{"matcher": "Edit|Write|MultiEdit",
    "hooks": [{"type": "command", "command": "canon hook"}]}]}}

  # .git/hooks/pre-commit
  canon hook $(git diff --cached --name-only --diff-filter=ACM)
```

### Options

```
  -h, --help                  help for hook
      --min-severity string   lowest severity that blocks: error, warning or suggestion (default "error")
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

