# HTTP APIs are spec-first with OpenAPI and generated code

When the spec is the source, the server, the client and the docs cannot disagree.

| | |
| --- | --- |
| ID | `go-openapi-spec-first` |
| Strength | must (error) |
| Status | accepted |
| Scope | go |
| Applies to | `api/**/*.yaml`, `**/*.gen.go` |
| Tags | api, openapi, codegen |
| Updated | 2026-10-27 |

## Context

Hand-written handlers and clients drift from each other and from the docs.

## Decision

The OpenAPI 3.x spec in `api/openapi.yaml` is the source of truth. Server
interfaces, types and clients are generated with `oapi-codegen`; `*.gen.go`
files are never edited. CI runs `go generate ./... && git diff --exit-code`.

## Consequences

Every API change starts as a spec change, which is the first thing reviewed.

## Enforced by

No automated rule yet; reviewers and `canon decide` carry this one.

Read it from an agent with `canon explain go-openapi-spec-first`, or the `canon://records/go-openapi-spec-first` resource.
