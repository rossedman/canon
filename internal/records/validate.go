package records

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Problem is one thing wrong with a record.
type Problem struct {
	Record  string `json:"record"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// String formats the problem for terminal output.
func (p Problem) String() string {
	return fmt.Sprintf("%s: %s", p.Path, p.Message)
}

// idPattern is the kebab-case shape of a record ID.
var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// datePattern is an ISO date.
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Validate checks every record in s. ruleExists reports whether a rule ID is
// defined; pass nil to skip rule resolution.
func Validate(s *Set, ruleExists func(string) bool) []Problem {
	var out []Problem
	for _, r := range s.All() {
		add := func(format string, args ...any) {
			out = append(out, Problem{Record: r.ID, Path: r.Path, Message: fmt.Sprintf(format, args...)})
		}
		validateFields(r, add)
		for _, sec := range RequiredSections {
			if strings.TrimSpace(r.Section(sec)) == "" {
				add("missing required section %q", "## "+sec)
			}
		}
		for _, id := range r.Supersedes {
			if _, ok := s.Get(id); !ok {
				add("supersedes unknown record %q", id)
			}
		}
		if ruleExists != nil {
			for _, id := range r.Rules {
				if !ruleExists(id) {
					add("rule %q is not defined in rules.yaml", id)
				}
			}
		}
		if r.Status == Superseded && len(s.SupersededBy(r.ID)) == 0 {
			add("status is superseded but no record supersedes it")
		}
	}
	for _, cycle := range supersedeCycles(s) {
		out = append(out, Problem{Record: cycle[0], Path: mustPath(s, cycle[0]),
			Message: "supersedes cycle: " + strings.Join(cycle, " -> ")})
	}
	return out
}

// validateFields checks the frontmatter of one record.
func validateFields(r *Record, add func(string, ...any)) {
	if !idPattern.MatchString(r.ID) {
		add("id %q is not kebab-case", r.ID)
	}
	if base := strings.TrimSuffix(path.Base(r.Path), ".md"); base != r.ID {
		add("file name %q does not match id %q", base+".md", r.ID)
	}
	if strings.TrimSpace(r.Title) == "" {
		add("title is empty")
	}
	if !slices.Contains([]Status{Proposed, Accepted, Superseded, Deprecated}, r.Status) {
		add("status %q is not one of proposed, accepted, superseded, deprecated", r.Status)
	}
	if !slices.Contains([]Strength{Must, Should, Prefer}, r.Strength) {
		add("strength %q is not one of must, should, prefer", r.Strength)
	}
	if len(r.Scope) == 0 {
		add("scope is empty")
	}
	for _, sc := range r.Scope {
		if !slices.Contains(Scopes, sc) {
			add("scope %q is not one of %s", sc, strings.Join(Scopes, ", "))
		}
	}
	if len(r.AppliesTo.Files) == 0 {
		add("applies_to.files is empty")
	}
	if !datePattern.MatchString(r.Updated) {
		add("updated %q is not a YYYY-MM-DD date", r.Updated)
	}
}

// mustPath returns the path of a record known to exist.
func mustPath(s *Set, id string) string {
	if r, ok := s.Get(id); ok {
		return r.Path
	}
	return ""
}

// supersedeCycles finds cycles in the supersedes graph, each reported once,
// starting from its smallest ID.
func supersedeCycles(s *Set) [][]string {
	const (
		unseen = iota
		active
		done
	)
	state := map[string]int{}
	var stack []string
	var cycles [][]string
	var visit func(id string)
	visit = func(id string) {
		state[id] = active
		stack = append(stack, id)
		r, _ := s.Get(id)
		for _, next := range r.Supersedes {
			if _, ok := s.Get(next); !ok {
				continue
			}
			switch state[next] {
			case unseen:
				visit(next)
			case active:
				i := slices.Index(stack, next)
				cycles = append(cycles, rotate(append(slices.Clone(stack[i:]), next)))
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
	}
	for _, r := range s.All() {
		if state[r.ID] == unseen {
			visit(r.ID)
		}
	}
	return cycles
}

// rotate turns a closed path a->b->c->a into one that starts at its smallest ID.
func rotate(cycle []string) []string {
	ring := cycle[:len(cycle)-1]
	minI := 0
	for i, id := range ring {
		if id < ring[minI] {
			minI = i
		}
	}
	out := append(slices.Clone(ring[minI:]), ring[:minI]...)
	return append(out, out[0])
}
