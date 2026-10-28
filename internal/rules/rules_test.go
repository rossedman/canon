package rules

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rossedman/canon/canon"
)

// yamlDoc is a small rules file.
const yamlDoc = `rules:
  - id: b-exact
    record: rec-b
    checker: lint
    codes: [E1]
  - id: a-wild
    record: rec-a
    checker: lint
    codes: ["*"]
concerns:
  new-dependency: rec-a
  ghost: rec-missing
`

func TestParseAndResolve(t *testing.T) {
	t.Parallel()
	reg, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	if ids := []string{reg.All()[0].ID, reg.All()[1].ID}; ids[0] != "a-wild" {
		t.Fatalf("not sorted: %v", ids)
	}
	if r, ok := reg.Resolve("lint", "E1", nil); !ok || r.ID != "b-exact" {
		t.Fatal("exact code should win")
	}
	if r, ok := reg.Resolve("lint", "E2", nil); !ok || r.ID != "a-wild" {
		t.Fatal("wildcard should catch other codes")
	}
	onlyWild := func(r *Rule) bool { return r.ID == "a-wild" }
	if _, ok := reg.Resolve("lint", "E1", onlyWild); ok {
		t.Fatal("a code whose exact rule is not in force must not fall to the wildcard")
	}
	if _, ok := reg.Resolve("lint", "E2", func(*Rule) bool { return false }); ok {
		t.Fatal("disallowed wildcard resolved")
	}
	if _, ok := reg.Resolve("other", "E1", nil); ok {
		t.Fatal("unknown checker resolved")
	}
	if got := reg.ForRecord("rec-b"); len(got) != 1 || !reg.Has("b-exact") || reg.Has("nope") {
		t.Fatal("ForRecord/Has wrong")
	}
	if _, ok := reg.Get("a-wild"); !ok {
		t.Fatal("Get failed")
	}
	if id, ok := reg.ConcernRecord("new-dependency"); !ok || id != "rec-a" || len(reg.Concerns()) != 2 {
		t.Fatal("concerns wrong")
	}
}

func TestNewErrors(t *testing.T) {
	t.Parallel()
	tests := map[string][]*Rule{
		"missing fields": {{ID: "x"}},
		"duplicate id":   {{ID: "x", Record: "r", Checker: "c"}, {ID: "x", Record: "r", Checker: "c"}},
		"duplicate code": {{ID: "x", Record: "r", Checker: "c", Codes: []string{"A"}}, {ID: "y", Record: "r", Checker: "c", Codes: []string{"A"}}},
	}
	for name, rs := range tests {
		if _, err := New(rs, nil); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if _, err := Parse([]byte("rules: [")); err == nil {
		t.Error("want yaml error")
	}
	if _, err := Load(fstest.MapFS{}, "rules.yaml"); err == nil {
		t.Error("want read error")
	}
	if reg, err := New(nil, nil); err != nil || reg.Concerns() == nil {
		t.Error("nil concerns should become an empty map")
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	reg, err := New([]*Rule{
		{ID: "ok", Record: "rec", Checker: "lint", Codes: []string{"A"}},
		{ID: "unlisted", Record: "rec", Checker: "lint", Codes: []string{"B"}},
		{ID: "orphan", Record: "gone", Checker: "nope"},
	}, map[string]string{"k": "gone"})
	if err != nil {
		t.Fatal(err)
	}
	records := map[string][]string{"rec": {"ok"}}
	ps := reg.Validate(func(id string) ([]string, bool) { r, ok := records[id]; return r, ok }, func(c string) bool { return c == "lint" })
	var text strings.Builder
	for _, p := range ps {
		text.WriteString(p.Rule + ": " + p.Message + "\n")
	}
	for _, want := range []string{"unlisted: record \"rec\" does not list this rule", "orphan: record \"gone\" does not exist", "orphan: checker \"nope\"", "orphan: codes is empty", "concern:k"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("missing %q in\n%s", want, text.String())
		}
	}
}

func TestEmbeddedRulesLoad(t *testing.T) {
	t.Parallel()
	reg, err := Load(canon.FS, "rules/rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Resolve("hadolint", "DL3006", nil); !ok {
		t.Fatal("DL3006 should map to a rule")
	}
}
