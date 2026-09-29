package ml

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// #264 (ML-2): one name for each thing. A replaced name stays for one
// release, marked Deprecated, with its old meaning.

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

func TestDeprecatedMLNamesSayWhatReplacedThem(t *testing.T) {
	const removal = "Removed in the release after the one that deprecated it."
	deprecated := []struct{ file, name, notice string }{
		{"onnx_export.go", "WriteONNX", "Deprecated: use ExportONNX"},
		{"decision_tree.go", "DecisionTreeClassifierOptions", "Deprecated: use DecisionTreeOptions"},
		{"decision_tree.go", "DecisionTreeRegressorOptions", "Deprecated: use DecisionTreeOptions"},
	}
	for _, d := range deprecated {
		doc, ok := mlDocs(t, d.file)[d.name]
		if !ok {
			t.Errorf("%s not found in %s", d.name, d.file)
			continue
		}
		if !strings.Contains(doc, d.notice) || !strings.Contains(doc, removal) {
			t.Errorf("%s doc is %q; want it to contain %q and %q", d.name, doc, d.notice, removal)
		}
	}
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

func TestWriteONNXKeepsItsMeaning(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("x"))
	y := insyra.NewDataList(2.0, 4.1, 5.9, 8.2)
	model, err := FitLinearRegression(x, y)
	if err != nil {
		t.Fatal(err)
	}
	var exported, written bytes.Buffer
	if err := ExportONNX(&exported, model); err != nil {
		t.Fatal(err)
	}
	if err := WriteONNX(&written, model); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(exported.Bytes(), written.Bytes()) {
		t.Error("WriteONNX wrote different bytes than ExportONNX")
	}
	for name, value := range map[string]any{"a string": "model", "nil": nil} {
		var buffer bytes.Buffer
		if err := WriteONNX(&buffer, value); err == nil || buffer.Len() != 0 {
			t.Errorf("WriteONNX(%s): err = %v, wrote %d bytes; want an error and nothing written", name, err, buffer.Len())
		}
	}
	var buffer bytes.Buffer
	if err := ExportONNX(&buffer, nil); err == nil || buffer.Len() != 0 {
		t.Errorf("ExportONNX(nil): err = %v, wrote %d bytes; want an error and nothing written", err, buffer.Len())
	}
}

func TestDeprecatedTreeOptionsAreTheSameType(t *testing.T) {
	x := insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0).SetName("a"),
		insyra.NewDataList(8.0, 1.0, 7.0, 2.0, 6.0, 3.0, 5.0, 4.0).SetName("b"),
	)
	labels := insyra.NewDataList("n", "n", "n", "y", "y", "n", "y", "y")
	targets := insyra.NewDataList(1.0, 1.5, 2.0, 4.0, 4.5, 2.5, 5.0, 5.5)

	legacyClassifier, err := FitDecisionTreeClassifier(x, labels, DecisionTreeClassifierOptions{MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	classifier, err := FitDecisionTreeClassifier(x, labels, DecisionTreeOptions{MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(legacyClassifier.LeafValues(), classifier.LeafValues()) {
		t.Errorf("classifier leaves %v, want %v", legacyClassifier.LeafValues(), classifier.LeafValues())
	}

	legacyRegressor, err := FitDecisionTreeRegressor(x, targets, DecisionTreeRegressorOptions{MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	regressor, err := FitDecisionTreeRegressor(x, targets, DecisionTreeOptions{MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(legacyRegressor.LeafValues(), regressor.LeafValues()) {
		t.Errorf("regressor leaves %v, want %v", legacyRegressor.LeafValues(), regressor.LeafValues())
	}
}
