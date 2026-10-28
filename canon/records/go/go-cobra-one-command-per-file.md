---
id: go-cobra-one-command-per-file
title: One cobra command per file in cmd/
summary: A command's flags, help and wiring live in one file named after it, so they are easy to find and review.
status: accepted
scope: [go]
applies_to:
  files: ["cmd/*.go"]
tags: [cli, cobra, layout]
rules: [canon-one-command-per-file]
strength: must
updated: 2026-10-27
---

## Context
CLIs that define several commands in one file grow into a tangle of shared
flag variables and `init()` functions that register each other's commands.

## Decision
Each non-test file in `cmd/` defines exactly one `*cobra.Command` and registers
it in its own `init()`. The file is named after the command: `cmd/serve.go`,
`cmd/config_get.go` for `config get`. Commands stay thin and call into
`internal/`.

## Consequences
Adding a command is adding a file. `cmd/` holds wiring only, so logic stays
testable without cobra.
