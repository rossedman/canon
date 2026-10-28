// Package scaffold renders project and component templates from the canon.
// A template is a directory with a template.yaml and files; names ending in
// .tmpl are rendered with text/template and lose the suffix.
package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v3"
)

// Template describes one scaffold kind.
type Template struct {
	Kind        string            `yaml:"-" json:"kind"`
	Description string            `yaml:"description" json:"description"`
	Root        string            `yaml:"root" json:"root,omitempty"`
	Records     []string          `yaml:"records" json:"records"`
	Vars        map[string]string `yaml:"vars" json:"vars,omitempty"`
	Next        []string          `yaml:"next" json:"next,omitempty"`
}

// File is one rendered file, with a slash path relative to the destination.
type File struct {
	Path    string `json:"path"`
	Content []byte `json:"-"`
}

// Plan is a rendered template, ready to write.
type Plan struct {
	Template *Template         `json:"template"`
	Root     string            `json:"root"`
	Vars     map[string]string `json:"vars"`
	Files    []File            `json:"files"`
	Next     []string          `json:"next,omitempty"`
}

// namePattern is what a scaffold name may look like.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// ErrUnknownKind is returned for a kind with no template.
var ErrUnknownKind = errors.New("unknown scaffold kind")

// List returns every template in fsys, sorted by kind.
func List(fsys fs.FS) ([]*Template, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	var out []*Template
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := Load(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// Load reads one template's metadata.
func Load(fsys fs.FS, kind string) (*Template, error) {
	if !namePattern.MatchString(kind) {
		return nil, fmt.Errorf("%w %q", ErrUnknownKind, kind)
	}
	data, err := fs.ReadFile(fsys, path.Join(kind, "template.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w %q", ErrUnknownKind, kind)
	}
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", kind, err)
	}
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse template %s: %w", kind, err)
	}
	t.Kind = kind
	return &t, nil
}

// funcs are the helpers templates can call.
var funcs = template.FuncMap{
	"snake": func(s string) string { return strings.ReplaceAll(s, "-", "_") },
	"upper": strings.ToUpper,
	"title": func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
}

// Render renders template kind for name. vars override the template's
// defaults; defaults may refer to name and to each other in declaration-free
// order, since each is rendered once with name and the overrides available.
func Render(fsys fs.FS, kind, name string, vars map[string]string) (*Plan, error) {
	if !namePattern.MatchString(name) {
		return nil, fmt.Errorf("name %q must be lowercase letters, digits and hyphens, starting with a letter", name)
	}
	t, err := Load(fsys, kind)
	if err != nil {
		return nil, err
	}
	resolved := map[string]string{"name": name}
	maps.Copy(resolved, vars)
	for _, k := range slices.Sorted(maps.Keys(t.Vars)) {
		if _, set := resolved[k]; set {
			continue
		}
		v, err := renderString(k, t.Vars[k], resolved)
		if err != nil {
			return nil, err
		}
		resolved[k] = v
	}
	resolved["name"] = name
	root, err := renderString("root", t.Root, resolved)
	if err != nil {
		return nil, err
	}
	plan := &Plan{Template: t, Root: root, Vars: resolved}
	for _, n := range t.Next {
		s, err := renderString("next", n, resolved)
		if err != nil {
			return nil, err
		}
		plan.Next = append(plan.Next, s)
	}
	err = fs.WalkDir(fsys, kind, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, kind), "/")
		if d.IsDir() || rel == "template.yaml" {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		out, err := renderString(rel, rel, resolved)
		if err != nil {
			return err
		}
		if trimmed, ok := strings.CutSuffix(out, ".tmpl"); ok {
			out = trimmed
			s, err := renderString(rel, string(data), resolved)
			if err != nil {
				return err
			}
			data = []byte(s)
		}
		if !safePath(out) {
			return fmt.Errorf("template %s renders an unsafe path %q", kind, out)
		}
		plan.Files = append(plan.Files, File{Path: path.Join(root, out), Content: data})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", kind, err)
	}
	slices.SortFunc(plan.Files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	return plan, nil
}

// renderString executes one template string against vars. A missing variable
// is an error, so typos fail loudly instead of rendering "<no value>".
func renderString(name, text string, vars map[string]string) (string, error) {
	if !strings.Contains(text, "{{") {
		return text, nil
	}
	tmpl, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, vars); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return b.String(), nil
}

// safePath reports whether a rendered path stays inside the destination.
func safePath(p string) bool {
	if p == "" || path.IsAbs(p) || strings.Contains(p, "\\") {
		return false
	}
	clean := path.Clean(p)
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

// Paths returns the plan's file paths.
func (p *Plan) Paths() []string {
	out := make([]string, len(p.Files))
	for i, f := range p.Files {
		out[i] = f.Path
	}
	return out
}

// ErrExists is returned when writing would overwrite a file.
var ErrExists = errors.New("file already exists")

// Write creates the plan's files under dest. It refuses to overwrite any
// existing file, and checks all of them before writing any.
func (p *Plan) Write(dest string) error {
	for _, f := range p.Files {
		target := filepath.Join(dest, filepath.FromSlash(f.Path))
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("%w: %s", ErrExists, target)
		}
	}
	for _, f := range p.Files {
		target := filepath.Join(dest, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // project directories are meant to be readable
			return fmt.Errorf("create directory for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(target, f.Content, 0o644); err != nil { //nolint:gosec // generated project files are meant to be readable
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}
	return nil
}
