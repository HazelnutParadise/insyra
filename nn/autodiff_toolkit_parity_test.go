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
)

//go:embed testdata/toolkit_fixture.py
var autodiffToolkitFixtureScript string

type autodiffToolkitReference struct {
	Steps []autodiffToolkitStep `json:"steps"`
}

type autodiffToolkitStep struct {
	LearningRate float32              `json:"lr"`
	Loss         float32              `json:"loss"`
	GradNorm     float32              `json:"grad_norm"`
	Parameters   map[string][]float32 `json:"parameters"`
}

func TestAutodiffToolkitParity(t *testing.T) {
	python := requireAutodiffReference(t)
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "autodiff-toolkit.safetensors")
	command := exec.Command(python, "-c", autodiffToolkitFixtureScript, weightsPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("autodiff toolkit fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var want autodiffToolkitReference
	if err := json.Unmarshal(stdout.Bytes(), &want); err != nil {
		t.Fatalf("decode autodiff toolkit stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	if len(want.Steps) != 6 {
		t.Fatalf("fixture returned %d steps, want 6", len(want.Steps))
	}

	file, err := os.Open(weightsPath)
	if err != nil {
		t.Fatalf("open autodiff toolkit weights: %v", err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close autodiff toolkit weights: %v", closeErr)
	}

	const parameterNames = "w1 b1 w2 b2"
	tape := NewTape()
	parameters := make(map[string]*Parameter)
	for _, name := range strings.Fields(parameterNames) {
		parameter, err := tape.Param(weights[name])
		if err != nil {
			t.Fatalf("Param %s: %v", name, err)
		}
		parameters[name] = parameter
	}
	input := mustTestTensor(t, []int{5, 3}, []float32{
		0.2, -0.7, 1.1,
		0.4, 0.3, -0.8,
		-0.6, 0.9, 0.5,
		1.2, -0.1, -0.4,
		-0.3, 0.8, 0.6,
	})
	targets := mustTestTensor(t, []int{5, 2}, []float32{
		1, 0,
		0, 1,
		1, 1,
		0, 0,
		1, 0,
	})
	schedule, err := NewCosineAnnealingLR(1e-2, 6)
	if err != nil {
		t.Fatal(err)
	}

	for step, reference := range want.Steps {
		tape.ops = nil
		tape.grads = make(map[*Tensor]*Tensor)
		hidden, err := tape.MatMul(input, parameters["w1"].Value())
		if err != nil {
			t.Fatalf("step %d hidden MatMul: %v", step, err)
		}
		hidden, err = tape.Add(hidden, parameters["b1"].Value())
		if err != nil {
			t.Fatalf("step %d hidden Add: %v", step, err)
		}
		hidden, err = tape.Tanh(hidden)
		if err != nil {
			t.Fatalf("step %d hidden Tanh: %v", step, err)
		}
		logits, err := tape.MatMul(hidden, parameters["w2"].Value())
		if err != nil {
			t.Fatalf("step %d output MatMul: %v", step, err)
		}
		logits, err = tape.Add(logits, parameters["b2"].Value())
		if err != nil {
			t.Fatalf("step %d output Add: %v", step, err)
		}
		loss, err := tape.BCEWithLogitsLoss(logits, targets)
		if err != nil {
			t.Fatalf("step %d BCEWithLogitsLoss: %v", step, err)
		}
		if err := tape.Backward(loss); err != nil {
			t.Fatalf("step %d Backward: %v", step, err)
		}
		gradNorm, err := tape.ClipGradNorm(1)
		if err != nil {
			t.Fatalf("step %d ClipGradNorm: %v", step, err)
		}
		// Torch reads this rate before optimizer.step(); scheduler.step() runs after it.
		lr := schedule.LR(step)
		if err := tape.SGDMomentum(lr, 0.9); err != nil {
			t.Fatalf("step %d SGDMomentum: %v", step, err)
		}

		assertAutodiffValue(t, "step "+itoa(step)+" lr", []float32{lr}, []float32{reference.LearningRate})
		assertAutodiffValue(t, "step "+itoa(step)+" loss", loss.data, []float32{reference.Loss})
		assertAutodiffValue(t, "step "+itoa(step)+" pre-clip grad norm", []float32{gradNorm}, []float32{reference.GradNorm})
		maxParameterDifference := float32(0)
		for _, name := range strings.Fields(parameterNames) {
			got := parameters[name].Value().data
			want := reference.Parameters[name]
			assertAutodiffValue(t, "step "+itoa(step)+" "+name, got, want)
			for index := range got {
				maxParameterDifference = maxFloat32(maxParameterDifference, float32(math.Abs(float64(got[index]-want[index]))))
			}
		}
		t.Logf("toolkit step=%d lr=%.9g torch_lr=%.9g loss=%.9g torch_loss=%.9g grad_norm=%.9g torch_grad_norm=%.9g max_param_abs_diff=%.9g", step, lr, reference.LearningRate, loss.data[0], reference.Loss, gradNorm, reference.GradNorm, maxParameterDifference)
	}
}
