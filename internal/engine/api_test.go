package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/canon/internal/checkers"
	"github.com/rossedman/canon/internal/fakerun"
	"github.com/rossedman/canon/internal/scaffold"
)

func TestDecide(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	tests := []struct {
		q     string
		scope []string
		want  string
	}{
		{"how do we handle errors in Go?", nil, "go-errors-wrap"},
		{"what should the runtime image of a container be", nil, "containers-distroless-base"},
		{"pin the base image", nil, "containers-pin-base-images"},
		{"should github actions be pinned", []string{"ci"}, "ci-pin-actions-sha"},
		{"how should a CLI load config", nil, "go-config-viper"},
		{"python dependencies lock file", nil, "python-uv"},
	}
	for _, tt := range tests {
		res, err := e.Decide(DecideRequest{Question: tt.q, Scope: tt.scope})
		if err != nil {
			t.Fatal(err)
		}
		if res.Gap || res.Matches[0].ID != tt.want {
			t.Errorf("%q: got %+v, want %s first", tt.q, res.Matches, tt.want)
		}
		if res.Matches[0].Decision == "" || res.Matches[0].Why == "" || len(res.Matches) > 3 {
			t.Errorf("%q: match incomplete or over limit", tt.q)
		}
	}
	res, _ := e.Decide(DecideRequest{Question: "how do we bake sourdough bread", Limit: 1})
	if !res.Gap || len(res.Matches) != 0 || res.Message == "" {
		t.Fatalf("expected a gap, got %+v", res)
	}
	res, _ = e.Decide(DecideRequest{Question: "alpine runtime images"})
	for _, m := range res.Matches {
		if m.ID == "containers-alpine-base" {
			t.Fatal("superseded record served as advice")
		}
	}
	if _, err := e.Decide(DecideRequest{Question: "  "}); err == nil {
		t.Fatal("empty question: want error")
	}
}

