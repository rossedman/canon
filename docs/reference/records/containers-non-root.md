# Containers run as a non-root user

A process that escapes a non-root container starts with no privileges on the host.

| | |
| --- | --- |
| ID | `containers-non-root` |
| Strength | must (error) |
| Status | accepted |
| Scope | containers |
| Applies to | `Dockerfile`, `**/Dockerfile`, `*.Dockerfile`, `**/*.Dockerfile` |
| Tags | security |
| Updated | 2026-10-27 |

## Context

Root in a container is root on the host for any kernel or runtime escape.

## Decision

The final stage sets `USER 65532:65532` or the base image's `nonroot` user,
and never `USER root`.

## Consequences

Files the app writes must live in a directory that user owns.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-final-stage-user` | canon | The final stage sets a non-root USER (or uses a nonroot base) |
| `hadolint-user` | hadolint | hadolint checks the last USER is not root |

Read it from an agent with `canon explain containers-non-root`, or the `canon://records/containers-non-root` resource.
