package review

import (
	"strings"
	"testing"
	"testing/fstest"
)

// gitDiff covers new, deleted, renamed and modified files, a no-newline
// marker, and a removed line that looks like a file header.
const gitDiff = `diff --git a/go.mod b/go.mod
index 1..2 100644
--- a/go.mod
+++ b/go.mod
@@ -3,4 +3,6 @@ go 1.27
 require (
-	github.com/spf13/cobra v1.9.0
+	github.com/spf13/cobra v1.10.2
+	github.com/google/uuid v1.6.0
+	golang.org/x/sys v0.47.0 // indirect
 )
diff --git a/Dockerfile b/Dockerfile
new file mode 100644
--- /dev/null
+++ b/Dockerfile
@@ -0,0 +1,2 @@
+FROM scratch
+COPY a /a
\ No newline at end of file
diff --git a/old.txt b/old.txt
deleted file mode 100644
--- a/old.txt
+++ /dev/null
@@ -1 +0,0 @@
--- this line starts with dashes
diff --git a/a.go b/b.go
similarity index 90%
rename from a.go
rename to b.go
diff --git a/pkg/api.go b/pkg/api.go
--- a/pkg/api.go
+++ b/pkg/api.go
@@ -10,3 +10,4 @@
-func Run(x int) error {
+func Run(x int, y int) error {
+func New() *T {
-type Old struct{}
 }
`

func TestParseDiff(t *testing.T) {
	t.Parallel()
	ds := ParseDiff(gitDiff)
	if len(ds) != 5 {
		t.Fatalf("got %d files: %+v", len(ds), ds)
	}
	gomod, docker, old, renamed, api := ds[0], ds[1], ds[2], ds[3], ds[4]
	if gomod.Status != Modified || len(gomod.Added) != 3 || gomod.Added[0].N != 4 || gomod.Removed[0].N != 4 {
		t.Fatalf("go.mod = %+v", gomod)
	}
	if docker.Status != Added || docker.Path != "Dockerfile" || len(docker.Added) != 2 || docker.Added[1].N != 2 {
		t.Fatalf("Dockerfile = %+v", docker)
	}
	if old.Status != Deleted || old.Path != "old.txt" || len(old.Removed) != 1 || old.Removed[0].Text != "-- this line starts with dashes" {
		t.Fatalf("old.txt = %+v", old)
	}
	if renamed.Status != Renamed || renamed.Path != "b.go" || renamed.OldPath != "a.go" {
		t.Fatalf("rename = %+v", renamed)
	}
	if api.Path != "pkg/api.go" || len(api.Added) != 2 || len(api.Removed) != 2 {
		t.Fatalf("api = %+v", api)
	}
}

func TestParsePlainUnifiedDiff(t *testing.T) {
	t.Parallel()
	plain := "--- a.txt\t2026-01-01\n+++ a.txt\t2026-01-02\n@@ -1 +1 @@\n-x\n+y\n--- b.txt\n+++ b.txt\n@@ -1,2 +1,2 @@\n\n-a\n+b\n@@ bogus\n"
	ds := ParseDiff(plain)
	if len(ds) != 2 || ds[0].Path != "a.txt" || ds[1].Path != "b.txt" || ds[1].Added[0].N != 2 {
		t.Fatalf("plain = %+v", ds)
	}
	if ds := ParseDiff("garbage\n+++ orphan\n"); len(ds) != 0 {
		t.Fatalf("garbage = %+v", ds)
	}
	if ds := ParseDiff("diff --git weird\n"); len(ds) != 0 {
		t.Fatalf("header without paths = %+v", ds)
	}
	if _, ok := parseHunk("@@ -x +1 @@"); ok {
		t.Fatal("bad hunk start accepted")
	}
	if _, ok := parseHunk("@@ -1,x +1 @@"); ok {
		t.Fatal("bad hunk count accepted")
	}
	if _, ok := parseHunk("@@ -1"); ok {
		t.Fatal("short hunk accepted")
	}
	if p := diffPath("/dev/null"); p != "" {
		t.Fatal("dev null not empty")
	}
}

// kinds lists concern kinds and details for assertions.
func kinds(cs []Concern) string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Kind+" "+c.Path+" "+c.Detail)
	}
	return strings.Join(out, "\n")
}