func TestExplain(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	res, err := e.Explain(ExplainRequest{Rule: "canon-final-stage-base"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Rule.ID != "canon-final-stage-base" || res.Record.ID != "containers-distroless-base" || !strings.HasPrefix(res.Markdown, "---\n") {
		t.Fatalf("explain rule = %+v", res)
	}
	if len(res.Supersedes) != 1 || res.Supersedes[0].ID != "containers-alpine-base" || len(res.Rules) != 1 {
		t.Fatalf("relations = %+v / %+v", res.Supersedes, res.Rules)
	}
	old, err := e.Explain(e.Resolve("containers-alpine-base"))
	if err != nil || len(old.SupersededBy) != 1 || len(old.Rules) != 0 || old.Rules == nil {
		t.Fatalf("explain old = %+v, %v", old, err)
	}
	if req := e.Resolve("hadolint-user"); req.Rule != "hadolint-user" {
		t.Fatal("Resolve should spot a rule ID")
	}
	for _, req := range []ExplainRequest{{}, {Rule: "a", Record: "b"}} {
		if _, err := e.Explain(req); err == nil {
			t.Errorf("%+v: want error", req)
		}
	}
	for _, req := range []ExplainRequest{{Rule: "ghost"}, {Record: "ghost"}} {
		if _, err := e.Explain(req); !errors.Is(err, ErrNotFound) {
			t.Errorf("%+v: want ErrNotFound, got %v", req, err)
		}
	}
}

func TestProfile(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	e.Runner = fakerun.New().Missing("hadolint")
	res, err := e.Profile(repo("bad"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile != "go-cli" || len(res.Records) == 0 || strings.Join(res.Missing, ",") != "hadolint" {
		t.Fatalf("profile = %+v", res)
	}
	for _, c := range res.Checkers {
		if c.Name == "canon" && (!c.Installed || c.Binary != "") {
			t.Fatal("built-in checker should always be installed")
		}
	}
	v, err := e.ProfileRecords("tf-module")
	if err != nil || v["profile"] == nil || len(v["records"].([]RecordRef)) == 0 {
		t.Fatalf("ProfileRecords = %v, %v", v, err)
	}
	if _, err := e.ProfileRecords("ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatal("want ErrNotFound")
	}
}

// reviewDiff adds a Dockerfile change and a new dependency to the bad repo.
const reviewDiff = `diff --git a/Dockerfile b/Dockerfile
new file mode 100644
--- /dev/null
+++ b/Dockerfile
@@ -0,0 +1 @@
+FROM debian:bookworm
diff --git a/go.mod b/go.mod
--- a/go.mod
+++ b/go.mod
@@ -1 +1,2 @@
+require github.com/google/uuid v1.6.0
diff --git a/removed.go b/removed.go
deleted file mode 100644
--- a/removed.go
+++ /dev/null
@@ -1 +0,0 @@
-package x
`

func TestReview(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	res, err := e.Review(context.Background(), ReviewRequest{Root: repo("bad"), Diff: reviewDiff})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Files, ",") != "Dockerfile,go.mod,removed.go" || res.Profile != "go-cli" {
		t.Fatalf("files = %v", res.Files)
	}
	var kinds []string
	for _, c := range res.Concerns {
		kinds = append(kinds, c.Kind+"→"+c.Record)
	}
	if strings.Join(kinds, ",") != "new-dockerfile→containers-distroless-base,new-dependency→go-lint-baseline" {
		t.Fatalf("concerns = %v", kinds)
	}
	touched := strings.Join(res.RecordsTouched, ",")
	if !strings.Contains(touched, "containers-distroless-base") || !strings.Contains(touched, "containers-pin-base-images") {
		t.Fatalf("records touched = %s", touched)
	}
	if len(res.Findings) == 0 || res.Summary.Error == 0 {
		t.Fatal("findings on changed files are missing")
	}
}

func TestReviewComputesDiff(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	e.Runner = fakerun.New().Missing(external...).
		On("git merge-base", checkers.Result{Stdout: []byte("abc")}).
		On("git diff --no-color", checkers.Result{Stdout: []byte(reviewDiff)}).
		On("git ls-files", checkers.Result{})
	res, err := e.Review(context.Background(), ReviewRequest{Root: repo("bad")})
	if err != nil || len(res.Concerns) != 2 {
		t.Fatalf("review = %+v, %v", res, err)
	}
	empty, err := e.Review(context.Background(), ReviewRequest{Root: repo("bad"), Diff: "nothing here"})
	if err != nil || len(empty.Findings) != 0 || len(empty.Files) != 0 {
		t.Fatalf("empty diff = %+v, %v", empty, err)
	}
	e.Runner = fakerun.New()
	if _, err := e.Review(context.Background(), ReviewRequest{Root: repo("bad"), Base: "main"}); err == nil {
		t.Fatal("git failure: want error")
	}
	e.Profiles = e.Profiles[:1]
	if _, err := e.Review(context.Background(), ReviewRequest{Root: t.TempDir(), Diff: reviewDiff}); err == nil {
		t.Fatal("no profile: want error")
	}
}

func TestScaffoldEveryKindPassesItsOwnCanon(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	kinds, err := e.Kinds()
	if err != nil || len(kinds) != 6 {
		t.Fatalf("kinds = %v, %v", kinds, err)
	}
	for _, k := range kinds {
		t.Run(k.Kind, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			dry, err := e.Scaffold(context.Background(), ScaffoldRequest{Root: root, Kind: k.Kind, Name: "demo", DryRun: true})
			if err != nil || len(dry.Files) == 0 || len(dry.Records) == 0 || dry.Check != nil {
				t.Fatalf("dry run = %+v, %v", dry, err)
			}
			if _, err := os.Stat(filepath.Join(root, dry.Files[0])); err == nil {
				t.Fatal("dry run wrote files")
			}
			res, err := e.Scaffold(context.Background(), ScaffoldRequest{Root: root, Kind: k.Kind, Name: "demo", Dest: "out"})
			if err != nil {
				t.Fatal(err)
			}
			if !res.Passed || res.Check == nil {
				t.Fatalf("generated %s fails its own canon: %+v", k.Kind, res.Check.Findings)
			}
			for _, f := range res.Check.Findings {
				if k.Kind == "py-service" && f.Rule == "canon-uv-lock" {
					continue // uv.lock needs a resolver; the template's next step runs uv lock
				}
				t.Errorf("finding in generated %s: %s:%d %s %s", k.Kind, f.Path, f.Line, f.Rule, f.Message)
			}
			if _, err := os.Stat(filepath.Join(root, res.Files[0])); err != nil {
				t.Fatalf("file not written: %v", err)
			}
			if _, err := e.Scaffold(context.Background(), ScaffoldRequest{Root: root, Kind: k.Kind, Name: "demo", Dest: "out"}); !errors.Is(err, scaffold.ErrExists) {
				t.Fatalf("second write should refuse to overwrite: %v", err)
			}
		})
	}
}

func TestScaffoldErrors(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	ctx := context.Background()
	if _, err := e.Scaffold(ctx, ScaffoldRequest{Root: t.TempDir(), Kind: "nope", Name: "x", DryRun: true}); err == nil {
		t.Error("unknown kind: want error")
	}
	if _, err := e.Scaffold(ctx, ScaffoldRequest{Root: t.TempDir(), Kind: "go-cli", Name: "x", Dest: "../escape", DryRun: true}); err == nil {
		t.Error("dest outside root: want error")
	}
	if got := mustAbs(""); !filepath.IsAbs(got) {
		t.Errorf("mustAbs(\"\") = %q", got)
	}
}
