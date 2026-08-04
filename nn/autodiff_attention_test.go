package nn

import (
	"fmt"
	"math"
	"testing"
)

func TestAttentionVJPsFiniteDifference(t *testing.T) {
	tests := []struct {
		name  string
		param *Tensor
		build func(*testing.T, *Tensor) (*Tape, *Tensor)
	}{
		{
			name:  "batched matmul broadcast",
			param: mustTestTensor(t, []int{2, 1, 2, 3}, scaledRange(12, 0.07)),
			build: func(t *testing.T, a *Tensor) (*Tape, *Tensor) {
				b := mustTestTensor(t, []int{1, 4, 3, 2}, scaledRange(24, -0.05))
				tape := NewTape()
				a = mustTapeParameter(t, tape, a)
				b = mustTapeParameter(t, tape, b)
				output, err := tape.MatMul(a, b)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "axis softmax",
			param: mustTestTensor(t, []int{2, 3}, []float32{-0.7, 0.2, 1.1, 0.4, -0.9, 0.8}),
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.Softmax(input, 1)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "layer normalization input",
			param: mustTestTensor(t, []int{2, 3}, []float32{-1.2, 0.3, 1.1, 0.7, -0.4, 1.5}),
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				scale := mustTestTensor(t, []int{3}, []float32{0.8, 1.1, -0.6})
				bias := mustTestTensor(t, []int{3}, []float32{0.1, -0.2, 0.3})
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.LayerNormalization(input, scale, bias, -1, 1e-5)
				if err != nil {
					t.Fatal(err)
				}
				return tape, weightedLoss(t, tape, output, []float32{0.7, -0.2, 1.3, -0.8, 0.4, 1.1})
			},
		},
		{
			name:  "layer normalization scale",
			param: mustTestTensor(t, []int{3}, []float32{0.8, 1.1, -0.6}),
			build: func(t *testing.T, scale *Tensor) (*Tape, *Tensor) {
				input := mustTestTensor(t, []int{2, 3}, []float32{-1.2, 0.3, 1.1, 0.7, -0.4, 1.5})
				bias := mustTestTensor(t, []int{3}, []float32{0.1, -0.2, 0.3})
				tape := NewTape()
				scale = mustTapeParameter(t, tape, scale)
				output, err := tape.LayerNormalization(input, scale, bias, -1, 1e-5)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "layer normalization bias",
			param: mustTestTensor(t, []int{3}, []float32{0.1, -0.2, 0.3}),
			build: func(t *testing.T, bias *Tensor) (*Tape, *Tensor) {
				input := mustTestTensor(t, []int{2, 3}, []float32{-1.2, 0.3, 1.1, 0.7, -0.4, 1.5})
				scale := mustTestTensor(t, []int{3}, []float32{0.8, 1.1, -0.6})
				tape := NewTape()
				bias = mustTapeParameter(t, tape, bias)
				output, err := tape.LayerNormalization(input, scale, bias, -1, 1e-5)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "gelu",
			param: mustTestTensor(t, []int{2, 2}, []float32{-1.2, -0.3, 0.4, 1.1}),
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.Gelu(input)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "erf",
			param: mustTestTensor(t, []int{2, 2}, []float32{-1.2, -0.3, 0.4, 1.1}),
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.Erf(input)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "sqrt",
			param: mustTestTensor(t, []int{2, 2}, []float32{0.5, 1.2, 2.3, 4.1}),
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.Sqrt(input)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "pow left",
			param: mustTestTensor(t, []int{2, 1}, []float32{0.7, 1.4}),
			build: func(t *testing.T, left *Tensor) (*Tape, *Tensor) {
				right := mustTestTensor(t, []int{1, 3}, []float32{1.2, 0.8, 1.7})
				tape := NewTape()
				left = mustTapeParameter(t, tape, left)
				output, err := tape.Pow(left, right)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:  "pow right",
			param: mustTestTensor(t, []int{1, 3}, []float32{1.2, 0.8, 1.7}),
			build: func(t *testing.T, right *Tensor) (*Tape, *Tensor) {
				left := mustTestTensor(t, []int{2, 1}, []float32{0.7, 1.4})
				tape := NewTape()
				right = mustTapeParameter(t, tape, right)
				output, err := tape.Pow(left, right)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkFiniteDifference(t, test.param, test.build)
		})
	}
}

func TestReduceMeanVJPsFiniteDifferenceBothKeepdims(t *testing.T) {
	for _, keepdims := range []bool{false, true} {
		t.Run(fmt.Sprintf("keepdims=%t", keepdims), func(t *testing.T) {
			parameter := mustTestTensor(t, []int{2, 3, 2}, scaledRange(12, 0.11))
			checkFiniteDifference(t, parameter, func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.ReduceMean(input, []int{0, 2}, keepdims)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			})
		})
	}
}

func TestShapeVJPChainFiniteDifference(t *testing.T) {
	parameter := mustTestTensor(t, []int{2, 3}, []float32{0.2, -0.4, 0.7, 1.1, -0.8, 0.5})
	checkFiniteDifference(t, parameter, func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
		tape := NewTape()
		input = mustTapeParameter(t, tape, input)
		value, err := tape.Transpose(input, []int{1, 0})
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Reshape(value, []int{1, 3, 2})
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Unsqueeze(value, []int{1})
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Squeeze(value, []int{1})
		if err != nil {
			t.Fatal(err)
		}
		parts, err := tape.Split(value, []int{1, 2}, 1)
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Concat(parts, 1)
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Slice(value, []int64{0}, []int64{3}, []int64{1}, []int64{1})
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Flatten(value, 1)
		if err != nil {
			t.Fatal(err)
		}
		value, err = tape.Reshape(value, []int{2, 3})
		if err != nil {
			t.Fatal(err)
		}
		return tape, meanLoss(t, tape, value)
	})
}

func TestAutodiffMatMulRankOneEdges(t *testing.T) {
	for _, test := range []struct {
		name string
		a    []int
		b    []int
	}{
		{name: "vector-vector", a: []int{3}, b: []int{3}},
		{name: "vector-matrix", a: []int{3}, b: []int{3, 2}},
		{name: "matrix-vector", a: []int{2, 3}, b: []int{3}},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := mustTestTensor(t, test.a, scaledRange(product(test.a), 0.13))
			b := mustTestTensor(t, test.b, scaledRange(product(test.b), -0.09))
			tape := NewTape()
			a = mustTapeParameter(t, tape, a)
			b = mustTapeParameter(t, tape, b)
			output, err := tape.MatMul(a, b)
			if err != nil {
				t.Fatal(err)
			}
			var loss *Tensor
			if len(output.shape) == 0 {
				loss = output
			} else {
				loss = meanLoss(t, tape, output)
			}
			if err := tape.Backward(loss); err != nil {
				t.Fatal(err)
			}
			for _, value := range []*Tensor{a, b} {
				gradient, err := tape.Grad(value)
				if err != nil || !sameShape(gradient.shape, value.shape) {
					t.Fatalf("gradient shape = %v, want %v, err=%v", gradient.Shape(), value.Shape(), err)
				}
			}
		})
	}
}

