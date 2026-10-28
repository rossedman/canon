// Package engine is the one core behind every front door: the MCP server,
// the CLI and the hook binary all call it. It never holds opinions itself;
// it reads them from a canon directory and delegates mechanical checks to
// real linters.
package engine

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/rossedman/canon/canon"
	"github.com/rossedman/canon/internal/checkers"
	"github.com/rossedman/canon/internal/profiles"
	"github.com/rossedman/canon/internal/records"
	"github.com/rossedman/canon/internal/rules"
	"github.com/rossedman/canon/internal/search"
)

// Engine holds a loaded canon and the tools to apply it.
type Engine struct {
	Records    *records.Set
	Rules      *rules.Registry
	Profiles   []*profiles.Profile
	Principles string
	// Templates is the templates/ directory of the canon.
	Templates fs.FS
	// Configs is the rules/ directory, holding per-checker configs.
	Configs  fs.FS
	Checkers map[string]checkers.Checker
	Runner   checkers.Runner
	// MinDecideScore is the BM25 score below which decide reports a gap.
	MinDecideScore float64

	index *search.Index
}

// Default loads the canon embedded in the binary.
func Default() (*Engine, error) {
	return Load(canon.FS)
}

// LoadDir loads a canon from a directory on disk, so records can live in
// their own repo.
func LoadDir(dir string) (*Engine, error) {
	return Load(os.DirFS(dir))
}

// Load reads a canon from fsys, which holds records/, rules/rules.yaml,
// profiles/, templates/ and principles.md.
func Load(fsys fs.FS) (*Engine, error) {
	set, err := records.Load(fsys, "records")
	if err != nil {
		return nil, fmt.Errorf("load canon: %w", err)
	}
	reg, err := rules.Load(fsys, "rules/rules.yaml")
	if err != nil {
		return nil, fmt.Errorf("load canon: %w", err)
	}
	ps, err := profiles.Load(fsys, "profiles")
	if err != nil {
		return nil, fmt.Errorf("load canon: %w", err)
	}
	principles, err := fs.ReadFile(fsys, "principles.md")
	if err != nil {
		return nil, fmt.Errorf("read principles: %w", err)
	}
	tmpl, err := fs.Sub(fsys, "templates")
	if err != nil {
		return nil, fmt.Errorf("open templates: %w", err)
	}
	cfgs, err := fs.Sub(fsys, "rules")
	if err != nil {
		return nil, fmt.Errorf("open rules: %w", err)
	}
	e := &Engine{
		Records:        set,
		Rules:          reg,
		Profiles:       ps,
		Principles:     string(principles),
		Templates:      tmpl,
		Configs:        cfgs,
		Checkers:       checkers.All(),
		Runner:         checkers.ExecRunner{},
		MinDecideScore: 2.0,
	}
	e.index = buildIndex(set)
	return e, nil
}

// buildIndex indexes records for decide, weighting the title and tags above the body.
func buildIndex(set *records.Set) *search.Index {
	docs := make([]search.Doc, 0, len(set.All()))
	for _, r := range set.All() {
		docs = append(docs, search.Doc{ID: r.ID, Fields: []search.Field{
			{Text: r.Title, Weight: 3},
			{Text: strings.Join(r.Tags, " ") + " " + strings.Join(r.Scope, " "), Weight: 2},
			{Text: r.Summary, Weight: 2},
			{Text: r.Body, Weight: 1},
		}})
	}
	return search.New(docs)
}

// RecordRef is a short reference to a record.
type RecordRef struct {
	ID       string           `json:"id"`
	Title    string           `json:"title"`
	Status   records.Status   `json:"status"`
	Strength records.Strength `json:"strength"`
	Scope    []string         `json:"scope"`
}

// refOf builds a RecordRef.
func refOf(r *records.Record) RecordRef {
	return RecordRef{ID: r.ID, Title: r.Title, Status: r.Status, Strength: r.Strength, Scope: r.Scope}
}

// Index returns a short reference to every record, for the canon://records resource.
func (e *Engine) Index() []RecordRef {
	out := make([]RecordRef, 0, len(e.Records.All()))
	for _, r := range e.Records.All() {
		out = append(out, refOf(r))
	}
	return out
}

// Problem is one thing wrong with the canon.
type Problem struct {
	Kind    string `json:"kind"`
	Ref     string `json:"ref"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// Lint validates the whole canon: records, rules, profiles and templates.
func (e *Engine) Lint() []Problem {
	var out []Problem
	for _, p := range records.Validate(e.Records, e.Rules.Has) {
		out = append(out, Problem{Kind: "record", Ref: p.Record, Path: p.Path, Message: p.Message})
	}
	recordRules := func(id string) ([]string, bool) {
		r, ok := e.Records.Get(id)
		if !ok {
			return nil, false
		}
		return r.Rules, true
	}
	known := func(name string) bool { _, ok := e.Checkers[name]; return ok }
	for _, p := range e.Rules.Validate(recordRules, known) {
		out = append(out, Problem{Kind: "rule", Ref: p.Rule, Path: "rules/rules.yaml", Message: p.Message})
	}
	builtinCodes := checkers.Builtin{}.Codes()
	for _, r := range e.Rules.All() {
		if r.Checker != "canon" {
			continue
		}
		for _, c := range r.Codes {
			if !slices.Contains(builtinCodes, c) {
				out = append(out, Problem{Kind: "rule", Ref: r.ID, Path: "rules/rules.yaml", Message: fmt.Sprintf("canon has no built-in check %q", c)})
			}
		}
	}
	for _, p := range profiles.Validate(e.Profiles, e.Records) {
		out = append(out, Problem{Kind: "profile", Ref: p.Profile, Message: p.Message})
	}
	out = append(out, e.lintTemplates()...)
	return out
}
