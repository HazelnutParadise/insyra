package nn_test

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/nn"
)

// What insyra writes, insyra reads: every model family ml exports must load
// and run in nn, reproducing the fitted model's own predictions. This half of
// the closure is pure Go and runs everywhere — no reference toolchain gate —
// which is the point: the loop closes even on a machine with nothing but this
// repository. ml's own export tests already pin these same files against
// onnxruntime, and nn's parity harness pins every kernel against it, so the
// external reference holds transitively while this test holds directly.
func TestMLExportsReadBackAndReproduceTheirOwnPredictions(t *testing.T) {
	features, regressionTarget, classificationTarget := roundtripFixture()

	regressors := map[string]func() (ml.Model, error){
		"linear": func() (ml.Model, error) { return ml.FitLinearRegression(features, regressionTarget) },
		"ridge":  func() (ml.Model, error) { return ml.FitRidgeRegression(features, regressionTarget, 0.7) },
		"lasso":  func() (ml.Model, error) { return ml.FitLassoRegression(features, regressionTarget, 0.05) },
		"wls": func() (ml.Model, error) {
			weights := make([]any, features.NumRows())
			for i := range weights {
				weights[i] = 1.0 + float64(i%3)
			}
			return ml.FitWeightedLinearRegression(features, regressionTarget, insyra.NewDataList(weights...))
		},
		"tree-regressor": func() (ml.Model, error) {
			return ml.FitDecisionTreeRegressor(features, regressionTarget, ml.DecisionTreeOptions{MaxDepth: 3})
		},
		"forest-regressor": func() (ml.Model, error) {
			seed := int64(5)
			return ml.FitRandomForestRegressor(features, regressionTarget,
				ml.RandomForestOptions{Trees: 8, Seed: &seed, Tree: ml.DecisionTreeOptions{MaxDepth: 3}})
		},
		"boosted-regressor": func() (ml.Model, error) {
			return ml.FitGradientBoostingRegressor(features, regressionTarget, ml.GradientBoostingOptions{Stages: 12})
		},
	}
	for name, fit := range regressors {
		t.Run(name, func(t *testing.T) {
			model, err := fit()
			if err != nil {
				t.Fatal(err)
			}
			loaded := exportAndLoad(t, model)
			want, err := model.Predict(features)
			if err != nil {
				t.Fatal(err)
			}
			got := runExportedGraph(t, loaded, features)
			compareLists(t, "prediction", got["prediction"], want, 1e-4)
		})
	}

	classifiers := map[string]func() (ml.Model, error){
		"logistic": func() (ml.Model, error) { return ml.FitLogisticRegression(features, classificationTarget) },
		"tree-classifier": func() (ml.Model, error) {
			return ml.FitDecisionTreeClassifier(features, classificationTarget, ml.DecisionTreeOptions{MaxDepth: 3})
		},
		"forest-classifier": func() (ml.Model, error) {
			seed := int64(5)
			return ml.FitRandomForestClassifier(features, classificationTarget,
				ml.RandomForestOptions{Trees: 8, Seed: &seed, Tree: ml.DecisionTreeOptions{MaxDepth: 3}})
		},
		"boosted-classifier": func() (ml.Model, error) {
			return ml.FitGradientBoostingClassifier(features, classificationTarget, ml.GradientBoostingOptions{Stages: 12})
		},
	}
	for name, fit := range classifiers {
		t.Run(name, func(t *testing.T) {
			model, err := fit()
			if err != nil {
				t.Fatal(err)
			}
			loaded := exportAndLoad(t, model)
			wantLabels, err := model.Predict(features)
			if err != nil {
				t.Fatal(err)
			}
			got := runExportedGraph(t, loaded, features)
			compareLists(t, "label", got["label"], wantLabels, 0)

			proba, isProba := model.(ml.ProbaModel)
			if !isProba {
				return
			}
			wantProba, err := proba.PredictProba(features)
			if err != nil {
				t.Fatal(err)
			}
			gotProba := got["probabilities"]
			if gotProba == nil {
				t.Fatal("the exported classifier produced no probabilities output")
			}
			shape := gotProba.Shape()
			if len(shape) != 2 || shape[0] != wantProba.NumRows() || shape[1] != wantProba.NumCols() {
				t.Fatalf("probabilities shape %v, want [%d %d]", shape, wantProba.NumRows(), wantProba.NumCols())
			}
			values, err := gotProba.Float32Data()
			if err != nil {
				t.Fatal(err)
			}
			for row := 0; row < wantProba.NumRows(); row++ {
				for col := 0; col < wantProba.NumCols(); col++ {
					want, _ := insyra.ToFloat64Safe(wantProba.GetElementByNumberIndex(row, col))
					gotValue := float64(values[row*shape[1]+col])
					if math.Abs(gotValue-want) > 1e-4 {
						t.Fatalf("probability [%d %d] = %v, ml's own %v", row, col, gotValue, want)
					}
				}
			}
		})
	}
}

