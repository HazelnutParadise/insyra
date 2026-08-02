package dl_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/dl"
	"github.com/HazelnutParadise/insyra/internal/reftest"
	"github.com/HazelnutParadise/insyra/ml"
)

//go:embed testdata/onnx_parity.py
var roundTripHelper string

type roundTripCase struct {
	name  string
	model func(*testing.T) ml.Model
	input func() *insyra.DataTable
}

type roundTripReference struct {
	Name  string `json:"name"`
	Shape []int  `json:"shape"`
	DType string `json:"dtype"`
	Data  []any  `json:"data"`
}

func TestMLONNXRoundTripsThroughDLAndONNXRuntime(t *testing.T) {
	python := requireRoundTripReference(t)
	if python == "" {
		return
	}
	cases := []roundTripCase{
		{name: "linear", model: mustLinearRoundTripModel, input: onnxNumericFeatures},
		{name: "ridge", model: mustRidgeRoundTripModel, input: onnxNumericFeatures},
		{name: "lasso", model: mustLassoRoundTripModel, input: onnxNumericFeatures},
		{name: "wls", model: mustWLSRoundTripModel, input: onnxNumericFeatures},
		{name: "logistic", model: mustLogisticRoundTripModel, input: onnxNumericFeatures},
		{name: "decision-tree-regressor", model: mustTreeRegressorRoundTripModel, input: onnxNumericFeatures},
		{name: "decision-tree-classifier", model: mustTreeClassifierRoundTripModel, input: onnxTreeInput},
		{name: "random-forest-regressor", model: mustForestRegressorRoundTripModel, input: onnxNumericFeatures},
		{name: "random-forest-classifier", model: mustForestClassifierRoundTripModel, input: onnxNumericFeatures},
		{name: "gradient-boosting-regressor", model: mustBoostedRegressorRoundTripModel, input: onnxNumericFeatures},
		{name: "gradient-boosting-classifier", model: mustBoostedClassifierRoundTripModel, input: onnxNumericFeatures},
		{name: "pipeline", model: mustPipelineRoundTripModel, input: onnxPipelineInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.model(t)
			input := tc.input()
			originalPrediction, err := model.Predict(input)
			if err != nil {
				t.Fatalf("fitted model Predict: %v", err)
			}
			originalProbabilities, isClassifier := model.(ml.ProbaModel)
			var originalProba *insyra.DataTable
			if isClassifier {
				originalProba, err = originalProbabilities.PredictProba(input)
				if err != nil {
					t.Fatalf("fitted model PredictProba: %v", err)
				}
			}

			modelPath := filepath.Join(t.TempDir(), "model.onnx")
			var exported bytes.Buffer
			exporter, ok := model.(ml.Exporter)
			if !ok {
				t.Fatalf("%T does not implement ml.Exporter", model)
			}
			if err := exporter.ExportONNX(&exported); err != nil {
				t.Fatalf("ExportONNX: %v", err)
			}
			if err := os.WriteFile(modelPath, exported.Bytes(), 0o600); err != nil {
				t.Fatalf("write ONNX: %v", err)
			}

			loaded, err := dl.LoadONNX(bytes.NewReader(exported.Bytes()))
			if err != nil {
				t.Fatalf("dl.LoadONNX: %v", err)
			}
			inputs, err := dlInputsForTable(loaded, model.Features(), input)
			if err != nil {
				t.Fatalf("build dl inputs: %v", err)
			}
			dlOutputs, err := loaded.Run(inputs)
			if err != nil {
				t.Fatalf("dl.Run: %v", err)
			}

			payloadPath := filepath.Join(t.TempDir(), "inputs.json")
			payload, err := runtimePayload(loaded, model.Features(), input)
			if err != nil {
				t.Fatalf("build runtime payload: %v", err)
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal runtime payload: %v", err)
			}
			if err := os.WriteFile(payloadPath, payloadBytes, 0o600); err != nil {
				t.Fatalf("write runtime payload: %v", err)
			}
			reference := runRoundTripRuntime(t, python, modelPath, payloadPath)

			if isClassifier {
				assertClassifierOutputs(t, tc.name, loaded, dlOutputs, reference, originalPrediction, originalProba)
			} else {
				assertFloatTensorAgainstDataList(t, tc.name+" dl prediction", dlOutputs["prediction"], originalPrediction.Data())
				assertFloatReferenceAgainstDataList(t, tc.name+" runtime prediction", reference["prediction"], originalPrediction.Data())
				assertFloatTensorAgainstReference(t, tc.name+" dl/runtime prediction", dlOutputs["prediction"], reference["prediction"])
			}
			t.Logf("round-trip family=%s outputs=%v", tc.name, outputNames(loaded))
		})
	}
}

