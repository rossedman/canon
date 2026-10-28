# Python projects are managed with uv and a committed lock file

A committed uv.lock makes installs reproducible on every machine and in CI.

| | |
| --- | --- |
| ID | `python-uv` |
| Strength | should (warning) |
| Status | accepted |
| Scope | python |
| Applies to | `**/pyproject.toml` |
| Tags | python, dependencies, reproducibility |
| Updated | 2026-10-27 |

## Context

`pip install -r requirements.txt` without hashes resolves differently over
time, so a green build today says little about tomorrow's.

## Decision

Declare dependencies in `pyproject.toml`, manage them with `uv`, and commit
`uv.lock`. CI installs with `uv sync --locked`.

## Consequences

Dependency changes show up as lock file diffs in review.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-uv-lock` | canon | pyproject.toml has a committed uv.lock beside it |

Read it from an agent with `canon explain python-uv`, or the `canon://records/python-uv` resource.
