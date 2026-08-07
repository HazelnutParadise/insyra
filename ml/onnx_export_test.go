package ml_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/reftest"
	"github.com/HazelnutParadise/insyra/ml"
)

func TestONNXExportWritesSupportedModels(t *testing.T) {
	features := onnxNumericFeatures()
	linear, err := ml.FitLinearRegression(features, onnxRegressionTargets())
	if err != nil {
		t.Fatal(err)
	}
	logistic, err := ml.FitLogisticRegression(features, onnxClassificationTargets())
	if err != nil {
		t.Fatal(err)
	}
	tree, err := ml.FitDecisionTreeRegressor(features, onnxRegressionTargets(), ml.DecisionTreeOptions{MaxDepth: 3})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		model any
		kind  string
	}{
		{name: "linear", model: linear, kind: "LinearRegressor"},
		{name: "logistic", model: logistic, kind: "LinearClassifier"},
		{name: "tree", model: tree, kind: "TreeEnsembleRegressor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := ml.ExportONNX(&output, tc.model); err != nil {
				t.Fatal(err)
			}
			if output.Len() == 0 {
				t.Fatal("export wrote no bytes")
			}
			if !bytes.Contains(output.Bytes(), []byte(tc.kind)) {
				t.Fatalf("export does not contain %s node", tc.kind)
			}
			if _, ok := tc.model.(ml.Exporter); !ok {
				t.Fatalf("%T does not implement ml.Exporter", tc.model)
			}
		})
	}
}

func TestONNXExportPipelineContainsPreprocessingGraph(t *testing.T) {
	input := onnxPipelineInput()
	pipeline := onnxPipeline()
	fitted, err := pipeline.Fit(input, onnxRegressionTargets())
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := ml.ExportONNX(&output, fitted); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"Sub", "OneHotEncoder", "LinearRegressor"} {
		if !bytes.Contains(output.Bytes(), []byte(kind)) {
			t.Fatalf("pipeline export does not contain %s", kind)
		}
	}
	if _, ok := fitted.(ml.Exporter); !ok {
		t.Fatalf("fitted pipeline %T does not implement ml.Exporter", fitted)
	}
}

func TestONNXExportRefusesUnsupportedModelWithoutWriting(t *testing.T) {
	model, err := ml.FitKMeans(onnxNumericFeatures(), 2)
	if err != nil {
		t.Fatal(err)
	}
	writer := &recordingWriter{}
	err = ml.ExportONNX(writer, model)
	if err == nil || !strings.Contains(err.Error(), "KMeansModel") {
		t.Fatalf("error = %v, want model name", err)
	}
	if writer.writes != 0 {
		t.Fatalf("unsupported export wrote %d times", writer.writes)
	}
}

