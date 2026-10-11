package ccl_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/engine/ccl"
)

// #259 (EN-1, CCL-29): engine/ccl reached the scalar and aggregate registries
// but not the sequence one, so a whole-column function could not be added from
// outside the module.
func TestRegisterSequenceFunctionReachesTheEvaluator(t *testing.T) {
	fn := ccl.SeqFunc(func(args ...[]any) ([]any, error) {
		col := args[0]
		out := make([]any, len(col))
		for i := range col {
			out[i] = col[len(col)-1-i]
		}
		return out, nil
	})
	ccl.RegisterSequenceFunction("ENGINETESTREVERSE", fn)

	ctx := mapContext(t)
	node, err := ccl.CompileExpression("ENGINETESTREVERSE(A)")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}
	bound, err := ccl.Bind(node, map[string]int{"A": 0, "B": 1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	// A sequence function runs once over the whole column, not once per row, so
	// Evaluate hands back the column; a DataTable gives each row its own cell.
	if ccl.IsRowDependent(bound) {
		t.Error("a registered sequence function is row dependent")
	}
	got, err := ccl.Evaluate(bound, ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if want := []any{3, 2, 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("ENGINETESTREVERSE(A): got %v (%T), want %v", got, got, want)
	}
}

// engineCCLDocs maps the functions of ccl.go to their doc comments, with runs of
// white space folded to one space.
func engineCCLDocs(t *testing.T) map[string]string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "ccl.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse ccl.go: %v", err)
	}
	docs := map[string]string{}
	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok && d.Recv == nil {
			docs[d.Name.Name] = strings.Join(strings.Fields(d.Doc.Text()), " ")
		}
	}
	return docs
}

// The two resets have done nothing since the depth counters moved onto the call
// stack, so they say so where go doc and an editor show it.
func TestNoOpResetsAreDeprecated(t *testing.T) {
	docs := engineCCLDocs(t)
	for _, name := range []string{"ResetEvalDepth", "ResetFuncCallDepth"} {
		doc, ok := docs[name]
		if !ok {
			t.Errorf("%s is missing from ccl.go", name)
			continue
		}
		if !strings.Contains(doc, "Deprecated: it does nothing") {
			t.Errorf("%s doc %q has no Deprecated notice saying it does nothing", name, doc)
		}
	}
}

// Every registration function says that the registry is shared by the whole
// process, safe to use from several goroutines, and keyed by a case-folded
// name a later registration replaces.
func TestRegisterFunctionsDocumentTheRegistry(t *testing.T) {
	docs := engineCCLDocs(t)
	for _, name := range []string{"RegisterFunction", "RegisterAggregateFunction", "RegisterSequenceFunction"} {
		doc := docs[name]
		for _, want := range []string{"every goroutine", "any letter case", "replaces"} {
			if !strings.Contains(doc, want) {
				t.Errorf("%s doc %q does not say %q", name, doc, want)
			}
		}
	}
}
