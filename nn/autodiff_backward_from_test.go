package nn

import (
	"testing"
)

func TestBackwardFromNonScalarOutput(t *testing.T) {
	x := mustTestTensor(t, []int{2, 3}, []float32{0.2, -1.1, 0.7, -0.5, 0.3, 0.9})
	W := mustTestTensor(t, []int{3, 4}, []float32{
		0.1, -0.2, 0.3, 0.4,
		-0.5, 0.6, -0.7, 0.8,
		0.9, 0.2, -0.1, 0.5,
	})
	g := mustTestTensor(t, []int{2, 4}, []float32{
		0.5, -0.2, 0.3, 0.1,
		-0.4, 0.6, 0.2, -0.7,
	})
	upstreamBefore := g.Data()

	tape := NewTape()
	W = mustTapeParameter(t, tape, W)
	y, err := tape.MatMul(x, W)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, g); err != nil {
		t.Fatal(err)
	}

	gradient, err := tape.Grad(W)
	if err != nil {
		t.Fatal(err)
	}

	xData := x.Data()
	xtData := make([]float32, 3*2)
	for row := 0; row < 2; row++ {
		for col := 0; col < 3; col++ {
			xtData[col*2+row] = xData[row*3+col]
		}
	}
	xt := mustTestTensor(t, []int{3, 2}, xtData)
	want, err := MatMul(xt, g)
	if err != nil {
		t.Fatal(err)
	}
	if !equalFloat32Slices(gradient.data, want.data) {
		t.Fatalf("W gradient = %v, want %v", gradient.data, want.data)
	}

	upstreamAfter := g.Data()
	if len(upstreamAfter) != len(upstreamBefore) {
		t.Fatalf("upstream length changed: %d to %d", len(upstreamBefore), len(upstreamAfter))
	}
	for index := range upstreamBefore {
		if upstreamAfter[index] != upstreamBefore[index] {
			t.Fatalf("upstream[%d] = %g, want original %g: BackwardFrom mutated the caller's upstream", index, upstreamAfter[index], upstreamBefore[index])
		}
	}
}

func TestBackwardFromScalarMatchesBackward(t *testing.T) {
	newGraph := func(t *testing.T) (*Tape, *Tensor, *Tensor) {
		t.Helper()
		x := mustTestTensor(t, []int{1, 2}, []float32{0.4, -0.6})
		W := mustTestTensor(t, []int{2, 2}, []float32{0.2, -0.3, 0.5, 0.1})
		target := mustTestTensor(t, []int{1, 2}, []float32{0.1, 0.8})
		tape := NewTape()
		W = mustTapeParameter(t, tape, W)
		hidden, err := tape.MatMul(x, W)
		if err != nil {
			t.Fatal(err)
		}
		activated, err := tape.Tanh(hidden)
		if err != nil {
			t.Fatal(err)
		}
		loss, err := tape.MSELoss(activated, target)
		if err != nil {
			t.Fatal(err)
		}
		return tape, W, loss
	}

	backwardTape, backwardW, backwardLoss := newGraph(t)
	if err := backwardTape.Backward(backwardLoss); err != nil {
		t.Fatal(err)
	}
	backwardGrad, err := backwardTape.Grad(backwardW)
	if err != nil {
		t.Fatal(err)
	}

	fromTape, fromW, fromLoss := newGraph(t)
	one, err := newFloat32Tensor(nil, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	if err := fromTape.BackwardFrom(fromLoss, one); err != nil {
		t.Fatal(err)
	}
	fromGrad, err := fromTape.Grad(fromW)
	if err != nil {
		t.Fatal(err)
	}
	if !equalFloat32Slices(fromGrad.data, backwardGrad.data) {
		t.Fatalf("BackwardFrom gradient %v differs from Backward gradient %v", fromGrad.data, backwardGrad.data)
	}
}

func TestBackwardFromRefuses(t *testing.T) {
	newTapeWithOutput := func(t *testing.T) (*Tape, *Tensor, *Tensor) {
		t.Helper()
		x := mustTestTensor(t, []int{2, 3}, []float32{0.2, -1.1, 0.7, -0.5, 0.3, 0.9})
		W := mustTestTensor(t, []int{3, 4}, []float32{
			0.1, -0.2, 0.3, 0.4,
			-0.5, 0.6, -0.7, 0.8,
			0.9, 0.2, -0.1, 0.5,
		})
		tape := NewTape()
		W = mustTapeParameter(t, tape, W)
		y, err := tape.MatMul(x, W)
		if err != nil {
			t.Fatal(err)
		}
		return tape, W, y
	}
	g := func(t *testing.T) *Tensor {
		t.Helper()
		return mustTestTensor(t, []int{2, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8})
	}

	t.Run("nil-output", func(t *testing.T) {
		tape, _, _ := newTapeWithOutput(t)
		if err := tape.BackwardFrom(nil, g(t)); err == nil {
			t.Fatal("BackwardFrom(nil, ...) succeeded, want error")
		}
	})

	t.Run("nil-upstream", func(t *testing.T) {
		tape, _, y := newTapeWithOutput(t)
		if err := tape.BackwardFrom(y, nil); err == nil {
			t.Fatal("BackwardFrom(..., nil) succeeded, want error")
		}
	})

	t.Run("bool-upstream", func(t *testing.T) {
		tape, _, y := newTapeWithOutput(t)
		boolUpstream, err := NewBoolTensor([]int{2, 4}, []bool{true, false, true, false, true, false, true, false})
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, boolUpstream); err == nil {
			t.Fatal("BackwardFrom(y, bool upstream) succeeded, want error")
		}
	})

	t.Run("shape-mismatch", func(t *testing.T) {
		tape, _, y := newTapeWithOutput(t)
		wrongShape := mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6})
		if err := tape.BackwardFrom(y, wrongShape); err == nil {
			t.Fatal("BackwardFrom with mismatched upstream shape succeeded, want error")
		}
	})

	t.Run("output-foreign", func(t *testing.T) {
		tape, W, y := newTapeWithOutput(t)
		if err := tape.BackwardFrom(y, g(t)); err != nil {
			t.Fatalf("first successful BackwardFrom: %v", err)
		}
		gradient, err := tape.Grad(W)
		if err != nil {
			t.Fatal(err)
		}
		gradientData := append([]float32(nil), gradient.data...)

		foreign := mustTestTensor(t, []int{2, 4}, []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5})
		if err := tape.BackwardFrom(foreign, g(t)); err == nil {
			t.Fatal("BackwardFrom with a foreign output succeeded, want error")
		}
		after, err := tape.Grad(W)
		if err != nil {
			t.Fatal(err)
		}
		if !equalFloat32Slices(after.data, gradientData) {
			t.Fatalf("W gradient changed after a refused BackwardFrom: %v, want %v", after.data, gradientData)
		}
	})
}
