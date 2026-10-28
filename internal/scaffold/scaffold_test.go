package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rossedman/canon/canon"
)

// templates returns the embedded templates directory.
func templates(t *testing.T) fstest.MapFS {
	t.Helper()
	return fstest.MapFS{
		"kit/template.yaml":         {Data: []byte("description: d\nroot: \"{{.name}}\"\nrecords: [r]\nvars:\n  module: \"example.com/{{.name}}\"\n  pkg: \"{{snake .name}}\"\nnext: [\"cd {{.name}}\"]\n")},
		"kit/{{.pkg}}/main.go.tmpl": {Data: []byte("package {{.pkg}} // {{.module}} {{upper .name}} {{title .name}} {{title \"\"}}")},
		"kit/static.txt":            {Data: []byte("{{ not rendered }}")},
		"bad/template.yaml":         {Data: []byte("vars: [")},
		"esc/template.yaml":         {Data: []byte("root: \"\"\n")},
		"esc/{{.name}}.tmpl":        {Data: []byte("x")},
		"miss/template.yaml":        {Data: []byte("root: \"\"\n")},
		"miss/a.tmpl":               {Data: []byte("{{.nope}}")},
		"parse/template.yaml":       {Data: []byte("root: \"{{\"\n")},
	}
}

func TestRender(t *testing.T) {
	t.Parallel()
	p, err := Render(templates(t), "kit", "my-tool", map[string]string{"module": "github.com/me/my-tool"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Root != "my-tool" || len(p.Next) != 1 || p.Next[0] != "cd my-tool" {
		t.Fatalf("plan = %+v", p)
	}
	paths := strings.Join(p.Paths(), ",")
	if paths != "my-tool/my_tool/main.go,my-tool/static.txt" {
		t.Fatalf("paths = %s", paths)
	}
	if got := string(p.Files[0].Content); got != "package my_tool // github.com/me/my-tool MY-TOOL My-tool " {
		t.Fatalf("content = %q", got)
	}
	if got := string(p.Files[1].Content); got != "{{ not rendered }}" {
		t.Fatalf("static file was rendered: %q", got)
	}
}

func TestRenderErrors(t *testing.T) {
	t.Parallel()
	fsys := templates(t)
	tests := []struct{ kind, name string }{
		{"kit", "Bad Name"},
		{"nope", "x"},
		{"../kit", "x"},
		{"bad", "x"},
		{"miss", "x"},
		{"parse", "x"},
	}
	for _, tt := range tests {
		if _, err := Render(fsys, tt.kind, tt.name, nil); err == nil {
			t.Errorf("Render(%s, %s): want error", tt.kind, tt.name)
		}
	}
	if _, err := Render(fsys, "nope", "x", nil); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("want ErrUnknownKind, got %v", err)
	}
	if _, err := Render(fsys, "esc", "x", map[string]string{"name": "../../etc"}); err != nil {
		t.Errorf("name override must be ignored: %v", err)
	}
	for _, p := range []string{"", "/abs", "../up", "a/../../up", `a\b`} {
		if safePath(p) {
			t.Errorf("safePath(%q) = true", p)
		}
	}
	if _, err := Load(fstest.MapFS{"x/template.yaml": {Data: nil}}, "x"); err != nil {
		t.Errorf("empty template.yaml should load: %v", err)
	}
	if _, err := Load(fstest.MapFS{"x/template.yaml/inner": {}}, "x"); err == nil {
		t.Error("unreadable template.yaml should fail")
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	ts, err := List(canon.FS)
	if err == nil || ts != nil {
		t.Fatal("the canon root is not a templates dir; expect a parse error from a non-template")
	}
	sub, _ := fsSub(t)
	ts, err = List(sub)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, tpl := range ts {
		kinds = append(kinds, tpl.Kind)
	}
	if strings.Join(kinds, ",") != "dockerfile,gha-workflow,go-cli,go-service,py-service,tf-module" {
		t.Fatalf("kinds = %v", kinds)
	}
	for _, k := range kinds {
		if _, err := Render(sub, k, "sample", nil); err != nil {
			t.Errorf("%s: %v", k, err)
		}
	}
}

// fsSub returns the embedded templates directory.
func fsSub(t *testing.T) (fstest.MapFS, error) {
	t.Helper()
	out := fstest.MapFS{}
	entries, err := canonFiles("templates")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range entries {
		out[strings.TrimPrefix(name, "templates/")] = &fstest.MapFile{Data: data}
	}
	return out, nil
}

func TestWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := &Plan{Files: []File{{Path: "a/b.txt", Content: []byte("x")}, {Path: "c.txt", Content: []byte("y")}}}
	if err := p.Write(dir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a/b.txt")); string(b) != "x" {
		t.Fatal("file not written")
	}
	if err := p.Write(dir); !errors.Is(err, ErrExists) {
		t.Fatalf("overwrite not refused: %v", err)
	}
	blocked := &Plan{Files: []File{{Path: "c.txt/inside", Content: nil}}}
	if err := blocked.Write(dir); err == nil {
		t.Fatal("mkdir over a file should fail")
	}
	ro := filepath.Join(dir, "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() != 0 {
		if err := (&Plan{Files: []File{{Path: "f"}}}).Write(ro); err == nil {
			t.Fatal("write into read-only dir should fail")
		}
	}
}
