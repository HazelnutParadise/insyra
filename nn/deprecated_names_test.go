package nn

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// #265 (NN-1): one name for each thing.

// nnDocs maps the functions, types, constants and methods (as Type.Method) of
// one source file to their doc comments, with runs of white space folded to
// one space.
func nnDocs(t *testing.T, file string) map[string]string {
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
			name := d.Name.Name
			if d.Recv != nil {
				recv := d.Recv.List[0].Type
				if star, ok := recv.(*ast.StarExpr); ok {
					recv = star.X
				}
				ident, ok := recv.(*ast.Ident)
				if !ok {
					continue
				}
				name = ident.Name + "." + name
			}
			docs[name] = text(d.Doc)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					doc := s.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					for _, n := range s.Names {
						docs[n.Name] = text(doc)
					}
				case *ast.TypeSpec:
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

func TestNNNamesThatStayAreNotDeprecated(t *testing.T) {
	// The names that stay are not deprecated. The four activation and shape
	// layers keep their New prefix because the bare name is a kernel.
	kept := []struct{ file, name string }{
		{"layers.go", "Dense"}, {"layers.go", "ReLU"}, {"layers.go", "Dropout"}, {"layers.go", "Func"},
		{"layers.go", "NewSigmoid"}, {"layers.go", "NewTanh"}, {"layers.go", "NewGelu"}, {"layers.go", "NewFlatten"},
		{"layers_attention.go", "MultiHeadAttention"},
		{"layers_catalog.go", "Conv2D"}, {"layers_catalog.go", "MaxPool2D"}, {"layers_catalog.go", "AvgPool2D"},
		{"layers_catalog.go", "GlobalAvgPool"}, {"layers_catalog.go", "BatchNorm2D"},
		{"layers_catalog.go", "LayerNorm"}, {"layers_catalog.go", "Embedding"},
		{"fit.go", "CrossEntropy"}, {"fit.go", "MSE"}, {"fit.go", "BCEWithLogits"},
		{"protocol.go", "BoundClassifier"}, {"protocol.go", "BoundRegressor"},
		{"kernels.go", "PoolOptions"},
		{"tensor.go", "DType"}, {"tensor.go", "NewTensor"}, {"tensor.go", "Tensor.Float32Data"},
	}
	for _, k := range kept {
		doc, ok := nnDocs(t, k.file)[k.name]
		if !ok {
			t.Errorf("%s not found in %s", k.name, k.file)
			continue
		}
		if strings.Contains(doc, "Deprecated") {
			t.Errorf("%s is the name that stays, but its doc says %q", k.name, doc)
		}
	}
}
