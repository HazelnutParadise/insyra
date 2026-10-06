package commands

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// engine/dsl is where a program registers its own command, so the names here
// that do the same are Deprecated for one release under the one-name rule of
// #211. The rest of this package has no counterpart there and stays.
func TestNamesEngineDSLNowProvidesAreDeprecated(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "commands.go", nil, parser.ParseComments)
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
				switch s := spec.(type) {
				case *ast.TypeSpec:
					docs[s.Name.Name] = strings.Join(strings.Fields(d.Doc.Text()), " ")
				case *ast.ValueSpec:
					for _, n := range s.Names {
						docs[n.Name] = strings.Join(strings.Fields(d.Doc.Text()), " ")
					}
				}
			}
		}
	}
	for _, name := range []string{"ExecContext", "CommandHandler", "CommandFlag", "ArgLimit", "Register", "MaxArgs", "FormArgs", "FormArgsAt", "OpenArgs"} {
		if want := "Deprecated: use " + name + " from engine/dsl instead"; !strings.Contains(docs[name], want) {
			t.Errorf("%s doc %q does not contain %q", name, docs[name], want)
		}
	}
	for _, name := range []string{"Registry", "Dispatch", "LookupCommand", "SnapshotRegistry", "DBConn", "SanitizeHistoryLine", "CloseAllDBConns", "SaveEnvState"} {
		if strings.Contains(docs[name], "Deprecated:") {
			t.Errorf("%s is deprecated, but engine/dsl has no counterpart for it", name)
		}
	}
}
