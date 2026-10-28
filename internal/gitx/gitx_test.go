package gitx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rossedman/canon/internal/checkers"
	"github.com/rossedman/canon/internal/fakerun"
)

// ok wraps stdout in a successful result.
func ok(s string) checkers.Result { return checkers.Result{Stdout: []byte(s)} }

func TestMergeBase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := fakerun.New().On("git merge-base origin/main HEAD", ok("abc\n")).On("git rev-parse HEAD", ok("head\n"))
	if mb, err := MergeBase(ctx, r, ".", "origin/main"); err != nil || mb != "abc" {
		t.Fatalf("MergeBase = %q, %v", mb, err)
	}
	r.On("git merge-base nope", checkers.Result{ExitCode: 128, Stderr: []byte("bad ref")})
	if mb, _ := MergeBase(ctx, r, ".", "nope"); mb != "head" {
		t.Fatalf("fallback = %q", mb)
	}
	if mb, _ := MergeBase(ctx, r, ".", ""); mb != "head" {
		t.Fatalf("empty base = %q", mb)
	}
	bad := fakerun.New().On("git rev-parse", checkers.Result{ExitCode: 128, Stderr: []byte("not a repo")})
	if _, err := MergeBase(ctx, bad, ".", ""); err == nil || !strings.Contains(err.Error(), "not a repo") {
		t.Fatalf("err = %v", err)
	}
	failing := fakerun.New().Fail("git", errors.New("no git"))
	if _, err := MergeBase(ctx, failing, ".", ""); err == nil {
		t.Fatal("want error when git cannot run")
	}
}

func TestChangedFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := fakerun.New().On("git merge-base", ok("mb")).
		On("git diff --name-only", ok("b.go\na.go\n")).
		On("git ls-files", ok("new.go\na.go\n"))
	got, err := ChangedFiles(ctx, r, ".", "main")
	if err != nil || strings.Join(got, ",") != "a.go,b.go,new.go" {
		t.Fatalf("ChangedFiles = %v, %v", got, err)
	}
	for _, failing := range []string{"git merge-base", "git diff", "git ls-files"} {
		r := fakerun.New().On("git", ok("")).On(failing, checkers.Result{ExitCode: 1}).On("git rev-parse", ok("h"))
		if failing == "git merge-base" {
			r.On("git rev-parse", checkers.Result{ExitCode: 1})
		}
		if _, err := ChangedFiles(ctx, r, ".", "main"); err == nil {
			t.Errorf("%s failing: want error", failing)
		}
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := fstest.MapFS{"new.txt": {Data: []byte("one\ntwo\n")}, "bin": {Data: []byte{0, 1}}, "empty": {}}
	r := fakerun.New().On("git merge-base", ok("mb")).
		On("git diff --no-color", ok("diff --git a/x b/x\n")).
		On("git ls-files", ok("new.txt\nbin\nempty\ngone\n"))
	d, err := Diff(ctx, r, ".", "main", repo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d, "+++ b/new.txt\n@@ -0,0 +1,2 @@\n+one\n+two\n") || strings.Contains(d, "b/bin") || !strings.Contains(d, "b/empty\n") {
		t.Fatalf("diff =\n%s", d)
	}
	for _, failing := range []string{"git diff", "git ls-files"} {
		r := fakerun.New().On("git", ok("")).On(failing, checkers.Result{ExitCode: 1})
		if _, err := Diff(ctx, r, ".", "main", repo); err == nil {
			t.Errorf("%s failing: want error", failing)
		}
	}
	if _, err := Diff(ctx, fakerun.New(), ".", "", repo); err == nil {
		t.Error("no git: want error")
	}
}
