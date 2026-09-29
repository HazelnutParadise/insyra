package nn

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"slices"
	"strings"
	"testing"
)

// #265 (NN-1): one name for each thing. A replaced name stays for one release,
// marked Deprecated, with its old meaning.

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

func TestDeprecatedNNNamesSayWhatReplacedThem(t *testing.T) {
	const removal = "Removed in the release after the one that deprecated it."
	deprecated := []struct{ file, name, notice string }{
		{"layers.go", "NewDense", "Deprecated: use Dense"},
		{"layers.go", "NewReLU", "Deprecated: use ReLU"},
		{"layers.go", "NewDropout", "Deprecated: use Dropout"},
		{"layers.go", "NewFunc", "Deprecated: use Func"},
		{"layers_attention.go", "NewMultiHeadAttention", "Deprecated: use MultiHeadAttention"},
		{"layers_catalog.go", "NewConv2D", "Deprecated: use Conv2D"},
		{"layers_catalog.go", "NewMaxPool2D", "Deprecated: use MaxPool2D"},
		{"layers_catalog.go", "NewAvgPool2D", "Deprecated: use AvgPool2D"},
		{"layers_catalog.go", "NewGlobalAvgPool", "Deprecated: use GlobalAvgPool"},
		{"layers_catalog.go", "NewBatchNorm2D", "Deprecated: use BatchNorm2D"},
		{"layers_catalog.go", "NewLayerNorm", "Deprecated: use LayerNorm"},
		{"layers_catalog.go", "NewEmbedding", "Deprecated: use Embedding"},
		{"fit.go", "SoftmaxCrossEntropy", "Deprecated: use CrossEntropy"},
		{"fit.go", "MSELoss", "Deprecated: use MSE"},
		{"fit.go", "BCEWithLogitsLoss", "Deprecated: use BCEWithLogits"},
		{"protocol.go", "Classifier", "Deprecated: use BoundClassifier"},
		{"protocol.go", "Regressor", "Deprecated: use BoundRegressor"},
		{"kernels.go", "MaxPoolOptions", "Deprecated: use PoolOptions"},
		{"kernels.go", "AveragePoolOptions", "Deprecated: use PoolOptions"},
		{"tensor.go", "DataType", "Deprecated: use DType"},
		{"tensor.go", "Float32", "Deprecated: use DTypeFloat32"},
		{"tensor.go", "Float16", "Deprecated: use DTypeFloat16"},
		{"tensor.go", "Float64", "Deprecated: use DTypeFloat64"},
		{"tensor.go", "NewFloat32Tensor", "Deprecated: use NewTensor"},
		{"tensor.go", "NewTensorWithDType", "Deprecated: use NewTensor"},
		{"tensor.go", "Tensor.Data", "Deprecated: use Float32Data"},
	}
	for _, d := range deprecated {
		doc, ok := nnDocs(t, d.file)[d.name]
		if !ok {
			t.Errorf("%s not found in %s", d.name, d.file)
			continue
		}
		if !strings.Contains(doc, d.notice) || !strings.Contains(doc, removal) {
			t.Errorf("%s doc is %q; want it to contain %q and %q", d.name, doc, d.notice, removal)
		}
	}

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

// buildOne builds a one-layer Sequential on a tape seeded with 11 and returns
// its parameters by name, as kind and shape and value bits.
func buildOne(t *testing.T, layer Layer) (string, map[string]string) {
	t.Helper()
	model, err := NewSequential(NewTape(11), layer)
	if err != nil {
		t.Fatalf("NewSequential: %v", err)
	}
	parameters := make(map[string]string)
	for name, parameter := range model.NamedParameters() {
		value := parameter.Value()
		bits := make([]uint32, len(value.data))
		for i, v := range value.data {
			bits[i] = math.Float32bits(v)
		}
		parameters[name] = fmt.Sprint(value.shape, bits)
	}
	return sequentialLayerKind(layer), parameters
}

func TestDeprecatedLayerTwinsBuildTheSameLayers(t *testing.T) {
	identity := func(_ *Tape, x *Tensor) (*Tensor, error) { return x, nil }
	pairs := []struct {
		name        string
		old, stayed Layer
	}{
		{"Dense", NewDense(3, 4), Dense(3, 4)},
		{"ReLU", NewReLU(), ReLU()},
		{"Dropout", NewDropout(0.25), Dropout(0.25)},
		{"Func", NewFunc(identity), Func(identity)},
		{"MultiHeadAttention", NewMultiHeadAttention(4, 2), MultiHeadAttention(4, 2)},
		{"Conv2D", NewConv2D(1, 2, 3), Conv2D(1, 2, 3)},
		{"MaxPool2D", NewMaxPool2D(2), MaxPool2D(2)},
		{"AvgPool2D", NewAvgPool2D(2), AvgPool2D(2)},
		{"GlobalAvgPool", NewGlobalAvgPool(), GlobalAvgPool()},
		{"BatchNorm2D", NewBatchNorm2D(2), BatchNorm2D(2)},
		{"LayerNorm", NewLayerNorm(4), LayerNorm(4)},
		{"Embedding", NewEmbedding(5, 3), Embedding(5, 3)},
	}
	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			oldKind, oldParameters := buildOne(t, p.old)
			kind, parameters := buildOne(t, p.stayed)
			if oldKind != kind {
				t.Errorf("deprecated constructor builds a %s, want %s", oldKind, kind)
			}
			if fmt.Sprint(oldParameters) != fmt.Sprint(parameters) {
				t.Errorf("parameters differ:\ndeprecated %v\nstayed     %v", oldParameters, parameters)
			}
		})
	}
}

