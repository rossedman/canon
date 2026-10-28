// Command doccheck fails when a function or method has no doc comment, or a
// package has no package comment. Test and generated files are skipped.
package main

import (
	"fmt"
	"go/ast"
	"os"
	"path"
	"sort"

	"github.com/rossedman/canon/tools/internal/gofiles"
)

// main runs the check over the packages named on the command line.
func main() {
	problems, err := check(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		fmt.Printf("%d missing doc comments\n", len(problems))
		os.Exit(1)
	}
}

// check returns one line per missing doc comment.
func check(patterns []string) ([]string, error) {
	files, err := gofiles.Walk(patterns)
	if err != nil {
		return nil, fmt.Errorf("doccheck: %w", err)
	}
	var out []string
	pkgDoc := map[string]bool{}
	pkgFirst := map[string]string{}
	for _, f := range files {
		if f.Generated {
			continue
		}
		dir := path.Dir(f.Path)
		if f.AST.Doc != nil {
			pkgDoc[dir] = true
		}
		if _, ok := pkgFirst[dir]; !ok {
			pkgFirst[dir] = f.Path
		}
		out = append(out, funcsWithoutDoc(f)...)
	}
	for dir, first := range pkgFirst {
		if !pkgDoc[dir] {
			out = append(out, fmt.Sprintf("%s:1: package in %s has no package comment", first, dir))
		}
	}
	sort.Strings(out)
	return out, nil
}

// funcsWithoutDoc lists the functions and methods in f with no doc comment.
func funcsWithoutDoc(f gofiles.File) []string {
	var out []string
	for _, d := range f.AST.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Doc != nil {
			continue
		}
		pos := f.Fset.Position(fn.Pos())
		out = append(out, fmt.Sprintf("%s:%d: %s has no doc comment", f.Path, pos.Line, fn.Name.Name))
	}
	return out
}
