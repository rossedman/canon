// Package ui holds every terminal style canon uses, and renders engine
// results for people. Machine output (JSON, SARIF) never passes through here.
package ui

import (
	"io"
	"log/slog"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
)

// Brand colors. Each reads on dark and light terminals.
var (
	Brand = lipgloss.Color("#E8A33D")
	OK    = lipgloss.Color("#3FB97F")
	Warn  = lipgloss.Color("#D9A21B")
	Bad   = lipgloss.Color("#E5534B")
	Info  = lipgloss.Color("#4C9BE8")
	Dim   = lipgloss.Color("#8B8F99")
)

// Styles used across commands.
var (
	Title      = lipgloss.NewStyle().Bold(true).Foreground(Brand)
	Heading    = lipgloss.NewStyle().Bold(true)
	Muted      = lipgloss.NewStyle().Foreground(Dim)
	Code       = lipgloss.NewStyle().Foreground(Info)
	ErrorText  = lipgloss.NewStyle().Bold(true).Foreground(Bad)
	WarnText   = lipgloss.NewStyle().Bold(true).Foreground(Warn)
	SuggestTxt = lipgloss.NewStyle().Foreground(Info)
	Success    = lipgloss.NewStyle().Bold(true).Foreground(OK)
)

// Writer wraps w so styles are downsampled to what the terminal supports,
// and stripped entirely for pipes, files and NO_COLOR.
func Writer(w io.Writer) io.Writer {
	return colorprofile.NewWriter(w, os.Environ())
}

// NewLogger returns a slog logger that writes human-friendly lines to w
// through charmbracelet/log.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	l := log.NewWithOptions(w, log.Options{Level: log.Level(level), Prefix: "canon"})
	return slog.New(l)
}
