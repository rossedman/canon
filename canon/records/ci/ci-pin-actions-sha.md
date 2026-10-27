---
id: ci-pin-actions-sha
title: Pin third-party GitHub Actions to a full commit SHA
summary: A tag can be moved to malicious code; a commit SHA cannot.
status: accepted
scope: [ci]
applies_to:
  files: [".github/workflows/*.yml", ".github/workflows/*.yaml", "action.yml", "action.yaml"]
tags: [ci, supply-chain, security]
rules: [canon-actions-pinned]
strength: must
updated: 2026-10-27
---

## Context
`uses: some/action@v3` runs whatever that tag points at today, with the
workflow's token. Compromised action tags have leaked secrets from thousands
of repos.

## Decision
Every `uses:` that references another repository pins a 40-character commit
SHA, with the version as a trailing comment:
`uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1`.
Dependabot's `github-actions` ecosystem keeps the pins and comments current.

## Consequences
Local actions (`./.github/actions/x`) and `docker://` references with a digest
are exempt.
