# Terraform modules are typed, documented and linted

Typed, described variables and outputs make a module usable without reading its source.

| | |
| --- | --- |
| ID | `terraform-module-conventions` |
| Strength | should (warning) |
| Status | accepted |
| Scope | terraform |
| Applies to | `**/*.tf` |
| Tags | terraform, docs, lint |
| Updated | 2026-10-27 |

## Context

Modules with untyped variables and undocumented outputs push their contract
into the heads of whoever wrote them.

## Decision

Every variable has a `type`, a `description` and `validation` for constrained
formats; every output has a `description`. Modules never configure providers
or backends. tflint runs the `terraform` ruleset with the `recommended` preset.

## Consequences

`terraform-docs` can generate the README's inputs and outputs tables.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `tflint-conventions` | tflint | tflint's recommended terraform ruleset |

Read it from an agent with `canon explain terraform-module-conventions`, or the `canon://records/terraform-module-conventions` resource.
