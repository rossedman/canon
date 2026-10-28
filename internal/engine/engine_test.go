package engine

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rossedman/canon/canon"
	"github.com/rossedman/canon/internal/checkers"
	"github.com/rossedman/canon/internal/fakerun"
	"github.com/rossedman/canon/internal/findings"
)

// external lists the binaries of every external checker.
var external = []string{"golangci-lint", "ruff", "hadolint", "tflint", "checkov", "yamllint"}

// newEngine loads the embedded canon with a fake runner where every
// external checker is missing unless scripted.
func newEngine(t testing.TB) (*Engine, *fakerun.Runner) {
	t.Helper()
	e, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	r := fakerun.New().Missing(external...)
	e.Runner = r
	return e, r
}

// repo returns the path of a fixture repo.
func repo(name string) string {
	return filepath.Join("..", "..", "testdata", "repos", name)
}

// ruleIDs lists the rule of each finding.
func ruleIDs(fs []findings.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

func TestLintEmbeddedCanon(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	if ps := e.Lint(); len(ps) > 0 {
		t.Fatalf("embedded canon has problems: %+v", ps)
	}
	if len(e.Index()) != len(e.Records.All()) {
		t.Fatal("index size mismatch")
	}
}

// copyCanon copies the embedded canon to a temp dir so tests can break it.
func copyCanon(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	err := fs.WalkDir(canon.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, p)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fs.ReadFile(canon.FS, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLintFindsProblems(t *testing.T) {
	t.Parallel()
	dir := copyCanon(t)
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	rec := filepath.Join(dir, "records/go/go-errors-wrap.md")
	b, _ := os.ReadFile(rec)
	must(os.WriteFile(rec, []byte(strings.Replace(string(b), "## Consequences", "## Later", 1)), 0o644))
	rulesPath := filepath.Join(dir, "rules/rules.yaml")
	rb, _ := os.ReadFile(rulesPath)
	rs := strings.Replace(string(rb), "codes: [one-command-per-file]", "codes: [no-such-check]", 1)
	rs = strings.Replace(rs, "checker: yamllint", "checker: mystery", 1)
	must(os.WriteFile(rulesPath, []byte(rs), 0o644))
	must(os.WriteFile(filepath.Join(dir, "profiles/extra.yaml"), []byte("name: extra\ndetect: {all: [x]}\nscopes: [cobol]\n"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "templates/dockerfile/template.yaml"), []byte("root: \"\"\nrecords: [ghost]\n"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "templates/broken"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "templates/broken/template.yaml"), []byte("root: \"{{.nope}}\"\n"), 0o644))
	e, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, p := range e.Lint() {
		text.WriteString(p.Kind + " " + p.Ref + ": " + p.Message + "\n")
	}
	for _, want := range []string{
		`record go-errors-wrap: missing required section "## Consequences"`,
		`rule canon-one-command-per-file: canon has no built-in check "no-such-check"`,
		`rule yamllint-baseline: checker "mystery" is not known`,
		`profile extra: scope "cobol"`,
		`template dockerfile: record "ghost" does not exist`,
		"template broken:",
	} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("missing %q in\n%s", want, text.String())
		}
	}
	must(os.WriteFile(filepath.Join(dir, "templates/broken/template.yaml"), []byte("vars: ["), 0o644))
	e, _ = LoadDir(dir)
	if ps := e.Lint(); !slices.ContainsFunc(ps, func(p Problem) bool { return p.Kind == "template" && p.Ref == "" }) {
		t.Errorf("unlistable templates not reported: %+v", ps)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()
	full := fstest.MapFS{}
	_ = fs.WalkDir(canon.FS, ".", func(p string, d fs.DirEntry, _ error) error {
		if !d.IsDir() {
			b, _ := fs.ReadFile(canon.FS, p)
			full[p] = &fstest.MapFile{Data: b}
		}
		return nil
	})
	without := func(prefix string) fstest.MapFS {
		out := fstest.MapFS{}
		for k, v := range full {
			if !strings.HasPrefix(k, prefix) {
				out[k] = v
			}
		}
		return out
	}
	for _, missing := range []string{"records/", "rules/rules.yaml", "profiles/", "principles.md"} {
		if _, err := Load(without(missing)); err == nil {
			t.Errorf("without %s: want error", missing)
		}
	}
	if _, err := Load(full); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(t.TempDir()); err == nil {
		t.Error("empty dir: want error")
	}
}

func TestCheckBadRepoBuiltinOnly(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	res, err := e.Check(context.Background(), CheckRequest{Root: repo("bad"), MinSeverity: findings.Suggestion})
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile != "go-cli" || res.FilesChecked < 8 {
		t.Fatalf("profile %s, %d files", res.Profile, res.FilesChecked)
	}
	got := strings.Join(ruleIDs(res.Findings), " ")
	for _, want := range []string{"canon-workflow-permissions", "canon-actions-pinned", "canon-release-signing", "canon-oss-files", "canon-base-digest", "canon-final-stage-base", "canon-final-stage-user", "canon-one-command-per-file", "canon-go-doc-comments", "canon-go-ux-libraries"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	if strings.Contains(got, "canon-uv-lock") {
		t.Error("python records are not in the go-cli profile")
	}
	var skipped []string
	for _, s := range res.Skipped {
		skipped = append(skipped, s.Checker)
	}
	if strings.Join(skipped, ",") != "golangci-lint,hadolint,yamllint" {
		t.Fatalf("skipped = %v", skipped)
	}
	f := res.Findings[0]
	if f.Why == "" || f.Record == "" || f.Severity == "" {
		t.Fatalf("finding lacks trace: %+v", f)
	}
	if res.Summary.Total() != len(res.Findings) {
		t.Fatal("summary does not match findings")
	}
}

func TestCheckGoodRepoIsClean(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	res, err := e.Check(context.Background(), CheckRequest{Root: repo("good"), MinSeverity: findings.Suggestion})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("good repo has findings: %+v", res.Findings)
	}
}

func TestCheckMapsExternalIssues(t *testing.T) {
	t.Parallel()
	e, r := newEngine(t)
	hadolint, err := os.ReadFile("../checkers/testdata/hadolint.json")
	if err != nil {
		t.Fatal(err)
	}
	e.Runner = fakerun.New().Missing("golangci-lint", "ruff", "tflint", "checkov").
		On("hadolint", checkers.Result{Stdout: append(hadolint[:len(hadolint)-1:len(hadolint)-1], []byte(`,{"code":"DL3006","file":"other/Dockerfile","line":1,"message":"outside"},{"code":"DL3002","file":"Dockerfile","line":2,"message":"root"}]`)...)}).
		On("yamllint", checkers.Result{ExitCode: 3, Stderr: []byte("broken config")})
	_ = r
	res, err := e.Check(context.Background(), CheckRequest{Root: repo("bad"), Paths: []string{"Dockerfile", ".github"}, Scope: []string{"containers", "yaml"}})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(ruleIDs(res.Findings), " ")
	if !strings.Contains(got, "hadolint-hygiene") || !strings.Contains(got, "hadolint-user") || strings.Contains(got, "canon-actions-pinned") {
		t.Fatalf("rules = %s", got)
	}
	for _, f := range res.Findings {
		if f.Path == "other/Dockerfile" {
			t.Fatal("issue outside the requested files was kept")
		}
		if f.Rule == "hadolint-hygiene" && !strings.HasPrefix(f.Message, "DL30") {
			t.Fatalf("message lacks code: %q", f.Message)
		}
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Checker != "yamllint" || !strings.Contains(res.Skipped[0].Reason, "broken config") {
		t.Fatalf("skipped = %+v", res.Skipped)
	}
}

func TestCheckUnmappedCodes(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	e.Runner = fakerun.New().Missing("hadolint", "ruff", "tflint", "checkov", "yamllint").
		On("golangci-lint", checkers.Result{Stdout: []byte(`{"Issues":[{"FromLinter":"gocritic","Text":"x","Pos":{"Filename":"internal/thing/thing.go","Line":1}}]}`)})
	res, err := e.Check(context.Background(), CheckRequest{Root: repo("bad"), Paths: []string{"internal"}, Scope: []string{"go"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Unmapped != 1 {
		t.Fatalf("unmapped = %d", res.Unmapped)
	}
}

func TestCheckRequests(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	ctx := context.Background()
	abs, _ := filepath.Abs(filepath.Join(repo("bad"), "Dockerfile"))
	res, err := e.Check(ctx, CheckRequest{Root: repo("bad"), Paths: []string{abs, "Dockerfile"}, MinSeverity: findings.Error})
	if err != nil || res.FilesChecked != 1 {
		t.Fatalf("abs path: %v %+v", err, res)
	}
	for _, f := range res.Findings {
		if f.Severity != findings.Error {
			t.Fatal("min severity not applied")
		}
	}
	for _, bad := range [][]string{{"../good"}, {"missing.go"}, {"/"}} {
		if _, err := e.Check(ctx, CheckRequest{Root: repo("bad"), Paths: bad}); err == nil {
			t.Errorf("paths %v: want error", bad)
		}
	}
	if _, err := e.Check(ctx, CheckRequest{Root: repo("bad"), Profile: "nope"}); err == nil {
		t.Error("unknown profile: want error")
	}
	res, err = e.Check(ctx, CheckRequest{Root: repo("bad"), Profile: "py-service", Paths: []string{"py"}})
	if err != nil || res.Profile != "py-service" || !slices.Contains(ruleIDs(res.Findings), "canon-uv-lock") {
		t.Fatalf("forced profile: %v %+v", err, res)
	}
}

func TestCheckChangedOnly(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	r := fakerun.New().Missing(external...).
		On("git merge-base", checkers.Result{Stdout: []byte("abc")}).
		On("git diff --name-only", checkers.Result{Stdout: []byte("Dockerfile\ndeleted.go\n")}).
		On("git ls-files", checkers.Result{})
	e.Runner = r
	res, err := e.Check(context.Background(), CheckRequest{Root: repo("bad"), ChangedOnly: true, Base: "origin/main"})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesChecked != 1 || slices.Contains(ruleIDs(res.Findings), "canon-oss-files") {
		t.Fatalf("changed-only = %+v", res)
	}
	e.Runner = fakerun.New()
	if _, err := e.Check(context.Background(), CheckRequest{Root: repo("bad"), ChangedOnly: true}); err == nil {
		t.Fatal("git failure should fail the check")
	}
}

func TestCheckCancelled(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Check(ctx, CheckRequest{Root: repo("bad")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestNoCatchAllProfile(t *testing.T) {
	t.Parallel()
	e, _ := newEngine(t)
	e.Profiles = e.Profiles[:1]
	if _, err := e.Check(context.Background(), CheckRequest{Root: t.TempDir()}); err == nil {
		t.Fatal("want error with no matching profile")
	}
	if _, err := e.Profile(t.TempDir()); err == nil {
		t.Fatal("profile: want error with no matching profile")
	}
}

func BenchmarkCheckBuiltin(b *testing.B) {
	e, _ := newEngine(b)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := e.Check(ctx, CheckRequest{Root: repo("bad")}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecide(b *testing.B) {
	e, _ := newEngine(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := e.Decide(DecideRequest{Question: "how do we handle errors in Go?"}); err != nil {
			b.Fatal(err)
		}
	}
}
