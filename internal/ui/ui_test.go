package ui

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/records"
	"github.com/rossedman/canon/internal/review"
	"github.com/rossedman/canon/internal/rules"
)

// sample findings across all severities, one with a multi-line fix.
var sample = []findings.Finding{
	{Rule: "r1", Record: "rec", Severity: findings.Error, Path: "Dockerfile", Line: 3, Column: 2, Message: "bad base", Why: "because", Fix: &findings.Fix{Suggestion: "line one\nline two"}},
	{Rule: "r2", Record: "rec", Severity: findings.Warning, Path: "a.go", Line: 1, Message: "warn", Why: "w"},
	{Rule: "r3", Record: "rec", Severity: findings.Suggestion, Path: "LICENSE", Message: "missing", Why: "s"},
}

// render runs fn and returns its plain output; a bytes.Buffer is not a
// terminal, so Writer strips every style.
func render(t *testing.T, fn func(w io.Writer) error) string {
	t.Helper()
	var b bytes.Buffer
	if err := fn(Writer(&b)); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// contains fails unless out has every want.
func contains(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in\n%s", w, out)
		}
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()
	out := render(t, func(b io.Writer) error {
		return Check(b, &engine.CheckResult{Findings: sample, Summary: findings.Summarize(sample), FilesChecked: 1, Profile: "go-cli", Skipped: []engine.Skipped{{Checker: "ruff", Reason: "ruff is not installed"}}})
	})
	contains(t, out, "Dockerfile:3:2 error bad base", "r1 → rec: because", "fix: line one\n       line two", "a.go:1 warning", "LICENSE suggestion", "skipped ruff: ruff is not installed", "canon 1 error, 1 warning, 1 suggestion in 1 file (profile go-cli)")
	if strings.Contains(out, "\x1b[") {
		t.Fatal("styles leaked into non-terminal output")
	}
	clean := render(t, func(b io.Writer) error { return Check(b, &engine.CheckResult{FilesChecked: 2, Profile: "generic"}) })
	contains(t, clean, "no findings in 2 files")
}

func TestDecideExplainProfile(t *testing.T) {
	t.Parallel()
	out := render(t, func(b io.Writer) error {
		return Decide(b, &engine.DecideResult{Gap: true, Message: "nothing", Matches: []engine.Decision{
			{ID: "a", Title: "A", Strength: "must", Decision: "Do A.\nAlways.", Exceptions: "Rarely\nnever."}, {ID: "b", Title: "B"},
		}})
	})
	contains(t, out, "gap nothing", "a (must) A", "  Do A.\n  Always.", "exceptions: Rarely never.", "\nb () B")
	rec := &records.Record{ID: "rec", Title: "T", Status: "accepted", Strength: "must", Body: "## Context\nx", Summary: "why"}
	out = render(t, func(b io.Writer) error {
		return Explain(b, &engine.ExplainResult{Rule: &rules.Rule{ID: "r", Description: "d", Checker: "canon"}, Record: rec, Why: "why",
			Rules: []*rules.Rule{{ID: "r", Checker: "canon", Description: "d"}}, Supersedes: []engine.RecordRef{{ID: "old", Title: "Old"}}, SupersededBy: []engine.RecordRef{{ID: "new", Title: "New"}}})
	})
	contains(t, out, "r d (canon)", "rec (accepted, must) T", "Enforced by", "supersedes old Old", "superseded by new New")
	out = render(t, func(b io.Writer) error {
		return Profile(b, &engine.ProfileResult{Profile: "go-cli", Description: "d", Records: []engine.RecordRef{{ID: "x", Strength: "must", Title: "X"}},
			Checkers: []engine.CheckerStatus{{Name: "canon", Installed: true}, {Name: "hadolint"}}})
	})
	contains(t, out, "profile go-cli", "1 record in force", "canon          installed", "hadolint       missing")
}

func TestReviewScaffoldLint(t *testing.T) {
	t.Parallel()
	out := render(t, func(b io.Writer) error {
		return Review(b, &engine.ReviewResult{Findings: sample[:1], Summary: findings.Summarize(sample[:1]), Files: []string{"a", "b"},
			Concerns:       []review.Concern{{Kind: "new-dockerfile", Path: "Dockerfile", Line: 1, Detail: "adds a Dockerfile", Record: "rec"}, {Kind: "missing-tests", Path: "pkg", Detail: "d", Record: "r2"}},
			RecordsTouched: []string{"rec", "r2"}, Skipped: []engine.Skipped{{Checker: "x", Reason: "y"}}})
	})
	contains(t, out, "Concerns", "new-dockerfile Dockerfile:1 adds a Dockerfile → rec", "missing-tests pkg d", "canon review 1 error, 0 warnings, 0 suggestions, 2 concerns across 2 files", "records touched: rec, r2")
	out = render(t, func(b io.Writer) error {
		return Scaffold(b, &engine.ScaffoldResult{Kind: "go-cli", DryRun: true, Files: []string{"a/go.mod"}, Records: []engine.RecordRef{{ID: "r"}}, Next: []string{"go mod tidy"}})
	})
	contains(t, out, "go-cli would write 1 file", "records applied: r", "dry run: pass --write", "next: go mod tidy")
	out = render(t, func(b io.Writer) error {
		return Scaffold(b, &engine.ScaffoldResult{Kind: "x", Check: &engine.CheckResult{}})
	})
	contains(t, out, "x wrote 0 files", "check: no findings")
	out = render(t, func(b io.Writer) error {
		return Lint(b, []engine.Problem{{Kind: "rule", Ref: "r", Message: "bad"}, {Path: "p.md", Message: "worse"}}, 3)
	})
	contains(t, out, "rule r bad", "p.md worse", "2 problems")
	contains(t, render(t, func(b io.Writer) error { return Lint(b, nil, 3) }), "ok 3 records are valid")
}

// failWriter fails every write.
type failWriter struct{}

// Write always fails.
func (failWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestWriteErrors(t *testing.T) {
	t.Parallel()
	w := failWriter{}
	calls := map[string]error{
		"findings": Findings(w, sample),
		"check":    Check(w, &engine.CheckResult{Findings: sample}),
		"check2":   Check(w, &engine.CheckResult{}),
		"decide":   Decide(w, &engine.DecideResult{}),
		"explain":  Explain(w, &engine.ExplainResult{Record: &records.Record{}}),
		"profile":  Profile(w, &engine.ProfileResult{}),
		"review":   Review(w, &engine.ReviewResult{Findings: sample}),
		"review2":  Review(w, &engine.ReviewResult{}),
		"scaffold": Scaffold(w, &engine.ScaffoldResult{}),
		"lint":     Lint(w, nil, 0),
	}
	for name, err := range calls {
		if err == nil {
			t.Errorf("%s: write error swallowed", name)
		}
	}
}

func TestLogger(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	NewLogger(&b, slog.LevelInfo).Info("hello", "k", "v")
	contains(t, b.String(), "canon", "hello", "k=v")
}
