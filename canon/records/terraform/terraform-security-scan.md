---
id: terraform-security-scan
title: Terraform passes a security scan before merge
summary: Misconfigurations like public buckets are cheapest to fix before they are applied.
status: accepted
scope: [terraform]
applies_to:
  files: ["**/*.tf"]
tags: [terraform, security]
rules: [checkov-terraform]
strength: must
updated: 2026-10-27
---

## Context
Public storage, unencrypted volumes and open security groups are easy to
write and hard to spot in a plan.

## Decision
Run checkov (or trivy config) in CI with no failed checks, or a documented
skip comment naming the check and the reason.

## Consequences
Defaults in modules are safe: private, encrypted, least privilege.
