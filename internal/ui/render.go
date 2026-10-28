package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/findings"
)

// flush writes a rendered block to w.
func flush(w io.Writer, b *strings.Builder) error {
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// severityStyle returns the label style for a severity.
func severityStyle(s findings.Severity) string {
	switch s {
	case findings.Error:
		return ErrorText.Render("error")
	case findings.Warning:
		return WarnText.Render("warning")
	}
	return SuggestTxt.Render("suggestion")
}

// location renders path:line:col.
func location(f findings.Finding) string {
	loc := f.Path
	if f.Line > 0 {
		loc += fmt.Sprintf(":%d", f.Line)
		if f.Column > 0 {
			loc += fmt.Sprintf(":%d", f.Column)
		}
	}
	return loc
}

// Findings prints findings, one block each, followed by a summary line.
func Findings(w io.Writer, fs []findings.Finding) error {
	var b strings.Builder
	for _, f := range fs {
		fmt.Fprintf(&b, "%s %s %s\n", Heading.Render(location(f)), severityStyle(f.Severity), f.Message)
		fmt.Fprintf(&b, "  %s %s\n", Muted.Render(f.Rule+" → "+f.Record+":"), f.Why)
		if f.Fix != nil {
			lines := strings.Split(f.Fix.Suggestion, "\n")
			for i, l := range lines {
				lines[i] = Code.Render(l)
			}
			fmt.Fprintf(&b, "  %s %s\n", Muted.Render("fix:"), strings.Join(lines, "\n       "))
		}
	}
	return flush(w, &b)
}

// summaryLine renders the error/warning/suggestion counts.
func summaryLine(s findings.Summary) string {
	if s.Total() == 0 {
		return Success.Render("no findings")
	}
	return fmt.Sprintf("%s, %s, %s",
		ErrorText.Render(plural(s.Error, "error")),
		WarnText.Render(plural(s.Warning, "warning")),
		SuggestTxt.Render(plural(s.Suggestion, "suggestion")))
}

// plural renders "1 error" or "2 errors".
func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// skipped renders the checkers that did not run.
func skipped(b *strings.Builder, ss []engine.Skipped) {
	for _, s := range ss {
		fmt.Fprintf(b, "%s %s: %s\n", Muted.Render("skipped"), s.Checker, s.Reason)
	}
}

// Check prints a check result.
func Check(w io.Writer, r *engine.CheckResult) error {
	if err := Findings(w, r.Findings); err != nil {
		return err
	}
	var b strings.Builder
	if len(r.Findings) > 0 {
		b.WriteByte('\n')
	}
	skipped(&b, r.Skipped)
	fmt.Fprintf(&b, "%s %s in %s (profile %s)\n", Title.Render("canon"), summaryLine(r.Summary), plural(r.FilesChecked, "file"), r.Profile)
	return flush(w, &b)
}

// Decide prints the records that answer a question.
func Decide(w io.Writer, r *engine.DecideResult) error {
	var b strings.Builder
	if r.Gap {
		fmt.Fprintf(&b, "%s %s\n", WarnText.Render("gap"), r.Message)
	}
	for i, m := range r.Matches {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s %s %s\n", Title.Render(m.ID), Muted.Render("("+string(m.Strength)+")"), Heading.Render(m.Title))
		fmt.Fprintf(&b, "%s\n", indent(m.Decision))
		if m.Exceptions != "" {
			fmt.Fprintf(&b, "  %s %s\n", Muted.Render("exceptions:"), strings.Join(strings.Fields(m.Exceptions), " "))
		}
	}
	return flush(w, &b)
}

// indent prefixes every line with two spaces.
func indent(s string) string {
	return "  " + strings.ReplaceAll(strings.TrimSpace(s), "\n", "\n  ")
}

// Explain prints a record in full with the rules that enforce it.
func Explain(w io.Writer, r *engine.ExplainResult) error {
	var b strings.Builder
	if r.Rule != nil {
		fmt.Fprintf(&b, "%s %s (%s)\n\n", Title.Render(r.Rule.ID), r.Rule.Description, r.Rule.Checker)
	}
	fmt.Fprintf(&b, "%s %s %s\n", Title.Render(r.Record.ID), Muted.Render("("+string(r.Record.Status)+", "+string(r.Record.Strength)+")"), Heading.Render(r.Record.Title))
	fmt.Fprintf(&b, "%s\n\n%s\n", Muted.Render(r.Why), strings.TrimSpace(r.Record.Body))
	if len(r.Rules) > 0 {
		fmt.Fprintf(&b, "\n%s\n", Heading.Render("Enforced by"))
		for _, rule := range r.Rules {
			fmt.Fprintf(&b, "  %s %s %s\n", Code.Render(rule.ID), Muted.Render("("+rule.Checker+")"), rule.Description)
		}
	}
	for _, s := range r.Supersedes {
		fmt.Fprintf(&b, "%s %s %s\n", Muted.Render("supersedes"), s.ID, s.Title)
	}
	for _, s := range r.SupersededBy {
		fmt.Fprintf(&b, "%s %s %s\n", WarnText.Render("superseded by"), s.ID, s.Title)
	}
	return flush(w, &b)
}

// Profile prints the detected profile, its records and missing checkers.
func Profile(w io.Writer, r *engine.ProfileResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n%s\n\n", Title.Render("profile"), Heading.Render(r.Profile), Muted.Render(r.Description))
	fmt.Fprintf(&b, "%s\n", Heading.Render(plural(len(r.Records), "record")+" in force"))
	for _, rec := range r.Records {
		fmt.Fprintf(&b, "  %-34s %s %s\n", rec.ID, Muted.Render(fmt.Sprintf("%-6s", rec.Strength)), rec.Title)
	}
	fmt.Fprintf(&b, "\n%s\n", Heading.Render("Checkers"))
	for _, c := range r.Checkers {
		state := Success.Render("installed")
		if !c.Installed {
			state = WarnText.Render("missing")
		}
		fmt.Fprintf(&b, "  %-14s %s\n", c.Name, state)
	}
	return flush(w, &b)
}

// Review prints findings and concerns for a change.
func Review(w io.Writer, r *engine.ReviewResult) error {
	if err := Findings(w, r.Findings); err != nil {
		return err
	}
	var b strings.Builder
	if len(r.Concerns) > 0 {
		if len(r.Findings) > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s\n", Heading.Render("Concerns"))
		for _, c := range r.Concerns {
			loc := c.Path
			if c.Line > 0 {
				loc += fmt.Sprintf(":%d", c.Line)
			}
			fmt.Fprintf(&b, "  %s %s %s %s\n", Code.Render(c.Kind), loc, c.Detail, Muted.Render("→ "+c.Record))
		}
	}
	b.WriteByte('\n')
	skipped(&b, r.Skipped)
	fmt.Fprintf(&b, "%s %s, %s across %s\n", Title.Render("canon review"), summaryLine(r.Summary), plural(len(r.Concerns), "concern"), plural(len(r.Files), "file"))
	if len(r.RecordsTouched) > 0 {
		fmt.Fprintf(&b, "%s %s\n", Muted.Render("records touched:"), strings.Join(r.RecordsTouched, ", "))
	}
	return flush(w, &b)
}

// Scaffold prints the files a scaffold wrote or would write.
func Scaffold(w io.Writer, r *engine.ScaffoldResult) error {
	var b strings.Builder
	verb := "would write"
	if !r.DryRun {
		verb = "wrote"
	}
	fmt.Fprintf(&b, "%s %s %s\n", Title.Render(r.Kind), verb, plural(len(r.Files), "file"))
	for _, f := range r.Files {
		fmt.Fprintf(&b, "  %s\n", f)
	}
	if len(r.Records) > 0 {
		ids := make([]string, 0, len(r.Records))
		for _, rec := range r.Records {
			ids = append(ids, rec.ID)
		}
		fmt.Fprintf(&b, "%s %s\n", Muted.Render("records applied:"), strings.Join(ids, ", "))
	}
	if r.Check != nil {
		fmt.Fprintf(&b, "%s %s\n", Muted.Render("check:"), summaryLine(r.Check.Summary))
	}
	if r.DryRun {
		fmt.Fprintf(&b, "%s\n", Muted.Render("dry run: pass --write to create these files"))
	}
	for _, n := range r.Next {
		fmt.Fprintf(&b, "%s %s\n", Muted.Render("next:"), Code.Render(n))
	}
	return flush(w, &b)
}

// Lint prints canon validation problems.
func Lint(w io.Writer, ps []engine.Problem, records int) error {
	var b strings.Builder
	for _, p := range ps {
		where := p.Path
		if where == "" {
			where = p.Kind + " " + p.Ref
		}
		fmt.Fprintf(&b, "%s %s\n", ErrorText.Render(where), p.Message)
	}
	if len(ps) == 0 {
		fmt.Fprintf(&b, "%s %s are valid\n", Success.Render("ok"), plural(records, "record"))
	} else {
		fmt.Fprintf(&b, "%s\n", ErrorText.Render(plural(len(ps), "problem")))
	}
	return flush(w, &b)
}
