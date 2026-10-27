---
id: yaml-lint
title: YAML is linted with yamllint
summary: Consistent, valid YAML avoids the silent type and indentation surprises YAML is known for.
status: accepted
scope: [yaml]
applies_to:
  files: ["**/*.yml", "**/*.yaml"]
tags: [yaml, lint]
rules: [yamllint-baseline]
strength: should
updated: 2026-10-27
---

## Context
Duplicate keys, tabs and truthy values like `on`/`yes` parse differently
across tools.

## Decision
YAML passes yamllint with the config shipped in this canon: two-space
indentation, no duplicate keys, line length warnings at 160.

## Consequences
GitHub workflow files are exempt from the `truthy` rule because `on:` is required.
