---
id: terraform-module-conventions
title: Terraform modules are typed, documented and linted
summary: Typed, described variables and outputs make a module usable without reading its source.
status: accepted
scope: [terraform]
applies_to:
  files: ["**/*.tf"]
tags: [terraform, docs, lint]
rules: [tflint-conventions]
strength: should
updated: 2026-10-27
---

## Context
Modules with untyped variables and undocumented outputs push their contract
into the heads of whoever wrote them.

## Decision
Every variable has a `type`, a `description` and `validation` for constrained
formats; every output has a `description`. Modules never configure providers
or backends. tflint runs the `terraform` ruleset with the `recommended` preset.

## Consequences
`terraform-docs` can generate the README's inputs and outputs tables.
