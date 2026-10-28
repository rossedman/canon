# Dockerfiles follow hadolint's best practices

hadolint encodes the common Dockerfile mistakes that bloat images or break caching.

| | |
| --- | --- |
| ID | `containers-dockerfile-hygiene` |
| Strength | should (warning) |
| Status | accepted |
| Scope | containers |
| Applies to | `Dockerfile`, `**/Dockerfile`, `*.Dockerfile`, `**/*.Dockerfile` |
| Tags | containers, lint |
| Updated | 2026-10-27 |

## Context

Unpinned packages, missing `--no-install-recommends` and caches left in layers
make images large and builds unrepeatable.

## Decision

Dockerfiles pass hadolint with the config shipped in this canon. Use
`# syntax=docker/dockerfile:1`, multi-stage builds, cache mounts, and
exec-form `ENTRYPOINT`.

## Consequences

Suppressions are `# hadolint ignore=DLxxxx` comments with a reason on the line above.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `hadolint-hygiene` | hadolint | Every other hadolint best-practice check |

Read it from an agent with `canon explain containers-dockerfile-hygiene`, or the `canon://records/containers-dockerfile-hygiene` resource.
