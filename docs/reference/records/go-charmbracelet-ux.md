# Terminal UX uses the Charmbracelet libraries

One set of terminal libraries gives every tool the same look, accessibility and script-safe behavior.

| | |
| --- | --- |
| ID | `go-charmbracelet-ux` |
| Strength | should (warning) |
| Status | accepted |
| Scope | go |
| Applies to | `**/*.go` |
| Tags | cli, ux, accessibility |
| Updated | 2026-10-27 |

## Context

Hand-rolled ANSI codes and a mix of color and prompt libraries give each tool
different behavior with `NO_COLOR`, pipes and screen readers.

## Decision

Use `fang` for the cobra root, `lipgloss` for styled output, `huh` for prompts,
`bubbletea` for full-screen programs and `log` as the `slog` handler. Styles
live in one `internal/ui` package. Machine output (`--json`, pipes) is never
styled, and prompts run only when stdin and stdout are terminals.

## Consequences

`fatih/color`, `survey`, `promptui` and `tview` are not imported.

## Exceptions

An existing repo that already uses another library keeps it until someone
decides to switch.

## Enforced by

| Rule | Checker | What it checks |
| --- | --- | --- |
| `canon-go-ux-libraries` | canon | No non-Charmbracelet terminal UX libraries are imported |

Read it from an agent with `canon explain go-charmbracelet-ux`, or the `canon://records/go-charmbracelet-ux` resource.
