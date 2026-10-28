// Package docgen writes the generated reference pages for the docs site:
// the CLI reference from cobra and one page per decision record.
package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/records"
)

// Commands writes one Markdown page per command into dir, replacing what
// was there so removed commands lose their page.
func Commands(root *cobra.Command, dir string) error {
	if err := reset(dir); err != nil {
		return err
	}
	root.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(root, dir); err != nil {
		return fmt.Errorf("generate command docs: %w", err)
	}
	return nil
}

// Records writes an index and one page per record into dir.
func Records(e *engine.Engine, dir string) error {
	if err := reset(dir); err != nil {
		return err
	}
	all := e.Records.All()
	for _, r := range all {
		if err := write(filepath.Join(dir, r.ID+".md"), RecordPage(e, r)); err != nil {
			return err
		}
	}
	return write(filepath.Join(dir, "index.md"), IndexPage(all))
}

// reset empties dir and recreates it.
func reset(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // docs are meant to be readable
		return fmt.Errorf("create %s: %w", dir, err)
	}
	return nil
}

// write saves a page.
func write(p, content string) error {
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil { //nolint:gosec // docs are meant to be readable
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}

// IndexPage renders the record index, grouped by scope.
func IndexPage(all []*records.Record) string {
	var b strings.Builder
	b.WriteString("# Decision records\n\nEvery rule canon enforces traces to one of these records. ")
	fmt.Fprintf(&b, "There are %d: %d accepted, and %d kept for history.\n", len(all), countInForce(all), len(all)-countInForce(all))
	for _, scope := range records.Scopes {
		var rs []*records.Record
		for _, r := range all {
			if len(r.Scope) > 0 && r.Scope[0] == scope {
				rs = append(rs, r)
			}
		}
		if len(rs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n| Record | Strength | Status |\n| --- | --- | --- |\n", scopeTitle(scope))
		for _, r := range rs {
			fmt.Fprintf(&b, "| [%s](%s.md) | %s | %s |\n", r.Title, r.ID, r.Strength, r.Status)
		}
	}
	return b.String()
}

// countInForce counts accepted records.
func countInForce(all []*records.Record) int {
	n := 0
	for _, r := range all {
		if r.InForce() {
			n++
		}
	}
	return n
}

// scopeTitle names a scope for a heading.
func scopeTitle(s string) string {
	switch s {
	case "ci":
		return "CI"
	case "oss":
		return "Open source"
	case "yaml":
		return "YAML"
	case "go":
		return "Go"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// RecordPage renders one record with its metadata and the rules enforcing it.
func RecordPage(e *engine.Engine, r *records.Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", r.Title)
	if r.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", r.Summary)
	}
	if !r.InForce() {
		by := e.Records.SupersededBy(r.ID)
		fmt.Fprintf(&b, "!!! warning \"%s\"\n    This record is no longer served as current advice.", titleCase(string(r.Status)))
		if len(by) > 0 {
			fmt.Fprintf(&b, " See %s.", linkList(by))
		}
		b.WriteString("\n\n")
	}
	b.WriteString("| | |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| ID | `%s` |\n| Strength | %s (%s) |\n| Status | %s |\n| Scope | %s |\n", r.ID, r.Strength, r.Strength.Severity(), r.Status, strings.Join(r.Scope, ", "))
	fmt.Fprintf(&b, "| Applies to | %s |\n", codeList(r.AppliesTo.Files))
	if len(r.Tags) > 0 {
		fmt.Fprintf(&b, "| Tags | %s |\n", strings.Join(r.Tags, ", "))
	}
	if len(r.Supersedes) > 0 {
		fmt.Fprintf(&b, "| Supersedes | %s |\n", linkList(r.Supersedes))
	}
	fmt.Fprintf(&b, "| Updated | %s |\n\n", r.Updated)
	for _, name := range r.Order {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", name, r.Sections[name])
	}
	rs := e.Rules.ForRecord(r.ID)
	b.WriteString("## Enforced by\n\n")
	if len(rs) == 0 {
		b.WriteString("No automated rule yet; reviewers and `canon decide` carry this one.\n")
	} else {
		b.WriteString("| Rule | Checker | What it checks |\n| --- | --- | --- |\n")
		for _, rule := range rs {
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", rule.ID, rule.Checker, rule.Description)
		}
	}
	fmt.Fprintf(&b, "\nRead it from an agent with `canon explain %s`, or the `canon://records/%s` resource.\n", r.ID, r.ID)
	return b.String()
}

// codeList renders values as inline code, comma separated, escaping pipes.
func codeList(vs []string) string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = "`" + strings.ReplaceAll(v, "|", "\\|") + "`"
	}
	return strings.Join(out, ", ")
}

// linkList renders record IDs as links to their pages.
func linkList(ids []string) string {
	sorted := slices.Clone(ids)
	slices.Sort(sorted)
	out := make([]string, len(sorted))
	for i, id := range sorted {
		out[i] = fmt.Sprintf("[%s](%s.md)", id, id)
	}
	return strings.Join(out, ", ")
}

// titleCase capitalizes the first letter.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
