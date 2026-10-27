// Package mcp exposes the engine as an MCP server: six tools, four
// resources and three prompts, over stdio or streamable HTTP.
package mcp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/findings"
)

// schemas holds each tool's input schema, kept as JSON so it matches the
// published tool contract exactly.
//
//go:embed schemas/*.json
var schemas embed.FS

// instructions tells the client when to reach for each tool.
const instructions = `canon serves engineering standards as decision records.
Call decide before choosing how to do something ("how should this service load config?").
Call check after editing files, and review before opening a pull request; both return findings that carry rule, record and why.
Call explain with a finding's rule or record to read the full reasoning.
Call profile to see which records apply to a repo, and scaffold (dry_run first) to start a project that already follows them.
When decide reports a gap, say the canon has no answer rather than guessing.`

// Server binds the engine to a repository root.
type Server struct {
	engine *engine.Engine
	root   string
}

// New builds the MCP server for e, resolving tool paths against root.
func New(e *engine.Engine, root, version string) (*sdk.Server, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	s := &Server{engine: e, root: abs}
	srv := sdk.NewServer(&sdk.Implementation{Name: "canon", Title: "canon", Version: version}, &sdk.ServerOptions{Instructions: instructions})
	if err := s.addTools(srv); err != nil {
		return nil, err
	}
	s.addResources(srv)
	s.addPrompts(srv)
	return srv, nil
}

// schema loads a tool's input schema.
func schema(name string) (*jsonschema.Schema, error) {
	data, err := schemas.ReadFile("schemas/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("read %s schema: %w", name, err)
	}
	var sc jsonschema.Schema
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse %s schema: %w", name, err)
	}
	return &sc, nil
}

// readOnly marks a tool that never changes anything.
func readOnly(title string) *sdk.ToolAnnotations {
	f := false
	return &sdk.ToolAnnotations{Title: title, ReadOnlyHint: true, DestructiveHint: &f, OpenWorldHint: &f}
}

// CheckInput is the check tool's arguments.
type CheckInput struct {
	Paths       []string `json:"paths,omitempty"`
	Scope       []string `json:"scope,omitempty"`
	MinSeverity string   `json:"min_severity,omitempty"`
	ChangedOnly bool     `json:"changed_only,omitempty"`
}

