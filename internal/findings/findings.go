// Package findings defines the Finding type every check returns, and the
// helpers that filter, order, summarize and export findings.
package findings

import (
	"cmp"
	"fmt"
	"slices"
)

// Severity ranks how much a finding matters. It follows from the strength of
// the record the finding traces to.
type Severity string

// The three severities, from most to least serious.
const (
	Error      Severity = "error"
	Warning    Severity = "warning"
	Suggestion Severity = "suggestion"
)

// rank orders severities so higher is more serious; unknown values rank 0.
func (s Severity) rank() int {
	switch s {
	case Error:
		return 3
	case Warning:
		return 2
	case Suggestion:
		return 1
	}
	return 0
}

// AtLeast reports whether s is as serious as threshold or more.
func (s Severity) AtLeast(threshold Severity) bool {
	return s.rank() >= threshold.rank()
}

// ParseSeverity turns a flag or tool argument into a Severity. An empty
// string means warning, the default threshold.
func ParseSeverity(s string) (Severity, error) {
	switch Severity(s) {
	case "":
		return Warning, nil
	case Error, Warning, Suggestion:
		return Severity(s), nil
	}
	return "", fmt.Errorf("unknown severity %q (want error, warning or suggestion)", s)
}

// Fix is a machine-applicable suggestion attached to a finding.
type Fix struct {
	Kind       string `json:"kind"`
	Suggestion string `json:"suggestion"`
}

// Finding is one violation of a rule, traced to the record it enforces.
type Finding struct {
	Rule     string   `json:"rule"`
	Record   string   `json:"record"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Line     int      `json:"line,omitempty"`
	Column   int      `json:"column,omitempty"`
	Message  string   `json:"message"`
	Why      string   `json:"why"`
	Fix      *Fix     `json:"fix,omitempty"`
}

// Summary counts findings by severity.
type Summary struct {
	Error      int `json:"error"`
	Warning    int `json:"warning"`
	Suggestion int `json:"suggestion"`
}

// Total returns the number of findings counted.
func (s Summary) Total() int {
	return s.Error + s.Warning + s.Suggestion
}

// Summarize counts fs by severity.
func Summarize(fs []Finding) Summary {
	var s Summary
	for _, f := range fs {
		switch f.Severity {
		case Error:
			s.Error++
		case Warning:
			s.Warning++
		case Suggestion:
			s.Suggestion++
		}
	}
	return s
}

// Filter keeps the findings at threshold severity or above.
func Filter(fs []Finding, threshold Severity) []Finding {
	out := make([]Finding, 0, len(fs))
	for _, f := range fs {
		if f.Severity.AtLeast(threshold) {
			out = append(out, f)
		}
	}
	return out
}

// key identifies a finding for deduplication: the same rule at the same place
// with the same message is reported once.
type key struct {
	rule, path, message string
	line, column        int
}

// Normalize removes duplicates and sorts by path, line, column, then the most
// serious first, so output is stable between runs.
func Normalize(fs []Finding) []Finding {
	seen := make(map[key]bool, len(fs))
	out := make([]Finding, 0, len(fs))
	for _, f := range fs {
		k := key{f.Rule, f.Path, f.Message, f.Line, f.Column}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, f)
	}
	slices.SortStableFunc(out, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Column, b.Column),
			cmp.Compare(b.Severity.rank(), a.Severity.rank()),
			cmp.Compare(a.Rule, b.Rule),
		)
	})
	return out
}
