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

//go:embed testdata/attention_layer_fixture.py
var attentionLayerFixtureScript string

type attentionForwardReference struct {
	InputShape  []int     `json:"input_shape"`
	Input       []float32 `json:"input"`
	OutputShape []int     `json:"output_shape"`
	Output      []float32 `json:"output"`
}

type attentionEncoderReference struct {
	Loss           float32              `json:"loss"`
	ParameterNames []string             `json:"parameter_names"`
	Gradients      map[string][]float32 `json:"gradients"`
	Parameters     map[string][]float32 `json:"parameters"`
}

func TestMultiHeadAttentionTorchWeightRoundTrip(t *testing.T) {
	python := requireAttentionReference(t, "the nn MultiHeadAttention parity check")
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "attention.safetensors")
	var reference attentionForwardReference
	runAttentionFixture(t, python, "mha", weightsPath, &reference)
	weights := loadAttentionFixtureWeights(t, weightsPath)

	model, err := NewSequential(NewTape(11), MultiHeadAttention(16, 4))
	if err != nil {
		t.Fatalf("NewSequential: %v", err)
	}
	if err := model.LoadWeights(prefixAttentionWeights(weights, "0.")); err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	input := mustTestTensor(t, reference.InputShape, reference.Input)
	output, err := model.Predict(input)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	assertAutodiffValue(t, "MultiHeadAttention forward", output.data, reference.Output)
	for _, name := range []string{"0.in_proj_weight", "0.in_proj_bias", "0.out_proj.weight", "0.out_proj.bias"} {
		if model.NamedParameters()[name] == nil {
			t.Fatalf("NamedParameters missing %q", name)
		}
	}

	var saved bytes.Buffer
	if err := model.SaveWeights(&saved); err != nil {
		t.Fatalf("SaveWeights: %v", err)
	}
	reloadedWeights, err := LoadSafeTensors(bytes.NewReader(saved.Bytes()))
	if err != nil {
		t.Fatalf("LoadSafeTensors saved MHA: %v", err)
	}
	reloaded, err := NewSequential(NewTape(23), MultiHeadAttention(16, 4))
	if err != nil {
		t.Fatalf("NewSequential reloaded: %v", err)
	}
	if err := reloaded.LoadWeights(reloadedWeights); err != nil {
		t.Fatalf("LoadWeights reloaded: %v", err)
	}
	reloadedOutput, err := reloaded.Predict(input)
	if err != nil {
		t.Fatalf("Predict reloaded: %v", err)
	}
	if !sameFloat32(output.data, reloadedOutput.data) {
		t.Fatalf("saved/reloaded MHA output differs")
	}
}

func TestLayerBuiltEncoderAdamWParity(t *testing.T) {
	python := requireAttentionReference(t, "the nn layer-built encoder parity check")
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "attention-encoder.safetensors")
	var reference attentionEncoderReference
	runAttentionFixture(t, python, "encoder", weightsPath, &reference)
	weights := loadAttentionFixtureWeights(t, weightsPath)

	tape := NewTape(31)
	encoder, err := NewSequential(tape,
		Residual(MultiHeadAttention(16, 4)),
		LayerNorm(16),
		Residual(Dense(16, 32), NewGelu(), Dense(32, 16)),
		LayerNorm(16),
	)
	if err != nil {
		t.Fatalf("NewSequential encoder: %v", err)
	}
	if err := encoder.LoadWeights(selectWeights(weights, "0.", "1.", "2.", "3.")); err != nil {
		t.Fatalf("LoadWeights encoder: %v", err)
	}
	head, err := NewSequential(tape, Dense(16, 3))
	if err != nil {
		t.Fatalf("NewSequential head: %v", err)
	}
	if err := head.LoadWeights(rebaseWeights(selectWeights(weights, "4."), "4.", "0.")); err != nil {
		t.Fatalf("LoadWeights head: %v", err)
	}
	input := mustTestTensor(t, []int{2, 5, 16}, fixedAttentionInput())
	encoded, err := encoder.Forward(tape, input)
	if err != nil {
		t.Fatalf("encoder Forward: %v", err)
	}
	pooled, err := tape.ReduceMean(encoded, []int{1}, false)
	if err != nil {
		t.Fatalf("ReduceMean: %v", err)
	}
	logits, err := head.Forward(tape, pooled)
	if err != nil {
		t.Fatalf("head Forward: %v", err)
	}
	labels := mustTestInt64Tensor(t, []int{2}, []int64{1, 2})
	loss, err := tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		t.Fatalf("SoftmaxCrossEntropy: %v", err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	assertAutodiffValue(t, "layer-built encoder loss", loss.data, []float32{reference.Loss})
	parameters := make(map[string]*Parameter)
	for name, parameter := range encoder.NamedParameters() {
		parameters[name] = parameter
	}
	for name, parameter := range head.NamedParameters() {
		parameters["4."+strings.TrimPrefix(name, "0.")] = parameter
	}
	for _, name := range reference.ParameterNames {
		parameter := parameters[name]
		if parameter == nil {
			t.Fatalf("missing parameter %q", name)
		}
		assertAutodiffValue(t, name+" gradient", attentionTorchLayout(name, parameter.Grad()), reference.Gradients[name])
	}
	if err := tape.AdamW(1e-3, 1e-2); err != nil {
		t.Fatalf("AdamW: %v", err)
	}
	for _, name := range reference.ParameterNames {
		assertAttentionValue(t, name+" post-step parameter", attentionTorchLayout(name, parameters[name].Value()), reference.Parameters[name])
	}
}

