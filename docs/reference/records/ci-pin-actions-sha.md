# Pin third-party GitHub Actions to a full commit SHA

A tag can be moved to malicious code; a commit SHA cannot.

| | |
| --- | --- |
| ID | `ci-pin-actions-sha` |
| Strength | must (error) |
| Status | accepted |
| Scope | ci |
| Applies to | `.github/workflows/*.yml`, `.github/workflows/*.yaml`, `action.yml`, `action.yaml` |
| Tags | ci, supply-chain, security |
| Updated | 2026-10-27 |

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

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-actions-pinned` | canon | Every remote uses reference pins a 40-character commit SHA |

Read it from an agent with `canon explain ci-pin-actions-sha`, or the `canon://records/ci-pin-actions-sha` resource.
