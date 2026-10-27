---
id: ci-least-privilege
title: Workflows declare least-privilege token permissions
summary: An explicit permissions block limits what a compromised step can do with the workflow token.
status: accepted
scope: [ci]
applies_to:
  files: [".github/workflows/*.yml", ".github/workflows/*.yaml"]
tags: [ci, security]
rules: [canon-workflow-permissions]
strength: must
updated: 2026-10-27
---

## Context
Without a `permissions:` block, the token gets the repository default, which
is often read-write on everything.

## Decision
Every workflow sets `permissions:` at the top level (usually `{}` or
`contents: read`), and jobs that need more ask for exactly that.

## Consequences
A job that writes releases or Pages states it in the job, where review sees it.
