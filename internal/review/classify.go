package review

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/rossedman/canon/internal/glob"
)

// Concern is a record-level issue in a change that no linter reports.
type Concern struct {
	Kind   string `json:"kind"`
	Record string `json:"record"`
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`
	Detail string `json:"detail"`
}

// Concern kinds.
const (
	KindNewDependency = "new-dependency"
	KindNewDockerfile = "new-dockerfile"
	KindProviderBump  = "terraform-provider-bump"
	KindExportedAPI   = "exported-api-change"
	KindMissingTests  = "missing-tests"
	KindNewWorkflow   = "new-workflow"
	KindAPISpecChange = "api-spec-change"
)

// File patterns that trigger concerns.
var (
	dockerfilePatterns = []string{"Dockerfile", "**/Dockerfile", "*.Dockerfile", "**/*.Dockerfile", "Containerfile", "**/Containerfile"}
	workflowPatterns   = []string{".github/workflows/*.yml", ".github/workflows/*.yaml"}
	apiSpecPatterns    = []string{"api/**/*.yaml", "api/**/*.yml", "**/openapi.yaml", "**/openapi.yml"}
	terraformPatterns  = []string{"**/*.tf", "**/.terraform.lock.hcl"}
)

// Classify finds concerns in diffs. repo, when non-nil, is the working tree,
// used to tell a brand-new package from new files in an existing one.
func Classify(diffs []FileDiff, repo fs.FS) []Concern {
	var out []Concern
	for _, d := range diffs {
		if d.Status == Deleted {
			continue
		}
		out = append(out, dependencies(d)...)
		switch {
		case d.Status == Added && glob.MatchAny(dockerfilePatterns, d.Path):
			out = append(out, Concern{Kind: KindNewDockerfile, Path: d.Path, Line: 1, Detail: "adds a Dockerfile"})
		case d.Status == Added && glob.MatchAny(workflowPatterns, d.Path):
			out = append(out, Concern{Kind: KindNewWorkflow, Path: d.Path, Line: 1, Detail: "adds a GitHub Actions workflow"})
		case d.Status != Added && glob.MatchAny(apiSpecPatterns, d.Path):
			out = append(out, Concern{Kind: KindAPISpecChange, Path: d.Path, Detail: "changes the API spec; regenerate code and check for drift"})
		}
		if glob.MatchAny(terraformPatterns, d.Path) {
			out = append(out, providerBumps(d)...)
		}
		out = append(out, exportedAPI(d)...)
	}
	return append(out, missingTests(diffs, repo)...)
}

var (
	// goRequire matches a go.mod require line, inside or outside a block.
	goRequire = regexp.MustCompile(`^\s*(?:require\s+)?([A-Za-z0-9][\w.\-]*\.[\w.\-]+(?:/[\w.\-~]+)*)\s+(v\S+)`)
	// pyRequirement matches a requirements.txt line or a pyproject dependency string.
	pyRequirement = regexp.MustCompile(`^\s*"?([A-Za-z0-9][A-Za-z0-9._\-]*)(?:\[[^\]]*\])?\s*(?:[<>=~!]=?[^",]*)?"?,?\s*$`)
	// npmDependency matches a package.json dependency entry.
	npmDependency = regexp.MustCompile(`^\s*"(@?[\w.\-]+(?:/[\w.\-]+)?)":\s*"([\^~]?\d[^"]*)"`)
)

// dependencies reports dependencies that a manifest change adds. A module
// that also appears on a removed line is a version change, not a new
// dependency, and is skipped.
func dependencies(d FileDiff) []Concern {
	base := path.Base(d.Path)
	var re *regexp.Regexp
	switch {
	case base == "go.mod":
		re = goRequire
	case strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt"), base == "pyproject.toml":
		re = pyRequirement
	case base == "package.json":
		re = npmDependency
	default:
		return nil
	}
	removed := map[string]bool{}
	for _, l := range d.Removed {
		if m := re.FindStringSubmatch(l.Text); m != nil {
			removed[strings.ToLower(m[1])] = true
		}
	}
	var out []Concern
	for _, l := range d.Added {
		if strings.Contains(l.Text, "// indirect") || strings.HasPrefix(strings.TrimSpace(l.Text), "#") {
			continue
		}
		m := re.FindStringSubmatch(l.Text)
		if m == nil || removed[strings.ToLower(m[1])] || isManifestKeyword(base, m[1]) {
			continue
		}
		out = append(out, Concern{Kind: KindNewDependency, Path: d.Path, Line: l.N, Detail: "adds " + depName(re, m, l.Text)})
	}
	return out
}

// depName names a dependency for a concern: module and version for Go and
// npm, the requirement as written for Python.
func depName(re *regexp.Regexp, m []string, line string) string {
	if re == pyRequirement {
		return strings.Trim(strings.TrimSpace(line), `",`)
	}
	return m[1] + " " + m[2]
}

