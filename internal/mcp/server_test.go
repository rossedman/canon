package mcp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/fakerun"
)

// connect starts the server on an in-memory transport and returns a client session.
func connect(t *testing.T) *sdk.ClientSession {
	t.Helper()
	e, err := engine.Default()
	if err != nil {
		t.Fatal(err)
	}
	e.Runner = fakerun.New().Missing("golangci-lint", "ruff", "hadolint", "tflint", "checkov", "yamllint")
	srv, err := New(e, filepath.Join("..", "..", "testdata", "repos", "bad"), "test")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, ct := sdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call invokes a tool and decodes its structured output into out.
func call(t *testing.T, cs *sdk.ClientSession, name string, args map[string]any, out any) *sdk.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if out != nil && !res.IsError {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
	}
	return res
}

func TestToolsList(t *testing.T) {
	t.Parallel()
	cs := connect(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
		if tl.Description == "" || tl.InputSchema == nil || tl.Annotations == nil {
			t.Errorf("%s is missing description, schema or annotations", tl.Name)
		}
		schema, _ := json.Marshal(tl.InputSchema)
		if strings.Contains(string(schema), `"oneOf"`) {
			t.Errorf("%s: top-level oneOf breaks clients that reject it", tl.Name)
		}
		if tl.Annotations.ReadOnlyHint != (tl.Name != "scaffold") {
			t.Errorf("%s: read-only hint wrong", tl.Name)
		}
	}
	if strings.Join(names, ",") != "check,decide,explain,profile,review,scaffold" {
		t.Fatalf("tools = %v", names)
	}
}

func TestToolCalls(t *testing.T) {
	t.Parallel()
	cs := connect(t)
	var chk engine.CheckResult
	call(t, cs, "check", map[string]any{"paths": []string{"Dockerfile"}, "min_severity": "error"}, &chk)
	if chk.Profile != "go-cli" || chk.Summary.Error == 0 || chk.Findings[0].Why == "" {
		t.Fatalf("check = %+v", chk)
	}
	if res := call(t, cs, "check", map[string]any{"min_severity": "fatal"}, nil); !res.IsError {
		t.Fatal("invalid enum should be rejected by the schema")
	}
	if res := call(t, cs, "check", map[string]any{"paths": []string{"../escape"}}, nil); !res.IsError {
		t.Fatal("path outside root should be a tool error")
	}
	var dec engine.DecideResult
	call(t, cs, "decide", map[string]any{"question": "how do we handle errors in Go?"}, &dec)
	if dec.Gap || dec.Matches[0].ID != "go-errors-wrap" {
		t.Fatalf("decide = %+v", dec)
	}
	if res := call(t, cs, "decide", map[string]any{}, nil); !res.IsError {
		t.Fatal("missing question should be rejected")
	}
	if res := call(t, cs, "decide", map[string]any{"question": " "}, nil); !res.IsError {
		t.Fatal("blank question should be a tool error")
	}
	var exp engine.ExplainResult
	call(t, cs, "explain", map[string]any{"rule": "canon-final-stage-base"}, &exp)
	if exp.Record == nil || exp.Record.ID != "containers-distroless-base" || exp.Markdown == "" {
		t.Fatalf("explain = %+v", exp)
	}
	if res := call(t, cs, "explain", map[string]any{"rule": "a", "record": "b"}, nil); !res.IsError {
		t.Fatal("both rule and record should be a tool error")
	}
	var rev engine.ReviewResult
	call(t, cs, "review", map[string]any{"diff": "diff --git a/Dockerfile b/Dockerfile\nnew file mode 100644\n--- /dev/null\n+++ b/Dockerfile\n@@ -0,0 +1 @@\n+FROM debian\n"}, &rev)
	if len(rev.Concerns) != 1 || len(rev.Findings) == 0 {
		t.Fatalf("review = %+v", rev)
	}
	if res := call(t, cs, "review", map[string]any{}, nil); !res.IsError {
		t.Fatal("review without git should be a tool error")
	}
	var sc engine.ScaffoldResult
	call(t, cs, "scaffold", map[string]any{"kind": "go-cli", "name": "demo"}, &sc)
	if !sc.DryRun || len(sc.Files) == 0 || sc.Check != nil {
		t.Fatalf("scaffold should default to dry run: %+v", sc)
	}
	if res := call(t, cs, "scaffold", map[string]any{"kind": "cobol-app", "name": "demo"}, nil); !res.IsError {
		t.Fatal("unknown kind should be rejected by the schema")
	}
	if res := call(t, cs, "scaffold", map[string]any{"kind": "go-cli", "name": "Bad Name", "dry_run": true}, nil); !res.IsError {
		t.Fatal("bad name should be a tool error")
	}
	var prof engine.ProfileResult
	call(t, cs, "profile", map[string]any{}, &prof)
	if prof.Profile != "go-cli" {
		t.Fatalf("profile = %+v", prof)
	}
	abs, _ := filepath.Abs(filepath.Join("..", "..", "testdata", "repos", "good"))
	call(t, cs, "profile", map[string]any{"path": abs}, &prof)
	if prof.Profile != "go-cli" {
		t.Fatalf("profile abs = %+v", prof)
	}
	if res := call(t, cs, "profile", map[string]any{"path": "missing-dir"}, nil); res.IsError {
		t.Fatal("a missing dir resolves to the generic profile, not an error")
	}
}

