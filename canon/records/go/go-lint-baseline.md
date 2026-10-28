---
id: go-lint-baseline
title: Every Go repo runs the golangci-lint baseline
summary: One shared linter set catches the bugs reviewers miss and stops style debates before they start.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [lint, quality, security]
rules: [golangci-baseline]
strength: must
updated: 2026-10-27
---

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
