// Package hook turns canon into a Claude Code hook: it reads the hook's JSON
// from stdin, checks the file the agent just edited, and reports findings
// back so the agent fixes them before moving on.
package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/findings"
)

// ExitBlock is the exit code that makes Claude Code show stderr to the agent.
const ExitBlock = 2

// Input is the part of a Claude Code hook payload canon reads.
type Input struct {
	Cwd           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	} `json:"tool_input"`
}

// ParseInput reads a hook payload. Empty input is not an error; it yields an
// Input with no paths, so canon hook can also run from git hooks with args.
func ParseInput(r io.Reader) (*Input, error) {
	data, err := io.ReadAll(io.LimitReader(r, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read hook input: %w", err)
	}
	in := &Input{}
	if len(bytes.TrimSpace(data)) == 0 {
		return in, nil
	}
	if err := json.Unmarshal(data, in); err != nil {
		return nil, fmt.Errorf("parse hook input: %w", err)
	}
	return in, nil
}

// Paths returns the file the tool touched, if any.
func (in *Input) Paths() []string {
	for _, p := range []string{in.ToolInput.FilePath, in.ToolInput.NotebookPath} {
		if p != "" {
			return []string{p}
		}
	}
	return nil
}

// Result is what the hook decided.
type Result struct {
	// Code is the process exit code: 0 to continue, ExitBlock to send the
	// message to the agent.
	Code int
	// Message goes to stderr.
	Message string
	Check   *engine.CheckResult
}

// Run checks paths under root and decides whether to block. Paths outside
// root are ignored, so an edit to a scratch file never trips the hook.
func Run(ctx context.Context, e *engine.Engine, root string, paths []string, threshold findings.Severity) (*Result, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	var inside []string
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			p = filepath.Join(abs, p)
		}
		rel, err := filepath.Rel(abs, p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		inside = append(inside, rel)
	}
	if len(inside) == 0 {
		return &Result{}, nil
	}
	res, err := e.Check(ctx, engine.CheckRequest{Root: abs, Paths: inside, MinSeverity: threshold})
	if err != nil {
		return nil, fmt.Errorf("hook check: %w", err)
	}
	out := &Result{Check: res}
	if len(res.Findings) == 0 {
		return out, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "canon found %d issue(s) in %s. Fix them before continuing; run `canon explain <rule>` for the full reasoning.\n", len(res.Findings), strings.Join(inside, ", "))
	for _, f := range res.Findings {
		fmt.Fprintf(&b, "- %s:%d [%s %s] %s (why: %s)", f.Path, f.Line, f.Severity, f.Rule, f.Message, f.Why)
		if f.Fix != nil {
			fmt.Fprintf(&b, " fix: %s", strings.ReplaceAll(f.Fix.Suggestion, "\n", " "))
		}
		b.WriteByte('\n')
	}
	out.Code = ExitBlock
	out.Message = b.String()
	return out, nil
}