func TestScaffoldWrites(t *testing.T) {
	t.Parallel()
	e, _ := engine.Default()
	e.Runner = fakerun.New().Missing("golangci-lint", "ruff", "hadolint", "tflint", "checkov", "yamllint")
	root := t.TempDir()
	srv, err := New(e, root, "test")
	if err != nil {
		t.Fatal(err)
	}
	st, ct := sdk.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()
	var sc engine.ScaffoldResult
	call(t, cs, "scaffold", map[string]any{"kind": "gha-workflow", "name": "lint", "dry_run": false}, &sc)
	if sc.DryRun || !sc.Passed || sc.Check == nil {
		t.Fatalf("scaffold write = %+v", sc)
	}
}

func TestResources(t *testing.T) {
	t.Parallel()
	cs := connect(t)
	ctx := context.Background()
	list, err := cs.ListResources(ctx, nil)
	if err != nil || len(list.Resources) != 2 {
		t.Fatalf("resources = %+v, %v", list, err)
	}
	tmpl, err := cs.ListResourceTemplates(ctx, nil)
	if err != nil || len(tmpl.ResourceTemplates) != 2 {
		t.Fatalf("templates = %+v, %v", tmpl, err)
	}
	read := func(uri string) (string, error) {
		res, err := cs.ReadResource(ctx, &sdk.ReadResourceParams{URI: uri})
		if err != nil {
			return "", err
		}
		return res.Contents[0].Text, nil
	}
	for uri, want := range map[string]string{
		"canon://records":                "\"id\": \"go-errors-wrap\"",
		"canon://principles":             "# Principles",
		"canon://records/go-errors-wrap": "## Decision",
		"canon://profiles/go-cli":        "\"records\"",
	} {
		got, err := read(uri)
		if err != nil || !strings.Contains(got, want) {
			t.Errorf("%s = %.80q, %v", uri, got, err)
		}
	}
	for _, uri := range []string{"canon://records/ghost", "canon://profiles/ghost"} {
		if _, err := read(uri); err == nil {
			t.Errorf("%s: want not found", uri)
		}
	}
}

func TestPrompts(t *testing.T) {
	t.Parallel()
	cs := connect(t)
	ctx := context.Background()
	list, err := cs.ListPrompts(ctx, nil)
	if err != nil || len(list.Prompts) != 3 {
		t.Fatalf("prompts = %+v, %v", list, err)
	}
	for name, args := range map[string]map[string]string{
		"canon-review":     {"base": "main"},
		"canon-new-record": {"topic": "logging"},
		"canon-adopt":      {"path": "svc"},
	} {
		for _, a := range []map[string]string{args, nil} {
			res, err := cs.GetPrompt(ctx, &sdk.GetPromptParams{Name: name, Arguments: a})
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			text := res.Messages[0].Content.(*sdk.TextContent).Text
			for _, v := range a {
				if !strings.Contains(text, v) {
					t.Errorf("%s does not use its argument %q", name, v)
				}
			}
			if res.Description == "" || len(text) < 100 {
				t.Errorf("%s: empty prompt", name)
			}
		}
	}
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()
	e, _ := engine.Default()
	srv, err := New(e, ".", "test")
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- ServeHTTP(ctx, srv, addr) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/", nil)
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = res.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test"}, nil).Connect(ctx, &sdk.StreamableClientTransport{Endpoint: "http://" + addr + "/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := cs.ListTools(ctx, nil); err != nil || len(res.Tools) != 6 {
		t.Fatalf("tools over http = %v, %v", res, err)
	}
	// Shut down while the client session (and its event stream) is still
	// open: a connected client must not hold the server open.
	start := time.Now()
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("shutdown took %s with a client connected", d)
	}
	_ = cs.Close()
	if err := ServeHTTP(context.Background(), srv, "bad-addr:-1"); err == nil {
		t.Fatal("bad address: want error")
	}
}

func TestSchemaErrors(t *testing.T) {
	t.Parallel()
	if _, err := schema("ghost"); err == nil {
		t.Fatal("missing schema: want error")
	}
	if _, err := New(nil, string([]byte{0}), "x"); err != nil {
		t.Logf("bad root rejected: %v", err)
	}
}
