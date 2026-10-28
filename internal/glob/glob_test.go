package glob

import (
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"**", "anything/at/all", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/a/b.go", true},
		{"**/*.go", "internal/a/b.txt", false},
		{"cmd/*.go", "cmd/root.go", true},
		{"cmd/*.go", "cmd/sub/root.go", false},
		{"Dockerfile", "Dockerfile", true},
		{"Dockerfile", "svc/Dockerfile", false},
		{"**/Dockerfile", "svc/api/Dockerfile", true},
		{"**/Dockerfile", "Dockerfile", true},
		{"a/**/b", "a/b", true},
		{"a/**/b", "a/x/y/b", true},
		{"a/**/b", "a/x/y/c", false},
		{"a/**/**/b", "a/x/b", true},
		{"a/**", "a/x/y", true},
		{".github/workflows/*.yml", ".github/workflows/ci.yml", true},
		{"[", "[", false},
		{"*.tf", "main.tf", true},
		{"x/y", "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"~"+tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Match(tt.pattern, tt.name); got != tt.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}

func TestMatchAny(t *testing.T) {
	t.Parallel()
	if !MatchAny([]string{"*.md", "**/*.go"}, "a/b.go") {
		t.Fatal("want match")
	}
	if MatchAny(nil, "a") {
		t.Fatal("empty pattern list matched")
	}
}

func BenchmarkMatch(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Match("**/internal/**/*_test.go", "github.com/x/y/internal/a/b/c/d_test.go")
	}
}

// FuzzMatch checks that Match never panics and that "**" and a literal
// pattern behave as documented.
func FuzzMatch(f *testing.F) {
	for _, s := range [][2]string{{"**/*.go", "a/b.go"}, {"a/**/b", "a/x/b"}, {"[", "x"}, {"cmd/*.go", "cmd/x.go"}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, pattern, name string) {
		_ = Match(pattern, name)
		if !Match("**", name) {
			t.Fatal(`"**" must match everything`)
		}
		if !strings.ContainsAny(name, `*?[\`) && !Match(name, name) {
			t.Fatalf("literal %q does not match itself", name)
		}
	})
}
