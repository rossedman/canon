package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rossedman/canon/internal/checkers"
	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/gitx"
	"github.com/rossedman/canon/internal/profiles"
	"github.com/rossedman/canon/internal/records"
	"github.com/rossedman/canon/internal/rules"
)

// CheckRequest says what to check.
type CheckRequest struct {
	// Root is the repository root. Paths are relative to it.
	Root string
	// Paths are files or directories to check; empty means the whole repo.
	Paths []string
	// Scope limits records to these scopes; empty means all.
	Scope []string
	// MinSeverity drops findings below it.
	MinSeverity findings.Severity
	// ChangedOnly limits the check to files changed since Base.
	ChangedOnly bool
	// Base is the branch ChangedOnly compares against.
	Base string
	// Profile forces a profile instead of detecting one.
	Profile string
}

// Skipped is a checker that did not run, and why.
type Skipped struct {
	Checker string `json:"checker"`
	Reason  string `json:"reason"`
}

// CheckResult is what check returns.
type CheckResult struct {
	Findings     []findings.Finding `json:"findings"`
	Summary      findings.Summary   `json:"summary"`
	Profile      string             `json:"profile"`
	FilesChecked int                `json:"files_checked"`
	Skipped      []Skipped          `json:"skipped,omitempty"`
	Unmapped     int                `json:"unmapped,omitempty"`
}

// skipDirs are never walked into, unless named explicitly as a path.
var skipDirs = []string{".git", "node_modules", "vendor", ".venv", "venv", "__pycache__", "dist", "site", ".terraform", "testdata", ".ruff_cache"}

// Check runs the rules in force against the requested files.
func (e *Engine) Check(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	root, err := filepath.Abs(req.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	repo := os.DirFS(root)
	prof, err := e.profileFor(repo, req.Profile)
	if err != nil {
		return nil, err
	}
	var files []string
	whole := false
	if req.ChangedOnly {
		files, err = gitx.ChangedFiles(ctx, e.Runner, root, req.Base)
		if err != nil {
			return nil, fmt.Errorf("list changed files: %w", err)
		}
		files = existingFiles(repo, files)
	} else {
		files, whole, err = collectFiles(repo, root, req.Paths)
		if err != nil {
			return nil, err
		}
	}
	inForce := filterScope(prof.Resolve(e.Records), req.Scope)
	res, err := e.run(ctx, root, repo, files, whole, inForce)
	if err != nil {
		return nil, err
	}
	res.Profile = prof.Name
	threshold := req.MinSeverity
	if threshold == "" {
		threshold = findings.Warning
	}
	res.Findings = findings.Filter(res.Findings, threshold)
	res.Summary = findings.Summarize(res.Findings)
	return res, nil
}

// profileFor returns the named profile, or detects one.
func (e *Engine) profileFor(repo fs.FS, name string) (*profiles.Profile, error) {
	if name != "" {
		p, ok := profiles.Find(e.Profiles, name)
		if !ok {
			return nil, fmt.Errorf("unknown profile %q", name)
		}
		return p, nil
	}
	p, ok := profiles.DetectProfile(repo, e.Profiles)
	if !ok {
		return nil, errors.New("no profile matches this repository, and the canon has no catch-all profile")
	}
	return p, nil
}

// filterScope keeps records in any of scopes; empty keeps all.
func filterScope(rs []*records.Record, scopes []string) []*records.Record {
	if len(scopes) == 0 {
		return rs
	}
	var out []*records.Record
	for _, r := range rs {
		if r.HasScope(scopes) {
			out = append(out, r)
		}
	}
	return out
}

// plan is the work for one checker.
type plan struct {
	checker checkers.Checker
	rules   []*rules.Rule
	files   []string
}

// run maps records to rules to checkers, runs them, and turns issues into findings.
func (e *Engine) run(ctx context.Context, root string, repo fs.FS, files []string, whole bool, inForce []*records.Record) (*CheckResult, error) {
	byID := map[string]*records.Record{}
	for _, r := range inForce {
		byID[r.ID] = r
	}
	plans := map[string]*plan{}
	var order []string
	for _, rec := range inForce {
		for _, id := range rec.Rules {
			rule, ok := e.Rules.Get(id)
			if !ok {
				continue
			}
			c, ok := e.Checkers[rule.Checker]
			if !ok {
				continue
			}
			p := plans[rule.Checker]
			if p == nil {
				p = &plan{checker: c}
				plans[rule.Checker] = p
				order = append(order, rule.Checker)
			}
			p.rules = append(p.rules, rule)
		}
	}
	slices.Sort(order)
	res := &CheckResult{FilesChecked: len(files), Findings: []findings.Finding{}}
	cfgDir, err := os.MkdirTemp("", "canon-config-")
	if err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(cfgDir) }()
	for _, name := range order {
		p := plans[name]
		p.files = filesFor(p.rules, byID, files)
		if len(p.files) == 0 {
			continue
		}
		if bin := p.checker.Binary(); bin != "" {
			if _, err := e.Runner.LookPath(bin); err != nil {
				res.Skipped = append(res.Skipped, Skipped{Checker: name, Reason: bin + " is not installed"})
				continue
			}
		}
		cfg, err := e.writeConfig(cfgDir, p.checker)
		if err != nil {
			return nil, err
		}
		issues, err := p.checker.Run(ctx, e.Runner, checkers.Request{
			Root: root, FS: repo, Files: p.files, Codes: codesOf(p.rules), Config: cfg, WholeRepo: whole,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("check cancelled: %w", ctx.Err())
			}
			res.Skipped = append(res.Skipped, Skipped{Checker: name, Reason: err.Error()})
			continue
		}
		res.Findings = append(res.Findings, e.toFindings(name, p, issues, byID, &res.Unmapped)...)
	}
	res.Findings = findings.Normalize(res.Findings)
	return res, nil
}

