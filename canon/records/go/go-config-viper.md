---
id: go-config-viper
title: CLI configuration resolves flag, then env, then file
summary: One precedence order everywhere means nobody has to guess which setting won.
status: accepted
scope: [go]
applies_to:
  files: ["cmd/*.go"]
tags: [cli, config, viper]
rules: []
strength: should
updated: 2026-10-27
---

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
