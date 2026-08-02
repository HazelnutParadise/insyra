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
	Shape      []int     `json:"shape"`
	DType      string    `json:"dtype"`
	Data       []float32 `json:"data"`
	Int64Data  []int64   `json:"-"`
	BoolData   []bool    `json:"-"`
	StringData []string  `json:"-"`
}

type parityInput struct {
	Name  string          `json:"name"`
	Shape []int           `json:"shape"`
	DType string          `json:"dtype"`
	Data  json.RawMessage `json:"data"`
}

func (output *parityOutput) UnmarshalJSON(data []byte) error {
	var wire struct {
		Shape []int           `json:"shape"`
		DType string          `json:"dtype"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	output.Shape, output.DType = wire.Shape, wire.DType
	switch wire.DType {
	case "int64", "int32", "uint8":
		return json.Unmarshal(wire.Data, &output.Int64Data)
	case "bool":
		return json.Unmarshal(wire.Data, &output.BoolData)
	case "string":
		return json.Unmarshal(wire.Data, &output.StringData)
	default:
		return json.Unmarshal(wire.Data, &output.Data)
	}
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
					mustTestTensor(t, []int{2, 1, 2, 3}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}),
					mustTestTensor(t, []int{1, 3, 3, 2}, []float32{1, 2, 3, 4, 5, 6, 2, 1, 4, 3, 6, 5, 3, 2, 5, 4, 7, 6}),
				)
			},
		},
		{
			name: "ConvAutoPadNotSet",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 1, 5, 5}, scaledRange(25, 0.1)),
					mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, -1, 0.5, 2}),
					nil,
					ConvOptions{AutoPad: "NOTSET", Pads: []int{1, 0, 0, 1}},
				)
			},
		},
		{
			name: "ConvAutoPadSameUpper",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 1, 5, 5}, scaledRange(25, 0.1)),
					mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, -1, 0.5, 2}),
					nil,
					ConvOptions{AutoPad: "SAME_UPPER", Strides: []int{2, 2}},
				)
			},
		},
		{
			name: "ConvAutoPadSameLower",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 1, 5, 5}, scaledRange(25, 0.1)),
					mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, -1, 0.5, 2}),
					nil,
					ConvOptions{AutoPad: "SAME_LOWER", Strides: []int{2, 2}},
				)
			},
		},
		{
			name: "ConvAutoPadValid",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 1, 5, 5}, scaledRange(25, 0.1)),
					mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, -1, 0.5, 2}),
					nil,
					ConvOptions{AutoPad: "VALID"},
				)
			},
		},
		{
			name: "ConvStrides",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 1, 5, 5}, scaledRange(25, 0.1)),
					mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, -1, 0.5, 2}),
					nil,
					ConvOptions{AutoPad: "NOTSET", Strides: []int{2, 2}},
				)
			},
		},
		{
			name: "ConvDilations",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 1, 5, 5}, scaledRange(25, 0.1)),
					mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, -1, 0.5, 2}),
					nil,
					ConvOptions{AutoPad: "NOTSET", Dilations: []int{2, 2}},
				)
			},
		},
		{
			name: "ConvDepthwise",
			run: func() (*Tensor, error) {
				return Conv(
					mustTestTensor(t, []int{1, 2, 5, 5}, scaledRange(50, 0.1)),
					mustTestTensor(t, []int{2, 1, 2, 2}, []float32{1, 0, 0, -1, 2, 1, -1, 0.5}),
					nil,
					ConvOptions{AutoPad: "NOTSET", Group: 2, Pads: []int{1, 1, 1, 1}},
				)
			},
		},
		{
			name: "MaxPoolStridesPads",
			run: func() (*Tensor, error) {
				return MaxPool(
					mustTestTensor(t, []int{1, 1, 3, 3}, scaledRange(9, 1)),
					[]int{2, 2},
					PoolOptions{Pads: []int{1, 0, 0, 1}, Strides: []int{2, 1}},
				)
			},
		},
		{
			name: "AveragePoolExcludePad",
			run: func() (*Tensor, error) {
				return AveragePool(
					mustTestTensor(t, []int{1, 1, 3, 3}, scaledRange(9, 1)),
					[]int{2, 2},
					PoolOptions{Pads: []int{1, 0, 0, 1}, Strides: []int{2, 1}},
				)
			},
		},
		{
			name: "AveragePoolIncludePad",
			run: func() (*Tensor, error) {
				return AveragePool(
					mustTestTensor(t, []int{1, 1, 3, 3}, scaledRange(9, 1)),
					[]int{2, 2},
					PoolOptions{Pads: []int{1, 0, 0, 1}, Strides: []int{2, 1}, CountIncludePad: true},
				)
			},
		},
		{
			name: "GlobalAveragePool",
			run: func() (*Tensor, error) {
				return GlobalAveragePool(mustTestTensor(t, []int{1, 2, 2, 3}, scaledRange(12, 1)))
			},
		},
		{
			name: "BatchNormalizationEpsilon",
			run: func() (*Tensor, error) {
				return BatchNormalization(
					mustTestTensor(t, []int{1, 2, 1, 2}, []float32{1, 3, 10, 14}),
					mustTestTensor(t, []int{2}, []float32{2, 0.5}),
					mustTestTensor(t, []int{2}, []float32{1, -1}),
					mustTestTensor(t, []int{2}, []float32{1, 12}),
					mustTestTensor(t, []int{2}, []float32{4, 4}),
					0.001,
				)
			},
		},
		{
			name: "PadAttributes",
			run: func() (*Tensor, error) {
				return Pad(mustTestTensor(t, []int{1, 2}, []float32{1, 2}), []int{1, 0, 2, 1}, 0.5)
			},
		},
		{
			name: "PadInitializers",
			run: func() (*Tensor, error) {
				return Pad(mustTestTensor(t, []int{1, 2}, []float32{1, 2}), []int{1, 0, 2, 1}, 0.5)
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
			name: "Div",
			run: func() (*Tensor, error) {
				return Div(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{2, 4, 5}))
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
			name: "LayerNormalization",
			run: func() (*Tensor, error) {
				return LayerNormalization(
					mustTestTensor(t, []int{2, 3, 4}, []float32{-1, 0, 1, 2, 2, 1, 0, -1, 0.5, -0.5, 1.5, -1.5, 3, 2, 1, 0, -2, -1, 0, 1, 1.5, 0.5, -0.5, -1.5}),
					mustTestTensor(t, []int{4}, []float32{1, 0.5, 2, -1}),
					mustTestTensor(t, []int{4}, []float32{0.1, -0.2, 0.3, 0.4}),
					-1, 1e-5,
				)
			},
		},
		{
			name: "LayerNormalizationAxis1",
			run: func() (*Tensor, error) {
				return LayerNormalization(
					mustTestTensor(t, []int{2, 3, 4}, []float32{-1, 0, 1, 2, 2, 1, 0, -1, 0.5, -0.5, 1.5, -1.5, 3, 2, 1, 0, -2, -1, 0, 1, 1.5, 0.5, -0.5, -1.5}),
					mustTestTensor(t, []int{3, 4}, []float32{1, 0.5, 2, -1, 0.75, 1.25, -0.5, 0.25, 1.5, -0.25, 0.5, 2}),
					mustTestTensor(t, []int{3, 4}, []float32{0.1, -0.2, 0.3, 0.4, -0.1, 0.2, -0.3, 0.5, 0.25, -0.4, 0.15, 0.05}),
					1, 1e-5,
				)
			},
		},
		{
			name: "Gelu",
			run:  func() (*Tensor, error) { return Gelu(mustTestTensor(t, []int{2, 3}, []float32{-2, -1, 0, 0.5, 1, 2})) },
		},
		{
			name: "Erf",
			run:  func() (*Tensor, error) { return Erf(mustTestTensor(t, []int{2, 3}, []float32{-2, -1, 0, 0.5, 1, 2})) },
		},
		{
			name: "Sqrt",
			run:  func() (*Tensor, error) { return Sqrt(mustTestTensor(t, []int{2, 3}, []float32{0, 1, 4, 9, 16, 25})) },
		},
		{
			name: "Pow",
			run: func() (*Tensor, error) {
				return Pow(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{1, 2, 0.5}))
			},
		},
		{
			name: "ReduceMean",
			run: func() (*Tensor, error) {
				return ReduceMean(mustTestTensor(t, []int{2, 3, 2}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}), []int{-1}, true)
			},
		},
		{
			name: "ReduceMeanMultiAxes",
			run: func() (*Tensor, error) {
				return ReduceMean(mustTestTensor(t, []int{2, 3, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}), []int{0, 2}, true)
			},
		},
		{
			name: "ReduceMeanNoKeepdims",
			run: func() (*Tensor, error) {
				return ReduceMean(mustTestTensor(t, []int{2, 3, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}), []int{0, 2}, false)
			},
		},
		{
			name: "ReduceMeanInitializer",
			run: func() (*Tensor, error) {
				return ReduceMean(mustTestTensor(t, []int{2, 3, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}), []int{0, 2}, false)
			},
		},
		{
			name: "Softmax",
			run: func() (*Tensor, error) {
				return Softmax(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5}), 0)
			},
		},
		{
			name: "Identity",
			run: func() (*Tensor, error) {
				return Identity(mustTestTensor(t, []int{2, 3}, []float32{-1, 0, 1, 2, -2, 0.5}))
			},
		},
		{
			name: "Concat",
			run: func() (*Tensor, error) {
				return Concat([]*Tensor{
					mustTestTensor(t, []int{2, 1}, []float32{1, 2}),
					mustTestTensor(t, []int{2, 2}, []float32{3, 4, 5, 6}),
				}, 1)
			},
		},
		{
			name: "Unsqueeze",
			run: func() (*Tensor, error) {
				return Unsqueeze(mustTestTensor(t, []int{3}, []float32{1, 2, 3}), []int{-1})
			},
		},
		{
			name: "Squeeze",
			run: func() (*Tensor, error) {
				return Squeeze(mustTestTensor(t, []int{1, 2, 1}, []float32{1, 2}), []int{-1})
			},
		},
		{
			name: "Expand",
			run: func() (*Tensor, error) {
				return Expand(mustTestTensor(t, []int{2, 1}, []float32{1, 2}), []int{2, 3})
			},
		},
		{
			name: "Shape",
			run:  func() (*Tensor, error) { return Shape(mustTestTensor(t, []int{2, 3, 4}, make([]float32, 24))) },
		},
		{
			name: "Slice",
			run: func() (*Tensor, error) {
				return Slice(
					mustTestTensor(t, []int{3, 4}, []float32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
					[]int64{0, 1}, []int64{3, 4}, []int64{0, 1}, []int64{1, 2},
				)
			},
		},
		{
			name: "Equal",
			run: func() (*Tensor, error) {
				return Equal(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{1, 2, 4}))
			},
		},
		{
			name: "Greater",
			run: func() (*Tensor, error) {
				return Greater(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestTensor(t, []int{3}, []float32{0, 2, 4}))
			},
		},
		{
			name: "Gather",
			run: func() (*Tensor, error) {
				return Gather(mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}), mustTestInt64Tensor(t, []int{1}, []int64{2}), 1)
			},
		},
		{
			name: "GreaterOrEqual",
			run: func() (*Tensor, error) {
				return GreaterOrEqual(mustTestInt64Tensor(t, []int{3}, []int64{1, 2, 3}), mustTestInt64Tensor(t, []int{}, []int64{2}))
			},
		},
		{
			name: "Where",
			run: func() (*Tensor, error) {
				return Where(mustTestBoolTensor(t, []int{3}, []bool{true, false, true}), mustTestTensor(t, []int{3}, []float32{1, 2, 3}), mustTestTensor(t, []int{}, []float32{-1}))
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
				return Transpose(mustTestTensor(t, []int{2, 3, 2}, []float32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}), []int{2, 0, 1})
			},
		},
		{
			name: "Cast",
			run: func() (*Tensor, error) {
				return Cast(mustTestInt64Tensor(t, []int{1, 3}, []int64{-1, 0, 1}), DTypeFloat32)
			},
		},
		{
			name: "OneHotEncoder",
			run: func() (*Tensor, error) {
				return oneHotEncoder(mustTestStringTensor(t, []int{3}, []string{"red", "blue", "unknown"}), map[string]protoAttribute{
					"cats_strings": {strings: [][]byte{[]byte("red"), []byte("blue")}},
				})
			},
		},
		{
			name: "LabelEncoder",
			run: func() (*Tensor, error) {
				return labelEncoder(mustTestStringTensor(t, []int{3}, []string{"red", "blue", "unknown"}), map[string]protoAttribute{
					"keys_strings":  {strings: [][]byte{[]byte("red"), []byte("blue")}},
					"values_int64s": {ints: []int64{1, 2}},
					"default_int64": {intValue: -1, hasInt: true},
				})
			},
		},
		{
			name: "Scaler",
			run: func() (*Tensor, error) {
				return scaler(mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4}), map[string]protoAttribute{
					"offset": {floats: []float32{-1, 1}},
					"scale":  {floats: []float32{2, 3}},
				})
			},
		},
		{
			name: "LinearRegressor",
			run: func() (*Tensor, error) {
				return linearRegressor(mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4}), map[string]protoAttribute{
					"coefficients":   {floats: []float32{2, -1}},
					"intercepts":     {floats: []float32{0.5}},
					"targets":        {intValue: 1, hasInt: true},
					"post_transform": {string: []byte("NONE")},
				})
			},
		},
		{
			name: "LinearClassifier",
			run: func() (*Tensor, error) {
				_, probabilities, err := linearClassifier(mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4}), map[string]protoAttribute{
					"classlabels_ints": {ints: []int64{0, 1}},
					"coefficients":     {floats: []float32{2, -1, -1, 2}},
					"intercepts":       {floats: []float32{0.5, 0.1}},
					"post_transform":   {string: []byte("LOGISTIC")},
				})
				return probabilities, err
			},
		},
		{
			name: "TreeEnsembleRegressor",
			run: func() (*Tensor, error) {
				return treeEnsembleRegressor(mustTestTensor(t, []int{2, 1}, []float32{-1, 1}), simpleTreeRegressorAttributes())
			},
		},
		{
			name: "TreeEnsembleClassifier",
			run: func() (*Tensor, error) {
				_, probabilities, err := treeEnsembleClassifier(mustTestTensor(t, []int{2, 1}, []float32{-1, 1}), simpleTreeClassifierAttributes())
				return probabilities, err
			},
		},
		{
			name: "Constant",
			run:  func() (*Tensor, error) { return Constant(mustTestTensor(t, []int{2, 2}, []float32{1.5, -2, 3, 4.25})) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			modelPath := filepath.Join(t.TempDir(), tc.name+".onnx")
			payloadPath := filepath.Join(t.TempDir(), tc.name+"-feed.json")
			reference := runONNXParityPython(t, python, "one-op", tc.name, modelPath, payloadPath)
			if len(reference) != 1 {
				t.Fatalf("reference returned %d outputs, want 1", len(reference))
			}
			got, err := tc.run()
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			assertParityOutput(t, got, reference[0])
			modelBytes, err := os.ReadFile(modelPath)
			if err != nil {
				t.Fatalf("read generated %s model: %v", tc.name, err)
			}
			model, err := LoadONNX(bytes.NewReader(modelBytes))
			if err != nil {
				t.Fatalf("load generated %s model: %v", tc.name, err)
			}
			outputs, err := model.Run(readParityInputs(t, payloadPath))
			if err != nil {
				t.Fatalf("run generated %s model: %v", tc.name, err)
			}
			modelOutputs := model.Outputs()
			if len(modelOutputs) != 1 {
				t.Fatalf("generated %s model declares %d graph outputs, want 1", tc.name, len(modelOutputs))
			}
			assertParityOutput(t, outputs[modelOutputs[0].Name], reference[0])
		})
	}
}

func TestSplitParityAgainstONNXRuntime(t *testing.T) {
	python := requireONNXReference(t)
	if python == "" {
		return
	}
	reference := runONNXParityPython(t, python, "one-op", "Split")
	if len(reference) != 2 {
		t.Fatalf("Split reference returned %d outputs, want 2", len(reference))
	}
	outputs, err := Split(mustTestTensor(t, []int{2, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8}), []int{2, 2}, 1)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	for index, output := range outputs {
		assertParityOutput(t, output, reference[index])
	}
	modelPath := filepath.Join(t.TempDir(), "Split.onnx")
	payloadPath := filepath.Join(t.TempDir(), "Split-feed.json")
	runONNXParityPython(t, python, "one-op", "Split", modelPath, payloadPath)
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read generated Split model: %v", err)
	}
	model, err := LoadONNX(bytes.NewReader(modelBytes))
	if err != nil {
		t.Fatalf("load generated Split model: %v", err)
	}
	modelOutputs, err := model.Run(readParityInputs(t, payloadPath))
	if err != nil {
		t.Fatalf("run generated Split model: %v", err)
	}
	for index, spec := range model.Outputs() {
		assertParityOutput(t, modelOutputs[spec.Name], reference[index])
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
		want := parityOutput{Shape: []int{1, 2}, DType: reference[0].DType, Data: reference[0].Data[row*2 : row*2+2]}
		assertParityOutput(t, rowOutput["Z"], want)
	}
}

func TestWholeTransformerEncoderParity(t *testing.T) {
	python := requireONNXReference(t)
	if python == "" {
		return
	}
	modelPath := filepath.Join(t.TempDir(), "encoder.onnx")
	reference := runONNXParityPython(t, python, "encoder", modelPath)
	if len(reference) != 1 {
		t.Fatalf("encoder reference returned %d outputs, want 1", len(reference))
	}
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read generated encoder: %v", err)
	}
	model, err := LoadONNX(bytes.NewReader(modelBytes))
	if err != nil {
		t.Fatalf("LoadONNX encoder: %v", err)
	}
	input := mustTestTensor(t, []int{1, 2, 4}, []float32{0.25, -0.5, 1, 0.75, -1.25, 0.5, 0.25, 1.5})
	outputs, err := model.Run(map[string]*Tensor{"X": input})
	if err != nil {
		t.Fatalf("Run encoder: %v", err)
	}
	assertParityOutput(t, outputs["Y"], reference[0])
}

func TestWholeCNNParity(t *testing.T) {
	python := requireONNXReference(t)
	if python == "" {
		return
	}
	modelPath := filepath.Join(t.TempDir(), "cnn.onnx")
	reference := runONNXParityPython(t, python, "cnn", modelPath)
	if len(reference) != 1 {
		t.Fatalf("cnn reference returned %d outputs, want 1", len(reference))
	}
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read generated cnn: %v", err)
	}
	model, err := LoadONNX(bytes.NewReader(modelBytes))
	if err != nil {
		t.Fatalf("LoadONNX cnn: %v", err)
	}
	inputData := make([]float32, 64)
	for index := range inputData {
		inputData[index] = (float32(index) - 32) / 32
	}
	outputs, err := model.Run(map[string]*Tensor{"X": mustTestTensor(t, []int{1, 1, 8, 8}, inputData)})
	if err != nil {
		t.Fatalf("Run cnn: %v", err)
	}
	assertParityOutput(t, outputs["Y"], reference[0])
}

func TestRealModelSmoke(t *testing.T) {
	path := os.Getenv("INSYRA_DL_REAL_MODEL")
	if path == "" {
		t.Skip("INSYRA_DL_REAL_MODEL is not set")
	}
	modelFile, err := os.Open(path)
	if err != nil {
		t.Fatalf("open real model %q: %v", path, err)
	}
	defer func() {
		if closeErr := modelFile.Close(); closeErr != nil {
			t.Logf("close real model %q: %v", path, closeErr)
		}
	}()
	model, err := LoadONNX(modelFile)
	if err != nil {
		t.Fatalf("load real model %q: %v", path, err)
	}
	inputs := make(map[string]*Tensor, len(model.Inputs()))
	for _, spec := range model.Inputs() {
		inputs[spec.Name] = smokeInputTensor(t, spec)
	}
	outputs, err := model.Run(inputs)
	if err != nil {
		t.Fatalf("run real model %q: %v", path, err)
	}
	for _, spec := range model.Outputs() {
		output, present := outputs[spec.Name]
		if !present {
			t.Fatalf("real model output %q was not returned", spec.Name)
		}
		fmt.Printf("INSYRA_DL_REAL_MODEL output %q shape %v\n", spec.Name, output.Shape())
	}
}

func smokeInputTensor(t *testing.T, spec ValueInfo) *Tensor {
	t.Helper()
	shape := append([]int(nil), spec.Shape...)
	for index, dimension := range shape {
		if dimension < 0 {
			shape[index] = 1
		}
	}
	if !spec.HasShape {
		shape = []int{1}
	}
	count := elementCount(shape)
	switch spec.DType {
	case DTypeFloat32:
		return mustTestTensor(t, shape, make([]float32, count))
	case DTypeInt64:
		values := make([]int64, count)
		if strings.Contains(strings.ToLower(spec.Name), "mask") {
			for index := range values {
				values[index] = 1
			}
		}
		return mustTestInt64Tensor(t, shape, values)
	case DTypeBool:
		values := make([]bool, count)
		if strings.Contains(strings.ToLower(spec.Name), "mask") {
			for index := range values {
				values[index] = true
			}
		}
		return mustTestBoolTensor(t, shape, values)
	case DTypeString:
		return mustTestStringTensor(t, shape, make([]string, count))
	default:
		t.Fatalf("real model input %q has unsupported dtype %s", spec.Name, spec.DType)
		return nil
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

func readParityInputs(t *testing.T, path string) map[string]*Tensor {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read parity feed: %v", err)
	}
	var payload []parityInput
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode parity feed: %v", err)
	}
	inputs := make(map[string]*Tensor, len(payload))
	for _, input := range payload {
		var tensor *Tensor
		var tensorErr error
		switch input.DType {
		case "float32":
			var values []float32
			if err := json.Unmarshal(input.Data, &values); err != nil {
				t.Fatalf("decode float32 feed %q: %v", input.Name, err)
			}
			tensor, tensorErr = NewTensor(input.Shape, values)
		case "int64":
			var values []int64
			if err := json.Unmarshal(input.Data, &values); err != nil {
				t.Fatalf("decode int64 feed %q: %v", input.Name, err)
			}
			tensor, tensorErr = NewInt64Tensor(input.Shape, values)
		case "bool":
			var values []bool
			if err := json.Unmarshal(input.Data, &values); err != nil {
				t.Fatalf("decode bool feed %q: %v", input.Name, err)
			}
			tensor, tensorErr = NewBoolTensor(input.Shape, values)
		case "string":
			var values []string
			if err := json.Unmarshal(input.Data, &values); err != nil {
				t.Fatalf("decode string feed %q: %v", input.Name, err)
			}
			tensor, tensorErr = NewStringTensor(input.Shape, values)
		default:
			t.Fatalf("parity feed %q has unsupported dtype %q", input.Name, input.DType)
		}
		if tensorErr != nil {
			t.Fatalf("build parity feed %q: %v", input.Name, tensorErr)
		}
		inputs[input.Name] = tensor
	}
	return inputs
}

func assertParityOutput(t *testing.T, got *Tensor, want parityOutput) {
	t.Helper()
	if got == nil {
		t.Fatal("got nil tensor")
	}
	if fmt.Sprint(got.shape) != fmt.Sprint(want.Shape) {
		t.Fatalf("shape = %v, want %v", got.shape, want.Shape)
	}
	switch got.dtype {
	case DTypeFloat32:
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
	case DTypeInt64:
		if fmt.Sprint(got.int64Data) != fmt.Sprint(want.Int64Data) {
			t.Fatalf("data = %v, want %v", got.int64Data, want.Int64Data)
		}
	case DTypeBool:
		if fmt.Sprint(got.boolData) != fmt.Sprint(want.BoolData) {
			t.Fatalf("data = %v, want %v", got.boolData, want.BoolData)
		}
	case DTypeString:
		if fmt.Sprint(got.stringData) != fmt.Sprint(want.StringData) {
			t.Fatalf("data = %v, want %v", got.stringData, want.StringData)
		}
	default:
		t.Fatalf("unsupported output dtype %s", got.dtype)
	}
}

func mustTestInt64Tensor(t *testing.T, shape []int, data []int64) *Tensor {
	t.Helper()
	tensor, err := NewInt64Tensor(shape, data)
	if err != nil {
		t.Fatalf("NewInt64Tensor: %v", err)
	}
	return tensor
}

func scaledRange(count int, scale float32) []float32 {
	values := make([]float32, count)
	for index := range values {
		values[index] = float32(index+1) * scale
	}
	return values
}

func mustTestStringTensor(t *testing.T, shape []int, data []string) *Tensor {
	t.Helper()
	tensor, err := NewStringTensor(shape, data)
	if err != nil {
		t.Fatalf("NewStringTensor: %v", err)
	}
	return tensor
}

func mustTestBoolTensor(t *testing.T, shape []int, data []bool) *Tensor {
	t.Helper()
	tensor, err := NewBoolTensor(shape, data)
	if err != nil {
		t.Fatalf("NewBoolTensor: %v", err)
	}
	return tensor
}

func simpleTreeRegressorAttributes() map[string]protoAttribute {
	return map[string]protoAttribute{
		"nodes_treeids":                   {ints: []int64{0, 0, 0}},
		"nodes_nodeids":                   {ints: []int64{0, 1, 2}},
		"nodes_featureids":                {ints: []int64{0, 0, 0}},
		"nodes_values":                    {floats: []float32{0, 0, 0}},
		"nodes_modes":                     {strings: [][]byte{[]byte("BRANCH_LEQ"), []byte("LEAF"), []byte("LEAF")}},
		"nodes_truenodeids":               {ints: []int64{1, 0, 0}},
		"nodes_falsenodeids":              {ints: []int64{2, 0, 0}},
		"nodes_missing_value_tracks_true": {ints: []int64{0, 0, 0}},
		"target_treeids":                  {ints: []int64{0, 0}},
		"target_nodeids":                  {ints: []int64{1, 2}},
		"target_ids":                      {ints: []int64{0, 0}},
		"target_weights":                  {floats: []float32{-1, 2}},
		"n_targets":                       {intValue: 1, hasInt: true},
		"post_transform":                  {string: []byte("NONE")},
	}
}

func simpleTreeClassifierAttributes() map[string]protoAttribute {
	attributes := simpleTreeRegressorAttributes()
	attributes["classlabels_ints"] = protoAttribute{ints: []int64{0, 1}}
	delete(attributes, "target_treeids")
	delete(attributes, "target_nodeids")
	delete(attributes, "target_ids")
	delete(attributes, "target_weights")
	delete(attributes, "n_targets")
	attributes["class_treeids"] = protoAttribute{ints: []int64{0, 0, 0, 0}}
	attributes["class_nodeids"] = protoAttribute{ints: []int64{1, 1, 2, 2}}
	attributes["class_ids"] = protoAttribute{ints: []int64{0, 1, 0, 1}}
	attributes["class_weights"] = protoAttribute{floats: []float32{1, 0, 0, 1}}
	return attributes
}
