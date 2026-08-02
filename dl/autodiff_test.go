package dl

import (
	"fmt"
	"math"
	"testing"
)

func TestAutodiffFiniteDifference(t *testing.T) {
	parameters := finiteDifferenceParameters(t)
	tape, loss := finiteDifferenceGraph(t, parameters)
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	const h = float32(1e-2)
	tolerance := float32(5e-3 + 2*h*h)
	for name, parameter := range parameters {
		gradient, err := tape.Grad(parameter)
		if err != nil {
			t.Fatalf("gradient %s: %v", name, err)
		}
		for index, want := range gradient.data {
			original := parameter.data[index]
			parameter.data[index] = original + h
			plus := finiteDifferenceLoss(t, parameters)
			parameter.data[index] = original - h
			minus := finiteDifferenceLoss(t, parameters)
			parameter.data[index] = original
			finite := (plus - minus) / (2 * h)
			if math.Abs(float64(want-finite)) > float64(tolerance)+0.01*math.Abs(float64(finite)) {
				t.Fatalf("%s[%d] tape=%g finite-difference=%g tolerance=%g", name, index, want, finite, tolerance)
			}
		}
	}
}

func TestAutodiffGemmTransposeGradients(t *testing.T) {
	cases := []GemmOptions{
		{Alpha: 0.7, Beta: 1.2},
		{Alpha: -0.5, Beta: 0.4, TransA: true},
		{Alpha: 1.3, Beta: -0.8, TransB: true},
		{Alpha: 0.9, Beta: 0.6, TransA: true, TransB: true},
	}
	for _, options := range cases {
		t.Run(fmt.Sprintf("transA=%t/transB=%t", options.TransA, options.TransB), func(t *testing.T) {
			var aShape, bShape []int
			if options.TransA {
				aShape = []int{3, 2}
			} else {
				aShape = []int{2, 3}
			}
			if options.TransB {
				bShape = []int{4, 3}
			} else {
				bShape = []int{3, 4}
			}
			a := mustTestTensor(t, aShape, []float32{0.2, -0.4, 0.7, 0.3, -0.6, 0.5})
			b := mustTestTensor(t, bShape, []float32{0.1, -0.2, 0.3, 0.4, -0.5, 0.6, 0.7, -0.8, 0.9, 0.2, -0.1, 0.5})
			c := mustTestTensor(t, []int{4}, []float32{0.1, -0.2, 0.3, -0.4})
			tape := NewTape()
			a = mustTapeParameter(t, tape, a)
			b = mustTapeParameter(t, tape, b)
			c = mustTapeParameter(t, tape, c)
			output, err := tape.Gemm(a, b, c, options)
			if err != nil {
				t.Fatal(err)
			}
			loss, err := tape.SoftmaxCrossEntropy(output, mustTestInt64Tensor(t, []int{2}, []int64{1, 3}))
			if err != nil {
				t.Fatal(err)
			}
			if err := tape.Backward(loss); err != nil {
				t.Fatal(err)
			}
			for _, parameter := range []*Tensor{a, b, c} {
				gradient, err := tape.Grad(parameter)
				if err != nil || gradient.Len() != parameter.Len() {
					t.Fatalf("gradient shape for %v: %v, err %v", parameter.Shape(), gradient.Shape(), err)
				}
			}
		})
	}
}

func TestAutodiffSGD(t *testing.T) {
	parameter := mustTestTensor(t, []int{2, 2}, []float32{1, -2, 0.5, 0.25})
	tape := NewTape()
	marked, err := tape.Param(parameter)
	if err != nil {
		t.Fatalf("Param: %v", err)
	}
	parameter = marked.Value()
	input := mustTestTensor(t, []int{1, 2}, []float32{2, 3})
	logits, err := tape.MatMul(input, parameter)
	if err != nil {
		t.Fatalf("MatMul: %v", err)
	}
	loss, err := tape.SoftmaxCrossEntropy(logits, mustTestInt64Tensor(t, []int{1}, []int64{0}))
	if err != nil {
		t.Fatalf("SoftmaxCrossEntropy: %v", err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	gradient := marked.Grad()
	if gradient == nil {
		t.Fatal("Grad returned nil")
	}
	before := parameter.Data()
	if err := tape.SGD(0.1); err != nil {
		t.Fatalf("SGD: %v", err)
	}
	after := parameter.Data()
	for index := range after {
		want := before[index] - 0.1*gradient.data[index]
		if after[index] != want {
			t.Fatalf("parameter[%d] = %g, want %g", index, after[index], want)
		}
	}
}

func finiteDifferenceParameters(t *testing.T) map[string]*Tensor {
	t.Helper()
	return map[string]*Tensor{
		"w1": mustTestTensor(t, []int{2, 3}, []float32{0.4, -0.3, 0.2, 0.1, 0.5, -0.6}),
		"b1": mustTestTensor(t, []int{3}, []float32{0.1, -0.2, 0.3}),
		"w2": mustTestTensor(t, []int{3, 2}, []float32{0.2, -0.4, 0.5, 0.3, -0.6, 0.7}),
		"b2": mustTestTensor(t, []int{2}, []float32{0.05, -0.15}),
		"wg": mustTestTensor(t, []int{2, 2}, []float32{0.3, -0.2, 0.4, 0.1}),
		"bg": mustTestTensor(t, []int{2}, []float32{0.2, -0.1}),
	}
}

func finiteDifferenceGraph(t *testing.T, parameters map[string]*Tensor) (*Tape, *Tensor) {
	t.Helper()
	x := mustTestTensor(t, []int{3, 2}, []float32{0.2, -1, 1.1, 0.3, -0.7, 0.8})
	labels := mustTestInt64Tensor(t, []int{3}, []int64{0, 1, 0})
	tape := NewTape()
	w1 := mustTapeParameter(t, tape, parameters["w1"])
	b1 := mustTapeParameter(t, tape, parameters["b1"])
	w2 := mustTapeParameter(t, tape, parameters["w2"])
	b2 := mustTapeParameter(t, tape, parameters["b2"])
	wg := mustTapeParameter(t, tape, parameters["wg"])
	bg := mustTapeParameter(t, tape, parameters["bg"])
	first, err := tape.MatMul(x, w1)
	if err != nil {
		t.Fatal(err)
	}
	first, err = tape.Add(first, b1)
	if err != nil {
		t.Fatal(err)
	}
	first, err = tape.Relu(first)
	if err != nil {
		t.Fatal(err)
	}
	first, err = tape.Sigmoid(first)
	if err != nil {
		t.Fatal(err)
	}
	first, err = tape.Tanh(first)
	if err != nil {
		t.Fatal(err)
	}
	logits, err := tape.MatMul(first, w2)
	if err != nil {
		t.Fatal(err)
	}
	logits, err = tape.Add(logits, b2)
	if err != nil {
		t.Fatal(err)
	}
	gemm, err := tape.Gemm(x, wg, bg, GemmOptions{Alpha: 0.7, Beta: 1.2})
	if err != nil {
		t.Fatal(err)
	}
	logits, err = tape.Add(logits, gemm)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		t.Fatal(err)
	}
	return tape, loss
}

func finiteDifferenceLoss(t *testing.T, parameters map[string]*Tensor) float32 {
	t.Helper()
	_, loss := finiteDifferenceGraph(t, parameters)
	return loss.data[0]
}

func mustTapeParameter(t *testing.T, tape *Tape, value *Tensor) *Tensor {
	t.Helper()
	parameter, err := tape.Param(value)
	if err != nil {
		t.Fatal(err)
	}
	return parameter.Value()
}
