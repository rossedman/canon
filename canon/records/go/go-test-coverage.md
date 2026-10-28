---
id: go-test-coverage
title: Hand-written Go code keeps 90% test coverage
summary: A high, enforced bar keeps untested paths visible, and excluding generated code keeps the number honest.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [testing, quality, ci]
rules: []
strength: must
updated: 2026-10-27
---

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
