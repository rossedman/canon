# Commits and PR titles follow Conventional Commits

Typed commit messages make changelogs and version bumps mechanical.

| | |
| --- | --- |
| ID | `oss-conventional-commits` |
| Strength | should (warning) |
| Status | accepted |
| Scope | oss |
| Applies to | `**` |
| Tags | git, release |
| Updated | 2026-10-27 |

## Context

Release notes assembled by hand from free-form commit messages are slow and miss things.

## Decision

PR titles (which become squash commits) use `type(scope): summary`, with types
`feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci` and `chore`.
Breaking changes add `!` or a `BREAKING CHANGE:` footer.

## Consequences

GoReleaser groups the changelog by type, and the next version follows from the commits.

## Enforced by

No automated rule yet; reviewers and `canon decide` carry this one.

Read it from an agent with `canon explain oss-conventional-commits`, or the `canon://records/oss-conventional-commits` resource.
