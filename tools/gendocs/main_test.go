package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	out := t.TempDir()
	if err := run(out); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"commands/canon_check.md", "records/index.md", "records/go-errors-wrap.md"} {
		if _, err := os.Stat(filepath.Join(out, p)); err != nil {
			t.Error(err)
		}
	}
	file := filepath.Join(out, "file")
	_ = os.WriteFile(file, nil, 0o600)
	if err := run(file); err == nil {
		t.Fatal("output under a file: want error")
	}
}
