# Hand-written Go code keeps 90% test coverage

A high, enforced bar keeps untested paths visible, and excluding generated code keeps the number honest.

| | |
| --- | --- |
| ID | `go-test-coverage` |
| Strength | must (error) |
| Status | accepted |
| Scope | go |
| Applies to | `**/*.go` |
| Tags | testing, quality, ci |
| Updated | 2026-10-27 |

## Context

Coverage that only ever drifts down ends up meaning nothing. Counting generated
code or example programs can hide or fake the real number.

## Decision

CI runs `go test -race -coverprofile` and fails below 90% total coverage,
measured on hand-written code only. Tests are table-driven and use
`t.Parallel()` where safe. Commands are tested through `cmd.SetArgs` with
captured output.

## Consequences

Coverage is a floor, not a target: don't test trivial getters to move it.

## Enforced by

No automated rule yet; reviewers and `canon decide` carry this one.

Read it from an agent with `canon explain go-test-coverage`, or the `canon://records/go-test-coverage` resource.
