package plot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// #413: a renamed chart keeps its former name for one release, marked
// Deprecated so tools and readers point at the new one.
func TestFormerChartNamesAreDeprecated(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "deprecated.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	check := func(name string, doc *ast.CommentGroup) {
		count++
		if doc == nil || !strings.Contains(doc.Text(), "Deprecated: use ") {
			t.Errorf("%s has no \"Deprecated: use ...\" paragraph", name)
		}
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			check(d.Name.Name, d.Doc)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					check(ts.Name.Name, d.Doc)
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("deprecated.go declares nothing")
	}
}
