package repl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/engine/dsl"
)

// #260 (EN-2): the DSL session has one public name, in engine/dsl. The names it
// had here stay for one release, Deprecated, with their meaning.
func TestDeprecatedSessionNamesSayWhatReplacedThem(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "session.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	docs := map[string]string{}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			docs[d.Name.Name] = strings.Join(strings.Fields(d.Doc.Text()), " ")
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					docs[ts.Name.Name] = strings.Join(strings.Fields(d.Doc.Text()), " ")
				}
			}
		}
	}
	want := map[string]string{
		"DSLSession":    "Deprecated: use Session from engine/dsl instead",
		"NewDSLSession": "Deprecated: use NewSession from engine/dsl instead",
	}
	for name, notice := range want {
		if !strings.Contains(docs[name], notice) {
			t.Errorf("%s doc %q does not contain %q", name, docs[name], notice)
		}
	}
}

func TestDeprecatedNewDSLSessionKeepsItsMeaning(t *testing.T) {
	if _, err := NewDSLSession(nil, "default", nil); err == nil {
		t.Error("NewDSLSession accepted a nil manager")
	}
	session, err := NewDSLSession(dsl.NewManager(t.TempDir(), ""), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatal(err)
	}
	if session.Context().EnvName != "default" {
		t.Errorf("EnvName = %q, want default", session.Context().EnvName)
	}
}
