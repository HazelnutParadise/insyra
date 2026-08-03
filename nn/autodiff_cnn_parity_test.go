package nn

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/autodiff_cnn_fixture.py
var autodiffCNNFixtureScript string

type autodiffCNNReference struct {
	Loss           float32              `json:"loss"`
	ParameterNames []string             `json:"parameter_names"`
	Gradients      map[string][]float32 `json:"gradients"`
	Parameters     map[string][]float32 `json:"parameters"`
}

func TestAutodiffCNNAdamParity(t *testing.T) {
	python := requireAutodiffReference(t)
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "autodiff-cnn.safetensors")
	command := exec.Command(python, "-c", autodiffCNNFixtureScript, weightsPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("autodiff CNN fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	var want autodiffCNNReference
	if err := json.Unmarshal(stdout.Bytes(), &want); err != nil {
		t.Fatalf("decode autodiff CNN stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	file, err := os.Open(weightsPath)
	if err != nil {
		t.Fatalf("open autodiff CNN weights: %v", err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close autodiff CNN weights: %v", closeErr)
	}

	tape := NewTape()
	parameters := make(map[string]*Parameter, len(want.ParameterNames))
	for _, name := range want.ParameterNames {
		parameter, err := tape.Param(weights[name])
		if err != nil {
			t.Fatalf("Param %s: %v", name, err)
		}
		parameters[name] = parameter
	}
	loss := buildAutodiffCNN(t, tape, parameters, weights)
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	assertAutodiffValue(t, "CNN loss", loss.data, []float32{want.Loss})
	for _, name := range want.ParameterNames {
		assertAutodiffValue(t, name+" gradient", parameters[name].Grad().data, want.Gradients[name])
	}
	if err := tape.Adam(0.003); err != nil {
		t.Fatalf("Adam: %v", err)
	}
	for _, name := range want.ParameterNames {
		assertAutodiffValue(t, name+" post-step parameter", parameters[name].Value().data, want.Parameters[name])
	}
}

func buildAutodiffCNN(t *testing.T, tape *Tape, parameters map[string]*Parameter, weights map[string]*Tensor) *Tensor {
	t.Helper()
	parameter := func(name string) *Tensor {
		value := parameters[name]
		if value == nil {
			t.Fatalf("missing CNN parameter %q", name)
		}
		return value.Value()
	}
	inputData := make([]float32, 4*8*8)
	for index := range inputData {
		inputData[index] = float32(index%23-11) / 23
	}
	input := mustTestTensor(t, []int{4, 1, 8, 8}, inputData)

	hidden, err := tape.Conv(input, parameter("conv1.weight"), parameter("conv1.bias"), ConvOptions{Pads: []int{1, 1, 1, 1}})
	if err != nil {
		t.Fatalf("conv1: %v", err)
	}
	hidden, err = tape.BatchNormalization(hidden, parameter("bn.weight"), parameter("bn.bias"), weights["bn.running_mean"], weights["bn.running_var"], 1e-5)
	if err != nil {
		t.Fatalf("batch normalization: %v", err)
	}
	hidden, err = tape.Relu(hidden)
	if err != nil {
		t.Fatalf("relu1: %v", err)
	}
	hidden, err = tape.MaxPool(hidden, []int{2, 2}, PoolOptions{Strides: []int{2, 2}})
	if err != nil {
		t.Fatalf("max pool: %v", err)
	}
	hidden, err = tape.Conv(hidden, parameter("conv2.weight"), parameter("conv2.bias"), ConvOptions{Pads: []int{1, 1, 1, 1}})
	if err != nil {
		t.Fatalf("conv2: %v", err)
	}
	hidden, err = tape.Relu(hidden)
	if err != nil {
		t.Fatalf("relu2: %v", err)
	}
	hidden, err = tape.GlobalAveragePool(hidden)
	if err != nil {
		t.Fatalf("global average pool: %v", err)
	}
	hidden, err = tape.Flatten(hidden, 1)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	fcWeight, err := tape.Transpose(parameter("fc.weight"), []int{1, 0})
	if err != nil {
		t.Fatalf("linear weight transpose: %v", err)
	}
	logits, err := tape.MatMul(hidden, fcWeight)
	if err != nil {
		t.Fatalf("linear matmul: %v", err)
	}
	logits, err = tape.Add(logits, parameter("fc.bias"))
	if err != nil {
		t.Fatalf("linear bias: %v", err)
	}
	return mustTapeCrossEntropy(t, tape, logits, []int64{0, 3, 6, 9})
}

func mustTapeCrossEntropy(t *testing.T, tape *Tape, logits *Tensor, labels []int64) *Tensor {
	t.Helper()
	loss, err := tape.SoftmaxCrossEntropy(logits, mustTestInt64Tensor(t, []int{len(labels)}, labels))
	if err != nil {
		t.Fatalf("cross entropy: %v", err)
	}
	return loss
}
