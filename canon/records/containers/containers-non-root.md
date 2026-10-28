---
id: containers-non-root
title: Containers run as a non-root user
summary: A process that escapes a non-root container starts with no privileges on the host.
status: accepted
scope: [containers]
applies_to:
  files: ["Dockerfile", "**/Dockerfile", "*.Dockerfile", "**/*.Dockerfile"]
tags: [security]
rules: [hadolint-user, canon-final-stage-user]
strength: must
updated: 2026-10-27
---

## Context
Root in a container is root on the host for any kernel or runtime escape.

## Decision
The final stage sets `USER 65532:65532` or the base image's `nonroot` user,
and never `USER root`.

## Consequences
Files the app writes must live in a directory that user owns.
