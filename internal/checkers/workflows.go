package checkers

import (
	"bufio"
	"bytes"
	"io/fs"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rossedman/canon/internal/findings"
)

// usesLine matches a workflow or composite action "uses:" line.
var usesLine = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*["']?([^"'\s#]+)`)

// commitSHA is a full 40-character commit hash.
var commitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// UsesRef is one "uses:" reference found in a workflow.
type UsesRef struct {
	Ref  string
	Line int
}

// FindUses returns every uses: reference with its line number.
func FindUses(data []byte) []UsesRef {
	var out []UsesRef
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
		if m := usesLine.FindStringSubmatch(sc.Text()); m != nil {
			out = append(out, UsesRef{Ref: m[1], Line: n})
		}
	}
	return out
}

// Pinned reports whether a uses: reference is pinned immutably: a local
// action, a docker image by digest, or a remote action at a commit SHA.
func Pinned(ref string) bool {
	switch {
	case strings.HasPrefix(ref, "./"):
		return true
	case strings.HasPrefix(ref, "docker://"):
		return strings.Contains(ref, "@sha256:")
	}
	_, version, ok := strings.Cut(ref, "@")
	return ok && commitSHA.MatchString(version)
}

// checkActionsPinned reports remote actions that are not pinned to a commit SHA.
func checkActionsPinned(_ fs.FS, _ string, data []byte) []Issue {
	var out []Issue
	for _, u := range FindUses(data) {
		if Pinned(u.Ref) {
			continue
		}
		action, _, _ := strings.Cut(u.Ref, "@")
		out = append(out, Issue{
			Line:    u.Line,
			Message: u.Ref + " is not pinned to a commit SHA",
			Fix:     &findings.Fix{Kind: "pin", Suggestion: "uses: " + action + "@<40-char commit SHA> # <version>"},
		})
	}
	return out
}

// checkWorkflowPermissions reports a workflow with no top-level permissions block.
func checkWorkflowPermissions(_ fs.FS, _ string, data []byte) []Issue {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "permissions" {
			return nil
		}
	}
	return []Issue{{
		Line:    1,
		Message: "workflow has no top-level permissions block",
		Fix:     &findings.Fix{Kind: "insert", Suggestion: "permissions:\n  contents: read"},
	}}
}

// checkReleaseSigning reports a GoReleaser config that does not sign its
// checksums (or every artifact) with cosign.
func checkReleaseSigning(_ fs.FS, _ string, data []byte) []Issue {
	var cfg struct {
		Signs []struct {
			Cmd       string `yaml:"cmd"`
			Artifacts string `yaml:"artifacts"`
		} `yaml:"signs"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	for _, s := range cfg.Signs {
		if s.Cmd == "cosign" && (s.Artifacts == "checksum" || s.Artifacts == "all") {
			return nil
		}
	}
	return []Issue{{
		Line:    1,
		Message: "GoReleaser does not sign the checksums file with cosign",
		Fix: &findings.Fix{Kind: "insert", Suggestion: `signs:
  - cmd: cosign
    artifacts: checksum
    signature: "${artifact}.sigstore.json"
    args: ["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]`},
	}}
}
