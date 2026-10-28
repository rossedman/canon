---
id: go-openapi-spec-first
title: HTTP APIs are spec-first with OpenAPI and generated code
summary: When the spec is the source, the server, the client and the docs cannot disagree.
status: accepted
scope: [go]
applies_to:
  files: ["api/**/*.yaml", "**/*.gen.go"]
tags: [api, openapi, codegen]
rules: []
strength: must
updated: 2026-10-27
---

## Context
Hand-written handlers and clients drift from each other and from the docs.

## Decision
The OpenAPI 3.x spec in `api/openapi.yaml` is the source of truth. Server
interfaces, types and clients are generated with `oapi-codegen`; `*.gen.go`
files are never edited. CI runs `go generate ./... && git diff --exit-code`.

## Consequences
Every API change starts as a spec change, which is the first thing reviewed.
