package insyra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// rootDocComments returns the doc comment of each function and method declared
// in the named files, keyed by name, on one line.
func rootDocComments(t *testing.T, files ...string) map[string]string {
	t.Helper()
	docs := make(map[string]string)
	fset := token.NewFileSet()
	for _, name := range files {
		file, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
				docs[fn.Name.Name] = strings.Join(strings.Fields(fn.Doc.Text()), " ")
			}
		}
	}
	return docs
}
