---
id: containers-pin-base-images
title: Pin every base image by tag and digest
summary: A digest-pinned base builds the same image tomorrow, and the tag keeps the pin readable.
status: accepted
scope: [containers]
applies_to:
  files: ["Dockerfile", "**/Dockerfile", "*.Dockerfile", "**/*.Dockerfile"]
tags: [supply-chain, reproducibility]
rules: [hadolint-tags, canon-base-digest]
strength: must
updated: 2026-10-27
---

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
