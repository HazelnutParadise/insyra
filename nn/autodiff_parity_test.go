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

//go:embed testdata/autodiff_fixture.py
var autodiffFixtureScript string

type autodiffReference struct {
	Loss      float32              `json:"loss"`
	Gradients map[string][]float32 `json:"gradients"`
}

func TestAutodiffParityAgainstPyTorch(t *testing.T) {
	python := requireAutodiffReference(t)
	if python == "" {
		return
	}
	path := filepath.Join(t.TempDir(), "autodiff.safetensors")
	command := exec.Command(python, "-c", autodiffFixtureScript, path)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("autodiff fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var want autodiffReference
	if err := json.Unmarshal(stdout.Bytes(), &want); err != nil {
		t.Fatalf("decode autodiff fixture stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open autodiff weights: %v", err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close autodiff weights: %v", closeErr)
	}

	tape := NewTape()
	parameters := make(map[string]*Parameter, len(weights))
	for _, name := range []string{"w1", "b1", "w2", "b2"} {
		parameter, err := tape.Param(weights[name])
		if err != nil {
			t.Fatalf("Param %s: %v", name, err)
		}
		parameters[name] = parameter
	}
	input := mustTestTensor(t, []int{4, 3}, []float32{0.2, -0.7, 1.1, 0.4, 0.3, -0.8, -0.6, 0.9, 0.5, 1.2, -0.1, -0.4})
	labels := mustTestInt64Tensor(t, []int{4}, []int64{2, 0, 1, 2})
	hidden, err := tape.MatMul(input, parameters["w1"].Value())
	if err != nil {
		t.Fatalf("hidden MatMul: %v", err)
	}
	hidden, err = tape.Add(hidden, parameters["b1"].Value())
	if err != nil {
		t.Fatalf("hidden Add: %v", err)
	}
	hidden, err = tape.Tanh(hidden)
	if err != nil {
		t.Fatalf("hidden Relu: %v", err)
	}
	logits, err := tape.MatMul(hidden, parameters["w2"].Value())
	if err != nil {
		t.Fatalf("output MatMul: %v", err)
	}
	logits, err = tape.Add(logits, parameters["b2"].Value())
	if err != nil {
		t.Fatalf("output Add: %v", err)
	}
	loss, err := tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		t.Fatalf("SoftmaxCrossEntropy: %v", err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}

	assertAutodiffValue(t, "loss", loss.data, []float32{want.Loss})
	for _, name := range []string{"w1", "b1", "w2", "b2"} {
		assertAutodiffValue(t, name+" gradient", parameters[name].Grad().data, want.Gradients[name])
	}
}

func requireAutodiffReference(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		reftest.Missing(t, "python3", "the nn PyTorch autodiff parity check", err)
		return ""
	}
	command := exec.Command(python, "-c", "import torch, safetensors.torch")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with torch and safetensors", "the nn PyTorch autodiff parity check", err, stderr.Bytes())
		return ""
	}
	return python
}

func assertAutodiffValue(t *testing.T, name string, got, want []float32) {
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
