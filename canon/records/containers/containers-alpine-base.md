---
id: containers-alpine-base
title: Use alpine for small runtime images
summary: Alpine keeps runtime images small.
status: superseded
scope: [containers]
applies_to:
  files: ["Dockerfile", "**/Dockerfile"]
tags: [image-size]
rules: []
strength: should
updated: 2025-03-01
---

## Context
Debian-based images were several hundred megabytes and slow to pull.

## Decision
Use `alpine` as the runtime base.

## Consequences
musl differs from glibc in subtle ways, and the image still ships a shell and
`apk`. Superseded by `containers-distroless-base`, which is smaller still and
has neither.
