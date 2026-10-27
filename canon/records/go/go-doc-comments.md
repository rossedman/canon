---
id: go-doc-comments
title: Every function and method has a doc comment
summary: Doc comments say what a function promises, which is what a reader and a reviewer need first.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [docs, readability]
rules: [canon-go-doc-comments]
strength: should
updated: 2026-10-27
---

## Context
Unexported helpers are where most of a codebase's logic lives, and they are
the functions a new contributor reads first when fixing a bug.

## Decision
Every function and method, exported or not, has a doc comment that starts with
its name. Every package has a package comment. Test files and generated files
are exempt.

## Consequences
Writing the comment forces a name and a contract; a function that is hard to
describe in a sentence is usually doing too much.