func TestONNXIndependentRuntimeRoundTripOrExplicitlySkips(t *testing.T) {
	const verification = "the ONNX round trip against an independent runtime"
	python, err := exec.LookPath("python3")
	if err != nil {
		reftest.Missing(t, "python3", verification, err)
	}
	check := exec.Command(python, "-c", "import onnxruntime, numpy")
	if output, err := check.CombinedOutput(); err != nil {
		reftest.MissingOutput(t, "python3 with onnxruntime and numpy", verification, err, output)
	}

	cases := []struct {
		name  string
		model ml.Model
		input [][]any
	}{
		{name: "linear", model: mustLinearModel(t), input: numericInputs(onnxNumericFeatures())},
		{name: "logistic", model: mustLogisticModel(t), input: numericInputs(onnxNumericFeatures())},
		{name: "tree-regressor", model: mustTreeRegressor(t), input: numericInputs(onnxNumericFeatures())},
		{name: "tree-classifier", model: mustTreeClassifier(t), input: categoricalInputs(onnxTreeInput())},
		{name: "pipeline", model: mustPipelineModel(t), input: pipelineInputs(onnxPipelineInput())},
		{name: "ridge", model: mustRidgeModel(t), input: numericInputs(onnxNumericFeatures())},
		{name: "lasso", model: mustLassoModel(t), input: numericInputs(onnxNumericFeatures())},
		{name: "wls", model: mustWLSModel(t), input: numericInputs(onnxNumericFeatures())},
		{name: "forest-regressor", model: mustForestRegressor(t), input: numericInputs(onnxNumericFeatures())},
		{name: "forest-classifier", model: mustForestClassifier(t), input: numericInputs(onnxNumericFeatures())},
		{name: "boosted-regressor", model: mustBoostedRegressor(t), input: numericInputs(onnxNumericFeatures())},
		{name: "boosted-classifier", model: mustBoostedClassifier(t), input: numericInputs(onnxNumericFeatures())},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := tc.model.Predict(inputTableForModel(tc.model, tc.input))
			if err != nil {
				t.Fatal(err)
			}
			modelPath := filepath.Join(t.TempDir(), "model.onnx")
			file, err := os.Create(modelPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.model.(ml.Exporter).ExportONNX(file); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			payloadPath := filepath.Join(t.TempDir(), "input.json")
			payload, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(payloadPath, payload, 0o600); err != nil {
				t.Fatal(err)
			}
			got := runONNXRuntime(t, python, modelPath, payloadPath)
			if len(got) != expected.Len() {
				t.Fatalf("runtime rows = %d, want %d", len(got), expected.Len())
			}
			for index, value := range expected.Data() {
				// The runtime computes in float32 and this package in float64,
				// so numeric outputs are compared within single-precision
				// tolerance — which is what the spec's "within the tolerance
				// of the exchange format's own precision" means. Labels have
				// no tolerance and must match exactly.
				gotNum, gotOK := insyra.ToFloat64Safe(got[index])
				wantNum, wantOK := insyra.ToFloat64Safe(value)
				if gotOK && wantOK {
					diff := math.Abs(gotNum - wantNum)
					scale := math.Max(math.Abs(wantNum), 1)
					if diff > 1e-6*scale {
						t.Fatalf("row %d = %v, want %v (diff %g exceeds float32 tolerance)", index, gotNum, wantNum, diff)
					}
					continue
				}
				if fmt.Sprint(got[index]) != fmt.Sprint(value) {
					t.Fatalf("row %d = %v, want %v", index, got[index], value)
				}
			}
		})
	}
}

type recordingWriter struct{ writes int }

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.writes++
	return len(p), nil
}

func runONNXRuntime(t *testing.T, python, modelPath, payloadPath string) []any {
	t.Helper()
	script := `import json, sys
import numpy as np
import onnxruntime as ort
session = ort.InferenceSession(sys.argv[1], providers=["CPUExecutionProvider"])
payload = json.load(open(sys.argv[2]))
feed = {}
for i, value in enumerate(payload):
    spec = session.get_inputs()[i]
    kind = np.str_ if spec.type == "tensor(string)" else np.float32
    feed[spec.name] = np.asarray(value, dtype=kind)
result = session.run(None, feed)[0]
print(json.dumps(np.asarray(result).reshape(-1).tolist()))`
	// stdout carries the JSON and stderr carries the runtime's logging, so
	// they are captured separately. CombinedOutput here made the test pass on
	// a machine where onnxruntime happened to stay quiet and fail on one where
	// it warns — on the CI runner it emits a PCI bus-scan warning that landed
	// in front of the JSON and broke every case at once.
	cmd := exec.Command(python, "-c", script, modelPath, payloadPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("onnxruntime: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var values []any
	if err := json.Unmarshal(stdout.Bytes(), &values); err != nil {
		t.Fatalf("decode runtime output: %v\nstdout=%s\nstderr=%s", err,
			strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	return values
}

func onnxNumericFeatures() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0).SetName("x1"),
		insyra.NewDataList(1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0).SetName("x2"),
	)
}

func onnxRegressionTargets() *insyra.DataList {
	return insyra.NewDataList(1.0, 2.0, 5.0, 6.0, 9.0, 10.0, 13.0, 14.0).SetName("y")
}

func onnxClassificationTargets() *insyra.DataList {
	return insyra.NewDataList(0, 0, 0, 1, 1, 1, 1, 1).SetName("label")
}

func onnxPipelineInput() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0).SetName("x1"),
		insyra.NewDataList("red", "blue", "red", "green", "blue", "green", "red", "blue").SetName("color"),
	)
}

func onnxTreeInput() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList("red", "blue", "green", "red", "blue", "green", "red", "blue").SetName("color"),
		insyra.NewDataList(0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0).SetName("x1"),
	)
}

