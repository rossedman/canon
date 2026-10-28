# Pin Terraform and provider versions with bounded constraints

Bounded version constraints stop a provider major release from changing infrastructure on an ordinary plan.

| | |
| --- | --- |
| ID | `terraform-version-constraints` |
| Strength | must (error) |
| Status | accepted |
| Scope | terraform |
| Applies to | `**/*.tf` |
| Tags | terraform, reproducibility |
| Updated | 2026-10-27 |

## Context

An unconstrained provider upgrades on the next `terraform init`, and a major
release can rename or replace resources.

## Decision

`required_version` has a minimum and an upper major bound (`>= 1.9, < 2.0`).
Every provider in `required_providers` has a `source` and a pessimistic
constraint (`~> 6.0`). Commit `.terraform.lock.hcl` in root modules.

## Consequences

Provider upgrades are deliberate PRs that show the plan.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `tflint-versions` | tflint | tflint checks Terraform and provider version constraints |

Read it from an agent with `canon explain terraform-version-constraints`, or the `canon://records/terraform-version-constraints` resource.
