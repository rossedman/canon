package checkers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rossedman/canon/internal/findings"
)

// GolangciLint runs golangci-lint v2 with canon's config.
type GolangciLint struct{}

// Name returns the checker name.
func (GolangciLint) Name() string { return "golangci-lint" }

// Binary returns the executable name.
func (GolangciLint) Binary() string { return "golangci-lint" }

// ConfigName returns the config file under canon/rules/.
func (GolangciLint) ConfigName() string { return "golangci.yml" }

// Run lints the packages that contain the requested files.
func (g GolangciLint) Run(ctx context.Context, run Runner, req Request) ([]Issue, error) {
	if len(req.Files) == 0 {
		return nil, nil
	}
	args := []string{"run", "--output.json.path=stdout", "--output.text.path=stderr", "--show-stats=false", "--issues-exit-code=0", "--path-mode=abs", "--max-issues-per-linter=0", "--max-same-issues=0"}
	if req.Config != "" {
		args = append(args, "--config", req.Config)
	}
	pkgs := []string{"./..."}
	if !req.WholeRepo {
		pkgs = dirsOf(req.Files)
	}
	res, err := run.Run(ctx, req.Root, g.Binary(), append(args, pkgs...)...)
	if err != nil {
		return nil, fmt.Errorf("golangci-lint: %w", err)
	}
	issues, perr := ParseGolangci(req.Root, res.Stdout)
	if perr != nil || res.ExitCode != 0 && len(issues) == 0 {
		return nil, toolError(g.Name(), res, perr)
	}
	return withoutTypecheck(issues)
}

// withoutTypecheck drops golangci-lint's typecheck issues, which mean the
// code did not compile rather than that it broke a rule. When they are all
// there is, nothing was linted, so that is reported as an error instead of
// passing silently.
func withoutTypecheck(issues []Issue) ([]Issue, error) {
	var out []Issue
	var first *Issue
	for i := range issues {
		if issues[i].Code == "typecheck" {
			if first == nil {
				first = &issues[i]
			}
			continue
		}
		out = append(out, issues[i])
	}
	if len(out) == 0 && first != nil {
		return nil, fmt.Errorf("golangci-lint could not type-check %s:%d: %s", first.Path, first.Line, firstLine(first.Message))
	}
	return out, nil
}

// firstLine returns s up to its first newline.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// ParseGolangci reads golangci-lint's JSON output. The issue code is the
// linter name, so rules map whole linters.
func ParseGolangci(root string, out []byte) ([]Issue, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	var doc struct {
		Issues []struct {
			FromLinter string
			Text       string
			Pos        struct {
				Filename string
				Line     int
				Column   int
			}
		}
	}
	// golangci-lint can print more than one JSON document; the first is the report.
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse golangci-lint output: %w", err)
	}
	issues := make([]Issue, 0, len(doc.Issues))
	for _, i := range doc.Issues {
		issues = append(issues, Issue{
			Code:    i.FromLinter,
			Path:    relPath(root, i.Pos.Filename),
			Line:    i.Pos.Line,
			Column:  i.Pos.Column,
			Message: i.Text,
		})
	}
	return issues, nil
}

// Ruff runs ruff check with canon's config.
type Ruff struct{}

// Name returns the checker name.
func (Ruff) Name() string { return "ruff" }

// Binary returns the executable name.
func (Ruff) Binary() string { return "ruff" }

// ConfigName returns the config file under canon/rules/.
func (Ruff) ConfigName() string { return "ruff.toml" }

// Run checks the requested Python files.
func (r Ruff) Run(ctx context.Context, run Runner, req Request) ([]Issue, error) {
	if len(req.Files) == 0 {
		return nil, nil
	}
	args := []string{"check", "--output-format", "json", "--exit-zero", "--no-cache"}
	if req.Config != "" {
		args = append(args, "--config", req.Config)
	}
	res, err := run.Run(ctx, req.Root, r.Binary(), append(args, req.Files...)...)
	if err != nil {
		return nil, fmt.Errorf("ruff: %w", err)
	}
	issues, perr := ParseRuff(req.Root, res.Stdout)
	if perr != nil || res.ExitCode != 0 {
		return nil, toolError(r.Name(), res, perr)
	}
	return issues, nil
}

