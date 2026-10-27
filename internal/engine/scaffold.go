package engine

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/scaffold"
)

// ScaffoldRequest asks for a project or component.
type ScaffoldRequest struct {
	Root   string
	Kind   string
	Name   string
	Dest   string
	Vars   map[string]string
	DryRun bool
}

// ScaffoldResult lists what was (or would be) written, the records the
// template applies, and, after a real run, the check of the result.
type ScaffoldResult struct {
	Kind    string       `json:"kind"`
	Dest    string       `json:"dest"`
	DryRun  bool         `json:"dry_run"`
	Files   []string     `json:"files"`
	Records []RecordRef  `json:"records"`
	Next    []string     `json:"next,omitempty"`
	Check   *CheckResult `json:"check,omitempty"`
	Passed  bool         `json:"passed"`
}

// Scaffold renders a template and, unless DryRun, writes it under Root/Dest
// and checks the result, so generated code always passes its own canon.
func (e *Engine) Scaffold(ctx context.Context, req ScaffoldRequest) (*ScaffoldResult, error) {
	plan, err := scaffold.Render(e.Templates, req.Kind, req.Name, req.Vars)
	if err != nil {
		return nil, err
	}
	dest := req.Dest
	if dest == "" {
		dest = "."
	}
	rel, err := relToRoot(mustAbs(req.Root), dest)
	if err != nil {
		return nil, err
	}
	res := &ScaffoldResult{Kind: req.Kind, Dest: rel, DryRun: req.DryRun, Next: plan.Next, Records: []RecordRef{}, Passed: true}
	for _, p := range plan.Paths() {
		res.Files = append(res.Files, path.Join(rel, p))
	}
	for _, id := range plan.Template.Records {
		if r, ok := e.Records.Get(id); ok {
			res.Records = append(res.Records, refOf(r))
		}
	}
	if req.DryRun {
		return res, nil
	}
	destDir := filepath.Join(mustAbs(req.Root), filepath.FromSlash(rel))
	if err := plan.Write(destDir); err != nil {
		return nil, fmt.Errorf("scaffold %s: %w", req.Kind, err)
	}
	checkRoot := filepath.Join(destDir, filepath.FromSlash(plan.Root))
	written := make([]string, 0, len(plan.Files))
	for _, p := range plan.Paths() {
		r, err := filepath.Rel(checkRoot, filepath.Join(destDir, filepath.FromSlash(p)))
		if err != nil {
			return nil, fmt.Errorf("scaffold %s: %w", req.Kind, err)
		}
		written = append(written, filepath.ToSlash(r))
	}
	chk, err := e.Check(ctx, CheckRequest{Root: checkRoot, Paths: written, MinSeverity: findings.Suggestion})
	if err != nil {
		return nil, fmt.Errorf("check scaffolded files: %w", err)
	}
	res.Check = chk
	res.Passed = chk.Summary.Error == 0
	return res, nil
}

// mustAbs makes p absolute, falling back to p if the working directory is gone.
func mustAbs(p string) string {
	if p == "" {
		p = "."
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// lintTemplates checks that every template loads, renders with a sample
// name, and names records that exist.
func (e *Engine) lintTemplates() []Problem {
	ts, err := scaffold.List(e.Templates)
	if err != nil {
		return []Problem{{Kind: "template", Message: err.Error()}}
	}
	var out []Problem
	for _, t := range ts {
		for _, id := range t.Records {
			if _, ok := e.Records.Get(id); !ok {
				out = append(out, Problem{Kind: "template", Ref: t.Kind, Path: "templates/" + t.Kind + "/template.yaml", Message: fmt.Sprintf("record %q does not exist", id)})
			}
		}
		if _, err := scaffold.Render(e.Templates, t.Kind, "sample", nil); err != nil {
			out = append(out, Problem{Kind: "template", Ref: t.Kind, Path: "templates/" + t.Kind, Message: err.Error()})
		}
	}
	return out
}

// Kinds lists the scaffold kinds available.
func (e *Engine) Kinds() ([]*scaffold.Template, error) {
	return scaffold.List(e.Templates)
}
