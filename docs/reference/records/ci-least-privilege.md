# Workflows declare least-privilege token permissions

An explicit permissions block limits what a compromised step can do with the workflow token.

| | |
| --- | --- |
| ID | `ci-least-privilege` |
| Strength | must (error) |
| Status | accepted |
| Scope | ci |
| Applies to | `.github/workflows/*.yml`, `.github/workflows/*.yaml` |
| Tags | ci, security |
| Updated | 2026-10-27 |

## Context

Without a `permissions:` block, the token gets the repository default, which
is often read-write on everything.

## Decision

Every workflow sets `permissions:` at the top level (usually `{}` or
`contents: read`), and jobs that need more ask for exactly that.

## Consequences

A job that writes releases or Pages states it in the job, where review sees it.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-workflow-permissions` | canon | Every workflow declares a top-level permissions block |

Read it from an agent with `canon explain ci-least-privilege`, or the `canon://records/ci-least-privilege` resource.