// ParseRuff reads ruff's JSON output.
func ParseRuff(root string, out []byte) ([]Issue, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	var doc []struct {
		Code     *string `json:"code"`
		Message  string  `json:"message"`
		Filename string  `json:"filename"`
		Location struct {
			Row    int `json:"row"`
			Column int `json:"column"`
		} `json:"location"`
		Fix *struct {
			Message string `json:"message"`
		} `json:"fix"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("parse ruff output: %w", err)
	}
	issues := make([]Issue, 0, len(doc))
	for _, d := range doc {
		code := "syntax-error"
		if d.Code != nil && *d.Code != "" {
			code = *d.Code
		}
		i := Issue{
			Code:    code,
			Path:    relPath(root, d.Filename),
			Line:    d.Location.Row,
			Column:  d.Location.Column,
			Message: d.Message,
		}
		if d.Fix != nil && d.Fix.Message != "" {
			i.Fix = &findings.Fix{Kind: "ruff-fix", Suggestion: d.Fix.Message + " (ruff check --fix)"}
		}
		issues = append(issues, i)
	}
	return issues, nil
}

// Hadolint runs hadolint with canon's config.
type Hadolint struct{}

// Name returns the checker name.
func (Hadolint) Name() string { return "hadolint" }

// Binary returns the executable name.
func (Hadolint) Binary() string { return "hadolint" }

// ConfigName returns the config file under canon/rules/.
func (Hadolint) ConfigName() string { return "hadolint.yaml" }

// Run lints the requested Dockerfiles.
func (h Hadolint) Run(ctx context.Context, run Runner, req Request) ([]Issue, error) {
	if len(req.Files) == 0 {
		return nil, nil
	}
	args := []string{"--format", "json", "--no-fail"}
	if req.Config != "" {
		args = append(args, "--config", req.Config)
	}
	res, err := run.Run(ctx, req.Root, h.Binary(), append(args, req.Files...)...)
	if err != nil {
		return nil, fmt.Errorf("hadolint: %w", err)
	}
	issues, perr := ParseHadolint(req.Root, res.Stdout)
	if perr != nil || res.ExitCode != 0 && len(issues) == 0 {
		return nil, toolError(h.Name(), res, perr)
	}
	return issues, nil
}

// ParseHadolint reads hadolint's JSON output.
func ParseHadolint(root string, out []byte) ([]Issue, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	var doc []struct {
		Code    string `json:"code"`
		File    string `json:"file"`
		Line    int    `json:"line"`
		Column  int    `json:"column"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("parse hadolint output: %w", err)
	}
	issues := make([]Issue, 0, len(doc))
	for _, d := range doc {
		issues = append(issues, Issue{Code: d.Code, Path: relPath(root, d.File), Line: d.Line, Column: d.Column, Message: d.Message})
	}
	return issues, nil
}

// TFLint runs tflint recursively with canon's config.
type TFLint struct{}

// Name returns the checker name.
func (TFLint) Name() string { return "tflint" }

// Binary returns the executable name.
func (TFLint) Binary() string { return "tflint" }

// ConfigName returns the config file under canon/rules/.
func (TFLint) ConfigName() string { return "tflint.hcl" }

// Run lints every Terraform module under the root; the engine keeps only
// issues in the requested files.
func (t TFLint) Run(ctx context.Context, run Runner, req Request) ([]Issue, error) {
	if len(req.Files) == 0 {
		return nil, nil
	}
	args := []string{"--recursive", "--format", "json", "--force"}
	if req.Config != "" {
		args = append(args, "--config", req.Config)
	}
	res, err := run.Run(ctx, req.Root, t.Binary(), args...)
	if err != nil {
		return nil, fmt.Errorf("tflint: %w", err)
	}
	issues, perr := ParseTFLint(req.Root, res.Stdout)
	if perr != nil {
		return nil, toolError(t.Name(), res, perr)
	}
	return issues, nil
}