func assertClassifierOutputs(t *testing.T, family string, model *dl.Model, got map[string]*dl.Tensor, reference map[string]roundTripReference, originalPrediction *insyra.DataList, originalProba *insyra.DataTable) {
	t.Helper()
	probabilityWant := make([]any, 0, originalProba.NumRows()*originalProba.NumCols())
	for row := 0; row < originalProba.NumRows(); row++ {
		for column := 0; column < originalProba.NumCols(); column++ {
			probabilityWant = append(probabilityWant, originalProba.GetColByNumber(column).Get(row))
		}
	}
	assertFloatTensorAgainstDataList(t, family+" dl probabilities", got["probabilities"], probabilityWant)
	assertFloatReferenceAgainstDataList(t, family+" runtime probabilities", reference["probabilities"], probabilityWant)
	assertFloatTensorAgainstReference(t, family+" dl/runtime probabilities", got["probabilities"], reference["probabilities"])

	labels := tensorValues(got["label"])
	if len(labels) != originalPrediction.Len() {
		t.Fatalf("%s dl labels length = %d, want %d", family, len(labels), originalPrediction.Len())
	}
	for index, want := range originalPrediction.Data() {
		assertSameLabel(t, fmt.Sprintf("%s dl label row %d", family, index), labels[index], want)
	}
	runtimeLabels := reference["label"].Data
	if len(runtimeLabels) != len(labels) {
		t.Fatalf("%s runtime labels length = %d, want %d", family, len(runtimeLabels), len(labels))
	}
	for index := range labels {
		assertSameLabel(t, fmt.Sprintf("%s runtime label row %d", family, index), runtimeLabels[index], labels[index])
	}
	_ = model
}

func assertFloatTensorAgainstDataList(t *testing.T, name string, got *dl.Tensor, want []any) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil", name)
	}
	data, err := got.Float32Data()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if len(data) != len(want) {
		t.Fatalf("%s length = %d, want %d", name, len(data), len(want))
	}
	for index, value := range want {
		wantNumber, ok := insyra.ToFloat64Safe(value)
		if !ok {
			t.Fatalf("%s want[%d] = %v is not numeric", name, index, value)
		}
		assertFloatClose(t, fmt.Sprintf("%s[%d]", name, index), float64(data[index]), wantNumber)
	}
}

func assertFloatReferenceAgainstDataList(t *testing.T, name string, got roundTripReference, want []any) {
	t.Helper()
	if len(got.Data) != len(want) {
		t.Fatalf("%s length = %d, want %d", name, len(got.Data), len(want))
	}
	for index, value := range want {
		wantNumber, ok := insyra.ToFloat64Safe(value)
		if !ok {
			t.Fatalf("%s want[%d] = %v is not numeric", name, index, value)
		}
		gotNumber, ok := insyra.ToFloat64Safe(got.Data[index])
		if !ok {
			t.Fatalf("%s got[%d] = %v is not numeric", name, index, got.Data[index])
		}
		assertFloatClose(t, fmt.Sprintf("%s[%d]", name, index), gotNumber, wantNumber)
	}
}

func assertFloatTensorAgainstReference(t *testing.T, name string, got *dl.Tensor, want roundTripReference) {
	t.Helper()
	data, err := got.Float32Data()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if len(data) != len(want.Data) {
		t.Fatalf("%s length = %d, want %d", name, len(data), len(want.Data))
	}
	for index := range data {
		wantNumber, ok := insyra.ToFloat64Safe(want.Data[index])
		if !ok {
			t.Fatalf("%s want[%d] = %v is not numeric", name, index, want.Data[index])
		}
		assertFloatClose(t, fmt.Sprintf("%s[%d]", name, index), float64(data[index]), wantNumber)
	}
}

func assertFloatClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	difference := math.Abs(got - want)
	scale := math.Max(1, math.Abs(want))
	if difference > 1e-5*scale {
		t.Fatalf("%s = %g, want %g (difference %g)", name, got, want, difference)
	}
}

func assertSameLabel(t *testing.T, name string, got, want any) {
	t.Helper()
	gotNumber, gotNumeric := insyra.ToFloat64Safe(got)
	wantNumber, wantNumeric := insyra.ToFloat64Safe(want)
	if gotNumeric && wantNumeric {
		if gotNumber != wantNumber {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
		return
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func tensorValues(tensor *dl.Tensor) []any {
	if tensor == nil {
		return nil
	}
	switch tensor.DType() {
	case dl.DTypeInt64:
		values, _ := tensor.Int64Data()
		result := make([]any, len(values))
		for index, value := range values {
			result[index] = value
		}
		return result
	case dl.DTypeString:
		values, _ := tensor.StringData()
		result := make([]any, len(values))
		for index, value := range values {
			result[index] = value
		}
		return result
	case dl.DTypeFloat32:
		values, _ := tensor.Float32Data()
		result := make([]any, len(values))
		for index, value := range values {
			result[index] = value
		}
		return result
	default:
		return nil
	}
}

func dlInputsForTable(model *dl.Model, features []string, table *insyra.DataTable) (map[string]*dl.Tensor, error) {
	inputs := make(map[string]*dl.Tensor)
	for index, spec := range model.Inputs() {
		if index >= len(features) {
			return nil, fmt.Errorf("model has input %q without a feature name", spec.Name)
		}
		column := table.GetColByName(features[index])
		if column == nil {
			return nil, fmt.Errorf("input column %q is missing", features[index])
		}
		values := column.Data()
		shape := []int{len(values)}
		switch spec.DType {
		case dl.DTypeFloat32:
			data := make([]float32, len(values))
			for index, value := range values {
				number, ok := insyra.ToFloat64Safe(value)
				if !ok {
					return nil, fmt.Errorf("input %q row %d is not numeric: %v", spec.Name, index, value)
				}
				data[index] = float32(number)
			}
			tensor, err := dl.NewFloat32Tensor(shape, data)
			if err != nil {
				return nil, err
			}
			inputs[spec.Name] = tensor
		case dl.DTypeString:
			data := make([]string, len(values))
			for index, value := range values {
				data[index] = fmt.Sprint(value)
			}
			tensor, err := dl.NewStringTensor(shape, data)
			if err != nil {
				return nil, err
			}
			inputs[spec.Name] = tensor
		case dl.DTypeInt64:
			data := make([]int64, len(values))
			for index, value := range values {
				number, ok := insyra.ToFloat64Safe(value)
				if !ok {
					return nil, fmt.Errorf("input %q row %d is not integer: %v", spec.Name, index, value)
				}
				data[index] = int64(number)
			}
			tensor, err := dl.NewInt64Tensor(shape, data)
			if err != nil {
				return nil, err
			}
			inputs[spec.Name] = tensor
		default:
			return nil, fmt.Errorf("input %q dtype %s is not supported by test", spec.Name, spec.DType)
		}
	}
	return inputs, nil
}

func runtimePayload(model *dl.Model, features []string, table *insyra.DataTable) ([][]any, error) {
	payload := make([][]any, 0, len(model.Inputs()))
	for index, spec := range model.Inputs() {
		if index >= len(features) {
			return nil, fmt.Errorf("model has input %q without a feature name", spec.Name)
		}
		column := table.GetColByName(features[index])
		if column == nil {
			return nil, fmt.Errorf("input column %q is missing", features[index])
		}
		payload = append(payload, column.Data())
	}
	return payload, nil
}

func requireRoundTripReference(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		reftest.Missing(t, "python3", "the dl ML ONNX round trips", err)
		return ""
	}
	probe := exec.Command(python, "-c", "import numpy, onnx, onnxruntime")
	var stdout, stderr bytes.Buffer
	probe.Stdout = &stdout
	probe.Stderr = &stderr
	if err := probe.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with numpy, onnx, and onnxruntime", "the dl ML ONNX round trips", err, stderr.Bytes())
		return ""
	}
	return python
}

func runRoundTripRuntime(t *testing.T, python, modelPath, payloadPath string) map[string]roundTripReference {
	t.Helper()
	command := exec.Command(python, "-c", roundTripHelper, "roundtrip", modelPath, payloadPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("onnxruntime round trip: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var values []roundTripReference
	if err := json.Unmarshal(stdout.Bytes(), &values); err != nil {
		t.Fatalf("decode onnxruntime round trip: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	result := make(map[string]roundTripReference, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func outputNames(model *dl.Model) []string {
	outputs := model.Outputs()
	names := make([]string, len(outputs))
	for index, output := range outputs {
		names[index] = output.Name
	}
	return names
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

func onnxTreeInput() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList("red", "blue", "green", "red", "blue", "green", "red", "blue").SetName("color"),
		insyra.NewDataList(0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0).SetName("x1"),
	)
}

func onnxPipelineInput() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0).SetName("x1"),
		insyra.NewDataList("red", "blue", "red", "green", "blue", "green", "red", "blue").SetName("color"),
	)
}

func mustLinearRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitLinearRegression(onnxNumericFeatures(), onnxRegressionTargets())
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustRidgeRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitRidgeRegression(onnxNumericFeatures(), onnxRegressionTargets(), 0.7)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustLassoRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitLassoRegression(onnxNumericFeatures(), onnxRegressionTargets(), 0.05)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustWLSRoundTripModel(t *testing.T) ml.Model {
	weights := insyra.NewDataList(1.0, 2.0, 1.0, 3.0, 1.0, 2.0, 1.0, 2.0)
	model, err := ml.FitWeightedLinearRegression(onnxNumericFeatures(), onnxRegressionTargets(), weights)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustLogisticRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitLogisticRegression(onnxNumericFeatures(), onnxClassificationTargets())
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustTreeRegressorRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitDecisionTreeRegressor(onnxNumericFeatures(), onnxRegressionTargets(), ml.DecisionTreeOptions{MaxDepth: 3})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustTreeClassifierRoundTripModel(t *testing.T) ml.Model {
	labels := insyra.NewDataList("warm", "cool", "cold", "warm", "cool", "cold", "warm", "cool").SetName("label")
	model, err := ml.FitDecisionTreeClassifier(onnxTreeInput(), labels, ml.DecisionTreeOptions{CategoricalFeatures: []string{"color"}, MaxDepth: 3})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustForestRegressorRoundTripModel(t *testing.T) ml.Model {
	seed := int64(9)
	model, err := ml.FitRandomForestRegressor(onnxNumericFeatures(), onnxRegressionTargets(), ml.RandomForestOptions{Trees: 12, Seed: &seed, Tree: ml.DecisionTreeOptions{MaxDepth: 3}})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustForestClassifierRoundTripModel(t *testing.T) ml.Model {
	seed := int64(9)
	model, err := ml.FitRandomForestClassifier(onnxNumericFeatures(), onnxClassificationTargets(), ml.RandomForestOptions{Trees: 12, Seed: &seed, Tree: ml.DecisionTreeOptions{MaxDepth: 3}})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustBoostedRegressorRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitGradientBoostingRegressor(onnxNumericFeatures(), onnxRegressionTargets(), ml.GradientBoostingOptions{Stages: 20})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustBoostedClassifierRoundTripModel(t *testing.T) ml.Model {
	model, err := ml.FitGradientBoostingClassifier(onnxNumericFeatures(), onnxClassificationTargets(), ml.GradientBoostingOptions{Stages: 20})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func mustPipelineRoundTripModel(t *testing.T) ml.Model {
	pipeline := ml.NewPipeline([]ml.Step{
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
	model, err := pipeline.Fit(onnxPipelineInput(), onnxRegressionTargets())
	if err != nil {
		t.Fatal(err)
	}
	return model
}
