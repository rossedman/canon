# Python is linted and formatted with ruff

One fast tool for lint and format means CI and editors agree on what clean Python looks like.

| | |
| --- | --- |
| ID | `python-ruff` |
| Strength | must (error) |
| Status | accepted |
| Scope | python |
| Applies to | `**/*.py` |
| Tags | lint, python, quality |
| Updated | 2026-10-27 |

## Context

flake8, isort, black and pyupgrade each have their own config and their own
opinion about line length. Running four tools is slow enough that people skip it.

## Decision

Use `ruff check` and `ruff format` with the config shipped in this canon
(pycodestyle, pyflakes, isort, bugbear, pyupgrade and bandit rules). CI fails
on any finding.

## Consequences

Suppressions are inline `# noqa: <code>` comments with a reason.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `ruff-baseline` | ruff | ruff check with the shared rule selection |

Read it from an agent with `canon explain python-ruff`, or the `canon://records/python-ruff` resource.