func assertAttentionValue(t *testing.T, name string, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", name, len(got), len(want))
	}
	for index := range got {
		difference := math.Abs(float64(got[index] - want[index]))
		scale := math.Max(1, math.Abs(float64(want[index])))
		// The torch reference itself varies ~3e-5 between Linux/x86 (MKL)
		// and macOS/ARM builds, and AdamW's rsqrt amplifies it at the step;
		// a bound tighter than the reference's own platform variance asserts
		// noise. 1e-4 stays strict for a one-step encoder comparison.
		if difference > 1e-4*scale {
			t.Fatalf("%s[%d] = %g, want %g (difference %g)", name, index, got[index], want[index], difference)
		}
	}
}

func attentionTorchLayout(name string, value *Tensor) []float32 {
	if !strings.HasSuffix(name, "weight") || len(value.shape) != 2 {
		return value.data
	}
	return transposeSequentialWeight(value.data, value.shape[0], value.shape[1])
}

func TestMultiHeadAttentionFiniteDifferences(t *testing.T) {
	tape := NewTape(7)
	model, err := NewSequential(tape, MultiHeadAttention(4, 2))
	if err != nil {
		t.Fatalf("NewSequential: %v", err)
	}
	input := mustTestTensor(t, []int{1, 3, 4}, []float32{
		0.2, -0.4, 0.7, 1.1,
		-0.6, 0.3, 0.5, -0.8,
		0.9, -1.2, 0.4, 0.1,
	})
	target := mustTestTensor(t, []int{1, 3, 4}, []float32{
		0.5, -0.2, 0.7, -0.3,
		-0.4, 0.6, 0.1, 0.8,
		0.2, -0.9, 0.3, 0.4,
	})
	tape.ops = nil
	output, err := model.Forward(tape, input)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	loss, err := tape.ReduceMean(mustTapeMul(t, tape, output, target), []int{0, 1, 2}, false)
	if err != nil {
		t.Fatalf("ReduceMean: %v", err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	for parameterIndex, parameter := range model.Parameters() {
		for valueIndex, original := range parameter.value.data {
			const step = float32(1e-3)
			parameter.value.data[valueIndex] = original + step
			plus := attentionObjective(t, model, input, target)
			parameter.value.data[valueIndex] = original - step
			minus := attentionObjective(t, model, input, target)
			parameter.value.data[valueIndex] = original
			finite := (plus - minus) / (2 * step)
			analytic := parameter.Grad().data[valueIndex]
			if difference := math.Abs(float64(analytic - finite)); difference > 2e-2+2e-2*math.Abs(float64(finite)) {
				t.Fatalf("parameter %d gradient[%d] tape=%g finite-difference=%g", parameterIndex, valueIndex, analytic, finite)
			}
		}
	}
}

func TestMultiHeadAttentionConstructionResidualAndONNXRefusal(t *testing.T) {
	if _, err := NewSequential(NewTape(), MultiHeadAttention(5, 2)); err == nil || !strings.Contains(err.Error(), "MultiHeadAttention") || !strings.Contains(err.Error(), "divisible") {
		t.Fatalf("divisibility error = %v", err)
	}
	residual := Residual(BatchNorm2D(1))
	model, err := NewSequential(NewTape(), residual)
	if err != nil {
		t.Fatalf("NewSequential residual: %v", err)
	}
	normalization := residual.(*residualLayer).layers[0].(*batchNorm2DLayer)
	normalization.runningMean.data[0] = 1
	normalization.runningVariance.data[0] = 1
	input := mustTestTensor(t, []int{1, 1, 1, 1}, []float32{2})
	output, err := model.Predict(input)
	if err != nil {
		t.Fatalf("Residual Predict: %v", err)
	}
	if len(output.data) != 1 || math.Abs(float64(output.data[0]-3)) > 1e-5 {
		t.Fatalf("Residual EvalLayer output = %v, want [3]", output.data)
	}
	for _, tc := range []struct {
		name  string
		layer Layer
	}{
		{name: "MultiHeadAttention", layer: MultiHeadAttention(4, 2)},
		{name: "Residual", layer: Residual(Dense(4, 4))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model, err := NewSequential(NewTape(), Dense(4, 4), tc.layer)
			if err != nil {
				t.Fatal(err)
			}
			if err := model.ExportONNX(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "layer 1") || !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("ExportONNX error = %v", err)
			}
		})
	}
}

