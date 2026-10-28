---
id: python-uv
title: Python projects are managed with uv and a committed lock file
summary: A committed uv.lock makes installs reproducible on every machine and in CI.
status: accepted
scope: [python]
applies_to:
  files: ["**/pyproject.toml"]
tags: [python, dependencies, reproducibility]
rules: [canon-uv-lock]
strength: should
updated: 2026-10-27
---

## Context
`pip install -r requirements.txt` without hashes resolves differently over
time, so a green build today says little about tomorrow's.

## Decision
Declare dependencies in `pyproject.toml`, manage them with `uv`, and commit
`uv.lock`. CI installs with `uv sync --locked`.

## Consequences
Dependency changes show up as lock file diffs in review.
