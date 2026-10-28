package hook

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/fakerun"
	"github.com/rossedman/canon/internal/findings"
)

// newEngine returns the embedded canon with every external checker missing.
func newEngine(t *testing.T) *engine.Engine {
	t.Helper()
	e, err := engine.Default()
	if err != nil {
		t.Fatal(err)
	}
	e.Runner = fakerun.New().Missing("golangci-lint", "ruff", "hadolint", "tflint", "checkov", "yamllint")
	return e
}

// fixture is the path of the bad fixture repo.
var fixture = filepath.Join("..", "..", "testdata", "repos", "bad")

func TestParseInput(t *testing.T) {
	t.Parallel()
	in, err := ParseInput(strings.NewReader(`{"cwd":"/r","hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/r/a.go"}}`))
	if err != nil || in.Cwd != "/r" || strings.Join(in.Paths(), ",") != "/r/a.go" {
		t.Fatalf("in = %+v, %v", in, err)
	}
	nb, _ := ParseInput(strings.NewReader(`{"tool_input":{"notebook_path":"n.ipynb"}}`))
	if strings.Join(nb.Paths(), ",") != "n.ipynb" {
		t.Fatal("notebook path not read")
	}
	empty, err := ParseInput(strings.NewReader("  \n"))
	if err != nil || empty.Paths() != nil {
		t.Fatal("empty input should give no paths")
	}
	if _, err := ParseInput(strings.NewReader("{")); err == nil {
		t.Fatal("bad JSON: want error")
	}
	if _, err := ParseInput(iotest.ErrReader(context.Canceled)); err == nil {
		t.Fatal("read error: want error")
	}
}

func TestRun(t *testing.T) {
	t.Parallel()
	e := newEngine(t)
	ctx := context.Background()
	res, err := Run(ctx, e, fixture, []string{"Dockerfile"}, findings.Error)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != ExitBlock || !strings.Contains(res.Message, "canon-final-stage-base") || !strings.Contains(res.Message, "fix: FROM gcr.io/distroless") {
		t.Fatalf("res = %+v", res)
	}
	abs, _ := filepath.Abs(filepath.Join(fixture, "internal/thing/thing.go"))
	res, err = Run(ctx, e, fixture, []string{abs}, findings.Error)
	if err != nil || res.Code != 0 || res.Check == nil {
		t.Fatalf("warnings below min should not block: %+v %v", res, err)
	}
	res, err = Run(ctx, e, fixture, []string{"/somewhere/else.go", "../outside.go"}, findings.Error)
	if err != nil || res.Code != 0 || res.Check != nil {
		t.Fatalf("outside files should be ignored: %+v %v", res, err)
	}
	if _, err := Run(ctx, e, fixture, []string{"missing.go"}, findings.Error); err == nil {
		t.Fatal("missing file: want error")
	}
}

// FuzzParseInput checks that any payload either fails cleanly or yields at
// most one path.
func FuzzParseInput(f *testing.F) {
	f.Add([]byte(`{"cwd":"/r","tool_input":{"file_path":"/r/a.go"}}`))
	f.Add([]byte(`{"tool_input":{"notebook_path":"n.ipynb","file_path":""}}`))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, data []byte) {
		in, err := ParseInput(strings.NewReader(string(data)))
		if err != nil {
			return
		}
		if ps := in.Paths(); len(ps) > 1 || len(ps) == 1 && ps[0] == "" {
			t.Fatalf("Paths = %q", ps)
		}
	})
}