func requireAttentionReference(t *testing.T, purpose string) string {
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

func runAttentionFixture(t *testing.T, python, mode, weightsPath string, result interface{}) {
	t.Helper()
	command := exec.Command(python, "-c", attentionLayerFixtureScript, mode, weightsPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("attention fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	if err := json.Unmarshal(stdout.Bytes(), result); err != nil {
		t.Fatalf("decode attention fixture stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
}

func loadAttentionFixtureWeights(t *testing.T, path string) map[string]*Tensor {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open attention weights: %v", err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors attention weights: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close attention weights: %v", closeErr)
	}
	return weights
}

func prefixAttentionWeights(weights map[string]*Tensor, prefix string) map[string]*Tensor {
	prefixed := make(map[string]*Tensor, len(weights))
	for name, tensor := range weights {
		prefixed[prefix+name] = tensor
	}
	return prefixed
}

func selectWeights(weights map[string]*Tensor, prefixes ...string) map[string]*Tensor {
	selected := make(map[string]*Tensor)
	for name, tensor := range weights {
		for _, prefix := range prefixes {
			if strings.HasPrefix(name, prefix) {
				selected[name] = tensor
				break
			}
		}
	}
	return selected
}

func rebaseWeights(weights map[string]*Tensor, from, to string) map[string]*Tensor {
	rebased := make(map[string]*Tensor, len(weights))
	for name, tensor := range weights {
		rebased[to+strings.TrimPrefix(name, from)] = tensor
	}
	return rebased
}

func fixedAttentionInput() []float32 {
	values := make([]float32, 2*5*16)
	for index := range values {
		values[index] = float32(index-37) / 19
	}
	return values
}

func attentionObjective(t *testing.T, model *Sequential, input, target *Tensor) float32 {
	t.Helper()
	tape := model.tape
	tape.ops = nil
	tape.grads = make(map[*Tensor]*Tensor)
	output, err := model.Forward(tape, input)
	if err != nil {
		t.Fatalf("finite-difference Forward: %v", err)
	}
	product, err := Mul(output, target)
	if err != nil {
		t.Fatalf("finite-difference Mul: %v", err)
	}
	loss, err := ReduceMean(product, []int{0, 1, 2}, false)
	if err != nil {
		t.Fatalf("finite-difference ReduceMean: %v", err)
	}
	return loss.data[0]
}
