# Terraform passes a security scan before merge

Misconfigurations like public buckets are cheapest to fix before they are applied.

| | |
| --- | --- |
| ID | `terraform-security-scan` |
| Strength | must (error) |
| Status | accepted |
| Scope | terraform |
| Applies to | `**/*.tf` |
| Tags | terraform, security |
| Updated | 2026-10-27 |

## Context

Public storage, unencrypted volumes and open security groups are easy to
write and hard to spot in a plan.

## Decision

Run checkov (or trivy config) in CI with no failed checks, or a documented
skip comment naming the check and the reason.

## Consequences

Defaults in modules are safe: private, encrypted, least privilege.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `checkov-terraform` | checkov | checkov security checks for Terraform |

Read it from an agent with `canon explain terraform-security-scan`, or the `canon://records/terraform-security-scan` resource.
