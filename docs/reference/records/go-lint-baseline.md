# Every Go repo runs the golangci-lint baseline

One shared linter set catches the bugs reviewers miss and stops style debates before they start.

| | |
| --- | --- |
| ID | `go-lint-baseline` |
| Strength | must (error) |
| Status | accepted |
| Scope | go |
| Applies to | `**/*.go` |
| Tags | lint, quality, security |
| Updated | 2026-10-27 |

## Context

Unchecked errors, shadowed variables and unsafe file permissions are easy to
miss in review and cheap to catch with a linter. Repos that pick their own
linters drift apart, and moving between them means relearning what "clean" is.

## Decision

Run golangci-lint v2 with `govet`, `staticcheck`, `errcheck`, `revive`, `gosec`,
`misspell`, `ineffassign` and `unused` enabled, from the config shipped in this
canon. Format with `gofmt`. CI fails on any finding.

## Consequences

Lint is a gate, not advice: a finding is fixed or suppressed inline with a
`//nolint:<linter> // reason` comment that a reviewer can question.

## Exceptions

Generated files (`// Code generated ... DO NOT EDIT.`) are excluded.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `golangci-baseline` | golangci-lint | The shared golangci-lint baseline linters |

Read it from an agent with `canon explain go-lint-baseline`, or the `canon://records/go-lint-baseline` resource.
