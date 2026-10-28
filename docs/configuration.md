# Configuration

Every option resolves in one order, so you can set a default once and override it anywhere.

## Precedence

Highest wins:

1. **Flag**: `--min-severity error`
2. **Environment variable**: `CANON_CHECK_MIN_SEVERITY=error`
3. **Config file**
4. **Default**

Environment variables use the `CANON_` prefix. Dashes and dots become
underscores, and command options are named `<command>.<flag>`, so
`check.min-severity` is `CANON_CHECK_MIN_SEVERITY`, `hook.min-severity` is
`CANON_HOOK_MIN_SEVERITY`, and the global `--root` is `CANON_ROOT`.

## Config file

canon uses the first of these that exists:

1. `$XDG_CONFIG_HOME/canon/config.yaml` (the OS user config directory, so
   `~/.config/canon/config.yaml` on most Linux systems)
2. `~/.canon.yaml`
3. `./.canon.yaml` in the current directory

Pass `--config <file>` to use a specific file instead. If that file cannot be
read, canon prints a warning to stderr and carries on with flags,
environment and defaults.

Keys match the flag names. Command options are nested under the command:

```yaml
# ~/.config/canon/config.yaml
log-level: info
check:
  min-severity: warning
  fail-on: error
  scope: [go, containers]
hook:
  min-severity: error
serve:
  http: ":8080"
```

## Global flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--config` | none | Config file to read |
| `--canon-dir` | embedded | Load records, rules and profiles from this directory instead of the canon built into the binary |
| `--root` | `.` | Repository root that paths are relative to |
| `--log-level` | `warn` | `debug`, `info`, `warn` or `error` |

## Use your own canon

The binary embeds a default canon. Point `--canon-dir` at a directory with the
same layout to use your own, and check it with `canon lint --canon-dir ./canon`:

```text
canon/
  principles.md
  records/<scope>/*.md
  rules/rules.yaml
  profiles/*.yaml
  templates/<kind>/
```

See the [command reference](reference/commands/canon.md) for every flag.
