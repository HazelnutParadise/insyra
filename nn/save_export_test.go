package nn

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

//go:embed testdata/save_export_fixture.py
var saveExportFixtureScript string

func TestSaveSafeTensorsRoundTripIsDeterministic(t *testing.T) {
	weights, err := NewFloat32Tensor([]int{2, 2}, []float32{math.Float32frombits(0x7fc00001), -2.5, 0, 4.25})
	if err != nil {
		t.Fatal(err)
	}
	indices, err := NewInt64Tensor([]int{3}, []int64{-7, 0, 9})
	if err != nil {
		t.Fatal(err)
	}
	mask, err := NewBoolTensor([]int{4}, []bool{true, false, true, false})
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := NewFloat32Tensor(nil, []float32{3.5})
	if err != nil {
		t.Fatal(err)
	}
	tensors := map[string]*Tensor{"z.weight": weights, "a.indices": indices, "m.mask": mask, "scalar": scalar}

	var first, second bytes.Buffer
	if err := SaveSafeTensors(&first, tensors); err != nil {
		t.Fatalf("SaveSafeTensors: %v", err)
	}
	if err := SaveSafeTensors(&second, map[string]*Tensor{"m.mask": mask, "z.weight": weights, "a.indices": indices, "scalar": scalar}); err != nil {
		t.Fatalf("second SaveSafeTensors: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("saving the same tensors in a different map order changed the bytes")
	}
	loaded, err := LoadSafeTensors(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if got := loaded["z.weight"].Data(); len(got) != 4 || math.Float32bits(got[0]) != 0x7fc00001 || !reflect.DeepEqual(got[1:], []float32{-2.5, 0, 4.25}) {
		t.Fatalf("loaded weights = %#v", got)
	}
	gotIndices, err := loaded["a.indices"].Int64Data()
	if err != nil || !reflect.DeepEqual(gotIndices, []int64{-7, 0, 9}) {
		t.Fatalf("loaded indices = %v, err %v", gotIndices, err)
	}
	gotMask, err := loaded["m.mask"].BoolData()
	if err != nil || !reflect.DeepEqual(gotMask, []bool{true, false, true, false}) {
		t.Fatalf("loaded mask = %v, err %v", gotMask, err)
	}
	if got := loaded["scalar"]; got == nil || len(got.Shape()) != 0 || !reflect.DeepEqual(got.Data(), []float32{3.5}) {
		t.Fatalf("loaded scalar = %#v", got)
	}
}

func TestSaveSafeTensorsRefusesUnsupportedDTypeByName(t *testing.T) {
	stringTensor, err := NewStringTensor([]int{1}, []string{"no"})
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveSafeTensors(&bytes.Buffer{}, map[string]*Tensor{"text": stringTensor}); err == nil || !strings.Contains(err.Error(), `tensor "text"`) || !strings.Contains(err.Error(), "string") {
		t.Fatalf("error = %v, want tensor name and dtype", err)
	}
}

func TestSequentialSaveWeightsUsesTorchNamesAndLinearLayout(t *testing.T) {
	model, err := NewSequential(NewTape(17), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := model.SaveWeights(&buffer); err != nil {
		t.Fatalf("SaveWeights: %v", err)
	}
	weights, err := LoadSafeTensors(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	for _, name := range []string{"0.weight", "0.bias", "2.weight", "2.bias"} {
		if weights[name] == nil {
			t.Fatalf("missing saved state %q", name)
		}
	}
	if got := weights["0.weight"].Shape(); !reflect.DeepEqual(got, []int{3, 2}) {
		t.Fatalf("saved Linear shape = %v, want [3 2]", got)
	}
	input := mustTestTensor(t, []int{2, 2}, []float32{1, 2, -1, 0.5})
	want, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewSequential(NewTape(99), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.LoadWeights(weights); err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	got, err := reloaded.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Data(), want.Data()) {
		t.Fatalf("reloaded prediction = %v, want %v", got.Data(), want.Data())
	}
}

func TestSequentialSaveWeightsIncludesBatchNormRunningStatistics(t *testing.T) {
	model, err := NewSequential(NewTape(23), Conv2D(1, 2, 3), BatchNorm2D(2))
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := model.SaveWeights(&buffer); err != nil {
		t.Fatalf("SaveWeights: %v", err)
	}
	weights, err := LoadSafeTensors(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	for _, name := range []string{"0.weight", "0.bias", "1.weight", "1.bias", "1.running_mean", "1.running_var"} {
		if weights[name] == nil {
			t.Fatalf("missing saved state %q", name)
		}
	}
}

func TestSequentialExportONNXRunsInNN(t *testing.T) {
	mlp, err := NewSequential(NewTape(1), Dense(3, 4), ReLU(), Dropout(0.2), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	mlpInput := mustTestTensor(t, []int{2, 3}, []float32{0.25, -1, 2, 1.5, 0, -0.5})
	assertSequentialONNXRoundTrip(t, mlp, mlpInput)

	cnn, err := NewSequential(NewTape(3), Conv2D(1, 2, 3, ConvOptions{Pads: []int{1, 1, 1, 1}}), BatchNorm2D(2), ReLU(), MaxPool2D(2), NewFlatten(), Dense(8, 3))
	if err != nil {
		t.Fatal(err)
	}
	cnnInput := mustTestTensor(t, []int{2, 1, 4, 4}, sequenceFloat32(32))
	for step := 0; step < 2; step++ {
		cnn.tape.ops = nil
		cnn.tape.grads = make(map[*Tensor]*Tensor)
		if _, err := cnn.Forward(cnn.tape, cnnInput); err != nil {
			t.Fatalf("CNN training forward: %v", err)
		}
	}
	assertSequentialONNXRoundTrip(t, cnn, cnnInput)
}

func TestSequentialExportONNXRefusesUnmappedLayersByPosition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		layer Layer
	}{
		{name: "Func", layer: Func(func(_ *Tape, x *Tensor) (*Tensor, error) { return x, nil })},
		{name: "Embedding", layer: Embedding(4, 3)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model, err := NewSequential(NewTape(), tc.layer)
			if err != nil {
				t.Fatal(err)
			}
			if err := model.ExportONNX(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "layer 0") || !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("error = %v, want layer position and kind", err)
			}
		})
	}
}

func assertSequentialONNXRoundTrip(t *testing.T, model *Sequential, input *Tensor) {
	t.Helper()
	want, err := model.Predict(input)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	var buffer bytes.Buffer
	if err := model.ExportONNX(&buffer); err != nil {
		t.Fatalf("ExportONNX: %v", err)
	}
	loaded, err := LoadONNX(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("LoadONNX: %v", err)
	}
	if len(loaded.Inputs()) != 1 || loaded.Inputs()[0].Name != "input" {
		t.Fatalf("inputs = %#v, want one input named input", loaded.Inputs())
	}
	gotMap, err := loaded.Run(map[string]*Tensor{"input": input})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := gotMap["output"]
	if !reflect.DeepEqual(got.Shape(), want.Shape()) {
		t.Fatalf("output shape = %v, want %v", got.Shape(), want.Shape())
	}
	if !reflect.DeepEqual(got.Data(), want.Data()) {
		t.Fatalf("output = %v, want %v", got.Data(), want.Data())
	}
}

func sequenceFloat32(count int) []float32 {
	values := make([]float32, count)
	for index := range values {
		values[index] = float32(index-16) / 17
	}
	return values
}

func TestSequentialSaveWeightsTorchRoundTrip(t *testing.T) {
	python := requireSaveExportReference(t, "torch and safetensors", "the nn SaveWeights torch round-trip", "import torch, safetensors.torch")
	if python == "" {
		return
	}
	model, err := NewSequential(NewTape(7), Dense(3, 4), ReLU(), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	weightsPath := filepath.Join(t.TempDir(), "weights.safetensors")
	file, err := os.Create(weightsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.SaveWeights(file); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	input := mustTestTensor(t, []int{2, 3}, []float32{0.25, -1, 2, 1.5, 0, -0.5})
	want, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	reference := runSaveExportFixture(t, python, "weights", weightsPath)
	assertJSONFloat32(t, "torch weights output", reference, want)
}

func TestSequentialExportONNXRoundsTripThroughONNXRuntime(t *testing.T) {
	python := requireSaveExportReference(t, "numpy and onnxruntime", "the nn Sequential ONNX round-trip", "import numpy, onnxruntime")
	if python == "" {
		return
	}
	models := []struct {
		name      string
		model     *Sequential
		input     *Tensor
		inputSize string
	}{
		{name: "mlp", model: trainedExportMLP(t), input: mustTestTensor(t, []int{4, 3}, sequenceFloat32(12)), inputSize: "4,3"},
		{name: "cnn", model: trainedExportCNN(t), input: mustTestTensor(t, []int{4, 1, 4, 4}, sequenceFloat32(64)), inputSize: "4,1,4,4"},
	}
	for _, tc := range models {
		t.Run(tc.name, func(t *testing.T) {
			want, err := tc.model.Predict(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			modelPath := filepath.Join(t.TempDir(), tc.name+".onnx")
			file, err := os.Create(modelPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.model.ExportONNX(file); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			inputPath := filepath.Join(t.TempDir(), tc.name+".f32")
			if err := writeFloat32Fixture(inputPath, tc.input.data); err != nil {
				t.Fatal(err)
			}
			reference := runSaveExportFixture(t, python, "onnx", modelPath, inputPath, tc.inputSize)
			assertJSONFloat32(t, tc.name+" onnxruntime output", reference, want)
			loadedFile, err := os.Open(modelPath)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := LoadONNX(loadedFile)
			closeErr := loadedFile.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("LoadONNX: %v, close: %v", err, closeErr)
			}
			gotMap, err := loaded.Run(map[string]*Tensor{"input": tc.input})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotMap["output"].Data(), want.Data()) {
				t.Fatalf("nn output = %v, want %v", gotMap["output"].Data(), want.Data())
			}
		})
	}
}

type saveExportReference struct {
	Shape  []int     `json:"shape"`
	Values []float32 `json:"values"`
}

func requireSaveExportReference(t *testing.T, tool, verification, probe string) string {
	t.Helper()
	python := filepath.Join(os.Getenv("HOME"), ".cache", "insyra-crosslang-venv", "bin", "python3")
	if _, err := os.Stat(python); err != nil {
		python, err = exec.LookPath("python3")
		if err != nil {
			reftest.Missing(t, "python3", verification, err)
			return ""
		}
	}
	command := exec.Command(python, "-c", probe)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reftest.MissingOutput(t, tool, verification, err, stderr.Bytes())
		return ""
	}
	return python
}

func runSaveExportFixture(t *testing.T, python, mode string, args ...string) saveExportReference {
	t.Helper()
	commandArgs := append([]string{"-c", saveExportFixtureScript, mode}, args...)
	command := exec.Command(python, commandArgs...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("reference fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var reference saveExportReference
	if err := json.Unmarshal(stdout.Bytes(), &reference); err != nil {
		t.Fatalf("decode reference stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	return reference
}

func assertJSONFloat32(t *testing.T, name string, reference saveExportReference, want *Tensor) {
	t.Helper()
	if !reflect.DeepEqual(reference.Shape, want.Shape()) || len(reference.Values) != len(want.data) {
		t.Fatalf("%s shape/length = %v/%d, want %v/%d", name, reference.Shape, len(reference.Values), want.Shape(), len(want.data))
	}
	for index, value := range want.data {
		if math.Abs(float64(reference.Values[index]-value)) > 2e-5 {
			t.Fatalf("%s value[%d] = %v, want %v", name, index, reference.Values[index], value)
		}
	}
}

func trainedExportMLP(t *testing.T) *Sequential {
	t.Helper()
	tape := NewTape(11)
	model, err := NewSequential(tape, Dense(3, 4), ReLU(), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	input := mustTestTensor(t, []int{4, 3}, sequenceFloat32(12))
	labels := mustTestInt64Tensor(t, []int{4}, []int64{0, 1, 0, 1})
	trainExportModel(t, model, input, labels, 3)
	return model
}

func trainedExportCNN(t *testing.T) *Sequential {
	t.Helper()
	tape := NewTape(13)
	model, err := NewSequential(tape, Conv2D(1, 2, 3, ConvOptions{Pads: []int{1, 1, 1, 1}}), BatchNorm2D(2), ReLU(), MaxPool2D(2), NewFlatten(), Dense(8, 3))
	if err != nil {
		t.Fatal(err)
	}
	input := mustTestTensor(t, []int{4, 1, 4, 4}, sequenceFloat32(64))
	labels := mustTestInt64Tensor(t, []int{4}, []int64{0, 1, 2, 1})
	trainExportModel(t, model, input, labels, 3)
	return model
}

func trainExportModel(t *testing.T, model *Sequential, input, labels *Tensor, steps int) {
	t.Helper()
	for step := 0; step < steps; step++ {
		model.tape.ops = nil
		model.tape.grads = make(map[*Tensor]*Tensor)
		logits, err := model.Forward(model.tape, input)
		if err != nil {
			t.Fatalf("training forward: %v", err)
		}
		loss, err := model.tape.SoftmaxCrossEntropy(logits, labels)
		if err != nil {
			t.Fatalf("training loss: %v", err)
		}
		if err := model.tape.Backward(loss); err != nil {
			t.Fatalf("training backward: %v", err)
		}
		if err := model.tape.Adam(0.01); err != nil {
			t.Fatalf("training Adam: %v", err)
		}
	}
}

func writeFloat32Fixture(path string, values []float32) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	for _, value := range values {
		if err := binary.Write(file, binary.LittleEndian, math.Float32bits(value)); err != nil {
			_ = file.Close()
			return err
		}
	}
	return file.Close()
}
