// Package gitx asks git which files changed and what the diff is, through
// the same Runner the checkers use so tests never need a real repository.
package gitx

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/rossedman/canon/internal/checkers"
)

// git runs one git command in root and returns stdout, or an error that
// includes stderr.
func git(ctx context.Context, run checkers.Runner, root string, args ...string) (string, error) {
	res, err := run.Run(ctx, root, "git", args...)
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("git %s exited %d: %s", args[0], res.ExitCode, strings.TrimSpace(string(res.Stderr)))
	}
	return string(res.Stdout), nil
}

// MergeBase returns the commit where HEAD branched from base, or HEAD itself
// when base is empty or cannot be resolved (a fresh clone with no remote).
func MergeBase(ctx context.Context, run checkers.Runner, root, base string) (string, error) {
	if base != "" {
		if out, err := git(ctx, run, root, "merge-base", base, "HEAD"); err == nil {
			return strings.TrimSpace(out), nil
		}
	}
	out, err := git(ctx, run, root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ChangedFiles lists files added, copied, modified or renamed since the merge
// base with base, including uncommitted and untracked files, sorted.
func ChangedFiles(ctx context.Context, run checkers.Runner, root, base string) ([]string, error) {
	mb, err := MergeBase(ctx, run, root, base)
	if err != nil {
		return nil, err
	}
	diff, err := git(ctx, run, root, "diff", "--name-only", "--diff-filter=ACMR", mb)
	if err != nil {
		return nil, err
	}
	untracked, err := git(ctx, run, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return uniqueLines(diff + "\n" + untracked), nil
}

// Diff returns a unified diff of the working tree against the merge base with
// base. Untracked files are appended as new-file diffs read from repo, so a
// Dockerfile nobody has staged yet still shows up in review.
func Diff(ctx context.Context, run checkers.Runner, root, base string, repo fs.FS) (string, error) {
	mb, err := MergeBase(ctx, run, root, base)
	if err != nil {
		return "", err
	}
	diff, err := git(ctx, run, root, "diff", "--no-color", "--no-ext-diff", mb)
	if err != nil {
		return "", err
	}
	untracked, err := git(ctx, run, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(diff)
	for _, f := range uniqueLines(untracked) {
		data, err := fs.ReadFile(repo, f)
		if err != nil || bytes.IndexByte(data, 0) >= 0 {
			continue
		}
		b.WriteString(NewFileDiff(f, string(data)))
	}
	return b.String(), nil
}

// NewFileDiff renders content as the diff git would show for a new file.
func NewFileDiff(name, content string) string {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if content == "" {
		lines = nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n", name, name, name)
	if len(lines) > 0 {
		fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
		for _, l := range lines {
			b.WriteString("+" + l + "\n")
		}
	}
	return b.String()
}

// uniqueLines splits s into sorted, unique, non-empty lines.
func uniqueLines(s string) []string {
	var out []string
	for l := range strings.SplitSeq(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
