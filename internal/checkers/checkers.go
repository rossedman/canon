// Package checkers adapts real linters (golangci-lint, ruff, hadolint, tflint,
// checkov, yamllint) and canon's own built-in checks to one interface that
// returns structured issues. Checkers report facts; the engine maps each
// issue to a rule and a record.
package checkers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rossedman/canon/internal/findings"
)

// Result is the outcome of running an external command.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner runs external commands. Tests swap in a fake.
type Runner interface {
	// Run executes name with args in dir. A non-zero exit is reported in
	// Result.ExitCode, not as an error; err is for failures to start.
	Run(ctx context.Context, dir, name string, args ...string) (Result, error)
	// LookPath reports where a binary is, or an error if it is not installed.
	LookPath(name string) (string, error)
}

// ExecRunner runs commands with os/exec.
type ExecRunner struct{}

// Run executes the command and captures its output.
func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // checkers run fixed linter binaries
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	res := Result{Stdout: out.Bytes(), Stderr: errb.Bytes()}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		res.ExitCode = exit.ExitCode()
		return res, nil
	}
	if err != nil {
		return res, fmt.Errorf("run %s: %w", name, err)
	}
	return res, nil
}

// LookPath finds a binary on PATH.
func (ExecRunner) LookPath(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("find %s: %w", name, err)
	}
	return p, nil
}

// Issue is one problem a checker reported, before it is mapped to a rule.
type Issue struct {
	Code    string
	Path    string
	Line    int
	Column  int
	Message string
	Fix     *findings.Fix
}

// Request is what a checker is asked to look at.
type Request struct {
	// Root is the repo root on disk, where external tools run.
	Root string
	// FS is the repo, rooted at Root, for built-in checks.
	FS fs.FS
	// Files are repo-relative, slash-separated paths to check.
	Files []string
	// Codes are the codes in force; built-in checks run only these.
	Codes []string
	// Config is the path to the checker's config file, or "".
	Config string
	// WholeRepo is set when the whole repository is being checked, which
	// enables repo-level checks such as required files.
	WholeRepo bool
}

// Checker is one source of issues.
type Checker interface {
	// Name is the checker name used in rules.yaml.
	Name() string
	// Binary is the executable it needs, or "" when it is built in.
	Binary() string
	// ConfigName is the config file under canon/rules/ it reads, or "".
	ConfigName() string
	// Run checks the request's files.
	Run(ctx context.Context, run Runner, req Request) ([]Issue, error)
}

// All returns every checker canon knows, keyed by name.
func All() map[string]Checker {
	cs := []Checker{Builtin{}, GolangciLint{}, Ruff{}, Hadolint{}, TFLint{}, Checkov{}, Yamllint{}}
	out := make(map[string]Checker, len(cs))
	for _, c := range cs {
		out[c.Name()] = c
	}
	return out
}

// relPath turns a path a tool printed into a clean repo-relative slash path.
func relPath(root, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) && root != "" {
		if r, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(r, "..") {
			p = r
		}
	}
	p = path.Clean(filepath.ToSlash(p))
	return strings.TrimPrefix(p, "/")
}

// dirsOf returns the sorted unique directories of files, as "./dir" package
// patterns suitable for Go tools.
func dirsOf(files []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range files {
		d := "./" + path.Dir(f)
		if d == "./." {
			d = "."
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return out
}

// toolError builds an error for a tool that failed without parseable output.
func toolError(name string, res Result, err error) error {
	msg := strings.TrimSpace(string(res.Stderr))
	if msg == "" {
		msg = strings.TrimSpace(string(res.Stdout))
	}
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}
	if err != nil {
		return fmt.Errorf("%s exited %d: %s: %w", name, res.ExitCode, msg, err)
	}
	return fmt.Errorf("%s exited %d: %s", name, res.ExitCode, msg)
}
