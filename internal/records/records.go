// Package records loads, parses, validates and indexes decision records:
// Markdown files with YAML frontmatter that hold one opinion each, with its
// context, decision and consequences.
package records

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/glob"
)

// Status says whether a record is in force.
type Status string

// Record statuses. Only accepted records are served as current advice.
const (
	Proposed   Status = "proposed"
	Accepted   Status = "accepted"
	Superseded Status = "superseded"
	Deprecated Status = "deprecated"
)

// Strength says how firmly a record applies, and sets finding severity.
type Strength string

// Record strengths, from firmest to softest.
const (
	Must   Strength = "must"
	Should Strength = "should"
	Prefer Strength = "prefer"
)

// Severity maps a strength to the severity of findings against it.
func (s Strength) Severity() findings.Severity {
	switch s {
	case Must:
		return findings.Error
	case Should:
		return findings.Warning
	}
	return findings.Suggestion
}

// Scopes lists the areas a record can belong to.
var Scopes = []string{"go", "python", "containers", "terraform", "yaml", "ci", "oss"}

// RequiredSections are the body headings every record must have.
var RequiredSections = []string{"Context", "Decision", "Consequences"}

// AppliesTo says which files a record is about.
type AppliesTo struct {
	Files []string `yaml:"files" json:"files"`
}

// Record is one decision record.
type Record struct {
	ID         string    `yaml:"id" json:"id"`
	Title      string    `yaml:"title" json:"title"`
	Summary    string    `yaml:"summary" json:"summary,omitempty"`
	Status     Status    `yaml:"status" json:"status"`
	Supersedes []string  `yaml:"supersedes" json:"supersedes,omitempty"`
	Scope      []string  `yaml:"scope" json:"scope"`
	AppliesTo  AppliesTo `yaml:"applies_to" json:"applies_to"`
	Tags       []string  `yaml:"tags" json:"tags,omitempty"`
	Rules      []string  `yaml:"rules" json:"rules,omitempty"`
	Strength   Strength  `yaml:"strength" json:"strength"`
	Updated    string    `yaml:"updated" json:"updated"`

	// Path is where the record was loaded from, relative to the canon root.
	Path string `yaml:"-" json:"path"`
	// Raw is the file as loaded, frontmatter included.
	Raw string `yaml:"-" json:"-"`
	// Body is the Markdown after the frontmatter.
	Body string `yaml:"-" json:"-"`
	// Sections maps each "## Heading" in the body to its text.
	Sections map[string]string `yaml:"-" json:"-"`
	// Order lists section headings as they appear.
	Order []string `yaml:"-" json:"-"`
}

// ErrNoFrontmatter is returned when a file does not start with a YAML block.
var ErrNoFrontmatter = errors.New("record has no frontmatter")

// Parse reads one record from data. name is used in errors and stored as Path.
func Parse(name string, data []byte) (*Record, error) {
	front, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	var r Record
	dec := yaml.NewDecoder(bytes.NewReader(front))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: frontmatter: %w", name, err)
	}
	r.Path = name
	r.Raw = string(data)
	r.Body = strings.TrimSpace(string(body))
	r.Sections, r.Order = splitSections(r.Body)
	return &r, nil
}

// splitFrontmatter separates the YAML block between the leading "---" lines
// from the body.
func splitFrontmatter(data []byte) (front, body []byte, err error) {
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, ErrNoFrontmatter
	}
	rest := data[4:]
	if bytes.HasPrefix(rest, []byte("---\n")) || bytes.Equal(rest, []byte("---")) {
		return nil, bytes.TrimPrefix(rest[3:], []byte("\n")), nil
	}
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		if bytes.HasSuffix(rest, []byte("\n---")) {
			return rest[:len(rest)-4], nil, nil
		}
		return nil, nil, errors.New("frontmatter is not closed with ---")
	}
	return rest[:end+1], rest[end+5:], nil
}

// splitSections splits a Markdown body on level-two headings, ignoring
// headings inside fenced code blocks.
func splitSections(body string) (map[string]string, []string) {
	sections := map[string]string{}
	var order []string
	var cur string
	var buf strings.Builder
	inFence := false
	flush := func() {
		if cur != "" {
			sections[cur] = strings.TrimSpace(buf.String())
		}
		buf.Reset()
	}
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
		}
		if heading := strings.TrimSpace(strings.TrimPrefix(line, "## ")); !inFence && strings.HasPrefix(line, "## ") && heading != "" {
			flush()
			cur = heading
			if _, dup := sections[cur]; !dup {
				order = append(order, cur)
			}
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	flush()
	return sections, order
}

// Section returns a body section by heading, matching case-insensitively.
func (r *Record) Section(name string) string {
	if s, ok := r.Sections[name]; ok {
		return s
	}
	for k, v := range r.Sections {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}

// Why returns the one-line reason quoted in findings: the summary, or the
// first sentence of the Decision section.
func (r *Record) Why() string {
	if r.Summary != "" {
		return r.Summary
	}
	return firstSentence(r.Section("Decision"))
}

// firstSentence returns text up to the first ". " or the end of the first paragraph.
func firstSentence(text string) string {
	para, _, _ := strings.Cut(text, "\n\n")
	para = strings.Join(strings.Fields(para), " ")
	if i := strings.Index(para, ". "); i >= 0 {
		return para[:i+1]
	}
	return para
}

// Markdown returns the record as written, frontmatter included.
func (r *Record) Markdown() string {
	return r.Raw
}

// InForce reports whether the record is current advice.
func (r *Record) InForce() bool {
	return r.Status == Accepted
}

// Matches reports whether the record applies to the repo-relative path.
func (r *Record) Matches(p string) bool {
	return glob.MatchAny(r.AppliesTo.Files, p)
}

// HasScope reports whether the record belongs to any of scopes. An empty
// scope list matches every record.
func (r *Record) HasScope(scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, s := range r.Scope {
		if slices.Contains(scopes, s) {
			return true
		}
	}
	return false
}

// Set is an indexed collection of records.
type Set struct {
	byID map[string]*Record
	all  []*Record
}

// NewSet indexes records by ID. A duplicate ID is an error.
func NewSet(rs []*Record) (*Set, error) {
	s := &Set{byID: make(map[string]*Record, len(rs))}
	for _, r := range rs {
		if prev, dup := s.byID[r.ID]; dup {
			return nil, fmt.Errorf("record id %q is defined in both %s and %s", r.ID, prev.Path, r.Path)
		}
		s.byID[r.ID] = r
		s.all = append(s.all, r)
	}
	slices.SortFunc(s.all, func(a, b *Record) int { return strings.Compare(a.ID, b.ID) })
	return s, nil
}

// Load parses every .md file under dir in fsys.
func Load(fsys fs.FS, dir string) (*Set, error) {
	var rs []*Record
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path.Ext(p) != ".md" {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("read record: %w", err)
		}
		r, err := Parse(p, data)
		if err != nil {
			return err
		}
		rs = append(rs, r)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load records from %s: %w", dir, err)
	}
	return NewSet(rs)
}

// Get returns the record with id.
func (s *Set) Get(id string) (*Record, bool) {
	r, ok := s.byID[id]
	return r, ok
}

// All returns every record, sorted by ID.
func (s *Set) All() []*Record {
	return s.all
}

// SupersededBy returns the IDs of records that list id in supersedes.
func (s *Set) SupersededBy(id string) []string {
	var out []string
	for _, r := range s.all {
		if slices.Contains(r.Supersedes, id) {
			out = append(out, r.ID)
		}
	}
	return out
}
