package checkers

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// codes lists issue codes in order.
func codes(is []Issue) string {
	var out []string
	for _, i := range is {
		out = append(out, i.Code)
	}
	return strings.Join(out, ",")
}

func TestParsersOnRecordedOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		parse func([]byte) ([]Issue, error)
		file  string
		want  string
		first Issue
	}{
		{"golangci", func(b []byte) ([]Issue, error) { return ParseGolangci("", b) }, "golangci.json", "errorlint,gosec,revive,revive",
			Issue{Code: "errorlint", Path: "internal/thing/thing.go", Line: 11, Column: 33}},
		{"ruff", func(b []byte) ([]Issue, error) { return ParseRuff("", b) }, "ruff.json", "E401,I001,F401,S602",
			Issue{Code: "E401", Path: "py/app.py", Line: 1, Column: 1}},
		{"hadolint", func(b []byte) ([]Issue, error) { return ParseHadolint("", b) }, "hadolint.json", "DL3008,DL3009,DL3015",
			Issue{Code: "DL3008", Path: "Dockerfile", Line: 7, Column: 1}},
		{"tflint", func(b []byte) ([]Issue, error) { return ParseTFLint("", b) }, "tflint.json", "terraform_required_version,terraform_required_providers,terraform_typed_variables",
			Issue{Code: "terraform_required_version", Path: "infra/main.tf", Line: 1, Column: 1}},
		{"yamllint", func(b []byte) ([]Issue, error) { return ParseYamllint("", b), nil }, "yamllint.txt", "key-duplicates,colons,commas",
			Issue{Code: "key-duplicates", Path: "bad.yaml", Line: 2, Column: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			is, err := tt.parse(fixture(t, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			if got := codes(is); got != tt.want {
				t.Fatalf("codes = %s, want %s", got, tt.want)
			}
			f := is[0]
			if f.Code != tt.first.Code || f.Path != tt.first.Path || f.Line != tt.first.Line || f.Column != tt.first.Column || f.Message == "" {
				t.Fatalf("first issue = %+v, want %+v", f, tt.first)
			}
		})
	}
}

func TestParseCheckov(t *testing.T) {
	t.Parallel()
	is, err := ParseCheckov("", fixture(t, "checkov.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(is) < 3 || is[0].Path != "infra/main.tf" || is[0].Line != 11 || !strings.HasPrefix(is[0].Code, "CKV") {
		t.Fatalf("issues = %+v", is)
	}
	list, err := ParseCheckov("/r", []byte(`[{"results":{"failed_checks":[{"check_id":"CKV_1","check_name":"n","file_path":"/r/a.tf","file_line_range":[],"guideline":"https://g"}]}},{"results":{}}]`))
	if err != nil || len(list) != 1 || list[0].Line != 0 || list[0].Path != "a.tf" || !strings.Contains(list[0].Message, "https://g") {
		t.Fatalf("list form = %+v, %v", list, err)
	}
	if is, err := ParseCheckov("", []byte(`{"passed":0,"failed":0}`)); err != nil || len(is) != 0 {
		t.Fatalf("summary form = %v, %v", is, err)
	}
}

func TestParseErrorsAndEmpty(t *testing.T) {
	t.Parallel()
	parsers := map[string]func([]byte) ([]Issue, error){
		"golangci": func(b []byte) ([]Issue, error) { return ParseGolangci("", b) },
		"ruff":     func(b []byte) ([]Issue, error) { return ParseRuff("", b) },
		"hadolint": func(b []byte) ([]Issue, error) { return ParseHadolint("", b) },
		"tflint":   func(b []byte) ([]Issue, error) { return ParseTFLint("", b) },
		"checkov":  func(b []byte) ([]Issue, error) { return ParseCheckov("", b) },
	}
	for name, parse := range parsers {
		if is, err := parse([]byte("  \n")); err != nil || len(is) != 0 {
			t.Errorf("%s: empty output = %v, %v", name, is, err)
		}
		if _, err := parse([]byte("{nope")); err == nil {
			t.Errorf("%s: want parse error", name)
		}
	}
	if _, err := ParseCheckov("", []byte("[nope")); err == nil {
		t.Error("checkov list: want parse error")
	}
	if _, err := ParseTFLint("", []byte(`{"issues":[],"errors":[{"message":"boom"}]}`)); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("tflint errors not surfaced: %v", err)
	}
	is, err := ParseRuff("/repo", []byte(`[{"code":null,"message":"SyntaxError","filename":"/repo/a.py","location":{"row":1,"column":1},"fix":{"message":"Remove"}}]`))
	if err != nil || is[0].Code != "syntax-error" || is[0].Path != "a.py" || is[0].Fix == nil {
		t.Fatalf("ruff syntax error = %+v, %v", is, err)
	}
	if got := ParseYamllint("", []byte("not a match\na.yaml:1:1: [warning] bad thing\n")); len(got) != 1 || got[0].Code != "syntax" {
		t.Fatalf("yamllint without rule = %+v", got)
	}
}

func TestRelPathAndDirs(t *testing.T) {
	t.Parallel()
	tests := map[[2]string]string{
		{"/repo", "/repo/a/b.go"}: "a/b.go",
		{"/repo", "./a/b.go"}:     "a/b.go",
		{"/repo", "/other/x.go"}:  "other/x.go",
		{"", ""}:                  "",
	}
	for in, want := range tests {
		if got := relPath(in[0], in[1]); got != want {
			t.Errorf("relPath(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
	if got := strings.Join(dirsOf([]string{"main.go", "a/x.go", "a/y.go", "b/z.go"}), " "); got != ". ./a ./b" {
		t.Fatalf("dirsOf = %s", got)
	}
}

func TestExternalRun(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	req := Request{Root: "/repo", Files: []string{"a/x.go"}, Config: "/cfg"}
	tests := []struct {
		checker Checker
		res     Result
		want    string
		args    string
	}{
		{GolangciLint{}, Result{Stdout: fixture(t, "golangci.json")}, "errorlint,gosec,revive,revive", "--config /cfg ./a"},
		{Ruff{}, Result{Stdout: fixture(t, "ruff.json")}, "E401,I001,F401,S602", "--config /cfg a/x.go"},
		{Hadolint{}, Result{Stdout: fixture(t, "hadolint.json")}, "DL3008,DL3009,DL3015", "--config /cfg a/x.go"},
		{TFLint{}, Result{Stdout: fixture(t, "tflint.json"), ExitCode: 2}, "terraform_required_version,terraform_required_providers,terraform_typed_variables", "--config /cfg"},
		{Checkov{}, Result{Stdout: []byte(`{"results":{"failed_checks":[{"check_id":"CKV_1","file_path":"/a/x.go","file_line_range":[2,3]}]}}`)}, "CKV_1", "-d ."},
		{Yamllint{}, Result{Stdout: fixture(t, "yamllint.txt"), ExitCode: 1}, "key-duplicates,colons,commas", "-c /cfg a/x.go"},
	}
	for _, tt := range tests {
		t.Run(tt.checker.Name(), func(t *testing.T) {
			t.Parallel()
			run := &fakeRunner{results: map[string]Result{tt.checker.Binary(): tt.res}}
			is, err := tt.checker.Run(ctx, run, req)
			if err != nil {
				t.Fatal(err)
			}
			if got := codes(is); got != tt.want {
				t.Fatalf("codes = %s, want %s", got, tt.want)
			}
			if !strings.Contains(run.calls[0], tt.args) {
				t.Fatalf("call %q lacks %q", run.calls[0], tt.args)
			}
			if is, err := tt.checker.Run(ctx, run, Request{}); err != nil || is != nil {
				t.Fatal("no files should mean no run")
			}
			if tt.checker.ConfigName() == "" && tt.checker.Name() != "checkov" {
				t.Fatal("external checkers other than checkov ship a config")
			}
		})
	}
}

func TestExternalRunFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	req := Request{Root: "/repo", Files: []string{"x"}, WholeRepo: true}
	all := []Checker{GolangciLint{}, Ruff{}, Hadolint{}, TFLint{}, Checkov{}, Yamllint{}}
	for _, c := range all {
		run := &fakeRunner{errs: map[string]error{c.Binary(): errors.New("exec failed")}}
		if _, err := c.Run(ctx, run, req); err == nil {
			t.Errorf("%s: start failure not returned", c.Name())
		}
		run = &fakeRunner{results: map[string]Result{c.Binary(): {Stdout: []byte("garbage"), Stderr: []byte(strings.Repeat("e", 600)), ExitCode: 3}}}
		if _, err := c.Run(ctx, run, req); err == nil || !strings.Contains(err.Error(), "exited 3") {
			t.Errorf("%s: bad exit not reported: %v", c.Name(), err)
		}
	}
	run := &fakeRunner{results: map[string]Result{"golangci-lint": {Stdout: fixture(t, "golangci-typecheck.json")}}}
	if _, err := (GolangciLint{}).Run(ctx, run, req); err == nil || !strings.Contains(err.Error(), "could not type-check") {
		t.Fatalf("typecheck-only output should be an error, got %v", err)
	}
	if !strings.Contains(run.calls[0], "./...") {
		t.Fatalf("whole repo should lint ./..., got %q", run.calls[0])
	}
	mixed := []Issue{{Code: "typecheck"}, {Code: "errcheck"}}
	if out, err := withoutTypecheck(mixed); err != nil || len(out) != 1 {
		t.Fatalf("mixed = %v, %v", out, err)
	}
	run = &fakeRunner{results: map[string]Result{"ruff": {Stdout: []byte("[]"), ExitCode: 2, Stderr: []byte("config error")}}}
	if _, err := (Ruff{}).Run(ctx, run, req); err == nil {
		t.Fatal("ruff non-zero exit should fail")
	}
}

func TestAllAndExecRunner(t *testing.T) {
	t.Parallel()
	if len(All()) != 7 {
		t.Fatalf("All() = %d checkers", len(All()))
	}
	r := ExecRunner{}
	res, err := r.Run(context.Background(), ".", "sh", "-c", "echo out; echo err >&2; exit 3")
	if err != nil || res.ExitCode != 3 || strings.TrimSpace(string(res.Stdout)) != "out" || strings.TrimSpace(string(res.Stderr)) != "err" {
		t.Fatalf("Run = %+v, %v", res, err)
	}
	if _, err := r.Run(context.Background(), ".", "canon-no-such-binary"); err == nil {
		t.Fatal("missing binary should be an error")
	}
	if _, err := r.LookPath("sh"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LookPath("canon-no-such-binary"); err == nil {
		t.Fatal("LookPath found a ghost")
	}
	if err := toolError("x", Result{ExitCode: 1, Stdout: []byte("only stdout")}, errors.New("cause")); !strings.Contains(err.Error(), "only stdout") {
		t.Fatalf("toolError = %v", err)
	}
}

// FuzzParseToolOutput checks that no parser panics on arbitrary tool output,
// and that every issue a parser returns has a clean relative path.
func FuzzParseToolOutput(f *testing.F) {
	for _, name := range []string{"golangci.json", "ruff.json", "hadolint.json", "tflint.json", "checkov.json", "yamllint.txt"} {
		f.Add(fixture(f, name))
	}
	f.Add([]byte(`[{"results":{"failed_checks":[{"file_path":"/../x"}]}}]`))
	f.Fuzz(func(t *testing.T, out []byte) {
		var all []Issue
		for _, parse := range []func(string, []byte) ([]Issue, error){ParseGolangci, ParseRuff, ParseHadolint, ParseTFLint, ParseCheckov} {
			is, _ := parse("/repo", out)
			all = append(all, is...)
		}
		all = append(all, ParseYamllint("/repo", out)...)
		for _, i := range all {
			if strings.HasPrefix(i.Path, "/") || strings.HasPrefix(i.Path, "./") {
				t.Fatalf("unclean path %q", i.Path)
			}
		}
	})
}

func BenchmarkParseGolangci(b *testing.B) {
	data := fixture(b, "golangci.json")
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseGolangci("/repo", data); err != nil {
			b.Fatal(err)
		}
	}
}