// toFindings maps one checker's issues to findings. External tools may report
// on files outside the request (tflint scans whole modules), so those are
// dropped; built-in repo-level checks name files that do not exist yet, so
// theirs are kept.
func (e *Engine) toFindings(checker string, p *plan, issues []checkers.Issue, byID map[string]*records.Record, unmapped *int) []findings.Finding {
	requested := map[string]bool{}
	for _, f := range p.files {
		requested[f] = true
	}
	allowed := func(r *rules.Rule) bool { return slices.Contains(p.rules, r) }
	var out []findings.Finding
	for _, is := range issues {
		if p.checker.Binary() != "" && !requested[is.Path] {
			continue
		}
		rule, ok := e.Rules.Resolve(checker, is.Code, allowed)
		if !ok {
			*unmapped++
			continue
		}
		rec := byID[rule.Record]
		msg := is.Message
		if p.checker.Binary() != "" && is.Code != "" && !strings.Contains(msg, is.Code) {
			msg = is.Code + ": " + msg
		}
		out = append(out, findings.Finding{
			Rule: rule.ID, Record: rec.ID, Severity: rec.Strength.Severity(),
			Path: is.Path, Line: is.Line, Column: is.Column,
			Message: msg, Why: rec.Why(), Fix: is.Fix,
		})
	}
	return out
}

// codesOf lists every code claimed by rules.
func codesOf(rs []*rules.Rule) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Codes...)
	}
	return out
}

// filesFor returns the files any of the rules' records apply to.
func filesFor(rs []*rules.Rule, byID map[string]*records.Record, files []string) []string {
	var recs []*records.Record
	for _, r := range rs {
		if rec, ok := byID[r.Record]; ok && !slices.Contains(recs, rec) {
			recs = append(recs, rec)
		}
	}
	var out []string
	for _, f := range files {
		if slices.ContainsFunc(recs, func(r *records.Record) bool { return r.Matches(f) }) {
			out = append(out, f)
		}
	}
	return out
}

// writeConfig writes a checker's config from the canon into dir and returns
// its path, or "" when the checker takes none.
func (e *Engine) writeConfig(dir string, c checkers.Checker) (string, error) {
	name := c.ConfigName()
	if name == "" {
		return "", nil
	}
	data, err := fs.ReadFile(e.Configs, name)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s config: %w", c.Name(), err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return "", fmt.Errorf("write %s config: %w", c.Name(), err)
	}
	return p, nil
}

// collectFiles expands paths (files or directories, relative to root or
// absolute inside it) into repo-relative files. It reports whether the
// request covers the whole repository.
func collectFiles(repo fs.FS, root string, paths []string) ([]string, bool, error) {
	whole := len(paths) == 0
	if whole {
		paths = []string{"."}
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		rel, err := relToRoot(root, p)
		if err != nil {
			return nil, false, err
		}
		if rel == "." {
			whole = true
		}
		st, err := fs.Stat(repo, rel)
		if err != nil {
			return nil, false, fmt.Errorf("check %s: %w", p, err)
		}
		if !st.IsDir() {
			if !seen[rel] {
				seen[rel] = true
				out = append(out, rel)
			}
			continue
		}
		err = fs.WalkDir(repo, rel, func(fp string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if fp != rel && slices.Contains(skipDirs, d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if d.Type().IsRegular() && !seen[fp] {
				seen[fp] = true
				out = append(out, fp)
			}
			return nil
		})
		if err != nil {
			return nil, false, fmt.Errorf("walk %s: %w", p, err)
		}
	}
	slices.Sort(out)
	return out, whole, nil
}

// relToRoot turns a user-supplied path into a clean slash path inside root.
func relToRoot(root, p string) (string, error) {
	if filepath.IsAbs(p) {
		r, err := filepath.Rel(root, p)
		if err != nil {
			return "", fmt.Errorf("path %s: %w", p, err)
		}
		p = r
	}
	rel := path.Clean(filepath.ToSlash(p))
	if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return "", fmt.Errorf("path %s is outside the repository root %s", p, root)
	}
	return rel, nil
}

// existingFiles keeps the paths that are regular files in repo.
func existingFiles(repo fs.FS, files []string) []string {
	var out []string
	for _, f := range files {
		if st, err := fs.Stat(repo, f); err == nil && st.Mode().IsRegular() {
			out = append(out, f)
		}
	}
	return out
}
