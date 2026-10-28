package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/rossedman/canon/internal/records"
	"github.com/rossedman/canon/internal/rules"
)

// DecideRequest is a "how do we do X" question.
type DecideRequest struct {
	Question string
	Scope    []string
	Limit    int
}

// Decision is one record that answers a question.
type Decision struct {
	ID         string           `json:"id"`
	Title      string           `json:"title"`
	Strength   records.Strength `json:"strength"`
	Scope      []string         `json:"scope"`
	Why        string           `json:"why"`
	Decision   string           `json:"decision"`
	Exceptions string           `json:"exceptions,omitempty"`
	Score      float64          `json:"score"`
}

// DecideResult lists matching records, best first. Gap is set when nothing
// matched well enough, which marks a hole in the canon.
type DecideResult struct {
	Question string     `json:"question"`
	Matches  []Decision `json:"matches"`
	Gap      bool       `json:"gap"`
	Message  string     `json:"message,omitempty"`
}

// Decide answers a question from the records in force.
func (e *Engine) Decide(req DecideRequest) (*DecideResult, error) {
	q := strings.TrimSpace(req.Question)
	if q == "" {
		return nil, errors.New("question is empty")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}
	keep := func(id string) bool {
		r, ok := e.Records.Get(id)
		return ok && r.InForce() && r.HasScope(req.Scope)
	}
	hits := e.index.Search(q, limit, e.MinDecideScore, keep)
	res := &DecideResult{Question: q, Matches: []Decision{}}
	for _, h := range hits {
		r, _ := e.Records.Get(h.ID)
		res.Matches = append(res.Matches, Decision{
			ID: r.ID, Title: r.Title, Strength: r.Strength, Scope: r.Scope, Why: r.Why(),
			Decision: r.Section("Decision"), Exceptions: r.Section("Exceptions"), Score: h.Score,
		})
	}
	if len(res.Matches) == 0 {
		res.Gap = true
		res.Message = "No record in the canon answers this question. Say so rather than guessing, and consider drafting one with the canon-new-record prompt."
	}
	return res, nil
}

// ExplainRequest names a rule or a record to explain.
type ExplainRequest struct {
	Rule   string
	Record string
}

// ExplainResult is a record in full, with what enforces it and what it replaced.
type ExplainResult struct {
	Rule         *rules.Rule     `json:"rule,omitempty"`
	Record       *records.Record `json:"record"`
	Why          string          `json:"why"`
	Markdown     string          `json:"markdown"`
	Rules        []*rules.Rule   `json:"rules"`
	Supersedes   []RecordRef     `json:"supersedes,omitempty"`
	SupersededBy []RecordRef     `json:"superseded_by,omitempty"`
}

// ErrNotFound is returned for an unknown rule or record.
var ErrNotFound = errors.New("not found")

// Explain traces a rule or a record to its reasoning.
func (e *Engine) Explain(req ExplainRequest) (*ExplainResult, error) {
	if (req.Rule == "") == (req.Record == "") {
		return nil, errors.New("give exactly one of rule or record")
	}
	res := &ExplainResult{}
	id := req.Record
	if req.Rule != "" {
		rule, ok := e.Rules.Get(req.Rule)
		if !ok {
			return nil, fmt.Errorf("rule %q: %w", req.Rule, ErrNotFound)
		}
		res.Rule = rule
		id = rule.Record
	}
	rec, ok := e.Records.Get(id)
	if !ok {
		return nil, fmt.Errorf("record %q: %w", id, ErrNotFound)
	}
	res.Record = rec
	res.Why = rec.Why()
	res.Markdown = rec.Markdown()
	res.Rules = e.Rules.ForRecord(rec.ID)
	if res.Rules == nil {
		res.Rules = []*rules.Rule{}
	}
	for _, s := range rec.Supersedes {
		if r, ok := e.Records.Get(s); ok {
			res.Supersedes = append(res.Supersedes, refOf(r))
		}
	}
	for _, s := range e.Records.SupersededBy(rec.ID) {
		if r, ok := e.Records.Get(s); ok {
			res.SupersededBy = append(res.SupersededBy, refOf(r))
		}
	}
	return res, nil
}

// Resolve guesses whether id names a rule or a record, for the CLI's
// positional argument.
func (e *Engine) Resolve(id string) ExplainRequest {
	if e.Rules.Has(id) {
		return ExplainRequest{Rule: id}
	}
	return ExplainRequest{Record: id}
}

// CheckerStatus says whether a checker a profile needs is installed.
type CheckerStatus struct {
	Name      string   `json:"name"`
	Binary    string   `json:"binary,omitempty"`
	Installed bool     `json:"installed"`
	Rules     []string `json:"rules"`
}

// ProfileResult is the profile a repo matches and what it puts in force.
type ProfileResult struct {
	Profile     string          `json:"profile"`
	Description string          `json:"description"`
	Records     []RecordRef     `json:"records"`
	Checkers    []CheckerStatus `json:"checkers"`
	Missing     []string        `json:"missing"`
}

// Profile detects which profile applies to the repo at root, and which
// checkers it needs that are not installed.
func (e *Engine) Profile(root string) (*ProfileResult, error) {
	repo, err := repoFS(root)
	if err != nil {
		return nil, err
	}
	p, err := e.profileFor(repo, "")
	if err != nil {
		return nil, err
	}
	res := &ProfileResult{Profile: p.Name, Description: p.Description, Records: []RecordRef{}, Checkers: []CheckerStatus{}, Missing: []string{}}
	byChecker := map[string][]string{}
	for _, r := range p.Resolve(e.Records) {
		res.Records = append(res.Records, refOf(r))
		for _, id := range r.Rules {
			if rule, ok := e.Rules.Get(id); ok {
				byChecker[rule.Checker] = append(byChecker[rule.Checker], rule.ID)
			}
		}
	}
	names := make([]string, 0, len(byChecker))
	for n := range byChecker {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		c, ok := e.Checkers[n]
		if !ok {
			continue
		}
		st := CheckerStatus{Name: n, Binary: c.Binary(), Installed: true, Rules: byChecker[n]}
		if c.Binary() != "" {
			if _, err := e.Runner.LookPath(c.Binary()); err != nil {
				st.Installed = false
				res.Missing = append(res.Missing, c.Binary())
			}
		}
		res.Checkers = append(res.Checkers, st)
	}
	return res, nil
}

// ProfileRecords resolves a named profile's records, for the
// canon://profiles/{name} resource.
func (e *Engine) ProfileRecords(name string) (map[string]any, error) {
	for _, p := range e.Profiles {
		if p.Name != name {
			continue
		}
		refs := []RecordRef{}
		for _, r := range p.Resolve(e.Records) {
			refs = append(refs, refOf(r))
		}
		return map[string]any{"profile": p, "records": refs}, nil
	}
	return nil, fmt.Errorf("profile %q: %w", name, ErrNotFound)
}
