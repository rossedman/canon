# Critical paths have benchmarks, and input parsers have fuzz targets

Parsers see inputs nobody wrote a test for, and hot paths regress silently without a benchmark.

| | |
| --- | --- |
| ID | `go-fuzz-critical-paths` |
| Strength | should (warning) |
| Status | accepted |
| Scope | go |
| Applies to | `**/*.go` |
| Tags | testing, performance, security |
| Updated | 2026-10-27 |

## Context

Functions that parse external input, transform data or sit in hot loops are
where crashes and slowdowns come from.

## Decision

List those functions under "Critical paths" in `CONTRIBUTING.md`. Each has a
`BenchmarkXxx` using `b.ReportAllocs()` and `b.Loop()`. Each one that accepts
external input has a `FuzzXxx` with a seed corpus from real examples, asserting
properties (no panic, round trips, invariants). CI smoke-runs both.

## Consequences

Failing fuzz inputs are committed to `testdata/fuzz/` and become regression tests.

## Enforced by

No automated rule yet; reviewers and `canon decide` carry this one.

Read it from an agent with `canon explain go-fuzz-critical-paths`, or the `canon://records/go-fuzz-critical-paths` resource.
