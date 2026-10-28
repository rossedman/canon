package profiles

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rossedman/canon/canon"
	"github.com/rossedman/canon/internal/records"
)

func loadEmbedded(t *testing.T) []*Profile {
	t.Helper()
	ps, err := Load(canon.FS, "profiles")
	if err != nil {
		t.Fatal(err)
	}
	return ps
}

func TestDetect(t *testing.T) {
	t.Parallel()
	ps := loadEmbedded(t)
	tests := []struct {
		name string
		repo fstest.MapFS
		want string
	}{
		{"go cli", fstest.MapFS{"go.mod": {}, "cmd/root.go": {}}, "go-cli"},
		{"go service", fstest.MapFS{"go.mod": {}, "api/openapi.yaml": {}}, "go-service"},
		{"go library", fstest.MapFS{"go.mod": {}}, "go-library"},
		{"python", fstest.MapFS{"pyproject.toml": {}}, "py-service"},
		{"terraform", fstest.MapFS{"main.tf": {}}, "tf-module"},
		{"anything", fstest.MapFS{"README.md": {}}, "generic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, ok := DetectProfile(tt.repo, ps)
			if !ok || p.Name != tt.want {
				t.Fatalf("got %v, want %s", p, tt.want)
			}
		})
	}
	if _, ok := DetectProfile(fstest.MapFS{}, []*Profile{{Name: "x", Detect: Detect{All: []string{"go.mod"}}}}); ok {
		t.Fatal("matched without a catch-all")
	}
	if p, ok := Find(ps, "go-cli"); !ok || p.Name != "go-cli" {
		t.Fatal("Find failed")
	}
	if _, ok := Find(ps, "nope"); ok {
		t.Fatal("Find found a ghost")
	}
}

func TestMatchesDirectoryEntry(t *testing.T) {
	t.Parallel()
	p := &Profile{Detect: Detect{All: []string{"cmd/"}}}
	if p.Matches(fstest.MapFS{"cmd": {}}) {
		t.Fatal("a file named cmd is not a directory")
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	set, err := records.Load(canon.FS, "records")
	if err != nil {
		t.Fatal(err)
	}
	ps := loadEmbedded(t)
	lib, _ := Find(ps, "go-library")
	ids := map[string]bool{}
	for _, r := range lib.Resolve(set) {
		ids[r.ID] = true
	}
	if ids["go-cobra-one-command-per-file"] || !ids["go-errors-wrap"] || ids["python-ruff"] || ids["containers-alpine-base"] {
		t.Fatalf("go-library resolved %v", ids)
	}
	extra := &Profile{Name: "x", Records: []string{"python-ruff"}}
	if got := extra.Resolve(set); len(got) != 1 || got[0].ID != "python-ruff" {
		t.Fatalf("named records not added: %v", got)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()
	if _, err := Load(fstest.MapFS{}, "profiles"); err == nil {
		t.Fatal("want error for missing dir")
	}
	if _, err := Load(fstest.MapFS{"p/a.yaml": {Data: []byte("name: [")}}, "p"); err == nil {
		t.Fatal("want yaml error")
	}
	if _, err := Load(fstest.MapFS{"p/a.yaml": {Data: []byte("priority: 1")}}, "p"); err == nil {
		t.Fatal("want missing name error")
	}
	ps, err := Load(fstest.MapFS{"p/a.txt": {}, "p/sub/b.yaml": {}, "p/c.yml": {Data: []byte("name: c")}}, "p")
	if err != nil || len(ps) != 1 {
		t.Fatalf("got %v, %v", ps, err)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	set, _ := records.NewSet(nil)
	ps := []*Profile{{Name: "x", Detect: Detect{All: []string{"go.mod"}}, Scopes: []string{"cobol"}, Records: []string{"ghost"}}}
	var text strings.Builder
	for _, p := range Validate(ps, set) {
		text.WriteString(p.Message + "\n")
	}
	for _, want := range []string{`scope "cobol"`, `record "ghost"`, "no profile without detect rules"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("missing %q in %s", want, text.String())
		}
	}
	allSet, _ := records.Load(canon.FS, "records")
	if got := Validate(loadEmbedded(t), allSet); len(got) != 0 {
		t.Fatalf("embedded profiles invalid: %v", got)
	}
}
