# YAML is linted with yamllint

Consistent, valid YAML avoids the silent type and indentation surprises YAML is known for.

| | |
| --- | --- |
| ID | `yaml-lint` |
| Strength | should (warning) |
| Status | accepted |
| Scope | yaml |
| Applies to | `**/*.yml`, `**/*.yaml` |
| Tags | yaml, lint |
| Updated | 2026-10-27 |

## Context

Duplicate keys, tabs and truthy values like `on`/`yes` parse differently
across tools.

## Decision

YAML passes yamllint with the config shipped in this canon: two-space
indentation, no duplicate keys, line length warnings at 160.

## Consequences

GitHub workflow files are exempt from the `truthy` rule because `on:` is required.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `yamllint-baseline` | yamllint | yamllint with the shared config |

Read it from an agent with `canon explain yaml-lint`, or the `canon://records/yaml-lint` resource.
