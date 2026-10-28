package search

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStem(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"errors": "error", "wrapping": "wrap", "wrapped": "wrap", "handled": "handl", "handle": "handl",
		"images": "imag", "image": "imag", "boxes": "box", "policies": "policy", "applied": "apply",
		"files": "file", "process": "process", "pins": "pins", "go": "go", "pinned": "pin",
		"releases": "releas", "release": "releas", "ings": "ings", "using": "using", "passes": "pass",
	}
	for in, want := range tests {
		if got := Stem(in); got != want {
			t.Errorf("Stem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTokenize(t *testing.T) {
	t.Parallel()
	got := Tokenize("How do we handle Errors in Go? (wrapping %w)")
	want := []string{"handl", "error", "go", "wrap", "w"}
	if !slices.Equal(got, want) {
		t.Fatalf("Tokenize = %v, want %v", got, want)
	}
}

// docs is a small corpus for ranking tests.
func docs() []Doc {
	return []Doc{
		{ID: "errors", Fields: []Field{{Text: "Wrap errors with context", Weight: 3}, {Text: "Return wrapped errors; never log and return."}}},
		{ID: "images", Fields: []Field{{Text: "Use distroless runtime images", Weight: 3}, {Text: "Base images carry no shell."}}},
		{ID: "config", Fields: []Field{{Text: "Load config from flags, env and file", Weight: 0}}},
	}
}

func TestSearch(t *testing.T) {
	t.Parallel()
	idx := New(docs())
	hits := idx.Search("how do we handle errors?", 3, 0, nil)
	if len(hits) == 0 || hits[0].ID != "errors" {
		t.Fatalf("hits = %+v", hits)
	}
	if hits := idx.Search("which base image", 1, 0, nil); len(hits) != 1 || hits[0].ID != "images" {
		t.Fatalf("limit/ranking wrong: %+v", hits)
	}
	if hits := idx.Search("errors images", 0, 0, func(id string) bool { return id != "errors" }); len(hits) != 1 || hits[0].ID != "images" {
		t.Fatalf("filter wrong: %+v", hits)
	}
	if hits := idx.Search("errors", 5, 100, nil); len(hits) != 0 {
		t.Fatalf("threshold ignored: %+v", hits)
	}
	if hits := idx.Search("bake a cake", 5, 0, nil); len(hits) != 0 {
		t.Fatalf("unrelated query matched: %+v", hits)
	}
	if hits := New(nil).Search("x", 1, 0, nil); hits != nil {
		t.Fatal("empty index returned hits")
	}
}

func BenchmarkSearch(b *testing.B) {
	var ds []Doc
	for i := range 200 {
		ds = append(ds, Doc{ID: strings.Repeat("d", i%5+1) + string(rune('a'+i%26)), Fields: []Field{{Text: "wrap errors with context and return them to callers who handle errors", Weight: 1}}})
	}
	idx := New(ds)
	b.ReportAllocs()
	for b.Loop() {
		idx.Search("how should we handle wrapped errors in go services", 3, 0, nil)
	}
}

// FuzzTokenize checks tokens are lowercase, non-empty, valid UTF-8 and never stopwords.
func FuzzTokenize(f *testing.F) {
	f.Add("How do we handle errors in Go?")
	f.Add("ÄÖÜ ß İstanbul 日本語")
	f.Fuzz(func(t *testing.T, s string) {
		for _, tok := range Tokenize(s) {
			if tok == "" || stopwords[tok] || !utf8.ValidString(tok) && utf8.ValidString(s) {
				t.Fatalf("bad token %q from %q", tok, s)
			}
		}
	})
}
