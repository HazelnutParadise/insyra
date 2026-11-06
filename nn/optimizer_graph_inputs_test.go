package nn

import (
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func optimizerGraphResidentMatrix(t *testing.T, device accel.Device, values []float32) *Tensor {
	t.Helper()
	host, err := NewTensor([]int{2, 2}, values)
	if err != nil {
		t.Fatal(err)
	}
	resident, err := host.To(device)
	if err != nil {
		t.Fatal(err)
	}
	return resident
}

func TestRetainedGradientGraphInput(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	tape := NewTape()
	t.Cleanup(tape.Close)
	if err := tape.SetPrecision(accel.PrecisionFloat32); err != nil {
		t.Fatal(err)
	}

	seed, err := tape.Param(optimizerGraphResidentMatrix(t, device, []float32{1, 2, 3, 4}))
	if err != nil {
		t.Fatal(err)
	}
	customOutput := optimizerGraphResidentMatrix(t, device, []float32{1, 0, 0, 1})
	retainedGradient := optimizerGraphResidentMatrix(t, device, []float32{1, -2, 3, -4})
	if err := tape.Custom("resident-gradient", []*Tensor{seed.Value()}, customOutput,
		func(*Tensor) ([]*Tensor, error) { return []*Tensor{retainedGradient}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(customOutput, customOutput); err != nil {
		t.Fatal(err)
	}
	if seed.grad != retainedGradient || tape.marked[retainedGradient] != nil {
		t.Fatal("fixture must retain the Custom VJP result as a gradient, not register it as a parameter")
	}
	tape.Clear(ClearOptions{KeepGradients: true})
	if _, err := retainedGradient.ToHost(); err != nil {
		t.Fatalf("KeepGradients Clear released the resident matrix gradient: %v", err)
	}

	operand, err := tape.Param(optimizerGraphResidentMatrix(t, device, []float32{2, 0, 1, 3}))
	if err != nil {
		t.Fatal(err)
	}
	result, err := tape.MatMul(retainedGradient, operand.Value())
	if err != nil {
		t.Fatalf("MatMul could not record the retained gradient as a graph input: %v", err)
	}
	if tape.marked[retainedGradient] != nil || len(tape.params) != 2 {
		t.Fatal("retained graph input was promoted to a parameter")
	}

	var firstGradient []float32
	for pass := 0; pass < 2; pass++ {
		if err := tape.BackwardFrom(result, result); err != nil {
			t.Fatalf("BackwardFrom pass %d failed: %v", pass+1, err)
		}
		if _, err := retainedGradient.ToHost(); err != nil {
			t.Fatalf("BackwardFrom pass %d released a still-referenced graph input: %v", pass+1, err)
		}
		if operand.grad == nil {
			t.Fatalf("BackwardFrom pass %d did not produce the operand parameter gradient", pass+1)
		}
		hostGradient, err := operand.grad.ToHost()
		if err != nil {
			t.Fatalf("read operand gradient after pass %d: %v", pass+1, err)
		}
		got := append([]float32(nil), hostGradient.data...)
		if pass == 0 {
			firstGradient = got
		} else {
			sameBitsSlice(t, "repeated graph-input gradient", got, firstGradient)
		}
	}

	parameterGradient := operand.grad
	tape.Clear(ClearOptions{KeepGradients: true})
	if !retainedGradient.device.Released() {
		t.Fatal("second KeepGradients Clear retained an input that no parameter gradient references")
	}
	if parameterGradient.device.Released() {
		t.Fatal("second KeepGradients Clear released the live parameter gradient")
	}
	if _, err := retainedGradient.ToHost(); err == nil {
		t.Fatal("cleared graph input remained readable after its last owner was released")
	}

	tape.Clear()
	if !parameterGradient.device.Released() {
		t.Fatal("default Clear did not release the parameter gradient")
	}
	tape.Close()
	if got := accel.ResidentBytes(device); got != baseline {
		t.Fatalf("resident bytes after Clear and Close = %d, want baseline %d", got, baseline)
	}
}
