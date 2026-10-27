---
id: oss-conventional-commits
title: Commits and PR titles follow Conventional Commits
summary: Typed commit messages make changelogs and version bumps mechanical.
status: accepted
scope: [oss]
applies_to:
  files: ["**"]
tags: [git, release]
rules: []
strength: should
updated: 2026-10-27
---

## Context
Release notes assembled by hand from free-form commit messages are slow and miss things.

## Decision
PR titles (which become squash commits) use `type(scope): summary`, with types
`feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci` and `chore`.
Breaking changes add `!` or a `BREAKING CHANGE:` footer.

## Consequences
GoReleaser groups the changelog by type, and the next version follows from the commits.
