// Package gofiles walks Go packages for the repo check tools, so doccheck
// and generated agree on which files exist and which are generated.
package gofiles

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// File is one parsed Go source file.
type File struct {
	// Path is the slash path relative to the walk root.
	Path string
	// Generated reports a "// Code generated ... DO NOT EDIT." header.
	Generated bool
	AST       *ast.File
	Fset      *token.FileSet
}

// skip lists directories never walked.
var skip = map[string]bool{".git": true, "vendor": true, "testdata": true, "node_modules": true, "site": true, "dist": true}

// Walk parses every non-test Go file under the given patterns. A pattern is a
// directory, optionally ending in /... to include subdirectories.
func Walk(patterns []string) ([]File, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	var out []File
	fset := token.NewFileSet()
	for _, pat := range patterns {
		dir, recursive := strings.CutSuffix(pat, "/...")
		if dir == "" {
			dir = "."
		}
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if p != dir && (skip[d.Name()] || !recursive) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parse %s: %w", p, err)
			}
			out = append(out, File{Path: filepath.ToSlash(p), Generated: ast.IsGenerated(f), AST: f, Fset: fset})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", pat, err)
		}
	}
	return out, nil
}
