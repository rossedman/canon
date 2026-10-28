---
id: containers-dockerfile-hygiene
title: Dockerfiles follow hadolint's best practices
summary: hadolint encodes the common Dockerfile mistakes that bloat images or break caching.
status: accepted
scope: [containers]
applies_to:
  files: ["Dockerfile", "**/Dockerfile", "*.Dockerfile", "**/*.Dockerfile"]
tags: [containers, lint]
rules: [hadolint-hygiene]
strength: should
updated: 2026-10-27
---

## Context
Unpinned packages, missing `--no-install-recommends` and caches left in layers
make images large and builds unrepeatable.

## Decision
Dockerfiles pass hadolint with the config shipped in this canon. Use
`# syntax=docker/dockerfile:1`, multi-stage builds, cache mounts, and
exec-form `ENTRYPOINT`.

## Consequences
Suppressions are `# hadolint ignore=DLxxxx` comments with a reason on the line above.
