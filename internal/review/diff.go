// Package review parses unified diffs and classifies a change for concerns
// that linters cannot see: new dependencies, new Dockerfiles, provider
// bumps, exported API changes and new packages without tests.
package review

import (
	"strconv"
	"strings"
)

// Status says what happened to a file in a diff.
type Status string

// File statuses.
const (
	Added    Status = "added"
	Modified Status = "modified"
	Deleted  Status = "deleted"
	Renamed  Status = "renamed"
)

// Line is one added or removed line, numbered in the new file (added) or the
// old file (removed).
type Line struct {
	N    int
	Text string
}

// FileDiff is the change to one file.
type FileDiff struct {
	Path    string
	OldPath string
	Status  Status
	Added   []Line
	Removed []Line
}

// ParseDiff reads a unified diff, as printed by git diff or diff -u. Content
// it does not understand is skipped rather than rejected, since diffs arrive
// from agents and tools of every kind.
func ParseDiff(text string) []FileDiff {
	p := &diffParser{}
	for raw := range strings.SplitSeq(text, "\n") {
		p.line(strings.TrimSuffix(raw, "\r"))
	}
	out := p.files[:0]
	for _, f := range p.files {
		if f.Path == "" {
			f.Path = f.OldPath
		}
		if f.Path != "" {
			out = append(out, f)
		}
	}
	return out
}

// diffParser holds the state of ParseDiff between lines.
type diffParser struct {
	files            []FileDiff
	cur              *FileDiff
	sawPlus          bool
	oldN, newN       int
	oldLeft, newLeft int
}

// start begins a new file.
func (p *diffParser) start() {
	p.files = append(p.files, FileDiff{Status: Modified})
	p.cur = &p.files[len(p.files)-1]
	p.sawPlus = false
	p.oldN, p.newN = 0, 0
	p.oldLeft, p.newLeft = 0, 0
}

// line consumes one line of the diff.
func (p *diffParser) line(line string) {
	if p.oldLeft > 0 || p.newLeft > 0 {
		p.hunkLine(line)
		return
	}
	switch {
	case strings.HasPrefix(line, "diff --git "):
		p.start()
		if a, b, ok := splitGitHeader(strings.TrimPrefix(line, "diff --git ")); ok {
			p.cur.OldPath, p.cur.Path = a, b
		}
	case strings.HasPrefix(line, "--- "):
		if p.cur == nil || p.sawPlus {
			p.start()
		}
		if path := diffPath(strings.TrimPrefix(line, "--- ")); path == "" {
			p.cur.Status = Added
		} else {
			p.cur.OldPath = path
		}
	case p.cur == nil:
	case strings.HasPrefix(line, "+++ "):
		p.sawPlus = true
		if path := diffPath(strings.TrimPrefix(line, "+++ ")); path == "" {
			p.cur.Status = Deleted
		} else {
			p.cur.Path = path
		}
	case strings.HasPrefix(line, "new file mode"):
		p.cur.Status = Added
	case strings.HasPrefix(line, "deleted file mode"):
		p.cur.Status = Deleted
	case strings.HasPrefix(line, "rename from "):
		p.cur.OldPath = strings.TrimPrefix(line, "rename from ")
		p.cur.Status = Renamed
	case strings.HasPrefix(line, "rename to "):
		p.cur.Path = strings.TrimPrefix(line, "rename to ")
	case strings.HasPrefix(line, "@@"):
		if h, ok := parseHunk(line); ok {
			// Hunks run in order; a header that jumps backwards is malformed,
			// so numbering continues from where the last hunk ended.
			p.oldN, p.newN = max(h.oldStart, p.oldN), max(h.newStart, p.newN)
			p.oldLeft, p.newLeft = h.oldCount, h.newCount
		}
	}
}

// hunkLine consumes one line inside a hunk, counting down what the header promised.
func (p *diffParser) hunkLine(line string) {
	if line == "" {
		// Some tools strip the space from empty context lines.
		line = " "
	}
	switch line[0] {
	case '+':
		p.cur.Added = append(p.cur.Added, Line{N: p.newN, Text: line[1:]})
		p.newN++
		p.newLeft--
	case '-':
		p.cur.Removed = append(p.cur.Removed, Line{N: p.oldN, Text: line[1:]})
		p.oldN++
		p.oldLeft--
	case ' ':
		p.oldN++
		p.newN++
		p.oldLeft--
		p.newLeft--
	case '\\':
		// "\ No newline at end of file"
	default:
		p.oldLeft, p.newLeft = 0, 0
		p.line(line)
	}
}

// splitGitHeader splits "a/x b/y" into x and y. Paths with spaces are
// ambiguous in this header; the ---/+++ lines that follow correct them.
func splitGitHeader(s string) (string, string, bool) {
	i := strings.Index(s, " b/")
	if !strings.HasPrefix(s, "a/") || i < 0 {
		return "", "", false
	}
	return s[2:i], s[i+3:], true
}

// diffPath strips the a/ or b/ prefix and any trailing timestamp from a
// ---/+++ path. /dev/null becomes "".
func diffPath(s string) string {
	s, _, _ = strings.Cut(s, "\t")
	s = strings.TrimSpace(s)
	if s == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(s, "a/") || strings.HasPrefix(s, "b/") {
		return s[2:]
	}
	return s
}

// hunk is a parsed "@@ -a,b +c,d @@" header.
type hunk struct {
	oldStart, oldCount, newStart, newCount int
}

// parseHunk reads a hunk header. A missing count means 1, as in diff -u.
func parseHunk(line string) (hunk, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
		return hunk{}, false
	}
	os, oc, ok1 := hunkRange(fields[1][1:])
	ns, nc, ok2 := hunkRange(fields[2][1:])
	if !ok1 || !ok2 {
		return hunk{}, false
	}
	return hunk{oldStart: max(os, 1), oldCount: oc, newStart: max(ns, 1), newCount: nc}, true
}

// hunkRange parses "12,3" into 12 and 3, or "12" into 12 and 1.
func hunkRange(s string) (int, int, bool) {
	startS, countS, hasCount := strings.Cut(s, ",")
	start, err := strconv.Atoi(startS)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	count := 1
	if hasCount {
		if count, err = strconv.Atoi(countS); err != nil || count < 0 {
			return 0, 0, false
		}
	}
	return start, count, true
}