func onnxPipeline() ml.Estimator {
	return ml.NewPipeline([]ml.Step{
		{Name: "scale", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			scaler := insyra.NewStandardScaler()
			if err := scaler.Fit(x, "x1"); err != nil {
				return nil, err
			}
			return ml.NewColumnTransformer(scaler, "x1"), nil
		}},
		{Name: "encode", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			_, encoder, err := x.OneHotEncode(insyra.OneHotOptions{Columns: []string{"color"}, DropFirst: true, SortCategories: true})
			if err != nil {
				return nil, err
			}
			return ml.NewColumnTransformer(encoder, "color"), nil
		}},
	}, ml.Estimator{Name: "linear", Fit: ml.FitLinearRegression})
}

func mustLinearModel(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitLinearRegression(onnxNumericFeatures(), onnxRegressionTargets())
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustLogisticModel(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitLogisticRegression(onnxNumericFeatures(), onnxClassificationTargets())
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustTreeRegressor(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitDecisionTreeRegressor(onnxNumericFeatures(), onnxRegressionTargets(), ml.DecisionTreeOptions{MaxDepth: 3})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustTreeClassifier(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitDecisionTreeClassifier(onnxTreeInput(), insyra.NewDataList("warm", "cool", "cold", "warm", "cool", "cold", "warm", "cool").SetName("label"), ml.DecisionTreeOptions{CategoricalFeatures: []string{"color"}, MaxDepth: 3})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustPipelineModel(t *testing.T) ml.Model {
	t.Helper()
	model, err := onnxPipeline().Fit(onnxPipelineInput(), onnxRegressionTargets())
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustRidgeModel(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitRidgeRegression(onnxNumericFeatures(), onnxRegressionTargets(), 0.7)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustLassoModel(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitLassoRegression(onnxNumericFeatures(), onnxRegressionTargets(), 0.05)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustWLSModel(t *testing.T) ml.Model {
	t.Helper()
	weights := insyra.NewDataList(1.0, 2.0, 1.0, 3.0, 1.0, 2.0, 1.0, 2.0)
	model, err := ml.FitWeightedLinearRegression(onnxNumericFeatures(), onnxRegressionTargets(), weights)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustForestRegressor(t *testing.T) ml.Model {
	t.Helper()
	seed := int64(9)
	model, err := ml.FitRandomForestRegressor(onnxNumericFeatures(), onnxRegressionTargets(),
		ml.RandomForestOptions{Trees: 12, Seed: &seed, Tree: ml.DecisionTreeOptions{MaxDepth: 3}})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustForestClassifier(t *testing.T) ml.Model {
	t.Helper()
	seed := int64(9)
	model, err := ml.FitRandomForestClassifier(onnxNumericFeatures(), onnxClassificationTargets(),
		ml.RandomForestOptions{Trees: 12, Seed: &seed, Tree: ml.DecisionTreeOptions{MaxDepth: 3}})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustBoostedRegressor(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitGradientBoostingRegressor(onnxNumericFeatures(), onnxRegressionTargets(),
		ml.GradientBoostingOptions{Stages: 20})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustBoostedClassifier(t *testing.T) ml.Model {
	t.Helper()
	model, err := ml.FitGradientBoostingClassifier(onnxNumericFeatures(), onnxClassificationTargets(),
		ml.GradientBoostingOptions{Stages: 20})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func numericInputs(table *insyra.DataTable) [][]any {
	return [][]any{table.GetColByNumber(0).Data(), table.GetColByNumber(1).Data()}
}

func categoricalInputs(table *insyra.DataTable) [][]any {
	return [][]any{table.GetColByNumber(0).Data(), table.GetColByNumber(1).Data()}
}

func pipelineInputs(table *insyra.DataTable) [][]any {
	return [][]any{table.GetColByNumber(0).Data(), table.GetColByNumber(1).Data()}
}

func inputTableForModel(model ml.Model, input [][]any) *insyra.DataTable {
	features := model.Features()
	columns := make([]*insyra.DataList, len(features))
	for index, values := range input {
		columns[index] = insyra.NewDataList(values...).SetName(features[index])
	}
	return insyra.NewDataTable(columns...)
}
