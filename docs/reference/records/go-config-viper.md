# CLI configuration resolves flag, then env, then file

One precedence order everywhere means nobody has to guess which setting won.

| | |
| --- | --- |
| ID | `go-config-viper` |
| Strength | should (warning) |
| Status | accepted |
| Scope | go |
| Applies to | `cmd/*.go` |
| Tags | cli, config, viper |
| Updated | 2026-10-27 |

## Context

Users set options in whatever way suits the moment: a flag while testing, an
environment variable in CI, a config file on their laptop. Tools that honor
only some of these, or in a surprising order, cause "it works on my machine".

## Decision

Use viper in `cmd/root.go`. Search for the config file in
`$XDG_CONFIG_HOME/<name>`, the home directory and the current directory. Set
`viper.SetEnvPrefix` to the binary name, call `viper.AutomaticEnv()` with a
`-` to `_` key replacer, and bind every flag with `viper.BindPFlag`. Precedence
is flag, then environment, then file, then default.

## Consequences

Every flag is also an environment variable (`--min-severity` is
`<NAME>_MIN_SEVERITY`), which the docs state once.

## Enforced by

No automated rule yet; reviewers and `canon decide` carry this one.

Read it from an agent with `canon explain go-config-viper`, or the `canon://records/go-config-viper` resource.
