package insyra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestLookAlikeWindowMethodsAreDeprecated checks that Difference, MovingAverage,
// MovingStdev and WeightedMovingAverage carry a Deprecated notice naming their
// replacement and removal, and that ExponentialSmoothing does not.
func TestLookAlikeWindowMethodsAreDeprecated(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "datalist.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse datalist.go: %v", err)
	}

	docs := map[string]string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if ident, ok := star.X.(*ast.Ident); !ok || ident.Name != "DataList" {
			continue
		}
		docs[fn.Name.Name] = strings.Join(strings.Fields(fn.Doc.Text()), " ")
	}

	const removal = "Removed in the release after the one that deprecated it."
	deprecated := []struct{ name, notice string }{
		{"Difference", "Deprecated: use Diff(1)"},
		{"MovingAverage", "Deprecated: use Rolling(RollingOptions{Window: windowSize}).Mean()"},
		{"MovingStdev", "Deprecated: use Rolling(RollingOptions{Window: windowSize}).Std()"},
		{"WeightedMovingAverage", "Deprecated: use Rolling(RollingOptions{Window: windowSize, Weights: weights}).Mean()"},
	}
	for _, m := range deprecated {
		doc, found := docs[m.name]
		if !found {
			t.Fatalf("(*DataList).%s not found in datalist.go", m.name)
		}
		if !strings.Contains(doc, m.notice) {
			t.Errorf("(*DataList).%s doc lacks %q; doc is %q", m.name, m.notice, doc)
		}
		if !strings.Contains(doc, removal) {
			t.Errorf("(*DataList).%s doc lacks %q; doc is %q", m.name, removal, doc)
		}
	}

	doc, found := docs["ExponentialSmoothing"]
	if !found {
		t.Fatalf("(*DataList).ExponentialSmoothing not found in datalist.go")
	}
	if strings.Contains(doc, "Deprecated:") {
		t.Errorf("(*DataList).ExponentialSmoothing must not be deprecated; doc is %q", doc)
	}
}

// TestLookAlikeWindowMethodsAreDeprecatedOnIDataList checks that the same four
// methods are marked Deprecated in the IDataList interface, so a call through
// the interface is flagged too, and that ExponentialSmoothing is not.
func TestLookAlikeWindowMethodsAreDeprecatedOnIDataList(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "interfaces.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse interfaces.go: %v", err)
	}

	var iface *ast.InterfaceType
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "IDataList" {
			return true
		}
		iface, _ = spec.Type.(*ast.InterfaceType)
		return false
	})
	if iface == nil {
		t.Fatalf("type IDataList interface not found in interfaces.go")
	}

	fields := map[string]*ast.Field{}
	for _, field := range iface.Methods.List {
		for _, name := range field.Names {
			fields[name.Name] = field
		}
	}

	for _, name := range []string{"Difference", "MovingAverage", "MovingStdev", "WeightedMovingAverage"} {
		field, found := fields[name]
		if !found {
			t.Fatalf("IDataList.%s not found in interfaces.go", name)
		}
		if field.Doc == nil {
			t.Errorf("IDataList.%s has no doc comment; want one containing %q", name, "Deprecated: use")
			continue
		}
		if doc := field.Doc.Text(); !strings.Contains(doc, "Deprecated: use") {
			t.Errorf("IDataList.%s doc lacks %q; doc is %q", name, "Deprecated: use", doc)
		}
	}

	field, found := fields["ExponentialSmoothing"]
	if !found {
		t.Fatalf("IDataList.ExponentialSmoothing not found in interfaces.go")
	}
	if field.Doc != nil && strings.Contains(field.Doc.Text(), "Deprecated:") {
		t.Errorf("IDataList.ExponentialSmoothing must not be deprecated; doc is %q", field.Doc.Text())
	}
}
