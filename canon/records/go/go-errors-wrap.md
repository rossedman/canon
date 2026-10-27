---
id: go-errors-wrap
title: Wrap errors with %w and context; never log and return
summary: Wrapped errors keep the cause inspectable and say where it failed, and one owner handles each error.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [errors, reliability, debugging]
rules: [golangci-errors]
strength: must
updated: 2026-10-27
---

## Context
An error that crosses a package boundary without context ("open: no such file")
forces whoever reads the log to guess which call failed. Logging an error and
then returning it means the same failure shows up two or three times, at
different levels, with no clear owner.

## Decision
Return errors wrapped with `fmt.Errorf("doing x: %w", err)` when they cross a
package or a meaningful step. Compare with `errors.Is` and `errors.As`, never
with `==` or string matching. An error is either handled (logged, retried,
turned into a response) or returned, never both.

## Consequences
Error messages read as a chain of steps from the top caller down to the cause.
`wrapcheck` and `errorlint` enforce the mechanics; reviewers still check that
the added context says something useful.

## Exceptions
Sentinel errors returned unchanged from the package that defines them.

## Examples
```go
// good
if err := os.WriteFile(path, data, 0o600); err != nil {
	return fmt.Errorf("write config %s: %w", path, err)
}

// bad: logged and returned, and the cause is flattened to a string
if err != nil {
	log.Printf("write failed: %v", err)
	return fmt.Errorf("write failed: %v", err)
}
```
