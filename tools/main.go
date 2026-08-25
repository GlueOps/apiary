package main

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// directives are compiler instructions that merely look like comments.
// Stripping them changes what the program DOES, silently: losing //go:embed
// left internal/migrations with empty filesystems, so `migrate up` applied no
// migrations at all and still exited zero. They are parsed back in and kept.
func isDirective(text string) bool {
	return strings.HasPrefix(text, "//go:") ||
		strings.HasPrefix(text, "//line ") ||
		strings.HasPrefix(text, "/*line ") ||
		strings.HasPrefix(text, "//export ") ||
		// The pre-1.17 build constraint. //go:build is covered by the //go:
		// prefix above, but a file carrying only the legacy form and no
		// //go:build line would, once stripped, compile on EVERY platform
		// instead of the one it names. Filename suffixes (_linux.go) are not
		// comments and survive regardless, so the exposure is limited to files
		// that use the comment form alone — but it fails silently, and the
		// worst case is a _linux and _windows pair both compiling at once.
		strings.HasPrefix(text, "// +build") ||
		// golangci-lint suppressions. Not a compiler directive, but stripping
		// one silently reinstates a lint failure someone deliberately waived,
		// and this tool is meant to be run repeatedly.
		strings.HasPrefix(text, "//nolint")
}

func stripComments(path string) error {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	kept := make([]*ast.CommentGroup, 0, len(f.Comments))
	for _, group := range f.Comments {
		lines := make([]*ast.Comment, 0, len(group.List))
		for _, c := range group.List {
			if isDirective(c.Text) {
				lines = append(lines, c)
			}
		}
		if len(lines) > 0 {
			kept = append(kept, &ast.CommentGroup{List: lines})
		}
	}
	f.Comments = kept

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return printer.Fprint(out, fset, f)
}

func main() {
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Ext(path) == ".go" {
			return stripComments(path)
		}
		return err
	})
}