func TestDeprecatedLossAliasesTrainTheSameWay(t *testing.T) {
	x, err := NewTensor([]int{4, 2}, []float32{0, 1, 1, 0, 1, 1, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	regression, err := NewTensor([]int{4, 1}, []float32{1, 1, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	binary, err := NewTensor([]int{4, 1}, []float32{1, 1, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	labels, err := NewInt64Tensor([]int{4}, []int64{1, 1, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	losses := func(outputs int, y *Tensor, loss LossSpec) []float64 {
		t.Helper()
		model, err := NewSequential(NewTape(5), Dense(2, outputs))
		if err != nil {
			t.Fatal(err)
		}
		result, err := model.Fit(x, y, FitConfig{Epochs: 3, BatchSize: 2, Seed: 5, Optimizer: SGD{Rate: 0.1}, Loss: loss, Quiet: true})
		if err != nil {
			t.Fatalf("%T: %v", loss, err)
		}
		return result.TrainLosses
	}
	pairs := []struct {
		name        string
		outputs     int
		y           *Tensor
		old, stayed LossSpec
	}{
		{"CrossEntropy", 2, labels, SoftmaxCrossEntropy{}, CrossEntropy{}},
		{"MSE", 1, regression, MSELoss{}, MSE{}},
		{"BCEWithLogits", 1, binary, BCEWithLogitsLoss{}, BCEWithLogits{}},
	}
	for _, p := range pairs {
		if old, stayed := losses(p.outputs, p.y, p.old), losses(p.outputs, p.y, p.stayed); !slices.Equal(old, stayed) {
			t.Errorf("%s: deprecated alias losses %v, want %v", p.name, old, stayed)
		}
	}
}

func TestDeprecatedTensorNamesKeepTheirMeaning(t *testing.T) {
	data := []float32{1, 2, 3, 4, 5, 6}
	want, err := NewTensor([]int{2, 3}, data)
	if err != nil {
		t.Fatal(err)
	}
	viaFloat32, err := NewFloat32Tensor([]int{2, 3}, data)
	if err != nil {
		t.Fatal(err)
	}
	viaDType, err := NewTensorWithDType(DTypeFloat32, []int{2, 3}, data)
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]*Tensor{"NewFloat32Tensor": viaFloat32, "NewTensorWithDType": viaDType} {
		if got.DType() != want.DType() || !slices.Equal(got.Shape(), want.Shape()) || !slices.Equal(got.data, want.data) {
			t.Errorf("%s built %v %v %v, want %v %v %v", name, got.DType(), got.Shape(), got.data, want.DType(), want.Shape(), want.data)
		}
	}
	if _, err := NewTensorWithDType(DTypeInt64, []int{2}, []float32{1, 2}); err == nil {
		t.Error("NewTensorWithDType accepted DTypeInt64")
	}

	ints, err := NewInt64Tensor([]int{2}, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if got := ints.Data(); got != nil {
		t.Errorf("deprecated Data() on an int64 tensor = %v, want nil", got)
	}
	if _, err := ints.Float32Data(); err == nil || !strings.Contains(err.Error(), "int64") {
		t.Errorf("Float32Data() on an int64 tensor: err = %v, want one naming int64", err)
	}
	var missing *Tensor
	if got := missing.Data(); got != nil {
		t.Errorf("deprecated Data() on a nil tensor = %v, want nil", got)
	}
	if _, err := missing.Float32Data(); err == nil {
		t.Error("Float32Data() on a nil tensor returned no error")
	}

	if Float32 != DTypeFloat32 || Float16 != DTypeFloat16 || Float64 != DTypeFloat64 {
		t.Errorf("deprecated dtype constants %q %q %q changed value", Float32, Float16, Float64)
	}
	dtypes := []DataType{DTypeInt64}
	if dtypes[0] != DTypeInt64 {
		t.Error("DataType is no longer DType")
	}
	bound := []*Classifier{(*BoundClassifier)(nil)}
	regressors := []*Regressor{(*BoundRegressor)(nil)}
	if len(bound) != 1 || len(regressors) != 1 {
		t.Error("the deprecated binding aliases are no longer the bound types")
	}

	input, err := NewTensor([]int{1, 1, 2, 2}, []float32{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	maxOld, err := MaxPool(input, []int{2, 2}, MaxPoolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	maxStayed, err := MaxPool(input, []int{2, 2}, PoolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	averageOld, err := AveragePool(input, []int{2, 2}, AveragePoolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	averageStayed, err := AveragePool(input, []int{2, 2}, PoolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(maxOld.data, maxStayed.data) || !slices.Equal(averageOld.data, averageStayed.data) {
		t.Errorf("deprecated pool options changed a result: max %v vs %v, average %v vs %v", maxOld.data, maxStayed.data, averageOld.data, averageStayed.data)
	}
}
