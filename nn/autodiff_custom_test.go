package nn

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestCustomProductMatchesMul(t *testing.T) {
	w := mustTestTensor(t, []int{1}, []float32{3})
	x := mustTestTensor(t, []int{1}, []float32{2})
	zero := mustTestTensor(t, []int{1}, []float32{0})

	customTape := NewTape()
	customW := mustTapeParameter(t, customTape, w)
	y, err := Mul(customW, x)
	if err != nil {
		t.Fatal(err)
	}
	if err := customTape.Custom("product", []*Tensor{customW, x}, y, func(upstream *Tensor) ([]*Tensor, error) {
		dw, err := Mul(upstream, x)
		if err != nil {
			return nil, err
		}
		dx, err := Mul(upstream, customW)
		if err != nil {
			return nil, err
		}
		return []*Tensor{dw, dx}, nil
	}); err != nil {
		t.Fatal(err)
	}
	customLoss, err := customTape.MSELoss(y, zero)
	if err != nil {
		t.Fatal(err)
	}
	if err := customTape.Backward(customLoss); err != nil {
		t.Fatal(err)
	}

	builtInTape := NewTape()
	builtInW := mustTapeParameter(t, builtInTape, w)
	by, err := builtInTape.Mul(builtInW, x)
	if err != nil {
		t.Fatal(err)
	}
	builtInLoss, err := builtInTape.MSELoss(by, zero)
	if err != nil {
		t.Fatal(err)
	}
	if err := builtInTape.Backward(builtInLoss); err != nil {
		t.Fatal(err)
	}

	customGrad, err := customTape.Grad(customW)
	if err != nil {
		t.Fatal(err)
	}
	if len(customGrad.data) != 1 || customGrad.data[0] != 24 {
		t.Fatalf("custom gradient = %v, want [24]", customGrad.data)
	}
	builtInGrad, err := builtInTape.Grad(builtInW)
	if err != nil {
		t.Fatal(err)
	}
	for index := range customGrad.data {
		if customGrad.data[index] != builtInGrad.data[index] {
			t.Fatalf("custom gradient %v not bit-identical to built-in %v", customGrad.data, builtInGrad.data)
		}
	}
}

