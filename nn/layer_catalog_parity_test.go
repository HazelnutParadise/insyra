package nn

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

//go:embed testdata/batchnorm_catalog_fixture.py
var batchNormCatalogFixture string

//go:embed testdata/embedding_catalog_fixture.py
var embeddingCatalogFixture string

//go:embed testdata/cnn_catalog_fixture.py
var cnnCatalogFixture string

type batchNormCatalogReference struct {
	State   map[string][]float32            `json:"state"`
	Batches [][]float32                     `json:"batches"`
	Labels  [][]int64                       `json:"labels"`
	Steps   []batchNormCatalogReferenceStep `json:"steps"`
}

type batchNormCatalogReferenceStep struct {
	Loss        float32              `json:"loss"`
	Gradients   map[string][]float32 `json:"gradients"`
	Parameters  map[string][]float32 `json:"parameters"`
	RunningMean []float32            `json:"running_mean"`
	RunningVar  []float32            `json:"running_var"`
}

func TestBatchNormCatalogPyTorchParity(t *testing.T) {
	python := requireCatalogReference(t, "the nn BatchNorm catalog parity check")
	if python == "" {
		return
	}
	var reference batchNormCatalogReference
	runCatalogFixture(t, python, batchNormCatalogFixture, &reference)
	tape := NewTape(1)
	model, err := NewSequential(tape, Conv2D(1, 2, 3, ConvOptions{Pads: []int{1, 1, 1, 1}}), BatchNorm2D(2), ReLU(), GlobalAvgPool(), NewFlatten(), Dense(2, 3))
	if err != nil {
		t.Fatal(err)
	}
	if err := model.LoadWeights(catalogWeights(t, reference.State)); err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	bn := model.layers[1].(*batchNorm2DLayer)
	for step, expected := range reference.Steps {
		tape.ops = nil
		tape.grads = make(map[*Tensor]*Tensor)
		input := mustTestTensor(t, []int{2, 1, 4, 4}, reference.Batches[step])
		labels := mustTestInt64Tensor(t, []int{2}, reference.Labels[step])
		logits, err := model.Forward(tape, input)
		if err != nil {
			t.Fatalf("step %d Forward: %v", step, err)
		}
		loss, err := tape.SoftmaxCrossEntropy(logits, labels)
		if err != nil {
			t.Fatalf("step %d loss: %v", step, err)
		}
		if err := tape.Backward(loss); err != nil {
			t.Fatalf("step %d Backward: %v", step, err)
		}
		assertCatalogClose(t, fmt.Sprintf("step %d loss", step), loss.data, []float32{expected.Loss}, 2e-5)
		for name, values := range expected.Gradients {
			parameter := model.NamedParameters()[name]
			if parameter == nil {
				t.Fatalf("step %d missing parameter %q", step, name)
			}
			assertCatalogClose(t, fmt.Sprintf("step %d gradient %s", step, name), torchStateValues(name, parameter.Grad()), values, 5e-5)
		}
		if err := tape.SGD(0.02); err != nil {
			t.Fatalf("step %d SGD: %v", step, err)
		}
		for name, values := range expected.Parameters {
			parameter := model.NamedParameters()[name]
			assertCatalogClose(t, fmt.Sprintf("step %d parameter %s", step, name), torchStateValues(name, parameter.Value()), values, 5e-5)
		}
		assertCatalogClose(t, fmt.Sprintf("step %d running mean", step), bn.runningMean.data, expected.RunningMean, 5e-5)
		assertCatalogClose(t, fmt.Sprintf("step %d running var", step), bn.runningVariance.data, expected.RunningVar, 5e-5)
	}
}

type embeddingCatalogReference struct {
	Table    []float32 `json:"table"`
	Indices  []int64   `json:"indices"`
	Upstream []float32 `json:"upstream"`
	Output   []float32 `json:"output"`
	Loss     float32   `json:"loss"`
	Gradient []float32 `json:"gradient"`
}

