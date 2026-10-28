package records

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rossedman/canon/canon"
	"github.com/rossedman/canon/internal/findings"
)

// valid is a minimal record that passes validation.
const valid = `---
id: go-thing
title: Do the thing
summary: Because it is better.
status: accepted
scope: [go]
applies_to:
  files: ["**/*.go"]
tags: [x]
rules: [r1]
strength: must
updated: 2026-10-27
---

## Context
Why.

## Decision
Do the thing. Then more.

## Consequences
Some.

## Examples
` + "```md\n## Not a heading\n```\n"

func TestParse(t *testing.T) {
	t.Parallel()
	r, err := Parse("go/go-thing.md", []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "go-thing" || r.Strength != Must || r.Status != Accepted || r.Path != "go/go-thing.md" {
		t.Fatalf("frontmatter = %+v", r)
	}
	if got := r.Order; strings.Join(got, ",") != "Context,Decision,Consequences,Examples" {
		t.Fatalf("sections = %v", got)
	}
	if !strings.Contains(r.Section("examples"), "## Not a heading") {
		t.Fatal("heading inside a code fence split the section")
	}
	if r.Why() != "Because it is better." || r.Markdown() != valid {
		t.Fatalf("Why = %q", r.Why())
	}
	if !r.Matches("a/b.go") || r.Matches("a/b.py") || !r.InForce() {
		t.Fatal("Matches/InForce wrong")
	}
	if !r.HasScope(nil) || !r.HasScope([]string{"python", "go"}) || r.HasScope([]string{"python"}) {
		t.Fatal("HasScope wrong")
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"no frontmatter": "# hi",
		"unclosed":       "---\nid: x\n",
		"bad yaml":       "---\nid: [\n---\n",
		"unknown field":  "---\nid: x\nbogus: 1\n---\nbody",
	}
	for name, in := range tests {
		if _, err := Parse("x.md", []byte(in)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if _, err := Parse("x.md", []byte("# hi")); !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("want ErrNoFrontmatter, got %v", err)
	}
}

func TestParseEdgeFrontmatter(t *testing.T) {
	t.Parallel()
	r, err := Parse("x.md", []byte("\ufeff---\r\n---\r\nbody"))
	if err != nil || r.Body != "body" {
		t.Fatalf("empty frontmatter: %v %+v", err, r)
	}
	r, err = Parse("x.md", []byte("---\nid: only\n---"))
	if err != nil || r.ID != "only" {
		t.Fatalf("frontmatter at EOF: %v %+v", err, r)
	}
}

func TestWhyFallsBackToDecision(t *testing.T) {
	t.Parallel()
	r := &Record{Sections: map[string]string{"Decision": "Use X for\nall Y. Then Z."}}
	if got := r.Why(); got != "Use X for all Y." {
		t.Fatalf("Why = %q", got)
	}
	r = &Record{Sections: map[string]string{"Decision": "No full stop"}}
	if got := r.Why(); got != "No full stop" {
		t.Fatalf("Why = %q", got)
	}
}

func TestStrengthSeverity(t *testing.T) {
	t.Parallel()
	if Must.Severity() != findings.Error || Should.Severity() != findings.Warning || Prefer.Severity() != findings.Suggestion {
		t.Fatal("strength to severity mapping is wrong")
	}
}

func TestLoadAndSet(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"records/go/go-thing.md": {Data: []byte(valid)},
		"records/go/README.txt":  {Data: []byte("ignored")},
		"records/go/old.md":      {Data: []byte(strings.NewReplacer("id: go-thing", "id: old", "status: accepted", "status: superseded").Replace(valid))},
	}
	set, err := Load(fsys, "records")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.All()) != 2 {
		t.Fatalf("loaded %d records", len(set.All()))
	}
	if _, ok := set.Get("go-thing"); !ok {
		t.Fatal("Get failed")
	}
	if _, err := Load(fsys, "missing"); err == nil {
		t.Fatal("want error for missing dir")
	}
	bad := fstest.MapFS{"records/x.md": {Data: []byte("nope")}}
	if _, err := Load(bad, "records"); err == nil {
		t.Fatal("want parse error")
	}
	a, _ := Parse("a.md", []byte(valid))
	b, _ := Parse("b.md", []byte(valid))
	if _, err := NewSet([]*Record{a, b}); err == nil {
		t.Fatal("want duplicate id error")
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	mk := func(path, id string, edit func(*Record)) *Record {
		r, err := Parse(path, []byte(strings.Replace(valid, "id: go-thing", "id: "+id, 1)))
		if err != nil {
			t.Fatal(err)
		}
		edit(r)
		return r
	}
	rs := []*Record{
		mk("go/go-thing.md", "go-thing", func(*Record) {}),
		mk("go/wrong-name.md", "Bad_ID", func(r *Record) {
			r.Title, r.Status, r.Strength, r.Updated = "", "maybe", "always", "today"
			r.Scope, r.AppliesTo.Files = []string{"cobol"}, nil
			delete(r.Sections, "Consequences")
			r.Supersedes = []string{"ghost"}
		}),
		mk("go/a.md", "a", func(r *Record) { r.Supersedes = []string{"b"}; r.Rules = []string{"missing-rule"} }),
		mk("go/b.md", "b", func(r *Record) { r.Supersedes = []string{"a"}; r.Scope = nil }),
		mk("go/orphan.md", "orphan", func(r *Record) { r.Status = Superseded }),
	}
	set, err := NewSet(rs)
	if err != nil {
		t.Fatal(err)
	}
	problems := Validate(set, func(id string) bool { return id == "r1" })
	text := ""
	for _, p := range problems {
		text += p.String() + "\n"
	}
	for _, want := range []string{
		`id "Bad_ID" is not kebab-case`, `file name "wrong-name.md" does not match`, "title is empty",
		`status "maybe"`, `strength "always"`, `scope "cobol"`, "applies_to.files is empty", `updated "today"`,
		`missing required section "## Consequences"`, `supersedes unknown record "ghost"`,
		`rule "missing-rule" is not defined`, "scope is empty", "supersedes cycle: a -> b -> a",
		"status is superseded but no record supersedes it",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing problem %q in:\n%s", want, text)
		}
	}
	if n := len(Validate(set, nil)); n == 0 {
		t.Fatal("nil ruleExists should still validate fields")
	}
}