func TestAdamBiasCorrectionAndState(t *testing.T) {
	parameter := mustTestTensor(t, []int{2}, []float32{1, -2})
	tape := NewTape()
	marked, err := tape.Param(parameter)
	if err != nil {
		t.Fatal(err)
	}
	tape.grads[marked.Value()] = mustTestTensor(t, []int{2}, []float32{0.5, -0.25})
	if err := tape.Adam(0.1); err != nil {
		t.Fatal(err)
	}
	wantFirst := []float32{0.9, -1.9}
	assertAutodiffValue(t, "adam first step", parameter.data, wantFirst)
	if err := tape.Adam(0.1); err != nil {
		t.Fatal(err)
	}
	assertAutodiffValue(t, "adam second step", parameter.data, []float32{0.8, -1.8})
}

func TestAutodiffUnsupportedVJPNamesOperation(t *testing.T) {
	tape := NewTape()
	input := mustTestTensor(t, []int{1}, []float32{1})
	output := mustTestTensor(t, nil, []float32{1})
	tape.record("UnsupportedAttentionOp", []*Tensor{input}, output, nil)
	if err := tape.Backward(output); err == nil || !containsAll(err.Error(), "UnsupportedAttentionOp", "no VJP") {
		t.Fatalf("Backward error = %v, want named missing VJP", err)
	}
}

func checkFiniteDifference(t *testing.T, parameter *Tensor, build func(*testing.T, *Tensor) (*Tape, *Tensor)) {
	t.Helper()
	tape, loss := build(t, parameter)
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	gradient, err := tape.Grad(parameter)
	if err != nil {
		t.Fatal(err)
	}
	const h = float32(1e-3)
	for index, want := range gradient.data {
		original := parameter.data[index]
		parameter.data[index] = original + h
		plus := buildLoss(t, build, parameter)
		parameter.data[index] = original - h
		minus := buildLoss(t, build, parameter)
		parameter.data[index] = original
		finite := (plus - minus) / (2 * h)
		if math.Abs(float64(want-finite)) > 2e-2+2e-2*math.Abs(float64(finite)) {
			t.Fatalf("gradient[%d] tape=%g finite-difference=%g", index, want, finite)
		}
	}
}

func buildLoss(t *testing.T, build func(*testing.T, *Tensor) (*Tape, *Tensor), parameter *Tensor) float32 {
	t.Helper()
	_, loss := build(t, parameter)
	return loss.data[0]
}

func meanLoss(t *testing.T, tape *Tape, value *Tensor) *Tensor {
	t.Helper()
	loss, err := tape.ReduceMean(value, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return loss
}

func weightedLoss(t *testing.T, tape *Tape, value *Tensor, weights []float32) *Tensor {
	t.Helper()
	weight := mustTestTensor(t, value.shape, weights)
	product, err := tape.Mul(value, weight)
	if err != nil {
		t.Fatal(err)
	}
	return meanLoss(t, tape, product)
}

func product(shape []int) int {
	result := 1
	for _, dimension := range shape {
		result *= dimension
	}
	return result
}

func containsAll(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !contains(value, fragment) {
			return false
		}
	}
	return true
}

func contains(value, fragment string) bool {
	return len(value) >= len(fragment) && stringIndex(value, fragment) >= 0
}

func stringIndex(value, fragment string) int {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return index
		}
	}
	return -1
}
