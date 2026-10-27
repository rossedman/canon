---
id: containers-distroless-base
title: Use distroless or scratch for runtime images
summary: Runtime images carry no shell or package manager.
status: accepted
supersedes: [containers-alpine-base]
scope: [containers]
applies_to:
  files: ["Dockerfile", "**/Dockerfile", "*.Dockerfile", "**/*.Dockerfile"]
tags: [security, image-size, supply-chain]
rules: [canon-final-stage-base]
strength: must
updated: 2026-10-27
---

## Context
Shells and package managers in runtime images widen the attack surface
and pull in CVEs we then have to triage.

## Decision
Final stages use `gcr.io/distroless/static` (Go) or
`gcr.io/distroless/python3` (Python), `cgr.dev/chainguard/*`, or `scratch`.
Build tooling stays in earlier stages.

## Consequences
No `docker exec` shell for debugging; use ephemeral debug containers.

## Exceptions
Images that must run a shell entrypoint, with a comment explaining why.

## Examples
```dockerfile
# good
FROM golang:1.27 AS build
RUN CGO_ENABLED=0 go build -o /app .
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app /app

# bad: the runtime stage ships apt and bash
FROM debian:bookworm
```
