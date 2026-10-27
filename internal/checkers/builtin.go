package checkers

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/rossedman/canon/internal/glob"
)

// fileCheck is one built-in check that looks at files one at a time.
type fileCheck struct {
	code  string
	files []string
	run   func(fsys fs.FS, name string, data []byte) []Issue
}

// repoCheck is one built-in check that looks at the repository as a whole.
type repoCheck struct {
	code string
	run  func(fsys fs.FS) []Issue
}

// fileChecks are the built-in per-file checks, in the order they run.
var fileChecks = []fileCheck{
	{code: "one-command-per-file", files: []string{"cmd/*.go"}, run: checkOneCommand},
	{code: "go-doc-comments", files: []string{"**/*.go"}, run: checkDocComments},
	{code: "go-ux-libraries", files: []string{"**/*.go"}, run: checkUXLibraries},
	{code: "uv-lock", files: []string{"pyproject.toml", "**/pyproject.toml"}, run: checkUVLock},
	{code: "final-stage-base", files: dockerfiles, run: checkFinalStageBase},
	{code: "base-digest", files: dockerfiles, run: checkBaseDigest},
	{code: "final-stage-user", files: dockerfiles, run: checkFinalStageUser},
	{code: "actions-pinned", files: []string{".github/workflows/*.yml", ".github/workflows/*.yaml", "action.yml", "action.yaml", "**/action.yml", "**/action.yaml"}, run: checkActionsPinned},
	{code: "workflow-permissions", files: []string{".github/workflows/*.yml", ".github/workflows/*.yaml"}, run: checkWorkflowPermissions},
	{code: "release-signing", files: []string{".goreleaser.yaml", ".goreleaser.yml"}, run: checkReleaseSigning},
}

// repoChecks are the built-in checks that need the whole repo.
var repoChecks = []repoCheck{
	{code: "oss-files", run: checkOSSFiles},
}

// dockerfiles matches the file names canon treats as Dockerfiles.
var dockerfiles = []string{"Dockerfile", "**/Dockerfile", "*.Dockerfile", "**/*.Dockerfile", "Containerfile", "**/Containerfile"}

// Builtin runs canon's own checks, written in Go, for rules no linter covers.
type Builtin struct{}

// Name returns the checker name.
func (Builtin) Name() string { return "canon" }

// Binary returns "" because the checks are built in.
func (Builtin) Binary() string { return "" }

// ConfigName returns "" because built-in checks take no config.
func (Builtin) ConfigName() string { return "" }

// Codes lists every code the built-in checker can report.
func (Builtin) Codes() []string {
	var out []string
	for _, c := range fileChecks {
		out = append(out, c.code)
	}
	for _, c := range repoChecks {
		out = append(out, c.code)
	}
	return out
}

// Run applies the checks named in req.Codes to req.Files.
func (Builtin) Run(ctx context.Context, _ Runner, req Request) ([]Issue, error) {
	var issues []Issue
	for _, c := range fileChecks {
		if !slices.Contains(req.Codes, c.code) {
			continue
		}
		for _, f := range req.Files {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("canon checks: %w", err)
			}
			if !glob.MatchAny(c.files, f) {
				continue
			}
			data, err := fs.ReadFile(req.FS, f)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", f, err)
			}
			for _, i := range c.run(req.FS, f, data) {
				i.Code = c.code
				if i.Path == "" {
					i.Path = f
				}
				issues = append(issues, i)
			}
		}
	}
	if !req.WholeRepo {
		return issues, nil
	}
	for _, c := range repoChecks {
		if !slices.Contains(req.Codes, c.code) {
			continue
		}
		for _, i := range c.run(req.FS) {
			i.Code = c.code
			issues = append(issues, i)
		}
	}
	return issues, nil
}

// requiredFiles lists the community files and the places each may live.
var requiredFiles = []struct {
	name   string
	places []string
}{
	{"LICENSE", []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING"}},
	{"README.md", []string{"README.md", "README"}},
	{"SECURITY.md", []string{"SECURITY.md", ".github/SECURITY.md", "docs/SECURITY.md"}},
	{"CONTRIBUTING.md", []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md", "docs/CONTRIBUTING.md"}},
	{"CODE_OF_CONDUCT.md", []string{"CODE_OF_CONDUCT.md", ".github/CODE_OF_CONDUCT.md", "docs/CODE_OF_CONDUCT.md"}},
}

// checkOSSFiles reports each community file missing from the repo.
func checkOSSFiles(fsys fs.FS) []Issue {
	var out []Issue
	for _, rf := range requiredFiles {
		found := slices.ContainsFunc(rf.places, func(p string) bool {
			st, err := fs.Stat(fsys, p)
			return err == nil && !st.IsDir()
		})
		if !found {
			out = append(out, Issue{Path: rf.name, Message: rf.name + " is missing"})
		}
	}
	return out
}

// checkUVLock reports a pyproject.toml with no uv.lock beside it.
func checkUVLock(fsys fs.FS, name string, _ []byte) []Issue {
	lock := path.Join(path.Dir(name), "uv.lock")
	if _, err := fs.Stat(fsys, lock); err == nil {
		return nil
	}
	return []Issue{{Line: 1, Message: "no uv.lock next to " + name + "; run uv lock and commit it"}}
}
