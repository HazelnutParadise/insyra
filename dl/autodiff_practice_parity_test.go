package dl

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
)

//go:embed testdata/autodiff_practice_fixture.py
var autodiffPracticeFixtureScript string

type autodiffPracticeReference struct {
	Steps []autodiffPracticeStep `json:"steps"`
}

type autodiffPracticeStep struct {
	LearningRate float32              `json:"lr"`
	Loss         float32              `json:"loss"`
	Parameters   map[string][]float32 `json:"parameters"`
}

func TestAutodiffPracticeAdamWStepLRParity(t *testing.T) {
	python := requireAutodiffReference(t)
	if python == "" {
		return
	}
	want := runAutodiffPracticeFixture(t, python)
	if len(want.Steps) != 5 {
		t.Fatalf("fixture returned %d steps, want 5", len(want.Steps))
	}
	weightsPath := filepath.Join(t.TempDir(), "autodiff-practice.safetensors")
	if err := writeAutodiffPracticeWeights(t, python, weightsPath); err != nil {
		t.Fatal(err)
	}
	weights := loadPracticeWeights(t, weightsPath)

	decoupled := runPracticeTrajectory(t, weights, func(tape *Tape, rate float32) error {
		return tape.AdamW(rate, 1e-2)
	})
	for step, reference := range want.Steps {
		assertAutodiffValue(t, "practice loss", []float32{decoupled[step].loss}, []float32{reference.Loss})
		if decoupled[step].rate != reference.LearningRate {
			t.Fatalf("step %d lr = %g, want %g", step, decoupled[step].rate, reference.LearningRate)
		}
		for _, name := range strings.Fields(practiceParameterNames) {
			assertAutodiffValue(t, "step "+itoa(step)+" "+name, decoupled[step].parameters[name], reference.Parameters[name])
		}
		t.Logf("practice step=%d lr=%.9g loss=%.9g w1[0]=%.9g (reference %.9g)", step, decoupled[step].rate, decoupled[step].loss, decoupled[step].parameters["w1"][0], reference.Parameters["w1"][0])
	}

	coupled := runPracticeTrajectory(t, loadPracticeWeights(t, weightsPath), func(tape *Tape, rate float32) error {
		for _, parameter := range tape.params {
			gradient := tape.grads[parameter.value]
			for index := range gradient.data {
				gradient.data[index] += 1e-2 * parameter.value.data[index]
			}
		}
		return tape.Adam(rate)
	})
	maxDifference := float32(0)
	for _, name := range strings.Fields(practiceParameterNames) {
		decoupledValue := decoupled[len(decoupled)-1].parameters[name]
		coupledValue := coupled[len(coupled)-1].parameters[name]
		for index := range decoupledValue {
			maxDifference = maxFloat32(maxDifference, float32(math.Abs(float64(decoupledValue[index]-coupledValue[index]))))
		}
	}
	t.Logf("coupled-vs-decoupled step-5 max parameter difference=%.9g", maxDifference)
	if maxDifference <= 1e-6 {
		t.Fatalf("coupled L2 and decoupled AdamW did not diverge: max difference %g", maxDifference)
	}
}

const practiceParameterNames = "w1 b1 w2 b2"

type practiceTrajectoryStep struct {
	rate       float32
	loss       float32
	parameters map[string][]float32
}

func runPracticeTrajectory(t *testing.T, weights map[string]*Tensor, update func(*Tape, float32) error) []practiceTrajectoryStep {
	t.Helper()
	tape := NewTape()
	parameters := make(map[string]*Parameter)
	for _, name := range strings.Fields(practiceParameterNames) {
		parameter, err := tape.Param(weights[name])
		if err != nil {
			t.Fatalf("Param %s: %v", name, err)
		}
		parameters[name] = parameter
	}
	input := mustTestTensor(t, []int{4, 3}, []float32{0.2, -0.7, 1.1, 0.4, 0.3, -0.8, -0.6, 0.9, 0.5, 1.2, -0.1, -0.4})
	labels := mustTestInt64Tensor(t, []int{4}, []int64{2, 0, 1, 2})
	schedule, err := NewStepLR(1e-3, 0.5, 2)
	if err != nil {
		t.Fatal(err)
	}
	trajectory := make([]practiceTrajectoryStep, 0, 5)
	for step := 0; step < 5; step++ {
		tape.ops = nil
		tape.grads = make(map[*Tensor]*Tensor)
		hidden, err := tape.MatMul(input, parameters["w1"].Value())
		if err != nil {
			t.Fatal(err)
		}
		hidden, err = tape.Add(hidden, parameters["b1"].Value())
		if err != nil {
			t.Fatal(err)
		}
		hidden, err = tape.Tanh(hidden)
		if err != nil {
			t.Fatal(err)
		}
		logits, err := tape.MatMul(hidden, parameters["w2"].Value())
		if err != nil {
			t.Fatal(err)
		}
		logits, err = tape.Add(logits, parameters["b2"].Value())
		if err != nil {
			t.Fatal(err)
		}
		loss, err := tape.SoftmaxCrossEntropy(logits, labels)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.Backward(loss); err != nil {
			t.Fatal(err)
		}
		rate := schedule.LR(step)
		if err := update(tape, rate); err != nil {
			t.Fatalf("step %d optimizer: %v", step, err)
		}
		values := make(map[string][]float32, len(parameters))
		for _, name := range strings.Fields(practiceParameterNames) {
			values[name] = parameters[name].Value().Data()
		}
		trajectory = append(trajectory, practiceTrajectoryStep{rate: rate, loss: loss.data[0], parameters: values})
	}
	return trajectory
}

func runAutodiffPracticeFixture(t *testing.T, python string) autodiffPracticeReference {
	t.Helper()
	path := filepath.Join(t.TempDir(), "autodiff-practice.safetensors")
	command := exec.Command(python, "-c", autodiffPracticeFixtureScript, path)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("autodiff practice fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var reference autodiffPracticeReference
	if err := json.Unmarshal(stdout.Bytes(), &reference); err != nil {
		t.Fatalf("decode autodiff practice stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	return reference
}

func writeAutodiffPracticeWeights(t *testing.T, python, path string) error {
	t.Helper()
	command := exec.Command(python, "-c", autodiffPracticeFixtureScript, path)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("write autodiff practice weights: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func loadPracticeWeights(t *testing.T, path string) map[string]*Tensor {
	t.Helper()
	file, err := os.Open(path)
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
	return weights
}

func maxFloat32(left, right float32) float32 {
	if left > right {
		return left
	}
	return right
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
