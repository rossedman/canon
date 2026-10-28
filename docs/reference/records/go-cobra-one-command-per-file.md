# One cobra command per file in cmd/

A command's flags, help and wiring live in one file named after it, so they are easy to find and review.

| | |
| --- | --- |
| ID | `go-cobra-one-command-per-file` |
| Strength | must (error) |
| Status | accepted |
| Scope | go |
| Applies to | `cmd/*.go` |
| Tags | cli, cobra, layout |
| Updated | 2026-10-27 |

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

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-one-command-per-file` | canon | Each cmd/*.go file defines exactly one cobra.Command |

Read it from an agent with `canon explain go-cobra-one-command-per-file`, or the `canon://records/go-cobra-one-command-per-file` resource.
