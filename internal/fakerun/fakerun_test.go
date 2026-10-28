package fakerun

import (
	"context"
	"errors"
	"testing"

	"github.com/rossedman/canon/internal/checkers"
)

func TestRunner(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	r := New().On("git", checkers.Result{ExitCode: 1}).On("git diff", checkers.Result{Stdout: []byte("d")}).Fail("bad", boom).Missing("hadolint")
	ctx := context.Background()
	if res, err := r.Run(ctx, ".", "git", "diff", "HEAD"); err != nil || string(res.Stdout) != "d" {
		t.Fatalf("longest prefix not used: %+v %v", res, err)
	}
	if res, _ := r.Run(ctx, ".", "git", "status"); res.ExitCode != 1 {
		t.Fatal("shorter prefix not used")
	}
	if _, err := r.Run(ctx, ".", "bad"); !errors.Is(err, boom) {
		t.Fatal("scripted error not returned")
	}
	if _, err := r.Run(ctx, ".", "other"); !errors.Is(err, ErrUnscripted) {
		t.Fatal("unscripted command did not fail")
	}
	if len(r.Calls()) != 4 {
		t.Fatalf("calls = %v", r.Calls())
	}
	if _, err := r.LookPath("hadolint"); err == nil {
		t.Fatal("missing binary found")
	}
	if p, err := r.LookPath("ruff"); err != nil || p == "" {
		t.Fatal("present binary not found")
	}
}
