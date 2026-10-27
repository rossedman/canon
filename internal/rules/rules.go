// Package rules is the registry that maps checker output to rule IDs, and
// rule IDs to the records they enforce.
package rules

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Wildcard as a code claims every code from its checker that no other rule does.
const Wildcard = "*"

// Rule is a machine check that enforces part of a record.
type Rule struct {
	ID          string   `yaml:"id" json:"id"`
	Record      string   `yaml:"record" json:"record"`
	Checker     string   `yaml:"checker" json:"checker"`
	Codes       []string `yaml:"codes" json:"codes"`
	Description string   `yaml:"description" json:"description"`
}

// file is the shape of rules.yaml.
type file struct {
	Rules    []*Rule           `yaml:"rules"`
	Concerns map[string]string `yaml:"concerns"`
}

// Registry holds every rule and the concern-to-record map used by review.
type Registry struct {
	rules    []*Rule
	byID     map[string]*Rule
	byCode   map[string]map[string]*Rule
	concerns map[string]string
}

// Parse reads rules.yaml content.
func Parse(data []byte) (*Registry, error) {
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	return New(f.Rules, f.Concerns)
}

// Load reads rules.yaml at name in fsys.
func Load(fsys fs.FS, name string) (*Registry, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}
	return Parse(data)
}

// New indexes rules. Duplicate IDs, or two rules claiming the same code from
// the same checker, are errors.
func New(rs []*Rule, concerns map[string]string) (*Registry, error) {
	reg := &Registry{
		byID:     map[string]*Rule{},
		byCode:   map[string]map[string]*Rule{},
		concerns: concerns,
	}
	if reg.concerns == nil {
		reg.concerns = map[string]string{}
	}
	for _, r := range rs {
		if r.ID == "" || r.Checker == "" || r.Record == "" {
			return nil, fmt.Errorf("rule %q needs id, record and checker", r.ID)
		}
		if _, dup := reg.byID[r.ID]; dup {
			return nil, fmt.Errorf("rule id %q is defined twice", r.ID)
		}
		reg.byID[r.ID] = r
		codes := reg.byCode[r.Checker]
		if codes == nil {
			codes = map[string]*Rule{}
			reg.byCode[r.Checker] = codes
		}
		for _, c := range r.Codes {
			if prev, dup := codes[c]; dup {
				return nil, fmt.Errorf("%s code %q is claimed by both %s and %s", r.Checker, c, prev.ID, r.ID)
			}
			codes[c] = r
		}
		reg.rules = append(reg.rules, r)
	}
	slices.SortFunc(reg.rules, func(a, b *Rule) int { return strings.Compare(a.ID, b.ID) })
	return reg, nil
}

// All returns every rule, sorted by ID.
func (r *Registry) All() []*Rule {
	return r.rules
}

// Get returns a rule by ID.
func (r *Registry) Get(id string) (*Rule, bool) {
	rule, ok := r.byID[id]
	return rule, ok
}

// Has reports whether a rule ID is defined.
func (r *Registry) Has(id string) bool {
	_, ok := r.byID[id]
	return ok
}

// Resolve finds the rule that owns a code reported by checker, preferring an
// exact code over the checker's wildcard rule. allowed, when non-nil, limits
// the answer to rules in force; a code whose exact rule is not in force is not
// handed to the wildcard, so it stays unreported rather than mislabeled.
func (r *Registry) Resolve(checker, code string, allowed func(*Rule) bool) (*Rule, bool) {
	codes := r.byCode[checker]
	if rule, ok := codes[code]; ok {
		if allowed != nil && !allowed(rule) {
			return nil, false
		}
		return rule, true
	}
	if rule, ok := codes[Wildcard]; ok && (allowed == nil || allowed(rule)) {
		return rule, true
	}
	return nil, false
}

// ForRecord returns the rules that enforce a record.
func (r *Registry) ForRecord(id string) []*Rule {
	var out []*Rule
	for _, rule := range r.rules {
		if rule.Record == id {
			out = append(out, rule)
		}
	}
	return out
}

// ConcernRecord returns the record a review concern kind points at.
func (r *Registry) ConcernRecord(kind string) (string, bool) {
	id, ok := r.concerns[kind]
	return id, ok
}

// Concerns returns the concern-to-record map.
func (r *Registry) Concerns() map[string]string {
	return r.concerns
}

// Problem is one thing wrong with the rules file.
type Problem struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Validate checks that every rule's record exists and lists the rule back,
// every checker is known, and every concern points at a record.
func (r *Registry) Validate(recordRules func(id string) ([]string, bool), checkerKnown func(string) bool) []Problem {
	var out []Problem
	for _, rule := range r.rules {
		listed, ok := recordRules(rule.Record)
		switch {
		case !ok:
			out = append(out, Problem{rule.ID, fmt.Sprintf("record %q does not exist", rule.Record)})
		case !slices.Contains(listed, rule.ID):
			out = append(out, Problem{rule.ID, fmt.Sprintf("record %q does not list this rule in its rules", rule.Record)})
		}
		if !checkerKnown(rule.Checker) {
			out = append(out, Problem{rule.ID, fmt.Sprintf("checker %q is not known", rule.Checker)})
		}
		if len(rule.Codes) == 0 {
			out = append(out, Problem{rule.ID, "codes is empty"})
		}
	}
	kinds := make([]string, 0, len(r.concerns))
	for k := range r.concerns {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	for _, k := range kinds {
		if _, ok := recordRules(r.concerns[k]); !ok {
			out = append(out, Problem{"concern:" + k, fmt.Sprintf("record %q does not exist", r.concerns[k])})
		}
	}
	return out
}