func TestEmbeddingCatalogPyTorchParity(t *testing.T) {
	python := requireCatalogReference(t, "the nn Embedding catalog parity check")
	if python == "" {
		return
	}
	var reference embeddingCatalogReference
	runCatalogFixture(t, python, embeddingCatalogFixture, &reference)
	table := mustTestTensor(t, []int{4, 3}, reference.Table)
	indices, err := NewInt64Tensor([]int{len(reference.Indices)}, reference.Indices)
	if err != nil {
		t.Fatal(err)
	}
	upstream := mustTestTensor(t, []int{len(reference.Indices), 3}, reference.Upstream)
	tape := NewTape()
	parameter, err := tape.Param(table)
	if err != nil {
		t.Fatal(err)
	}
	output, err := tape.Embedding(table, indices)
	if err != nil {
		t.Fatal(err)
	}
	assertCatalogClose(t, "embedding output", output.data, reference.Output, 1e-6)
	product, err := tape.Mul(output, upstream)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.ReduceMean(product, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	assertCatalogClose(t, "embedding loss", loss.data, []float32{reference.Loss}, 1e-6)
	assertCatalogClose(t, "embedding gradient", parameter.Grad().data, reference.Gradient, 1e-6)
}

type cnnCatalogReference struct {
	Input  []float32 `json:"input"`
	Output []float32 `json:"output"`
}

func TestCNNCatalogLoadAndPredictPyTorchParity(t *testing.T) {
	python := requireCatalogReference(t, "the nn CNN LoadWeights parity check")
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "cnn.safetensors")
	var reference cnnCatalogReference
	runCatalogFixture(t, python, cnnCatalogFixture, &reference, weightsPath)
	file, err := os.Open(weightsPath)
	if err != nil {
		t.Fatal(err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	model, err := NewSequential(NewTape(1), Conv2D(1, 2, 3, ConvOptions{Pads: []int{1, 1, 1, 1}}), BatchNorm2D(2), ReLU(), MaxPool2D(2), NewFlatten(), Dense(8, 3))
	if err != nil {
		t.Fatal(err)
	}
	if err := model.LoadWeights(weights); err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	input := mustTestTensor(t, []int{2, 1, 4, 4}, reference.Input)
	output, err := model.Predict(input)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	assertCatalogClose(t, "CNN Predict", output.data, reference.Output, 2e-4)
}

func requireCatalogReference(t *testing.T, purpose string) string {
	t.Helper()
	python := filepath.Join(os.Getenv("HOME"), ".cache", "insyra-crosslang-venv", "bin", "python3")
	if _, err := os.Stat(python); err != nil {
		python, err = exec.LookPath("python3")
		if err != nil {
			reftest.Missing(t, "python3", purpose, err)
			return ""
		}
	}
	command := exec.Command(python, "-c", "import torch, safetensors.torch")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with torch and safetensors", purpose, err, stderr.Bytes())
		return ""
	}
	return python
}

func runCatalogFixture(t *testing.T, python, script string, result interface{}, args ...string) {
	t.Helper()
	commandArgs := append([]string{"-c", script}, args...)
	command := exec.Command(python, commandArgs...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("catalog fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	if err := json.Unmarshal(stdout.Bytes(), result); err != nil {
		t.Fatalf("decode catalog fixture stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
}

func catalogWeights(t *testing.T, values map[string][]float32) map[string]*Tensor {
	t.Helper()
	weights := make(map[string]*Tensor, len(values))
	for name, data := range values {
		if name == "1.num_batches_tracked" {
			continue
		}
		shape := catalogStateShape(name)
		weights[name] = mustTestTensor(t, shape, data)
	}
	return weights
}

func catalogStateShape(name string) []int {
	switch name {
	case "0.weight":
		return []int{2, 1, 3, 3}
	case "0.bias", "1.weight", "1.bias", "1.running_mean", "1.running_var":
		return []int{2}
	case "5.weight":
		return []int{3, 2}
	case "5.bias":
		return []int{3}
	default:
		panic("unknown catalog state " + name)
	}
}

func torchStateValues(name string, value *Tensor) []float32 {
	if name == "5.weight" {
		return transposeSequentialWeight(value.data, value.shape[0], value.shape[1])
	}
	return value.data
}

func assertCatalogClose(t *testing.T, name string, got, want []float32, tolerance float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", name, len(got), len(want))
	}
	for index := range got {
		difference := got[index] - want[index]
		if difference < -tolerance || difference > tolerance {
			t.Fatalf("%s[%d] = %g, want %g (tol %g)", name, index, got[index], want[index], tolerance)
		}
	}
}