func TestCustomBetweenBuiltIns(t *testing.T) {
	x := mustTestTensor(t, []int{1, 2}, []float32{0.5, -0.75})
	W := mustTestTensor(t, []int{2, 2}, []float32{0.2, -0.4, 0.7, 0.3})
	target := mustTestTensor(t, []int{1, 2}, []float32{0.1, 0.9})

	customTape := NewTape()
	customW := mustTapeParameter(t, customTape, W)
	h, err := MatMul(x, customW)
	if err != nil {
		t.Fatal(err)
	}
	if err := customTape.Custom("matmul", []*Tensor{x, customW}, h, func(upstream *Tensor) ([]*Tensor, error) {
		return matMulVJP(x, customW, upstream)
	}); err != nil {
		t.Fatal(err)
	}
	a, err := customTape.Tanh(h)
	if err != nil {
		t.Fatal(err)
	}
	customLoss, err := customTape.MSELoss(a, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := customTape.Backward(customLoss); err != nil {
		t.Fatal(err)
	}

	builtInTape := NewTape()
	builtInW := mustTapeParameter(t, builtInTape, W)
	bh, err := builtInTape.MatMul(x, builtInW)
	if err != nil {
		t.Fatal(err)
	}
	ba, err := builtInTape.Tanh(bh)
	if err != nil {
		t.Fatal(err)
	}
	builtInLoss, err := builtInTape.MSELoss(ba, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := builtInTape.Backward(builtInLoss); err != nil {
		t.Fatal(err)
	}

	customGrad, err := customTape.Grad(customW)
	if err != nil {
		t.Fatal(err)
	}
	builtInGrad, err := builtInTape.Grad(builtInW)
	if err != nil {
		t.Fatal(err)
	}
	if len(customGrad.data) != len(builtInGrad.data) {
		t.Fatalf("gradient lengths differ: %d and %d", len(customGrad.data), len(builtInGrad.data))
	}
	for index := range customGrad.data {
		if customGrad.data[index] != builtInGrad.data[index] {
			t.Fatalf("gradient %v not bit-identical to built-in %v", customGrad.data, builtInGrad.data)
		}
	}
}

func TestCustomRefusesAnOutputAlreadyOnTheTape(t *testing.T) {
	w := mustTestTensor(t, []int{1}, []float32{3})
	x := mustTestTensor(t, []int{1}, []float32{2})
	zero := mustTestTensor(t, []int{1}, []float32{0})

	tape := NewTape()
	param, err := tape.Param(w)
	if err != nil {
		t.Fatal(err)
	}
	customW := param.Value()
	y, err := tape.Mul(customW, x)
	if err != nil {
		t.Fatal(err)
	}
	before := len(tape.ops)
	err = tape.Custom("dup", []*Tensor{customW, x}, y, func(upstream *Tensor) ([]*Tensor, error) {
		dw, err := Mul(upstream, x)
		if err != nil {
			return nil, err
		}
		dx, err := Mul(upstream, customW)
		if err != nil {
			return nil, err
		}
		return []*Tensor{dw, dx}, nil
	})
	if err == nil {
		t.Fatal("Custom(...) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "dup") || !strings.Contains(err.Error(), "Mul") {
		t.Fatalf("error %q does not name the operation and its producer", err)
	}
	if after := len(tape.ops); after != before {
		t.Fatalf("Custom(...) recorded %d ops, want %d", after, before)
	}

	loss, err := tape.MSELoss(y, zero)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	gradient, err := tape.Grad(customW)
	if err != nil {
		t.Fatal(err)
	}
	if len(gradient.data) != 1 || gradient.data[0] != 24 {
		t.Fatalf("gradient = %v, want [24]", gradient.data)
	}
}

func surrogateSigmoid(value float32) float32 {
	sigmoid := 1 / (1 + float32(math.Exp(-float64(value))))
	return sigmoid * (1 - sigmoid)
}

func TestCustomSurrogateGradient(t *testing.T) {
	x := mustTestTensor(t, []int{3}, []float32{-1, 0.5, 2})
	yData := make([]float32, 3)
	for index, value := range x.data {
		if value > 0 {
			yData[index] = 1
		}
	}
	y := mustTestTensor(t, []int{3}, yData)
	target := mustTestTensor(t, []int{3}, []float32{0, 0, 0})

	tape := NewTape()
	param := mustTapeParameter(t, tape, x)
	if err := tape.Custom("step", []*Tensor{param}, y, func(upstream *Tensor) ([]*Tensor, error) {
		scale := make([]float32, 3)
		for index, value := range x.data {
			scale[index] = surrogateSigmoid(value)
		}
		multiplier, err := newFloat32Tensor([]int{3}, scale)
		if err != nil {
			return nil, err
		}
		gradient, err := Mul(upstream, multiplier)
		if err != nil {
			return nil, err
		}
		return []*Tensor{gradient}, nil
	}); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.MSELoss(y, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}

	gradient, err := tape.Grad(param)
	if err != nil {
		t.Fatal(err)
	}
	sawNonzero := false
	for index, value := range gradient.data {
		expected := (2 * yData[index] / 3) * surrogateSigmoid(x.data[index])
		if value != expected {
			t.Fatalf("gradient[%d] = %g, want %g", index, value, expected)
		}
		if value != 0 {
			sawNonzero = true
		}
	}
	if !sawNonzero {
		t.Fatal("surrogate gradient is zero everywhere")
	}
}

func TestCustomRefusesMalformedDeclarations(t *testing.T) {
	x := mustTestTensor(t, []int{1}, []float32{2})
	other := mustTestTensor(t, []int{1}, []float32{3})
	b, err := NewBoolTensor([]int{1}, []bool{true})
	if err != nil {
		t.Fatal(err)
	}
	vjp := func(upstream *Tensor) ([]*Tensor, error) { return []*Tensor{upstream}, nil }
	cases := []struct {
		name    string
		inputs  []*Tensor
		output  *Tensor
		vjp     func(*Tensor) ([]*Tensor, error)
		wantErr string
	}{
		{name: "", inputs: []*Tensor{x}, output: other, vjp: vjp, wantErr: "tape custom operation needs a name"},
		{name: "no-vjp", inputs: []*Tensor{x}, output: other, vjp: nil, wantErr: "tape custom no-vjp: vjp is nil"},
		{name: "nil-output", inputs: []*Tensor{x}, output: nil, vjp: vjp, wantErr: "tape custom nil-output output tensor is nil"},
		{name: "nil-input", inputs: []*Tensor{nil}, output: other, vjp: vjp, wantErr: "tape custom nil-input input 0 tensor is nil"},
		{name: "bool-input", inputs: []*Tensor{b}, output: other, vjp: vjp, wantErr: "tape custom bool-input input 0 has unsupported dtype bool"},
		{name: "self-input", inputs: []*Tensor{x}, output: x, vjp: vjp, wantErr: "tape custom self-input: output is also input 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tape := NewTape()
			before := len(tape.ops)
			err := tape.Custom(tc.name, tc.inputs, tc.output, tc.vjp)
			if err == nil {
				t.Fatalf("Custom(%q, ...) succeeded, want error", tc.name)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("Custom(%q, ...) error = %q, want %q", tc.name, err.Error(), tc.wantErr)
			}
			if after := len(tape.ops); after != before {
				t.Fatalf("Custom(%q, ...) recorded %d ops, want %d", tc.name, after, before)
			}
		})
	}
}

func TestCustomBackwardRejectsMalformedGradients(t *testing.T) {
	zero := mustTestTensor(t, []int{3}, []float32{0, 0, 0})
	badShape := mustTestTensor(t, []int{2}, []float32{1, 2})
	boolGradient, err := NewBoolTensor([]int{3}, []bool{true, false, true})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("wrong-shape", func(t *testing.T) {
		x := mustTestTensor(t, []int{3}, []float32{1, 2, 3})
		tape := NewTape()
		output := mustTestTensor(t, []int{3}, []float32{4, 5, 6})
		if err := tape.Custom("badshape", []*Tensor{x}, output, func(upstream *Tensor) ([]*Tensor, error) {
			return []*Tensor{badShape}, nil
		}); err != nil {
			t.Fatal(err)
		}
		loss, err := tape.MSELoss(output, zero)
		if err != nil {
			t.Fatal(err)
		}
		err = tape.Backward(loss)
		if err == nil {
			t.Fatal("Backward succeeded, want error")
		}
		if !strings.Contains(err.Error(), "badshape") {
			t.Fatalf("error %q does not name the operation", err)
		}
		if !strings.Contains(err.Error(), "[2]") || !strings.Contains(err.Error(), "[3]") {
			t.Fatalf("error %q does not name both shapes", err)
		}
	})

	t.Run("too-few", func(t *testing.T) {
		x := mustTestTensor(t, []int{3}, []float32{1, 2, 3})
		u := mustTestTensor(t, []int{3}, []float32{4, 5, 6})
		tape := NewTape()
		out, err := Add(x, u)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.Custom("shortvec", []*Tensor{x, u}, out, func(upstream *Tensor) ([]*Tensor, error) {
			return []*Tensor{upstream}, nil
		}); err != nil {
			t.Fatal(err)
		}
		loss, err := tape.MSELoss(out, zero)
		if err != nil {
			t.Fatal(err)
		}
		err = tape.Backward(loss)
		if err == nil {
			t.Fatal("Backward succeeded, want error")
		}
		if !strings.Contains(err.Error(), "shortvec") {
			t.Fatalf("error %q does not name the operation", err)
		}
	})

	t.Run("bool-gradient", func(t *testing.T) {
		x := mustTestTensor(t, []int{3}, []float32{1, 2, 3})
		tape := NewTape()
		output := mustTestTensor(t, []int{3}, []float32{4, 5, 6})
		if err := tape.Custom("boolgrad", []*Tensor{x}, output, func(upstream *Tensor) ([]*Tensor, error) {
			return []*Tensor{boolGradient}, nil
		}); err != nil {
			t.Fatal(err)
		}
		loss, err := tape.MSELoss(output, zero)
		if err != nil {
			t.Fatal(err)
		}
		err = tape.Backward(loss)
		if err == nil {
			t.Fatal("Backward succeeded, want error")
		}
		if !strings.Contains(err.Error(), "boolgrad") {
			t.Fatalf("error %q does not name the operation", err)
		}
	})

	t.Run("sentinel", func(t *testing.T) {
		x := mustTestTensor(t, []int{3}, []float32{1, 2, 3})
		tape := NewTape()
		output := mustTestTensor(t, []int{3}, []float32{4, 5, 6})
		sentinel := errors.New("vjp exploded")
		if err := tape.Custom("sentinelop", []*Tensor{x}, output, func(upstream *Tensor) ([]*Tensor, error) {
			return nil, sentinel
		}); err != nil {
			t.Fatal(err)
		}
		loss, err := tape.MSELoss(output, zero)
		if err != nil {
			t.Fatal(err)
		}
		err = tape.Backward(loss)
		if err == nil {
			t.Fatal("Backward succeeded, want error")
		}
		if !errors.Is(err, sentinel) {
			t.Fatalf("error %q does not wrap the sentinel", err)
		}
	})
}

func TestFailedBackwardKeepsPreviousGradients(t *testing.T) {
	w := mustTestTensor(t, []int{1}, []float32{3})
	x := mustTestTensor(t, []int{1}, []float32{2})
	zero := mustTestTensor(t, []int{1}, []float32{0})
	two := mustTestTensor(t, []int{1}, []float32{2})

	tape := NewTape()
	param, err := tape.Param(w)
	if err != nil {
		t.Fatal(err)
	}
	customW := param.Value()
	y, err := Mul(customW, x)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Custom("product", []*Tensor{customW, x}, y, func(upstream *Tensor) ([]*Tensor, error) {
		dw, err := Mul(upstream, x)
		if err != nil {
			return nil, err
		}
		dx, err := Mul(upstream, customW)
		if err != nil {
			return nil, err
		}
		return []*Tensor{dw, dx}, nil
	}); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.MSELoss(y, zero)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	firstTapeGrad, err := tape.Grad(customW)
	if err != nil {
		t.Fatal(err)
	}
	firstTapeGradData := append([]float32(nil), firstTapeGrad.data...)
	firstParamGrad := param.Grad()
	if firstParamGrad == nil {
		t.Fatal("parameter gradient is nil after success")
	}
	firstParamGradData := append([]float32(nil), firstParamGrad.data...)

	badOut, err := Mul(customW, two)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Custom("bad", []*Tensor{customW}, badOut, func(upstream *Tensor) ([]*Tensor, error) {
		return nil, errors.New("boom")
	}); err != nil {
		t.Fatal(err)
	}
	badLoss, err := tape.MSELoss(badOut, zero)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(badLoss); err == nil {
		t.Fatal("Backward succeeded, want error")
	}

	afterTapeGrad, err := tape.Grad(customW)
	if err != nil {
		t.Fatal(err)
	}
	if !equalFloat32Slices(afterTapeGrad.data, firstTapeGradData) {
		t.Fatalf("Tape.Grad after failure = %v, want %v", afterTapeGrad.data, firstTapeGradData)
	}
	afterParamGrad := param.Grad()
	if afterParamGrad == nil {
		t.Fatal("parameter gradient is nil after failure")
	}
	if !equalFloat32Slices(afterParamGrad.data, firstParamGradData) {
		t.Fatalf("Parameter.Grad after failure = %v, want %v", afterParamGrad.data, firstParamGradData)
	}
}

func equalFloat32Slices(left, right []float32) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