func TestEmbeddedCanonIsValid(t *testing.T) {
	t.Parallel()
	set, err := Load(canon.FS, "records")
	if err != nil {
		t.Fatal(err)
	}
	if ps := Validate(set, nil); len(ps) > 0 {
		t.Fatalf("embedded records have problems: %v", ps)
	}
	if by := set.SupersededBy("containers-alpine-base"); len(by) != 1 || by[0] != "containers-distroless-base" {
		t.Fatalf("SupersededBy = %v", by)
	}
}

func BenchmarkParse(b *testing.B) {
	data := []byte(valid)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Parse("go/go-thing.md", data); err != nil {
			b.Fatal(err)
		}
	}
}

// FuzzParse checks Parse never panics and that what it parses survives
// being parsed again from its own Markdown.
func FuzzParse(f *testing.F) {
	f.Add([]byte(valid))
	f.Add([]byte("---\n---\n"))
	f.Add([]byte("---\nid: x\n---\n## A\n```\n## B\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := Parse("f.md", data)
		if err != nil {
			return
		}
		again, err := Parse("f.md", []byte(r.Markdown()))
		if err != nil {
			t.Fatalf("re-parse of accepted input failed: %v", err)
		}
		if again.ID != r.ID || again.Body != r.Body || len(again.Order) != len(r.Order) {
			t.Fatal("re-parse changed the record")
		}
		for _, name := range r.Order {
			if _, ok := r.Sections[name]; !ok {
				t.Fatalf("section %q listed but missing", name)
			}
		}
	})
}
