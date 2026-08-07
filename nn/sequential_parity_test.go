package nn

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

//go:embed testdata/sequential_fixture.py
var sequentialFixtureScript string

type sequentialReference struct {
	Input          [][]float32          `json:"input"`
	Labels         []int64              `json:"labels"`
	Forward        [][]float32          `json:"forward"`
	Loss           float32              `json:"loss"`
	PostParameters map[string][]float32 `json:"post_parameters"`
}

func TestSequentialPyTorchInterop(t *testing.T) {
	python := requireSequentialReference(t)
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "sequential.safetensors")
	reference := runSequentialFixture(t, python, weightsPath)
	weights := loadSequentialWeights(t, weightsPath)

	model, err := NewSequential(NewTape(1), Dense(784, 128), ReLU(), Dense(128, 10))
	if err != nil {
		t.Fatal(err)
	}
	if err := model.LoadWeights(weights); err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	input := flattenReferenceInput(t, reference.Input)
	got, err := model.Predict(input)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	want := flattenReferenceOutput(t, reference.Forward)
	assertSequentialValues(t, "forward", got.Data(), want)

	tape := NewTape(1)
	trainModel, err := NewSequential(tape, Dense(784, 128), ReLU(), Dense(128, 10))
	if err != nil {
		t.Fatal(err)
	}
	if err := trainModel.LoadWeights(weights); err != nil {
		t.Fatalf("training LoadWeights: %v", err)
	}
	logits, err := trainModel.Forward(tape, input)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	labels, err := NewInt64Tensor([]int{len(reference.Labels)}, reference.Labels)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		t.Fatalf("SoftmaxCrossEntropy: %v", err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	if err := tape.AdamW(1e-3, 1e-2); err != nil {
		t.Fatalf("AdamW: %v", err)
	}
	assertSequentialValues(t, "loss", loss.Data(), []float32{reference.Loss})
	assertSequentialParameters(t, trainModel, reference.PostParameters)
}

func requireSequentialReference(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		reftest.Missing(t, "python3", "the nn Sequential PyTorch interop check", err)
		return ""
	}
	command := exec.Command(python, "-c", "import torch, safetensors.torch")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with torch and safetensors", "the nn Sequential PyTorch interop check", err, stderr.Bytes())
		return ""
	}
	return python
}

func runSequentialFixture(t *testing.T, python, weightsPath string) sequentialReference {
	t.Helper()
	command := exec.Command(python, "-c", sequentialFixtureScript, weightsPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("Sequential fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var reference sequentialReference
	if err := json.Unmarshal(stdout.Bytes(), &reference); err != nil {
		t.Fatalf("decode Sequential fixture stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	return reference
}

func loadSequentialWeights(t *testing.T, path string) map[string]*Tensor {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return weights
}

func flattenReferenceInput(t *testing.T, values [][]float32) *Tensor {
	t.Helper()
	if len(values) == 0 {
		t.Fatal("reference input is empty")
	}
	flat := make([]float32, 0, len(values)*len(values[0]))
	for row, values := range values {
		if len(values) != 784 {
			t.Fatalf("reference input row %d has %d values, want 784", row, len(values))
		}
		flat = append(flat, values...)
	}
	result, err := NewTensor([]int{len(values), 784}, flat)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func flattenReferenceOutput(t *testing.T, values [][]float32) []float32 {
	t.Helper()
	flat := make([]float32, 0, len(values)*10)
	for row, values := range values {
		if len(values) != 10 {
			t.Fatalf("reference output row %d has %d values, want 10", row, len(values))
		}
		flat = append(flat, values...)
	}
	return flat
}

func assertSequentialParameters(t *testing.T, model *Sequential, want map[string][]float32) {
	t.Helper()
	for name, reference := range want {
		parameter := model.NamedParameters()[name]
		if parameter == nil {
			t.Fatalf("post-step parameter %q is missing", name)
		}
		got := parameter.Value().Data()
		if strings.HasSuffix(name, ".weight") {
			got = transposeSequentialWeight(got, parameter.Value().Shape()[0], parameter.Value().Shape()[1])
		}
		assertSequentialValues(t, "post-step "+name, got, reference)
	}
}

func transposeSequentialWeight(values []float32, in, out int) []float32 {
	transposed := make([]float32, len(values))
	for input := 0; input < in; input++ {
		for output := 0; output < out; output++ {
			transposed[output*in+input] = values[input*out+output]
		}
	}
	return transposed
}

func assertSequentialValues(t *testing.T, name string, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", name, len(got), len(want))
	}
	for index := range got {
		difference := math.Abs(float64(got[index] - want[index]))
		scale := math.Max(1, math.Abs(float64(want[index])))
		if difference > 1e-5*scale {
			t.Fatalf("%s[%d] = %g, want %g (difference %g)", name, index, got[index], want[index], difference)
		}
	}
}
