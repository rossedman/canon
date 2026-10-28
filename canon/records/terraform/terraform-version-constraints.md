---
id: terraform-version-constraints
title: Pin Terraform and provider versions with bounded constraints
summary: Bounded version constraints stop a provider major release from changing infrastructure on an ordinary plan.
status: accepted
scope: [terraform]
applies_to:
  files: ["**/*.tf"]
tags: [terraform, reproducibility]
rules: [tflint-versions]
strength: must
updated: 2026-10-27
---

## Context
An unconstrained provider upgrades on the next `terraform init`, and a major
release can rename or replace resources.

## Decision
`required_version` has a minimum and an upper major bound (`>= 1.9, < 2.0`).
Every provider in `required_providers` has a `source` and a pessimistic
constraint (`~> 6.0`). Commit `.terraform.lock.hcl` in root modules.

## Consequences
Provider upgrades are deliberate PRs that show the plan.
