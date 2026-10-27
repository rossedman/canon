package engine

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/gitx"
	"github.com/rossedman/canon/internal/review"
)

// ReviewRequest is a change to grade.
type ReviewRequest struct {
	Root string
	// Base is compared against the working tree when Diff is empty.
	Base string
	// Diff is a unified diff; when empty it is computed from Base.
	Diff        string
	MinSeverity findings.Severity
}

// ReviewResult holds the facts about a change. The agent writes the narrative.
type ReviewResult struct {
	Findings       []findings.Finding `json:"findings"`
	Summary        findings.Summary   `json:"summary"`
	Concerns       []review.Concern   `json:"concerns"`
	RecordsTouched []string           `json:"records_touched"`
	Files          []string           `json:"files"`
	Profile        string             `json:"profile"`
	Skipped        []Skipped          `json:"skipped,omitempty"`
}

// repoFS opens root as a file system after resolving it.
func repoFS(root string) (fs.FS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	return os.DirFS(abs), nil
}

// Review checks the files a diff changes and classifies the change.
func (e *Engine) Review(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	root, err := filepath.Abs(req.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	repo := os.DirFS(root)
	diff := req.Diff
	if diff == "" {
		base := req.Base
		if base == "" {
			base = "origin/main"
		}
		if diff, err = gitx.Diff(ctx, e.Runner, root, base, repo); err != nil {
			return nil, fmt.Errorf("compute diff: %w", err)
		}
	}
	diffs := review.ParseDiff(diff)
	res := &ReviewResult{Findings: []findings.Finding{}, Concerns: []review.Concern{}, RecordsTouched: []string{}, Files: []string{}}
	var paths []string
	for _, d := range diffs {
		res.Files = append(res.Files, d.Path)
		if d.Status == review.Deleted {
			continue
		}
		if st, err := fs.Stat(repo, d.Path); err == nil && st.Mode().IsRegular() {
			paths = append(paths, d.Path)
		}
	}
	prof, err := e.profileFor(repo, "")
	if err != nil {
		return nil, err
	}
	res.Profile = prof.Name
	if len(paths) > 0 {
		chk, err := e.Check(ctx, CheckRequest{Root: root, Paths: paths, MinSeverity: req.MinSeverity})
		if err != nil {
			return nil, err
		}
		res.Findings, res.Summary, res.Skipped = chk.Findings, chk.Summary, chk.Skipped
	}
	for _, c := range review.Classify(diffs, repo) {
		if id, ok := e.Rules.ConcernRecord(c.Kind); ok {
			c.Record = id
		}
		res.Concerns = append(res.Concerns, c)
	}
	for _, f := range res.Findings {
		res.RecordsTouched = append(res.RecordsTouched, f.Record)
	}
	for _, c := range res.Concerns {
		if c.Record != "" {
			res.RecordsTouched = append(res.RecordsTouched, c.Record)
		}
	}
	slices.Sort(res.RecordsTouched)
	res.RecordsTouched = slices.Compact(res.RecordsTouched)
	return res, nil
}
