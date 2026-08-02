package dl

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

//go:embed testdata/autodiff_encoder_fixture.py
var autodiffEncoderFixtureScript string

type autodiffEncoderReference struct {
	Loss           float32              `json:"loss"`
	ParameterNames []string             `json:"parameter_names"`
	Gradients      map[string][]float32 `json:"gradients"`
	Parameters     map[string][]float32 `json:"parameters"`
}

func TestAutodiffEncoderAdamParity(t *testing.T) {
	python := requireAutodiffReference(t)
	if python == "" {
		return
	}
	weightsPath := filepath.Join(t.TempDir(), "autodiff-encoder.safetensors")
	command := exec.Command(python, "-c", autodiffEncoderFixtureScript, weightsPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("autodiff encoder fixture: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var want autodiffEncoderReference
	if err := json.Unmarshal(stdout.Bytes(), &want); err != nil {
		t.Fatalf("decode autodiff encoder stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}

	file, err := os.Open(weightsPath)
	if err != nil {
		t.Fatalf("open autodiff encoder weights: %v", err)
	}
	weights, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close autodiff encoder weights: %v", closeErr)
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
	loss := buildAutodiffEncoder(t, tape, parameters)
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	assertAutodiffValue(t, "encoder loss", loss.data, []float32{want.Loss})
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

func buildAutodiffEncoder(t *testing.T, tape *Tape, parameters map[string]*Parameter) *Tensor {
	t.Helper()
	parameter := func(name string) *Tensor {
		value := parameters[name]
		if value == nil {
			t.Fatalf("missing encoder parameter %q", name)
		}
		return value.Value()
	}
	input := mustTestTensor(t, []int{2, 2, 4}, []float32{
		0.25, -0.5, 1.0, 0.75,
		-1.25, 0.5, 0.25, 1.5,
		0.6, -0.2, 0.9, -0.7,
		-0.4, 1.2, -0.8, 0.3,
	})

	q := tapeEncoderMatMulAdd(t, tape, input, parameter("Wq"), parameter("bq"))
	q = tapeEncoderReshape(t, tape, q, []int{2, 2, 2, 2})
	q = tapeEncoderTranspose(t, tape, q, []int{0, 2, 1, 3})
	k := tapeEncoderMatMulAdd(t, tape, input, parameter("Wk"), parameter("bk"))
	k = tapeEncoderReshape(t, tape, k, []int{2, 2, 2, 2})
	k = tapeEncoderTranspose(t, tape, k, []int{0, 2, 3, 1})
	scores := tapeEncoderMatMul(t, tape, q, k)
	scale := mustTestTensor(t, nil, []float32{float32(math.Sqrt(2))})
	scaled := tapeEncoderDiv(t, tape, scores, scale)
	probability := tapeEncoderSoftmax(t, tape, scaled, -1)
	v := tapeEncoderMatMulAdd(t, tape, input, parameter("Wv"), parameter("bv"))
	v = tapeEncoderReshape(t, tape, v, []int{2, 2, 2, 2})
	v = tapeEncoderTranspose(t, tape, v, []int{0, 2, 1, 3})
	context := tapeEncoderMatMul(t, tape, probability, v)
	context = tapeEncoderTranspose(t, tape, context, []int{0, 2, 1, 3})
	context = tapeEncoderReshape(t, tape, context, []int{2, 2, 4})
	attention := tapeEncoderMatMulAdd(t, tape, context, parameter("Wo"), parameter("bo"))
	firstResidual := tapeEncoderAdd(t, tape, input, attention)
	firstNorm := tapeEncoderLayerNorm(t, tape, firstResidual, parameter("gamma1"), parameter("beta1"))
	hidden := tapeEncoderMatMulAdd(t, tape, firstNorm, parameter("W1"), parameter("b1"))
	hidden = tapeEncoderGelu(t, tape, hidden)
	feedForward := tapeEncoderMatMulAdd(t, tape, hidden, parameter("W2"), parameter("b2"))
	secondResidual := tapeEncoderAdd(t, tape, firstNorm, feedForward)
	encoded := tapeEncoderLayerNorm(t, tape, secondResidual, parameter("gamma2"), parameter("beta2"))
	pooled := tapeEncoderReduceMean(t, tape, encoded, []int{1}, false)
	logits := tapeEncoderMatMulAdd(t, tape, pooled, parameter("head_w"), parameter("head_b"))
	labels := mustTestInt64Tensor(t, []int{2}, []int64{1, 2})
	return tapeEncoderCrossEntropy(t, tape, logits, labels)
}

func tapeEncoderMatMulAdd(t *testing.T, tape *Tape, input, weight, bias *Tensor) *Tensor {
	return tapeEncoderAdd(t, tape, tapeEncoderMatMul(t, tape, input, weight), bias)
}

func tapeEncoderMatMul(t *testing.T, tape *Tape, left, right *Tensor) *Tensor {
	output, err := tape.MatMul(left, right)
	if err != nil {
		t.Fatalf("MatMul: %v", err)
	}
	return output
}

func tapeEncoderAdd(t *testing.T, tape *Tape, left, right *Tensor) *Tensor {
	output, err := tape.Add(left, right)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return output
}

func tapeEncoderDiv(t *testing.T, tape *Tape, left, right *Tensor) *Tensor {
	output, err := tape.Div(left, right)
	if err != nil {
		t.Fatalf("Div: %v", err)
	}
	return output
}

func tapeEncoderReshape(t *testing.T, tape *Tape, input *Tensor, shape []int) *Tensor {
	output, err := tape.Reshape(input, shape)
	if err != nil {
		t.Fatalf("Reshape: %v", err)
	}
	return output
}

func tapeEncoderTranspose(t *testing.T, tape *Tape, input *Tensor, perm []int) *Tensor {
	output, err := tape.Transpose(input, perm)
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	return output
}

func tapeEncoderSoftmax(t *testing.T, tape *Tape, input *Tensor, axis int) *Tensor {
	output, err := tape.Softmax(input, axis)
	if err != nil {
		t.Fatalf("Softmax: %v", err)
	}
	return output
}

func tapeEncoderLayerNorm(t *testing.T, tape *Tape, input, scale, bias *Tensor) *Tensor {
	output, err := tape.LayerNormalization(input, scale, bias, -1, 1e-5)
	if err != nil {
		t.Fatalf("LayerNormalization: %v", err)
	}
	return output
}

func tapeEncoderGelu(t *testing.T, tape *Tape, input *Tensor) *Tensor {
	output, err := tape.Gelu(input)
	if err != nil {
		t.Fatalf("Gelu: %v", err)
	}
	return output
}

func tapeEncoderReduceMean(t *testing.T, tape *Tape, input *Tensor, axes []int, keepdims bool) *Tensor {
	output, err := tape.ReduceMean(input, axes, keepdims)
	if err != nil {
		t.Fatalf("ReduceMean: %v", err)
	}
	return output
}

func tapeEncoderCrossEntropy(t *testing.T, tape *Tape, logits, labels *Tensor) *Tensor {
	loss, err := tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		t.Fatalf("SoftmaxCrossEntropy: %v", err)
	}
	return loss
}
