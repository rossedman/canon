# Use alpine for small runtime images

Alpine keeps runtime images small.

!!! warning "Superseded"
    This record is no longer served as current advice. See [containers-distroless-base](containers-distroless-base.md).

| | |
| --- | --- |
| ID | `containers-alpine-base` |
| Strength | should (warning) |
| Status | superseded |
| Scope | containers |
| Applies to | `Dockerfile`, `**/Dockerfile` |
| Tags | image-size |
| Updated | 2025-03-01 |

## Context

Debian-based images were several hundred megabytes and slow to pull.

## Decision

Use `alpine` as the runtime base.

## Consequences

musl differs from glibc in subtle ways, and the image still ships a shell and
`apk`. Superseded by `containers-distroless-base`, which is smaller still and
has neither.

## Enforced by

No automated rule yet; reviewers and `canon decide` carry this one.

Read it from an agent with `canon explain containers-alpine-base`, or the `canon://records/containers-alpine-base` resource.
