package checkers

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
)

// checkOneCommand reports a cmd/*.go file that does not define exactly one
// cobra.Command.
func checkOneCommand(_ fs.FS, name string, data []byte) []Issue {
	if strings.HasSuffix(name, "_test.go") {
		return nil
	}
	marker := []byte("&cobra.Command{")
	n := bytes.Count(data, marker)
	if n == 1 {
		return nil
	}
	line := 1
	if n > 1 {
		first := bytes.Index(data, marker)
		second := first + len(marker) + bytes.Index(data[first+len(marker):], marker)
		line = 1 + bytes.Count(data[:second], []byte("\n"))
	}
	return []Issue{{Line: line, Message: fmt.Sprintf("%s defines %d cobra commands; want exactly 1", name, n)}}
}

// checkDocComments reports functions and methods without a doc comment, and
// packages without a package comment. Test and generated files are skipped.
func checkDocComments(fsys fs.FS, name string, data []byte) []Issue {
	if strings.HasSuffix(name, "_test.go") {
		return nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, data, parser.ParseComments)
	if err != nil || ast.IsGenerated(f) {
		return nil
	}
	var out []Issue
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Doc != nil {
			continue
		}
		pos := fset.Position(fn.Pos())
		out = append(out, Issue{Line: pos.Line, Column: pos.Column, Message: funcLabel(fn) + " has no doc comment"})
	}
	if f.Doc == nil && !packageHasDoc(fsys, path.Dir(name), name) {
		out = append(out, Issue{Line: fset.Position(f.Package).Line, Message: "package " + f.Name.Name + " has no package comment"})
	}
	return out
}

// funcLabel names a function or method the way a reader would: "func Load"
// or "method (*Set).Get".
func funcLabel(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return "func " + fn.Name.Name
	}
	return fmt.Sprintf("method (%s).%s", typeString(fn.Recv.List[0].Type), fn.Name.Name)
}

// typeString renders a receiver type expression.
func typeString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return typeString(t.X)
	case *ast.IndexListExpr:
		return typeString(t.X)
	}
	return "?"
}

// packageHasDoc reports whether another non-test file in dir carries the
// package comment.
func packageHasDoc(fsys fs.FS, dir, except string) bool {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		p := path.Join(dir, e.Name())
		if e.IsDir() || p == except || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			continue
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, data, parser.ParseComments|parser.PackageClauseOnly)
		if err == nil && f.Doc != nil {
			return true
		}
	}
	return false
}

// bannedUX lists terminal UX imports replaced by the Charmbracelet libraries.
var bannedUX = []string{
	"github.com/fatih/color",
	"github.com/AlecAivazis/survey",
	"github.com/manifoldco/promptui",
	"github.com/rivo/tview",
	"github.com/gookit/color",
	"github.com/briandowns/spinner",
	"github.com/schollz/progressbar",
}

// checkUXLibraries reports imports of terminal libraries the canon replaces.
func checkUXLibraries(_ fs.FS, name string, data []byte) []Issue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, data, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	var out []Issue
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if slices.ContainsFunc(bannedUX, func(b string) bool { return p == b || strings.HasPrefix(p, b+"/") }) {
			out = append(out, Issue{
				Line:    fset.Position(imp.Pos()).Line,
				Message: "imports " + p + "; use the Charmbracelet libraries (lipgloss, huh, bubbletea)",
			})
		}
	}
	return out
}
