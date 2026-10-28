package main

import (
	"strings"
	"testing"

	"github.com/rossedman/canon/tools/internal/gofiles"
)

func TestGenerated(t *testing.T) {
	got := generated([]gofiles.File{{Path: "./internal/api/types.gen.go", Generated: true}, {Path: "main.go"}})
	if len(got) != 2 || !strings.HasSuffix(got[0], "__no_generated_files__") || got[1] != module+"/internal/api/types.gen.go" {
		t.Fatalf("generated = %v", got)
	}
}
