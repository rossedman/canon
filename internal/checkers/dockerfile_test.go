package checkers

import (
	"strings"
	"testing"
)

// multiStage builds on an alias that sets a non-root user in an earlier stage.
const multiStage = `# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27@sha256:abc AS build
RUN go build \
    -o /app .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:def AS base
user 65532

from base
COPY --from=build /app /app
`

func TestParseDockerfile(t *testing.T) {
	t.Parallel()
	st := ParseDockerfile([]byte(multiStage))
	if len(st) != 3 {
		t.Fatalf("stages = %+v", st)
	}
	if st[0].Image != "golang:1.27@sha256:abc" || st[0].Name != "build" || st[0].Line != 2 {
		t.Fatalf("stage 0 = %+v", st[0])
	}
	if st[1].Users[0].Value != "65532" || st[1].Users[0].Line != 7 || st[2].Image != "base" {
		t.Fatalf("stages = %+v", st)
	}
	img, users, ok := FinalBase(st)
	if !ok || !strings.HasPrefix(img, "gcr.io/distroless/") || len(users) != 1 {
		t.Fatalf("FinalBase = %q %v %v", img, users, ok)
	}
	if _, _, ok := FinalBase(nil); ok {
		t.Fatal("empty Dockerfile has no base")
	}
	trailing := ParseDockerfile([]byte("FROM a \\\n  AS b \\"))
	if len(trailing) != 1 || trailing[0].Name != "b" {
		t.Fatalf("unterminated continuation = %+v", trailing)
	}
	if st := ParseDockerfile([]byte("USER root\nFROM\n\n")); len(st) != 1 || st[0].Image != "" {
		t.Fatalf("degenerate = %+v", st)
	}
	if st := ParseDockerfile([]byte("FROM a\nRUN x \\\n# comment inside\n  y\nUSER\n")); len(st) != 1 || len(st[0].Users) != 0 {
		t.Fatalf("comment inside continuation = %+v", st)
	}
}

func TestDockerfileChecks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dockerfile string
		base, dig  int
		user       string
	}{
		{"good multi-stage", multiStage, 0, 0, ""},
		{"full OS runtime", "FROM golang:1.27 AS build\nFROM debian:bookworm\n", 1, 2, "sets no USER"},
		{"explicit root", "FROM scratch\nUSER root:root\n", 0, 0, "runs as root (USER root:root)"},
		{"uid zero after non-root", "FROM cgr.dev/chainguard/static@sha256:x\nUSER 1000\nUSER 0\n", 0, 0, "USER 0"},
		{"chainguard default", "FROM cgr.dev/chainguard/static@sha256:x\n", 0, 0, ""},
		{"variable base", "ARG BASE\nFROM ${BASE}\n", 0, 0, ""},
		{"scratch no user", "FROM scratch\nCOPY a /a\n", 0, 0, "sets no USER"},
		{"empty", "", 0, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := []byte(tt.dockerfile)
			if got := len(checkFinalStageBase(nil, "Dockerfile", data)); got != tt.base {
				t.Errorf("final-stage-base issues = %d, want %d", got, tt.base)
			}
			if got := len(checkBaseDigest(nil, "Dockerfile", data)); got != tt.dig {
				t.Errorf("base-digest issues = %d, want %d", got, tt.dig)
			}
			user := checkFinalStageUser(nil, "Dockerfile", data)
			if tt.user == "" && len(user) > 0 || tt.user != "" && (len(user) != 1 || !strings.Contains(user[0].Message, tt.user)) {
				t.Errorf("final-stage-user = %+v, want %q", user, tt.user)
			}
		})
	}
	is := checkFinalStageBase(nil, "Dockerfile", []byte("FROM golang:1.27 AS build\nFROM debian:bookworm\n"))
	if is[0].Message != "Final stage uses debian:bookworm" || is[0].Line != 2 || is[0].Fix == nil {
		t.Fatalf("finding = %+v", is[0])
	}
}

func BenchmarkParseDockerfile(b *testing.B) {
	data := []byte(strings.Repeat(multiStage, 20))
	b.ReportAllocs()
	for b.Loop() {
		ParseDockerfile(data)
	}
}

// FuzzParseDockerfile checks the parser never panics, line numbers are
// positive and increasing, and the final-stage checks agree with it.
func FuzzParseDockerfile(f *testing.F) {
	f.Add([]byte(multiStage))
	f.Add([]byte("FROM a AS b\nFROM b AS a\nFROM a\n"))
	f.Add([]byte("FROM x \\\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		stages := ParseDockerfile(data)
		last := 0
		for _, st := range stages {
			if st.Line <= last {
				t.Fatalf("stage lines not increasing: %+v", stages)
			}
			last = st.Line
		}
		_, _, ok := FinalBase(stages)
		if ok != (len(stages) > 0) {
			t.Fatal("FinalBase disagrees with stage count")
		}
		checkFinalStageBase(nil, "Dockerfile", data)
		checkBaseDigest(nil, "Dockerfile", data)
		checkFinalStageUser(nil, "Dockerfile", data)
	})
}