// DecideInput is the decide tool's arguments.
type DecideInput struct {
	Question string   `json:"question"`
	Scope    []string `json:"scope,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

// ExplainInput is the explain tool's arguments.
type ExplainInput struct {
	Rule   string `json:"rule,omitempty"`
	Record string `json:"record,omitempty"`
}

// ReviewInput is the review tool's arguments.
type ReviewInput struct {
	Base string `json:"base,omitempty"`
	Diff string `json:"diff,omitempty"`
}

// ScaffoldInput is the scaffold tool's arguments.
type ScaffoldInput struct {
	Kind   string            `json:"kind"`
	Name   string            `json:"name"`
	Dest   string            `json:"dest,omitempty"`
	Vars   map[string]string `json:"vars,omitempty"`
	DryRun *bool             `json:"dry_run,omitempty"`
}

// ProfileInput is the profile tool's arguments.
type ProfileInput struct {
	Path string `json:"path,omitempty"`
}

// tool is one registration: a tool definition and a function that adds it.
type tool struct {
	name  string
	desc  string
	notes *sdk.ToolAnnotations
	add   func(*sdk.Server, *sdk.Tool)
}

// addTools registers the six tools.
func (s *Server) addTools(srv *sdk.Server) error {
	f := false
	tools := []tool{
		{"check", "Run the canon's rules against paths and return findings. Each finding names its rule, the record it enforces, and why.", readOnly("Check against the canon"),
			func(srv *sdk.Server, tl *sdk.Tool) { sdk.AddTool(srv, tl, s.check) }},
		{"decide", `Answer a "how do we do X" question from decision records. Returns ranked records with their Decision and Exceptions, or a gap when the canon has no answer.`, readOnly("Decide from the canon"),
			func(srv *sdk.Server, tl *sdk.Tool) { sdk.AddTool(srv, tl, s.decide) }},
		{"explain", "Trace a rule or a record to its full reasoning: the record, the rules that enforce it, and anything it supersedes.", readOnly("Explain a rule or record"),
			func(srv *sdk.Server, tl *sdk.Tool) { sdk.AddTool(srv, tl, s.explain) }},
		{"review", "Grade a diff against the records that apply: check the changed files, then report concerns linters cannot see (new dependencies, new Dockerfiles, provider bumps, exported API changes, new packages without tests).", readOnly("Review a change"),
			func(srv *sdk.Server, tl *sdk.Tool) { sdk.AddTool(srv, tl, s.review) }},
		{"scaffold", "Generate a project or component from a profile's template. Defaults to dry_run, which lists files and records without writing. A real run never overwrites files and checks the result.", &sdk.ToolAnnotations{Title: "Scaffold from the canon", DestructiveHint: &f, IdempotentHint: false, OpenWorldHint: &f},
			func(srv *sdk.Server, tl *sdk.Tool) { sdk.AddTool(srv, tl, s.scaffold) }},
		{"profile", "Detect which profile and records apply to a repo, and which checkers are missing locally.", readOnly("Detect the repo profile"),
			func(srv *sdk.Server, tl *sdk.Tool) { sdk.AddTool(srv, tl, s.profile) }},
	}
	for _, tl := range tools {
		sc, err := schema(tl.name)
		if err != nil {
			return err
		}
		tl.add(srv, &sdk.Tool{Name: tl.name, Description: tl.desc, InputSchema: sc, Annotations: tl.notes})
	}
	return nil
}

// check runs the check tool.
func (s *Server) check(ctx context.Context, _ *sdk.CallToolRequest, in CheckInput) (*sdk.CallToolResult, *engine.CheckResult, error) {
	sev, err := findings.ParseSeverity(in.MinSeverity)
	if err != nil {
		return nil, nil, err
	}
	res, err := s.engine.Check(ctx, engine.CheckRequest{Root: s.root, Paths: in.Paths, Scope: in.Scope, MinSeverity: sev, ChangedOnly: in.ChangedOnly, Base: "origin/main"})
	if err != nil {
		return nil, nil, fmt.Errorf("check: %w", err)
	}
	return nil, res, nil
}

// decide runs the decide tool.
func (s *Server) decide(_ context.Context, _ *sdk.CallToolRequest, in DecideInput) (*sdk.CallToolResult, *engine.DecideResult, error) {
	res, err := s.engine.Decide(engine.DecideRequest{Question: in.Question, Scope: in.Scope, Limit: in.Limit})
	if err != nil {
		return nil, nil, fmt.Errorf("decide: %w", err)
	}
	return nil, res, nil
}

// explain runs the explain tool.
func (s *Server) explain(_ context.Context, _ *sdk.CallToolRequest, in ExplainInput) (*sdk.CallToolResult, *engine.ExplainResult, error) {
	res, err := s.engine.Explain(engine.ExplainRequest(in))
	if err != nil {
		return nil, nil, fmt.Errorf("explain: %w", err)
	}
	return nil, res, nil
}

// review runs the review tool.
func (s *Server) review(ctx context.Context, _ *sdk.CallToolRequest, in ReviewInput) (*sdk.CallToolResult, *engine.ReviewResult, error) {
	res, err := s.engine.Review(ctx, engine.ReviewRequest{Root: s.root, Base: in.Base, Diff: in.Diff})
	if err != nil {
		return nil, nil, fmt.Errorf("review: %w", err)
	}
	return nil, res, nil
}

// scaffold runs the scaffold tool.
func (s *Server) scaffold(ctx context.Context, _ *sdk.CallToolRequest, in ScaffoldInput) (*sdk.CallToolResult, *engine.ScaffoldResult, error) {
	dry := in.DryRun == nil || *in.DryRun
	res, err := s.engine.Scaffold(ctx, engine.ScaffoldRequest{Root: s.root, Kind: in.Kind, Name: in.Name, Dest: in.Dest, Vars: in.Vars, DryRun: dry})
	if err != nil {
		return nil, nil, fmt.Errorf("scaffold: %w", err)
	}
	return nil, res, nil
}

// profile runs the profile tool.
func (s *Server) profile(_ context.Context, _ *sdk.CallToolRequest, in ProfileInput) (*sdk.CallToolResult, *engine.ProfileResult, error) {
	p := in.Path
	if p == "" {
		p = "."
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(s.root, p)
	}
	res, err := s.engine.Profile(p)
	if err != nil {
		return nil, nil, fmt.Errorf("profile: %w", err)
	}
	return nil, res, nil
}

// addResources registers the canon resources.
func (s *Server) addResources(srv *sdk.Server) {
	srv.AddResource(&sdk.Resource{URI: "canon://records", Name: "records", Title: "Record index", Description: "Every record: id, title, status, strength, scope", MIMEType: "application/json"},
		func(_ context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			return jsonResource(req.Params.URI, s.engine.Index())
		})
	srv.AddResource(&sdk.Resource{URI: "canon://principles", Name: "principles", Title: "Principles", Description: "The core beliefs every record ladders up to", MIMEType: "text/markdown"},
		func(_ context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			return textResource(req.Params.URI, s.engine.Principles), nil
		})
	srv.AddResourceTemplate(&sdk.ResourceTemplate{URITemplate: "canon://records/{id}", Name: "record", Title: "Record", Description: "One record, full Markdown", MIMEType: "text/markdown"},
		func(_ context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			id := strings.TrimPrefix(req.Params.URI, "canon://records/")
			r, ok := s.engine.Records.Get(id)
			if !ok {
				return nil, sdk.ResourceNotFoundError(req.Params.URI)
			}
			return textResource(req.Params.URI, r.Markdown()), nil
		})
	srv.AddResourceTemplate(&sdk.ResourceTemplate{URITemplate: "canon://profiles/{name}", Name: "profile", Title: "Profile", Description: "A profile and its resolved record list", MIMEType: "application/json"},
		func(_ context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			name := strings.TrimPrefix(req.Params.URI, "canon://profiles/")
			v, err := s.engine.ProfileRecords(name)
			if errors.Is(err, engine.ErrNotFound) {
				return nil, sdk.ResourceNotFoundError(req.Params.URI)
			}
			if err != nil {
				return nil, err
			}
			return jsonResource(req.Params.URI, v)
		})
}

// textResource wraps Markdown as a resource result.
func textResource(uri, text string) *sdk.ReadResourceResult {
	return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: text}}}
}

// jsonResource encodes v as a JSON resource result.
func jsonResource(uri string, v any) (*sdk.ReadResourceResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", uri, err)
	}
	return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(b)}}}, nil
}

// ServeHTTP serves srv over streamable HTTP on addr until ctx is done.
func ServeHTTP(ctx context.Context, srv *sdk.Server, addr string) error {
	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv }, nil)
	hs := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- hs.ListenAndServe() }()
	select {
	case err := <-errc:
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := hs.Shutdown(shutdown); err != nil {
			return fmt.Errorf("shut down http: %w", err)
		}
		return nil
	}
}