func TestClassify(t *testing.T) {
	t.Parallel()
	diff := gitDiff + `diff --git a/requirements.txt b/requirements.txt
--- a/requirements.txt
+++ b/requirements.txt
@@ -1,2 +1,3 @@
-requests==2.31
+requests==2.32
+httpx>=0.27
+# a comment
diff --git a/pyproject.toml b/pyproject.toml
--- a/pyproject.toml
+++ b/pyproject.toml
@@ -1 +1,3 @@
+dependencies = [
+  "rich>=13",
+]
diff --git a/package.json b/package.json
--- a/package.json
+++ b/package.json
@@ -1 +1,2 @@
+    "left-pad": "^1.3.0",
diff --git a/.github/workflows/new.yml b/.github/workflows/new.yml
new file mode 100644
--- /dev/null
+++ b/.github/workflows/new.yml
@@ -0,0 +1 @@
+on: push
diff --git a/api/openapi.yaml b/api/openapi.yaml
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -1 +1 @@
-x
+y
diff --git a/infra/versions.tf b/infra/versions.tf
--- a/infra/versions.tf
+++ b/infra/versions.tf
@@ -1,2 +1,2 @@
-      version = "~> 5.0"
+      version = "~> 6.0"
       source = "hashicorp/aws"
diff --git a/infra/new.tf b/infra/new.tf
new file mode 100644
--- /dev/null
+++ b/infra/new.tf
@@ -0,0 +1 @@
+version = "1"
diff --git a/internal/x/x.go b/internal/x/x.go
--- a/internal/x/x.go
+++ b/internal/x/x.go
@@ -1 +1 @@
-func Hidden() {}
+func Hidden(a int) {}
diff --git a/newpkg/a.go b/newpkg/a.go
new file mode 100644
--- /dev/null
+++ b/newpkg/a.go
@@ -0,0 +1 @@
+package newpkg
diff --git a/tested/a.go b/tested/a.go
new file mode 100644
--- /dev/null
+++ b/tested/a.go
@@ -0,0 +1 @@
+package tested
diff --git a/tested/a_test.go b/tested/a_test.go
new file mode 100644
--- /dev/null
+++ b/tested/a_test.go
@@ -0,0 +1 @@
+package tested
diff --git a/existing/new.go b/existing/new.go
new file mode 100644
--- /dev/null
+++ b/existing/new.go
@@ -0,0 +1 @@
+package existing
`
	repo := fstest.MapFS{"newpkg/a.go": {}, "existing/new.go": {}, "existing/old.go": {}, "tested/a.go": {}}
	got := kinds(Classify(ParseDiff(diff), repo))
	want := []string{
		"new-dependency go.mod adds github.com/google/uuid v1.6.0",
		"new-dockerfile Dockerfile adds a Dockerfile",
		"exported-api-change pkg/api.go adds exported New",
		"exported-api-change pkg/api.go changes the signature of exported Run",
		"exported-api-change pkg/api.go removes exported Old",
		"new-dependency requirements.txt adds httpx>=0.27",
		"new-dependency pyproject.toml adds rich>=13",
		"new-dependency package.json adds left-pad ^1.3.0",
		"new-workflow .github/workflows/new.yml adds a GitHub Actions workflow",
		"api-spec-change api/openapi.yaml changes the API spec",
		`terraform-provider-bump infra/versions.tf sets version = "~> 6.0"`,
		"missing-tests newpkg adds package newpkg with no tests",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing concern %q in\n%s", w, got)
		}
	}
	for _, unwanted := range []string{"cobra", "golang.org/x/sys", "requests", "a comment", "Hidden", "infra/new.tf", "tested", "existing", "old.txt", "dependencies ="} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected %q in\n%s", unwanted, got)
		}
	}
	if !strings.Contains(kinds(Classify(ParseDiff(diff), nil)), "missing-tests existing") {
		t.Error("without a repo, the diff alone should decide")
	}
	if !newPackage(fstest.MapFS{}, "gone", nil) {
		t.Error("unreadable dir should count as new")
	}
}

func BenchmarkParseDiff(b *testing.B) {
	big := strings.Repeat(gitDiff, 50)
	b.ReportAllocs()
	for b.Loop() {
		ParseDiff(big)
	}
}

func BenchmarkClassify(b *testing.B) {
	ds := ParseDiff(strings.Repeat(gitDiff, 50))
	b.ReportAllocs()
	for b.Loop() {
		Classify(ds, nil)
	}
}

// FuzzParseDiff checks the parser never panics, every file has a path, and
// line numbers are positive and increasing within a file.
func FuzzParseDiff(f *testing.F) {
	f.Add(gitDiff)
	f.Add("--- a\n+++ b\n@@ -1,99999 +1 @@\n-x\n")
	f.Add("diff --git a/x b/x\n@@ -0,0 +1 @@\n+")
	f.Fuzz(func(t *testing.T, text string) {
		for _, d := range ParseDiff(text) {
			if d.Path == "" {
				t.Fatal("file without a path")
			}
			for _, ls := range [][]Line{d.Added, d.Removed} {
				prev := 0
				for _, l := range ls {
					if l.N < 1 || l.N <= prev {
						t.Fatalf("bad line numbers in %+v", d)
					}
					prev = l.N
				}
			}
			Classify([]FileDiff{d}, nil)
		}
	})
}
