---
id: oss-community-files
title: Open-source repos ship the standard community files
summary: Contributors and security reporters look for these files first, and GitHub's community profile checks for them.
status: accepted
scope: [oss]
applies_to:
  files: ["**"]
tags: [oss, community, security]
rules: [canon-oss-files]
strength: should
updated: 2026-10-27
---

## Context
A repo without a license cannot legally be used; one without a security policy
gets vulnerabilities reported in public issues.

## Decision
The repo root has `LICENSE`, `README.md`, `SECURITY.md`, `CONTRIBUTING.md` and
`CODE_OF_CONDUCT.md` (Contributor Covenant 2.1), written for the project
rather than copied boilerplate. The default license is Apache-2.0.

## Consequences
License text is fetched from the official source, never written from memory.