// isManifestKeyword filters words the loose Python pattern would mistake for
// package names, such as the go directive or a bare TOML key.
func isManifestKeyword(base, name string) bool {
	if base != "pyproject.toml" && !strings.HasPrefix(base, "requirements") {
		return false
	}
	return slices.Contains([]string{"dependencies", "requires", "name", "version", "description"}, strings.ToLower(name))
}

// tfVersion matches a version or required_version assignment.
var tfVersion = regexp.MustCompile(`^\s*(required_version|version)\s*=\s*"([^"]+)"`)

// providerBumps reports changed version constraints in Terraform files and
// the lock file.
func providerBumps(d FileDiff) []Concern {
	if d.Status == Added && !strings.HasSuffix(d.Path, ".terraform.lock.hcl") {
		return nil
	}
	old := map[string]bool{}
	for _, l := range d.Removed {
		if m := tfVersion.FindStringSubmatch(l.Text); m != nil {
			old[m[2]] = true
		}
	}
	var out []Concern
	for _, l := range d.Added {
		m := tfVersion.FindStringSubmatch(l.Text)
		if m == nil || old[m[2]] {
			continue
		}
		out = append(out, Concern{Kind: KindProviderBump, Path: d.Path, Line: l.N, Detail: fmt.Sprintf("sets %s = %q", m[1], m[2])})
	}
	return out
}

// goExported matches a top-level declaration of an exported Go identifier.
var goExported = regexp.MustCompile(`^(?:func\s+(?:\([^)]*\)\s*)?|type\s+|var\s+|const\s+)([A-Z]\w*)`)

// exportedAPI reports exported Go declarations a change adds, removes or
// rewrites, outside internal/ and main packages' cmd/ trees.
func exportedAPI(d FileDiff) []Concern {
	if !strings.HasSuffix(d.Path, ".go") || strings.HasSuffix(d.Path, "_test.go") {
		return nil
	}
	segs := strings.Split(d.Path, "/")
	if slices.Contains(segs, "internal") || slices.Contains(segs, "cmd") || slices.Contains(segs, "testdata") {
		return nil
	}
	added := map[string]Line{}
	for _, l := range d.Added {
		if m := goExported.FindStringSubmatch(l.Text); m != nil {
			added[m[1]] = l
		}
	}
	var out []Concern
	seen := map[string]bool{}
	for _, l := range d.Removed {
		m := goExported.FindStringSubmatch(l.Text)
		if m == nil {
			continue
		}
		name := m[1]
		seen[name] = true
		if a, ok := added[name]; ok {
			if strings.TrimSpace(a.Text) != strings.TrimSpace(l.Text) {
				out = append(out, Concern{Kind: KindExportedAPI, Path: d.Path, Line: a.N, Detail: "changes the signature of exported " + name})
			}
			continue
		}
		out = append(out, Concern{Kind: KindExportedAPI, Path: d.Path, Detail: "removes exported " + name})
	}
	for name, l := range added {
		if !seen[name] {
			out = append(out, Concern{Kind: KindExportedAPI, Path: d.Path, Line: l.N, Detail: "adds exported " + name})
		}
	}
	slices.SortFunc(out, func(a, b Concern) int { return strings.Compare(a.Detail, b.Detail) })
	return out
}

// missingTests reports new Go packages (directories whose every Go file is
// added by this change) that have no test file.
func missingTests(diffs []FileDiff, repo fs.FS) []Concern {
	added := map[string][]string{}
	hasTest := map[string]bool{}
	for _, d := range diffs {
		if !strings.HasSuffix(d.Path, ".go") || d.Status == Deleted {
			continue
		}
		dir := path.Dir(d.Path)
		if strings.HasSuffix(d.Path, "_test.go") {
			hasTest[dir] = true
			continue
		}
		if d.Status == Added {
			added[dir] = append(added[dir], d.Path)
		}
	}
	dirs := make([]string, 0, len(added))
	for dir := range added {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	var out []Concern
	for _, dir := range dirs {
		if hasTest[dir] || !newPackage(repo, dir, added[dir]) {
			continue
		}
		out = append(out, Concern{Kind: KindMissingTests, Path: dir, Detail: "adds package " + dir + " with no tests"})
	}
	return out
}

// newPackage reports whether every Go file in dir on disk is one of added,
// and none of them is a test. With no repo, the diff alone decides.
func newPackage(repo fs.FS, dir string, added []string) bool {
	if repo == nil {
		return true
	}
	entries, err := fs.ReadDir(repo, dir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		p := path.Join(dir, e.Name())
		if e.IsDir() || !strings.HasSuffix(p, ".go") {
			continue
		}
		if strings.HasSuffix(p, "_test.go") || !slices.Contains(added, p) {
			return false
		}
	}
	return true
}