// The exported-pipeline case: preprocessing and estimator as one graph.
func TestMLPipelineExportReadsBackAndReproducesItself(t *testing.T) {
	features, regressionTarget, _ := roundtripFixture()
	pipeline := ml.NewPipeline([]ml.Step{
		{Name: "scale", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			scaler := insyra.NewStandardScaler()
			if err := scaler.Fit(x, "x1", "x2"); err != nil {
				return nil, err
			}
			return scaler, nil
		}},
	}, ml.Estimator{Name: "linear", Fit: ml.FitLinearRegression})
	model, err := pipeline.Fit(features, regressionTarget)
	if err != nil {
		t.Fatal(err)
	}
	loaded := exportAndLoad(t, model)
	want, err := model.Predict(features)
	if err != nil {
		t.Fatal(err)
	}
	got := runExportedGraph(t, loaded, features)
	compareLists(t, "prediction", got["prediction"], want, 1e-4)
}

func roundtripFixture() (*insyra.DataTable, *insyra.DataList, *insyra.DataList) {
	const n = 24
	x1 := make([]any, n)
	x2 := make([]any, n)
	regression := make([]any, n)
	classification := make([]any, n)
	for i := 0; i < n; i++ {
		a := float64(i%6) + 0.5
		b := float64((i*5)%7) - 2
		x1[i] = a
		x2[i] = b
		regression[i] = 3*a - 1.5*b + 0.25*math.Sin(float64(i))
		if a-b > 2 {
			classification[i] = int64(1)
		} else {
			classification[i] = int64(0)
		}
	}
	features := insyra.NewDataTable(
		insyra.NewDataList(x1...).SetName("x1"),
		insyra.NewDataList(x2...).SetName("x2"),
	)
	return features, insyra.NewDataList(regression...), insyra.NewDataList(classification...)
}

func exportAndLoad(t *testing.T, model ml.Model) *nn.Model {
	t.Helper()
	exporter, ok := model.(ml.Exporter)
	if !ok {
		t.Fatalf("%T does not export", model)
	}
	var buffer bytes.Buffer
	if err := exporter.ExportONNX(&buffer); err != nil {
		t.Fatalf("export: %v", err)
	}
	loaded, err := nn.LoadONNX(&buffer)
	if err != nil {
		t.Fatalf("nn refused ml's own export: %v", err)
	}
	return loaded
}

// runExportedGraph feeds the graph ml's per-column inputs — one rank-1 tensor
// per feature, in the model's declared input order — and returns every output.
func runExportedGraph(t *testing.T, model *nn.Model, features *insyra.DataTable) map[string]*nn.Tensor {
	t.Helper()
	inputs := make(map[string]*nn.Tensor, len(model.Inputs()))
	for index, info := range model.Inputs() {
		if index >= features.NumCols() {
			t.Fatalf("model wants %d inputs, table has %d columns", len(model.Inputs()), features.NumCols())
		}
		column := features.GetColByNumber(index)
		values := make([]float32, column.Len())
		for i := 0; i < column.Len(); i++ {
			v, ok := insyra.ToFloat64Safe(column.Get(i))
			if !ok {
				t.Fatalf("column %d row %d is not numeric", index, i)
			}
			values[i] = float32(v)
		}
		tensor, err := nn.NewTensor([]int{column.Len()}, values)
		if err != nil {
			t.Fatal(err)
		}
		inputs[info.Name] = tensor
	}
	outputs, err := model.Run(inputs)
	if err != nil {
		t.Fatalf("nn run: %v", err)
	}
	return outputs
}

// compareLists checks a nn output tensor against ml's own DataList prediction.
// tolerance 0 means exact (labels); otherwise absolute-with-scale for the f32
// round trip through the exchange format.
func compareLists(t *testing.T, name string, got *nn.Tensor, want *insyra.DataList, tolerance float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("output %q is missing", name)
	}
	shape := got.Shape()
	rows := shape[0]
	if rows != want.Len() {
		t.Fatalf("%s rows = %d, want %d", name, rows, want.Len())
	}
	if tolerance == 0 {
		// ml exports integer class labels for this fixture; a string-labelled
		// model would come back through StringData the same way.
		labels, err := got.Int64Data()
		if err != nil {
			t.Fatalf("read %s labels: %v", name, err)
		}
		for i := 0; i < rows; i++ {
			if fmt.Sprint(labels[i]) != fmt.Sprint(want.Get(i)) {
				t.Fatalf("%s row %d = %v, ml's own %v", name, i, labels[i], want.Get(i))
			}
		}
		return
	}
	values, err := got.Float32Data()
	if err != nil {
		t.Fatalf("read %s values: %v", name, err)
	}
	for i := 0; i < rows; i++ {
		wantValue, _ := insyra.ToFloat64Safe(want.Get(i))
		gotValue := float64(values[i])
		if math.Abs(gotValue-wantValue) > tolerance*math.Max(math.Abs(wantValue), 1) {
			t.Fatalf("%s row %d = %v, ml's own %v", name, i, gotValue, wantValue)
		}
	}
}
