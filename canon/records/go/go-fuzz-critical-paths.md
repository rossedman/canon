---
id: go-fuzz-critical-paths
title: Critical paths have benchmarks, and input parsers have fuzz targets
summary: Parsers see inputs nobody wrote a test for, and hot paths regress silently without a benchmark.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [testing, performance, security]
rules: []
strength: should
updated: 2026-10-27
---

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
