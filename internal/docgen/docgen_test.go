package docgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rossedman/canon/internal/engine"
)

func TestRecordsAndCommands(t *testing.T) {
	t.Parallel()
	e, err := engine.Default()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Records(e, filepath.Join(dir, "records")); err != nil {
		t.Fatal(err)
	}
	idx, _ := os.ReadFile(filepath.Join(dir, "records", "index.md"))
	for _, want := range []string{"## Go", "## CI", "## Open source", "## YAML", "## Containers", "[Wrap errors with %w and context; never log and return](go-errors-wrap.md)", "24 accepted"} {
		if !strings.Contains(string(idx), want) {
			t.Errorf("index missing %q", want)
		}
	}
	page, _ := os.ReadFile(filepath.Join(dir, "records", "containers-distroless-base.md"))
	for _, want := range []string{"# Use distroless or scratch", "| Supersedes | [containers-alpine-base](containers-alpine-base.md) |", "`canon-final-stage-base`", "## Exceptions", "canon explain containers-distroless-base"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("page missing %q", want)
		}
	}
	old, _ := os.ReadFile(filepath.Join(dir, "records", "containers-alpine-base.md"))
	if !strings.Contains(string(old), `!!! warning "Superseded"`) || !strings.Contains(string(old), "No automated rule yet") {
		t.Errorf("superseded page wrong:\n%s", old)
	}
	root := &cobra.Command{Use: "tool"}
	root.AddCommand(&cobra.Command{Use: "sub", Run: func(*cobra.Command, []string) {}})
	if err := Commands(root, filepath.Join(dir, "commands")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "commands", "tool_sub.md")); err != nil {
		t.Fatal(err)
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	e, _ := engine.Default()
	if err := Records(e, filepath.Join(file, "sub")); err == nil {
		t.Error("records under a file: want error")
	}
	if err := Commands(&cobra.Command{Use: "x"}, filepath.Join(file, "sub")); err == nil {
		t.Error("commands under a file: want error")
	}
	if err := write(filepath.Join(dir, "missing", "x.md"), ""); err == nil {
		t.Error("write into missing dir: want error")
	}
	if titleCase("") != "" || scopeTitle("python") != "Python" {
		t.Error("helpers wrong")
	}
}
