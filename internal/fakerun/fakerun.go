// Package fakerun is a scripted checkers.Runner for tests, so engine, git
// and hook tests run without real linters or a git repository.
package fakerun

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/rossedman/canon/internal/checkers"
)

// Response is what a scripted command returns.
type Response struct {
	Result checkers.Result
	Err    error
}

// Runner answers commands by the longest matching "name arg..." prefix.
type Runner struct {
	mu        sync.Mutex
	responses map[string]Response
	missing   map[string]bool
	calls     []string
}

// New returns a Runner with no scripted commands.
func New() *Runner {
	return &Runner{responses: map[string]Response{}, missing: map[string]bool{}}
}

// On scripts the response for commands starting with prefix.
func (r *Runner) On(prefix string, res checkers.Result) *Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses[prefix] = Response{Result: res}
	return r
}

// Fail scripts an error for commands starting with prefix.
func (r *Runner) Fail(prefix string, err error) *Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses[prefix] = Response{Err: err}
	return r
}

// Missing marks binaries LookPath should not find.
func (r *Runner) Missing(names ...string) *Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range names {
		r.missing[n] = true
	}
	return r
}

// Calls returns every command run so far.
func (r *Runner) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

// ErrUnscripted is returned for a command with no scripted response.
var ErrUnscripted = errors.New("fakerun: no response scripted")

// Run returns the response with the longest prefix matching the command.
func (r *Runner) Run(_ context.Context, _, name string, args ...string) (checkers.Result, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, cmd)
	best, found := "", false
	for p := range r.responses {
		if strings.HasPrefix(cmd, p) && len(p) >= len(best) {
			best, found = p, true
		}
	}
	if !found {
		return checkers.Result{}, ErrUnscripted
	}
	res := r.responses[best]
	return res.Result, res.Err
}

// LookPath finds every binary not marked missing.
func (r *Runner) LookPath(name string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.missing[name] {
		return "", errors.New(name + " not found")
	}
	return "/fake/bin/" + name, nil
}
