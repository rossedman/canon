# Pin every base image by tag and digest

A digest-pinned base builds the same image tomorrow, and the tag keeps the pin readable.

| | |
| --- | --- |
| ID | `containers-pin-base-images` |
| Strength | must (error) |
| Status | accepted |
| Scope | containers |
| Applies to | `Dockerfile`, `**/Dockerfile`, `*.Dockerfile`, `**/*.Dockerfile` |
| Tags | supply-chain, reproducibility |
| Updated | 2026-10-27 |

## Context

`FROM golang:latest` or even `FROM golang:1.27` can change under you between
two builds of the same commit.

## Decision

Every `FROM` names a version tag and a digest:
`FROM golang:1.27.3@sha256:<digest>`. Dependabot's `docker` ecosystem keeps
the digests moving. Never write a digest from memory; resolve it with
`docker buildx imagetools inspect`.

## Consequences

Base image updates arrive as reviewable PRs.

## Exceptions

`FROM scratch` and references to earlier build stages.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-base-digest` | canon | Every external FROM pins a sha256 digest |
| `hadolint-tags` | hadolint | hadolint checks that base images name a version tag |

Read it from an agent with `canon explain containers-pin-base-images`, or the `canon://records/containers-pin-base-images` resource.
