# Writing records

Add a standard to your canon: the file format, the frontmatter, and how to validate it.

A record is one Markdown file at `canon/records/<scope>/<id>.md`. Here is a
real one, trimmed:

```markdown
---
id: go-errors-wrap
title: Wrap errors with %w and context; never log and return
summary: Wrapped errors keep the cause inspectable and say where it failed, and one owner handles each error.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [errors, reliability, debugging]
rules: [golangci-errors]
strength: must
updated: 2026-10-27
---

## Context
An error that crosses a package boundary without context forces whoever reads
the log to guess which call failed.

## Decision
Return errors wrapped with `fmt.Errorf("doing x: %w", err)` ...

## Consequences
Error messages read as a chain of steps from the top caller down to the cause.
```

## Frontmatter

| Field | Meaning |
| --- | --- |
| `id` | Kebab-case, starts with its scope, matches the file name. |
| `title` | The decision as an instruction. |
| `summary` | One sentence. It becomes the "why" in findings. |
| `status` | `proposed`, `accepted`, `superseded` or `deprecated`. |
| `supersedes` | ID of the record this replaces. No cycles. |
| `scope` | One or more of `go`, `python`, `containers`, `terraform`, `yaml`, `ci`, `oss`. |
| `applies_to.files` | Globs the record covers. |
| `tags` | Free-form, used by `decide` search. |
| `rules` | Rule IDs from `rules.yaml` that enforce it; `[]` if nothing checks it yet. |
| `strength` | `must` (error), `should` (warning) or `prefer` (suggestion). |
| `updated` | Date of the last change. |

## Sections

In order: `## Context`, `## Decision` and `## Consequences` are required.
`## Exceptions` and `## Examples` are strongly encouraged: exceptions are what
`decide` returns next to the decision, and examples show a good and a bad case.

## Validate

```bash
canon lint
```

checks every record has its required sections and valid frontmatter, every
rule ID resolves in both directions, `supersedes` has no cycles, profiles name
real scopes and records, and every template renders. Use
`canon lint --canon-dir ./canon` for your own canon.

## Let an agent draft it

The `canon-new-record` [MCP prompt](mcp.md#prompts) interviews you and drafts a
record in this format, after searching for an existing record to amend or supersede.
To enforce it, add a rule in `rules.yaml` that maps a checker code to the record.
