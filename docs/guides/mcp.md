# MCP server

Six tools, four resources and three prompts, over stdio or streamable HTTP.

```bash
claude mcp add canon -- canon serve   # stdio
canon serve --http :8080              # streamable HTTP
```

The server tells clients when to reach for each tool: `decide` before choosing
how to do something, `check` after editing files, `review` before opening a
pull request, `explain` for the reasoning behind a finding, `profile` to see
what applies, and `scaffold` (dry run first) to start a project. When `decide`
reports a gap, the agent should say the canon has no answer rather than guess.
Tool paths resolve against `--root`.

## Tools

All tools are read-only except `scaffold`, which defaults to a dry run and never overwrites files.

### check

Run the rules against paths and return findings with rule, record and why.

| Argument | Type | Default |
| --- | --- | --- |
| `paths` | array of string | repo root |
| `scope` | array of `go`, `python`, `containers`, `terraform`, `yaml`, `ci`, `oss` | all in profile |
| `min_severity` | `error`, `warning` or `suggestion` | `warning` |
| `changed_only` | boolean | `false` |

### decide

Answer "how do we do X" from the records. Returns ranked records with their Decision and Exceptions, or a gap.

| Argument | Type | Default |
| --- | --- | --- |
| `question` (required) | string | |
| `scope` | array of string | |
| `limit` | integer | `3` |

### explain

Trace a rule or a record to its full reasoning. Give exactly one of `rule` or `record`.

| Argument | Type |
| --- | --- |
| `rule` | string, a rule ID |
| `record` | string, a record ID |

### review

Grade a diff: check the changed files, then report concerns linters cannot see (new dependencies, new Dockerfiles, provider bumps, exported API changes, new packages without tests).

| Argument | Type | Default |
| --- | --- | --- |
| `base` | string | `origin/main` |
| `diff` | string, a unified diff | computed from `base` |

### scaffold

Generate a project or component from a template.

| Argument | Type | Default |
| --- | --- | --- |
| `kind` (required) | `go-cli`, `go-service`, `py-service`, `tf-module`, `dockerfile`, `gha-workflow` | |
| `name` (required) | string | |
| `dest` | string | `.` |
| `vars` | object of string to string | |
| `dry_run` | boolean | `true` |

### profile

Detect which profile and records apply, and which checkers are missing locally.

| Argument | Type | Default |
| --- | --- | --- |
| `path` | string | `.` |

## Resources

| URI | Contents |
| --- | --- |
| `canon://records` | Index of every record: id, title, status, strength, scope (JSON) |
| `canon://records/{id}` | One record, full Markdown |
| `canon://profiles/{name}` | A profile and its resolved record list (JSON) |
| `canon://principles` | The principles every record ladders up to (Markdown) |

## Prompts

| Prompt | Argument | What it does |
| --- | --- | --- |
| `canon-review` | `base` | Runs `review` on the branch and writes a PR-ready summary quoting each finding's why. |
| `canon-new-record` | `topic` | Interviews you about a decision and drafts a record. |
| `canon-adopt` | `path` | Runs `profile` and `check` on an existing repo and drafts a remediation plan by severity. |