// ParseTFLint reads tflint's JSON output. Errors tflint reports alongside
// issues (a module that failed to load) are returned as an error only when
// there are no issues at all.
func ParseTFLint(root string, out []byte) ([]Issue, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	var doc struct {
		Issues []struct {
			Rule struct {
				Name string `json:"name"`
			} `json:"rule"`
			Message string `json:"message"`
			Range   struct {
				Filename string `json:"filename"`
				Start    struct {
					Line   int `json:"line"`
					Column int `json:"column"`
				} `json:"start"`
			} `json:"range"`
		} `json:"issues"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("parse tflint output: %w", err)
	}
	if len(doc.Issues) == 0 && len(doc.Errors) > 0 {
		return nil, fmt.Errorf("tflint: %s", doc.Errors[0].Message)
	}
	issues := make([]Issue, 0, len(doc.Issues))
	for _, d := range doc.Issues {
		issues = append(issues, Issue{
			Code:    d.Rule.Name,
			Path:    relPath(root, d.Range.Filename),
			Line:    d.Range.Start.Line,
			Column:  d.Range.Start.Column,
			Message: d.Message,
		})
	}
	return issues, nil
}

// Checkov runs checkov's Terraform checks.
type Checkov struct{}

// Name returns the checker name.
func (Checkov) Name() string { return "checkov" }

// Binary returns the executable name.
func (Checkov) Binary() string { return "checkov" }

// ConfigName returns "" because checkov runs with its defaults.
func (Checkov) ConfigName() string { return "" }

// Run scans the root directory; the engine keeps only issues in the requested files.
func (c Checkov) Run(ctx context.Context, run Runner, req Request) ([]Issue, error) {
	if len(req.Files) == 0 {
		return nil, nil
	}
	args := []string{"-d", ".", "--framework", "terraform", "--output", "json", "--quiet", "--compact", "--soft-fail", "--skip-download"}
	res, err := run.Run(ctx, req.Root, c.Binary(), args...)
	if err != nil {
		return nil, fmt.Errorf("checkov: %w", err)
	}
	issues, perr := ParseCheckov(req.Root, res.Stdout)
	if perr != nil {
		return nil, toolError(c.Name(), res, perr)
	}
	return issues, nil
}

// checkovReport is one framework's report.
type checkovReport struct {
	Results struct {
		FailedChecks []struct {
			CheckID       string `json:"check_id"`
			CheckName     string `json:"check_name"`
			FilePath      string `json:"file_path"`
			FileLineRange []int  `json:"file_line_range"`
			Guideline     string `json:"guideline"`
		} `json:"failed_checks"`
	} `json:"results"`
}

// ParseCheckov reads checkov's JSON output, which is one report object, a
// list of them (one per framework), or a bare summary when nothing was scanned.
func ParseCheckov(root string, out []byte) ([]Issue, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	var reports []checkovReport
	if out[0] == '[' {
		if err := json.Unmarshal(out, &reports); err != nil {
			return nil, fmt.Errorf("parse checkov output: %w", err)
		}
	} else {
		var one checkovReport
		if err := json.Unmarshal(out, &one); err != nil {
			return nil, fmt.Errorf("parse checkov output: %w", err)
		}
		reports = []checkovReport{one}
	}
	var issues []Issue
	for _, r := range reports {
		for _, f := range r.Results.FailedChecks {
			line := 0
			if len(f.FileLineRange) > 0 {
				line = f.FileLineRange[0]
			}
			msg := f.CheckName
			if f.Guideline != "" {
				msg += " (" + f.Guideline + ")"
			}
			issues = append(issues, Issue{Code: f.CheckID, Path: relPath(root, f.FilePath), Line: line, Message: msg})
		}
	}
	return issues, nil
}

// Yamllint runs yamllint with canon's config.
type Yamllint struct{}

// Name returns the checker name.
func (Yamllint) Name() string { return "yamllint" }

// Binary returns the executable name.
func (Yamllint) Binary() string { return "yamllint" }

// ConfigName returns the config file under canon/rules/.
func (Yamllint) ConfigName() string { return "yamllint.yaml" }

// Run lints the requested YAML files.
func (y Yamllint) Run(ctx context.Context, run Runner, req Request) ([]Issue, error) {
	if len(req.Files) == 0 {
		return nil, nil
	}
	args := []string{"-f", "parsable"}
	if req.Config != "" {
		args = append(args, "-c", req.Config)
	}
	res, err := run.Run(ctx, req.Root, y.Binary(), append(args, req.Files...)...)
	if err != nil {
		return nil, fmt.Errorf("yamllint: %w", err)
	}
	issues := ParseYamllint(req.Root, res.Stdout)
	if res.ExitCode > 1 || res.ExitCode == 1 && len(issues) == 0 {
		return nil, toolError(y.Name(), res, nil)
	}
	return issues, nil
}

// yamllintLine matches "file:line:col: [level] message (rule)".
var yamllintLine = regexp.MustCompile(`^(.+?):(\d+):(\d+): \[(error|warning)\] (.*?)(?: \(([\w-]+)\))?$`)

// ParseYamllint reads yamllint's parsable output. Lines that do not match are
// ignored. A syntax error has no rule name and gets the code "syntax".
func ParseYamllint(root string, out []byte) []Issue {
	var issues []Issue
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		m := yamllintLine.FindStringSubmatch(strings.TrimRight(sc.Text(), "\r"))
		if m == nil {
			continue
		}
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		code := m[6]
		if code == "" {
			code = "syntax"
		}
		issues = append(issues, Issue{Code: code, Path: relPath(root, m[1]), Line: line, Column: col, Message: m[5]})
	}
	return issues
}
