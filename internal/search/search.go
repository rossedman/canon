// Package search ranks documents against a free-text question with BM25, so
// "how do we handle errors in Go?" finds the record about wrapping errors.
package search

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
)

// BM25 tuning constants, the usual defaults.
const (
	k1 = 1.2
	b  = 0.75
)

// Field is weighted text in a document; weight repeats its terms so that a
// match in the title counts more than one in the body.
type Field struct {
	Text   string
	Weight int
}

// Doc is a document to index.
type Doc struct {
	ID     string
	Fields []Field
}

// Hit is a scored match.
type Hit struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

// Index is an in-memory BM25 index.
type Index struct {
	ids    []string
	tf     []map[string]int
	length []int
	df     map[string]int
	avgLen float64
}

// New builds an index over docs.
func New(docs []Doc) *Index {
	idx := &Index{df: map[string]int{}}
	total := 0
	for _, d := range docs {
		tf := map[string]int{}
		n := 0
		for _, f := range d.Fields {
			w := max(f.Weight, 1)
			for _, t := range Tokenize(f.Text) {
				tf[t] += w
				n += w
			}
		}
		for t := range tf {
			idx.df[t]++
		}
		idx.ids = append(idx.ids, d.ID)
		idx.tf = append(idx.tf, tf)
		idx.length = append(idx.length, n)
		total += n
	}
	if len(docs) > 0 {
		idx.avgLen = float64(total) / float64(len(docs))
	}
	return idx
}

// Search scores every document against query and returns those scoring at
// least minScore, best first, at most limit (no limit when limit <= 0). keep,
// when non-nil, filters documents by ID before scoring.
func (idx *Index) Search(query string, limit int, minScore float64, keep func(id string) bool) []Hit {
	terms := unique(Tokenize(query))
	n := float64(len(idx.ids))
	var hits []Hit
	for i, id := range idx.ids {
		if keep != nil && !keep(id) {
			continue
		}
		score := 0.0
		for _, t := range terms {
			f := float64(idx.tf[i][t])
			if f == 0 {
				continue
			}
			df := float64(idx.df[t])
			idf := math.Log(1 + (n-df+0.5)/(df+0.5))
			norm := f * (k1 + 1) / (f + k1*(1-b+b*float64(idx.length[i])/idx.avgLen))
			score += idf * norm
		}
		if score > 0 && score >= minScore {
			hits = append(hits, Hit{ID: id, Score: math.Round(score*1000) / 1000})
		}
	}
	slices.SortStableFunc(hits, func(a, b Hit) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.ID, b.ID))
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// unique drops repeated terms so a question that says "error" twice does not
// count it twice.
func unique(ts []string) []string {
	seen := map[string]bool{}
	out := ts[:0:0]
	for _, t := range ts {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// stopwords are dropped from queries and documents.
var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true,
	"by": true, "do": true, "does": true, "for": true, "from": true, "how": true, "i": true,
	"in": true, "is": true, "it": true, "its": true, "of": true, "on": true, "or": true,
	"our": true, "should": true, "so": true, "that": true, "the": true, "this": true,
	"to": true, "we": true, "what": true, "when": true, "where": true, "which": true,
	"who": true, "why": true, "with": true, "you": true, "your": true, "can": true,
	"use": true, "us": true, "my": true, "me": true, "there": true, "than": true,
}

// Tokenize lowercases text, splits it on anything that is not a letter or
// digit, drops stopwords and reduces each word to a rough stem.
func Tokenize(text string) []string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(words))
	for _, w := range words {
		if stopwords[w] {
			continue
		}
		out = append(out, Stem(w))
	}
	return out
}

// Stem strips common English suffixes so "errors", "wrapping" and "wrapped"
// meet "error" and "wrap". It is deliberately light: words of four letters or
// fewer are left alone, and a final "e" is dropped so "handle" meets "handled".
func Stem(w string) string {
	s := stripSuffix(w)
	if len(s) > 4 && strings.HasSuffix(s, "e") {
		return s[:len(s)-1]
	}
	return s
}

// stripSuffix removes one inflectional suffix from w.
func stripSuffix(w string) string {
	if len(w) <= 4 {
		return w
	}
	for _, suf := range []string{"ing", "ies", "ied", "ed", "es", "s"} {
		if !strings.HasSuffix(w, suf) {
			continue
		}
		stem := w[:len(w)-len(suf)]
		if len(stem) < 3 {
			continue
		}
		switch suf {
		case "ies", "ied":
			return stem + "y"
		case "es":
			if !strings.HasSuffix(stem, "s") && !strings.HasSuffix(stem, "x") && !strings.HasSuffix(stem, "ch") && !strings.HasSuffix(stem, "sh") {
				return stem + "e"
			}
		case "s":
			if strings.HasSuffix(stem, "s") {
				return w
			}
		}
		return undouble(stem)
	}
	return w
}

// undouble turns "wrapp" back into "wrap" after stripping "ed" or "ing".
func undouble(s string) string {
	n := len(s)
	if n >= 4 && s[n-1] == s[n-2] && !strings.ContainsRune("aeiouls", rune(s[n-1])) {
		return s[:n-1]
	}
	return s
}
