package ml

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// #264 (ML-2): one name for each thing.

// mlDocs maps the functions and types of one source file to their doc
// comments, with runs of white space folded to one space.
func mlDocs(t *testing.T, file string) map[string]string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	text := func(g *ast.CommentGroup) string { return strings.Join(strings.Fields(g.Text()), " ") }
	docs := make(map[string]string)
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				docs[d.Name.Name] = text(d.Doc)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if s, ok := spec.(*ast.TypeSpec); ok {
					doc := s.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					docs[s.Name.Name] = text(doc)
				}
			}
		}
	}
	return docs
}

func TestMLNamesThatStayAreNotDeprecated(t *testing.T) {
	for _, kept := range []struct{ file, name string }{
		{"onnx_export.go", "ExportONNX"},
		{"decision_tree.go", "DecisionTreeOptions"},
	} {
		if doc := mlDocs(t, kept.file)[kept.name]; doc == "" || strings.Contains(doc, "Deprecated") {
			t.Errorf("%s is the name that stays, but its doc is %q", kept.name, doc)
		}
	}
}

func TestExportONNXTakesAModel(t *testing.T) {
	got := reflect.TypeOf(ExportONNX).In(1)
	if want := reflect.TypeOf((*Model)(nil)).Elem(); got != want {
		t.Fatalf("ExportONNX takes a %v, want %v", got, want)
	}
}

func TestExportONNXRefusesANilModel(t *testing.T) {
	var buffer bytes.Buffer
	if err := ExportONNX(&buffer, nil); err == nil || buffer.Len() != 0 {
		t.Errorf("ExportONNX(nil): err = %v, wrote %d bytes; want an error and nothing written", err, buffer.Len())
	}
}
