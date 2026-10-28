# Release assets are signed keylessly with cosign

Keyless signatures let anyone verify a download came from this repo's release workflow, with no keys to leak.

| | |
| --- | --- |
| ID | `oss-release-signing` |
| Strength | must (error) |
| Status | accepted |
| Scope | oss |
| Applies to | `.goreleaser.yaml`, `.goreleaser.yml` |
| Tags | release, supply-chain, security |
| Updated | 2026-10-27 |

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

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-release-signing` | canon | GoReleaser signs the checksums file with cosign |

Read it from an agent with `canon explain oss-release-signing`, or the `canon://records/oss-release-signing` resource.
