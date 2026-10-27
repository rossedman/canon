// Package profiles detects what kind of repo it is looking at and selects the
// records that apply to it.
package profiles

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rossedman/canon/internal/records"
)

// Detect lists the files that identify a profile. Every entry in All must be
// present, and at least one entry in Any when Any is set. An entry ending in
// "/" names a directory; one with glob characters matches with fs.Glob.
type Detect struct {
	All []string `yaml:"all" json:"all,omitempty"`
	Any []string `yaml:"any" json:"any,omitempty"`
}

// Profile is a named bundle of records for a kind of repo.
type Profile struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Priority    int      `yaml:"priority" json:"priority"`
	Detect      Detect   `yaml:"detect" json:"detect"`
	Scopes      []string `yaml:"scopes" json:"scopes"`
	Records     []string `yaml:"records" json:"records,omitempty"`
	Exclude     []string `yaml:"exclude" json:"exclude,omitempty"`
}

// Load reads every .yaml file in dir.
func Load(fsys fs.FS, dir string) ([]*Profile, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read profiles: %w", err)
	}
	var out []*Profile
	for _, e := range entries {
		if e.IsDir() || (path.Ext(e.Name()) != ".yaml" && path.Ext(e.Name()) != ".yml") {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read profile %s: %w", e.Name(), err)
		}
		var p Profile
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse profile %s: %w", e.Name(), err)
		}
		if p.Name == "" {
			return nil, fmt.Errorf("profile %s has no name", e.Name())
		}
		out = append(out, &p)
	}
	slices.SortFunc(out, func(a, b *Profile) int {
		return cmp.Or(cmp.Compare(b.Priority, a.Priority), cmp.Compare(a.Name, b.Name))
	})
	return out, nil
}

// Matches reports whether repo (rooted at the repo root) satisfies p's detection.
func (p *Profile) Matches(repo fs.FS) bool {
	for _, e := range p.Detect.All {
		if !exists(repo, e) {
			return false
		}
	}
	if len(p.Detect.Any) == 0 {
		return true
	}
	return slices.ContainsFunc(p.Detect.Any, func(e string) bool { return exists(repo, e) })
}

// exists reports whether a detection entry is present in repo.
func exists(repo fs.FS, entry string) bool {
	if strings.ContainsAny(entry, "*?[") {
		m, err := fs.Glob(repo, entry)
		return err == nil && len(m) > 0
	}
	if dir, ok := strings.CutSuffix(entry, "/"); ok {
		st, err := fs.Stat(repo, dir)
		return err == nil && st.IsDir()
	}
	_, err := fs.Stat(repo, entry)
	return err == nil
}

// Find returns the profile with name.
func Find(ps []*Profile, name string) (*Profile, bool) {
	for _, p := range ps {
		if p.Name == name {
			return p, true
		}
	}
	return nil, false
}

// DetectProfile returns the highest-priority profile that matches repo. ps must be
// sorted as Load returns them.
func DetectProfile(repo fs.FS, ps []*Profile) (*Profile, bool) {
	for _, p := range ps {
		if p.Matches(repo) {
			return p, true
		}
	}
	return nil, false
}

// Resolve lists the records in force under p: accepted records in one of its
// scopes or named in Records, minus Exclude.
func (p *Profile) Resolve(set *records.Set) []*records.Record {
	var out []*records.Record
	for _, r := range set.All() {
		if !r.InForce() || slices.Contains(p.Exclude, r.ID) {
			continue
		}
		if r.HasScope(p.Scopes) || slices.Contains(p.Records, r.ID) {
			out = append(out, r)
		}
	}
	return out
}

// Problem is one thing wrong with a profile.
type Problem struct {
	Profile string `json:"profile"`
	Message string `json:"message"`
}

// Validate checks that profiles name known scopes and records, and that a
// catch-all profile exists so every repo resolves to something.
func Validate(ps []*Profile, set *records.Set) []Problem {
	var out []Problem
	catchAll := false
	for _, p := range ps {
		if len(p.Detect.All) == 0 && len(p.Detect.Any) == 0 {
			catchAll = true
		}
		for _, s := range p.Scopes {
			if !slices.Contains(records.Scopes, s) {
				out = append(out, Problem{p.Name, fmt.Sprintf("scope %q is not known", s)})
			}
		}
		for _, id := range append(slices.Clone(p.Records), p.Exclude...) {
			if _, ok := set.Get(id); !ok {
				out = append(out, Problem{p.Name, fmt.Sprintf("record %q does not exist", id)})
			}
		}
	}
	if !catchAll {
		out = append(out, Problem{"", "no profile without detect rules; some repos will match nothing"})
	}
	return out
}
