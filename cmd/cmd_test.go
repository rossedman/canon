package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// bad and good are the fixture repos.
var (
	bad  = filepath.Join("..", "testdata", "repos", "bad")
	good = filepath.Join("..", "testdata", "repos", "good")
)

// resetFlags puts every flag back to its default, since cobra keeps values
// between executions in one process.
func resetFlags(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// clearConfig replaces any config file an earlier test loaded into viper's
// global state with an empty one.
func clearConfig(t *testing.T) {
	t.Helper()
	empty := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(empty, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	viper.SetConfigFile(empty)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
}

// run executes canon with args and returns stdout, stderr and the error.
func run(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	resetFlags(rootCmd)
	clearConfig(t)
	cfgFile = ""
	t.Setenv("PATH", t.TempDir()) // no external linters, so results are deterministic
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(args)
	err := rootCmd.ExecuteContext(context.Background())
	return out.String(), errb.String(), err
}

// exitCode returns the code carried by an exitError, or -1.
func exitCode(err error) int {
	var ee exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return -1
}

func TestVersion(t *testing.T) {
	out, _, err := run(t, "", "version")
	if err != nil || !strings.HasPrefix(out, "canon dev (commit none") {
		t.Fatalf("out = %q, %v", out, err)
	}
	if ee := (exitError{code: 3}); ee.Error() != "exit status 3" {
		t.Fatal(ee.Error())
	}
}

func TestCheck(t *testing.T) {
	out, _, err := run(t, "", "check", "--root", bad)
	if exitCode(err) != 1 || !strings.Contains(out, "canon-final-stage-base") || !strings.Contains(out, "skipped golangci-lint") {
		t.Fatalf("check bad: %v\n%s", err, out)
	}
	out, _, err = run(t, "", "check", "--root", good, "--min-severity", "suggestion")
	if err != nil || !strings.Contains(out, "no findings") {
		t.Fatalf("check good: %v\n%s", err, out)
	}
	out, _, err = run(t, "", "check", "--root", bad, "--json", "--fail-on", "none", "--scope", "containers", "Dockerfile")
	var res struct {
		Profile  string
		Findings []struct{ Rule string }
	}
	if err != nil || json.Unmarshal([]byte(out), &res) != nil || res.Profile != "go-cli" || len(res.Findings) == 0 {
		t.Fatalf("check json: %v\n%s", err, out)
	}
	out, _, err = run(t, "", "check", "--root", bad, "--sarif", "--fail-on", "warning", "Dockerfile")
	if exitCode(err) != 1 || !strings.Contains(out, `"version": "2.1.0"`) || !strings.Contains(out, `"canon-final-stage-base"`) {
		t.Fatalf("check sarif: %v\n%.200s", err, out)
	}
	for _, args := range [][]string{
		{"check", "--min-severity", "fatal"},
		{"check", "--root", bad, "--fail-on", "fatal"},
		{"check", "--root", bad, "missing.go"},
		{"check", "--canon-dir", t.TempDir()},
	} {
		if _, _, err := run(t, "", args...); err == nil || exitCode(err) != -1 {
			t.Errorf("%v: want a plain error, got %v", args, err)
		}
	}
}

func TestCheckEnvAndConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "canon.yaml")
	if err := os.WriteFile(cfg, []byte("check:\n  fail-on: none\n  min-severity: error\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(t, "", "check", "--config", cfg, "--root", bad)
	if err != nil || !strings.Contains(out, " 0 warnings") {
		t.Fatalf("config file not applied: %v\n%s", err, out)
	}
	t.Setenv("CANON_CHECK_MIN_SEVERITY", "suggestion")
	out, _, _ = run(t, "", "check", "--config", cfg, "--root", bad)
	if strings.Contains(out, " 0 warnings") {
		t.Fatalf("env should beat the config file:\n%s", out)
	}
	out, _, _ = run(t, "", "check", "--config", cfg, "--root", bad, "--min-severity", "error")
	if !strings.Contains(out, " 0 warnings") {
		t.Fatalf("flag should beat env:\n%s", out)
	}
	_, _, err = run(t, "", "check", "--config", filepath.Join(dir, "missing.yaml"), "--root", good)
	if err != nil {
		t.Fatalf("a missing --config is reported but not fatal: %v", err)
	}
}

func TestFindConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Chdir(t.TempDir())
	if got := findConfig(); got != "" {
		t.Fatalf("found %q with no config files", got)
	}
	if err := os.WriteFile(".canon.yaml", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := findConfig(); got != ".canon.yaml" {
		t.Fatalf("got %q", got)
	}
	if err := os.WriteFile(filepath.Join(home, ".canon.yaml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := findConfig(); got != filepath.Join(home, ".canon.yaml") {
		t.Fatalf("home should beat cwd, got %q", got)
	}
	xdg := filepath.Join(home, "xdg", "canon", "config.yaml")
	_ = os.MkdirAll(filepath.Dir(xdg), 0o755)
	if err := os.WriteFile(xdg, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := findConfig(); got != xdg {
		t.Fatalf("xdg should win, got %q", got)
	}
	if logLevel("debug").String() != "DEBUG" || logLevel("nope").String() != "WARN" {
		t.Fatal("logLevel wrong")
	}
}

func TestDecideExplainProfileLint(t *testing.T) {
	out, _, err := run(t, "", "decide", "how", "do", "we", "handle", "errors", "in", "Go")
	if err != nil || !strings.HasPrefix(out, "go-errors-wrap") {
		t.Fatalf("decide: %v\n%s", err, out)
	}
	out, _, _ = run(t, "", "decide", "--json", "--limit", "1", "--scope", "go", "bake", "bread")
	if !strings.Contains(out, `"gap": true`) {
		t.Fatalf("decide gap:\n%s", out)
	}
	if _, _, err := run(t, "", "decide", " "); err == nil {
		t.Fatal("blank question: want error")
	}
	out, _, err = run(t, "", "explain", "canon-final-stage-base")
	if err != nil || !strings.Contains(out, "containers-distroless-base") || !strings.Contains(out, "supersedes containers-alpine-base") {
		t.Fatalf("explain: %v\n%s", err, out)
	}
	out, _, _ = run(t, "", "explain", "--json", "go-errors-wrap")
	if !strings.Contains(out, `"markdown"`) {
		t.Fatalf("explain json:\n%s", out)
	}
	if _, _, err := run(t, "", "explain", "ghost"); err == nil {
		t.Fatal("explain ghost: want error")
	}
	out, _, err = run(t, "", "profile", "--root", "..", "testdata/repos/bad")
	if err != nil || !strings.Contains(out, "profile go-cli") || !strings.Contains(out, "hadolint       missing") {
		t.Fatalf("profile: %v\n%s", err, out)
	}
	abs, _ := filepath.Abs(good)
	out, _, _ = run(t, "", "profile", "--json", abs)
	if !strings.Contains(out, `"profile": "go-cli"`) {
		t.Fatalf("profile json:\n%s", out)
	}
	out, _, err = run(t, "", "lint")
	if err != nil || !strings.Contains(out, "25 records are valid") {
		t.Fatalf("lint: %v\n%s", err, out)
	}
	out, _, _ = run(t, "", "lint", "--json")
	if strings.TrimSpace(out) != "null" {
		t.Fatalf("lint json:\n%s", out)
	}
}

func TestLintBrokenCanon(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"records/go", "rules", "profiles", "templates"} {
		_ = os.MkdirAll(filepath.Join(dir, d), 0o755)
	}
	files := map[string]string{
		"records/go/x.md":   "---\nid: x\ntitle: X\nstatus: accepted\nscope: [go]\napplies_to: {files: ['**']}\nstrength: must\nupdated: 2026-01-01\n---\n## Context\nc\n",
		"rules/rules.yaml":  "rules: []\n",
		"profiles/all.yaml": "name: all\nscopes: [go]\n",
		"principles.md":     "# P\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	out, _, err := run(t, "", "lint", "--canon-dir", dir)
	if exitCode(err) != 1 || !strings.Contains(out, "missing required section") {
		t.Fatalf("lint broken: %v\n%s", err, out)
	}
}

func TestReview(t *testing.T) {
	diff := "diff --git a/Dockerfile b/Dockerfile\nnew file mode 100644\n--- /dev/null\n+++ b/Dockerfile\n@@ -0,0 +1 @@\n+FROM debian\n"
	out, _, err := run(t, diff, "review", "--root", bad, "--diff-file", "-")
	if err != nil || !strings.Contains(out, "new-dockerfile") || !strings.Contains(out, "records touched") {
		t.Fatalf("review: %v\n%s", err, out)
	}
	f := filepath.Join(t.TempDir(), "x.diff")
	if err := os.WriteFile(f, []byte(diff), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err = run(t, "", "review", "--root", bad, "--diff-file", f, "--json", "--fail-on", "error")
	if exitCode(err) != 1 || !strings.Contains(out, `"concerns"`) {
		t.Fatalf("review json: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"review", "--diff-file", "/no/such/file"},
		{"review", "--min-severity", "fatal"},
		{"review", "--root", bad},
	} {
		if _, _, err := run(t, "", args...); err == nil {
			t.Errorf("%v: want error", args)
		}
	}
}

func TestScaffold(t *testing.T) {
	root := t.TempDir()
	out, _, err := run(t, "", "scaffold", "--root", root, "go-cli", "hello")
	if err != nil || !strings.Contains(out, "would write") || !strings.Contains(out, "hello/cmd/root.go") {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
	out, _, err = run(t, "", "scaffold", "--root", root, "--write", "--var", "module=github.com/me/hello", "go-cli", "hello")
	if err != nil || !strings.Contains(out, "wrote") || !strings.Contains(out, "check: no findings") {
		t.Fatalf("write: %v\n%s", err, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "hello", "go.mod")); !strings.Contains(string(b), "module github.com/me/hello") {
		t.Fatalf("var not applied: %s", b)
	}
	out, _, _ = run(t, "", "scaffold", "--root", root, "--json", "dockerfile", "api", "--dest", "svc")
	if !strings.Contains(out, `"svc/Dockerfile"`) {
		t.Fatalf("json:\n%s", out)
	}
	if _, _, err := run(t, "", "scaffold", "--root", root, "--write", "go-cli", "hello"); err == nil {
		t.Fatal("overwrite: want error")
	}
	// A canon whose template fails its own checks makes scaffold exit 1.
	dir := t.TempDir()
	copyCanon(t, dir)
	tmpl := filepath.Join(dir, "templates", "dockerfile", "Dockerfile.tmpl")
	if err := os.WriteFile(tmpl, []byte("FROM debian:bookworm\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = run(t, "", "scaffold", "--canon-dir", dir, "--root", t.TempDir(), "--write", "dockerfile", "api")
	if exitCode(err) != 1 {
		t.Fatalf("failing template: %v", err)
	}
}

// copyCanon copies ../canon into dir.
func copyCanon(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(filepath.Join("..", "canon"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(filepath.Join("..", "canon"), p)
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHook(t *testing.T) {
	abs, _ := filepath.Abs(bad)
	payload := `{"cwd":"` + abs + `","hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(abs, "Dockerfile") + `"}}`
	_, stderr, err := run(t, payload, "hook")
	if exitCode(err) != 2 || !strings.Contains(stderr, "canon-final-stage-base") {
		t.Fatalf("hook: %v\n%s", err, stderr)
	}
	_, stderr, err = run(t, "", "hook", "--root", bad, "internal/thing/thing.go")
	if err != nil || stderr != "" {
		t.Fatalf("warnings should not block by default: %v %s", err, stderr)
	}
	_, _, err = run(t, "", "hook", "--root", bad, "--min-severity", "warning", "internal/thing/thing.go")
	if exitCode(err) != 2 {
		t.Fatalf("--min-severity warning should block: %v", err)
	}
	if _, _, err := run(t, "{", "hook"); err == nil {
		t.Fatal("bad payload: want error")
	}
	if _, _, err := run(t, "", "hook", "--min-severity", "x"); err == nil {
		t.Fatal("bad severity: want error")
	}
	if _, _, err := run(t, "", "hook", "--root", bad, "missing.go"); err == nil {
		t.Fatal("missing file: want error")
	}
	if _, _, err := run(t, "", "hook"); err != nil {
		t.Fatalf("empty payload should do nothing: %v", err)
	}
}

func TestServeHTTPStopsWithContext(t *testing.T) {
	resetFlags(rootCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rootCmd.SetArgs([]string{"serve", "--http", "127.0.0.1:0"})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("serve: %v", err)
	}
	rootCmd.SetArgs([]string{"serve", "--canon-dir", t.TempDir()})
	if err := rootCmd.ExecuteContext(context.Background()); err == nil {
		t.Fatal("bad canon: want error")
	}
	resetFlags(rootCmd)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs([]string{"serve"})
	_ = rootCmd.ExecuteContext(ctx)
	if Root() != rootCmd {
		t.Fatal("Root() wrong")
	}
}

func TestMustBindPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("want panic for a missing flag")
		}
	}()
	mustBind("x", nil)
}

func TestExecuteExitCodes(t *testing.T) {
	tests := []struct {
		args []string
		want int
	}{
		{[]string{"version"}, 0},
		{[]string{"check", "--root", bad}, 1},
		{[]string{"hook", "--root", bad, "Dockerfile"}, 2},
		{[]string{"no-such-command"}, 1},
	}
	for _, tt := range tests {
		resetFlags(rootCmd)
		clearConfig(t)
		cfgFile = ""
		t.Setenv("PATH", t.TempDir())
		var out bytes.Buffer
		rootCmd.SetOut(&out)
		rootCmd.SetErr(&out)
		rootCmd.SetArgs(tt.args)
		if got := execute(context.Background()); got != tt.want {
			t.Errorf("%v: exit %d, want %d\n%s", tt.args, got, tt.want, out.String())
		}
	}
	if repoRootWith("") != "." {
		t.Error("empty root should be .")
	}
}
