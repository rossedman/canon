# Open-source repos ship the standard community files

Contributors and security reporters look for these files first, and GitHub's community profile checks for them.

| | |
| --- | --- |
| ID | `oss-community-files` |
| Strength | should (warning) |
| Status | accepted |
| Scope | oss |
| Applies to | `**` |
| Tags | oss, community, security |
| Updated | 2026-10-27 |

## Context

A repo without a license cannot legally be used; one without a security policy
gets vulnerabilities reported in public issues.

## Decision

The repo root has `LICENSE`, `README.md`, `SECURITY.md`, `CONTRIBUTING.md` and
`CODE_OF_CONDUCT.md` (Contributor Covenant 2.1), written for the project
rather than copied boilerplate. The default license is Apache-2.0.

## Consequences

License text is fetched from the official source, never written from memory.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-oss-files` | canon | The repo root has LICENSE, README, SECURITY, CONTRIBUTING and CODE_OF_CONDUCT |

Read it from an agent with `canon explain oss-community-files`, or the `canon://records/oss-community-files` resource.
