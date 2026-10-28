package checkers

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeRunner returns canned results keyed by binary name.
type fakeRunner struct {
	results map[string]Result
	errs    map[string]error
	calls   []string
	missing map[string]bool
}

// Run records the call and returns the canned result.
func (f *fakeRunner) Run(_ context.Context, _, name string, args ...string) (Result, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if err := f.errs[name]; err != nil {
		return Result{}, err
	}
	return f.results[name], nil
}

// LookPath reports binaries as installed unless marked missing.
func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.missing[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

// fixture reads a recorded tool output.
func fixture(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
