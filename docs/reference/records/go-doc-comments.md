# Every function and method has a doc comment

Doc comments say what a function promises, which is what a reader and a reviewer need first.

| | |
| --- | --- |
| ID | `go-doc-comments` |
| Strength | should (warning) |
| Status | accepted |
| Scope | go |
| Applies to | `**/*.go` |
| Tags | docs, readability |
| Updated | 2026-10-27 |

## Context

Unexported helpers are where most of a codebase's logic lives, and they are
the functions a new contributor reads first when fixing a bug.

## Decision

Every function and method, exported or not, has a doc comment that starts with
its name. Every package has a package comment. Test files and generated files
are exempt.

## Consequences

Writing the comment forces a name and a contract; a function that is hard to
describe in a sentence is usually doing too much.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-go-doc-comments` | canon | Every function and method has a doc comment |

Read it from an agent with `canon explain go-doc-comments`, or the `canon://records/go-doc-comments` resource.
