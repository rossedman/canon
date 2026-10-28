---
id: go-charmbracelet-ux
title: Terminal UX uses the Charmbracelet libraries
summary: One set of terminal libraries gives every tool the same look, accessibility and script-safe behavior.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [cli, ux, accessibility]
rules: [canon-go-ux-libraries]
strength: should
updated: 2026-10-27
---

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
