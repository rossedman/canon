---
id: oss-release-signing
title: Release assets are signed keylessly with cosign
summary: Keyless signatures let anyone verify a download came from this repo's release workflow, with no keys to leak.
status: accepted
scope: [oss]
applies_to:
  files: [".goreleaser.yaml", ".goreleaser.yml"]
tags: [release, supply-chain, security]
rules: [canon-release-signing]
strength: must
updated: 2026-10-27
---

## Context
Unsigned release archives can be swapped on a mirror or in a compromised
release without anyone noticing.

## Decision
Sign the checksums file (which covers every asset) with
`cosign sign-blob --bundle` using the workflow's GitHub OIDC identity. The
release workflow has `id-token: write`. Add build provenance with
`actions/attest-build-provenance`. Document verification in `SECURITY.md`.

## Consequences
No signing keys or secrets to manage; verification checks the workflow identity.
