---
id: python-ruff
title: Python is linted and formatted with ruff
summary: One fast tool for lint and format means CI and editors agree on what clean Python looks like.
status: accepted
scope: [python]
applies_to:
  files: ["**/*.py"]
tags: [lint, python, quality]
rules: [ruff-baseline]
strength: must
updated: 2026-10-27
---

## Context
flake8, isort, black and pyupgrade each have their own config and their own
opinion about line length. Running four tools is slow enough that people skip it.

## Decision
Use `ruff check` and `ruff format` with the config shipped in this canon
(pycodestyle, pyflakes, isort, bugbear, pyupgrade and bandit rules). CI fails
on any finding.

## Consequences
Suppressions are inline `# noqa: <code>` comments with a reason.
