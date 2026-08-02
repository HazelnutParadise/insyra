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

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

//go:embed testdata/onnx_parity.py
var onnxParityScript string

type parityOutput struct {
	Shape []int     `json:"shape"`
	Data  []float32 `json:"data"`
}

func TestOneOpParityAgainstONNXRuntime(t *testing.T) {
	python := requireONNXReference(t)
	if python == "" {
		return
	}
	cases := []struct {
		name string
		run  func() (*Tensor, error)
	}{
		{
			name: "Gemm",
			run: func() (*Tensor, error) {
				return Gemm(
					mustTestTensor(t, []int{2, 3}, []float32{1, -2, 3, 4, 5, -6}),
					mustTestTensor(t, []int{3, 2}, []float32{1, 2, 3, 4, 5, 6}),
					mustTestTensor(t, []int{2}, []float32{0.5, -1}),
					GemmOptions{Alpha: 1.25, Beta: 0.5},
				)
			},
		},
		{
			name: "MatMul",
			run: func() (*Tensor, error) {
				return MatMul(
					mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}),
					mustTestTensor(t, []int{3, 2}, []float32{1, 2, 3, 4, 5, 6}),
				)
			},
		},
		{
			name: "Add",
			run: func() (*Tensor, error) {
				return Add(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{10, 20, 30}))
			},
		},
		{
			name: "Sub",
			run: func() (*Tensor, error) {
				return Sub(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{10, 20, 30}))
			},
		},
		{
			name: "Mul",
			run: func() (*Tensor, error) {
				return Mul(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{10, 20, 30}))
			},
		},
		{
			name: "Relu",
			run:  func() (*Tensor, error) { return Relu(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5})) },
		},
		{
			name: "Sigmoid",
			run: func() (*Tensor, error) {
				return Sigmoid(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5}))
			},
		},
		{
			name: "Tanh",
			run:  func() (*Tensor, error) { return Tanh(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5})) },
		},
		{
			name: "Softmax",
			run: func() (*Tensor, error) {
				return Softmax(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5}), 1)
			},
		},
		{
			name: "Identity",
			run: func() (*Tensor, error) {
				return Identity(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5}))
			},
		},
		{
			name: "Reshape",
			run: func() (*Tensor, error) {
				return Reshape(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), []int{3, 2})
			},
		},
		{
			name: "Flatten",
			run: func() (*Tensor, error) {
				return Flatten(mustTestTensor(t, []int{2, 3, 2}, []float32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}), 1)
			},
		},
		{
			name: "Transpose",
			run: func() (*Tensor, error) {
				return Transpose(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), []int{1, 0})
			},
		},
		{
			name: "Cast",
			run: func() (*Tensor, error) {
				return Cast(mustTestTensor(t, []int{1, 3}, []float32{-1, 0, 1}), DTypeFloat32)
			},
		},
		{
			name: "Constant",
			run:  func() (*Tensor, error) { return Constant(mustTestTensor(t, []int{2, 2}, []float32{1.5, -2, 3, 4.25})) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reference := runONNXParityPython(t, python, "one-op", tc.name)
			if len(reference) != 1 {
				t.Fatalf("reference returned %d outputs, want 1", len(reference))
			}
			got, err := tc.run()
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			assertParityOutput(t, got, reference[0])
		})
	}
}

func TestWholeMLPParityAndBatchInvariance(t *testing.T) {
	python := requireONNXReference(t)
	if python == "" {
		return
	}
	modelPath := filepath.Join(t.TempDir(), "mlp.onnx")
	reference := runONNXParityPython(t, python, "mlp", modelPath)
	if len(reference) != 1 {
		t.Fatalf("reference returned %d outputs, want 1", len(reference))
	}
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read generated model: %v", err)
	}
	model, err := LoadONNX(bytes.NewReader(modelBytes))
	if err != nil {
		t.Fatalf("LoadONNX: %v", err)
	}
	inputData := []float32{0.5, -1, 2, 1.5, 0.25, -0.75, -2, 1, 0.5}
	input := mustTestTensor(t, []int{3, 3}, inputData)
	outputs, err := model.Run(map[string]*Tensor{"X": input})
	if err != nil {
		t.Fatalf("Run batch: %v", err)
	}
	assertParityOutput(t, outputs["Z"], reference[0])

	for row := 0; row < 3; row++ {
		rowInput := mustTestTensor(t, []int{1, 3}, inputData[row*3:row*3+3])
		rowOutput, runErr := model.Run(map[string]*Tensor{"X": rowInput})
		if runErr != nil {
			t.Fatalf("Run row %d: %v", row, runErr)
		}
		want := parityOutput{Shape: []int{1, 2}, Data: reference[0].Data[row*2 : row*2+2]}
		assertParityOutput(t, rowOutput["Z"], want)
	}
}

func requireONNXReference(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		reftest.Missing(t, "python3", "the dl ONNX parity checks", err)
		return ""
	}
	command := exec.Command(python, "-c", "import numpy, onnx, onnxruntime")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with numpy, onnx, and onnxruntime", "the dl ONNX parity checks", err, stderr.Bytes())
		return ""
	}
	return python
}

func runONNXParityPython(t *testing.T, python string, args ...string) []parityOutput {
	t.Helper()
	commandArgs := append([]string{"-c", onnxParityScript}, args...)
	command := exec.Command(python, commandArgs...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("onnxruntime helper: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var result []parityOutput
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode helper stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	return result
}

func assertParityOutput(t *testing.T, got *Tensor, want parityOutput) {
	t.Helper()
	if got == nil {
		t.Fatal("got nil tensor")
	}
	if fmt.Sprint(got.shape) != fmt.Sprint(want.Shape) {
		t.Fatalf("shape = %v, want %v", got.shape, want.Shape)
	}
	if len(got.data) != len(want.Data) {
		t.Fatalf("data length = %d, want %d", len(got.data), len(want.Data))
	}
	for index := range want.Data {
		difference := math.Abs(float64(got.data[index] - want.Data[index]))
		scale := math.Max(1, math.Abs(float64(want.Data[index])))
		if difference > 1e-5*scale {
			t.Fatalf("data[%d] = %g, want %g (difference %g)", index, got.data[index], want.Data[index], difference)
		}
	}
}
