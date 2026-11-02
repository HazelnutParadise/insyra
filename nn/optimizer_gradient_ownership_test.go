package nn

import (
	"errors"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func retainedOutputGradient(t *testing.T) (*Tape, *Parameter, *Parameter, *Tensor) {
	t.Helper()
	device := residentTestDevice(t)
	tape := NewTape()
	t.Cleanup(tape.Close)
	if err := tape.SetPrecision(accel.PrecisionFloat32); err != nil {
		t.Fatal(err)
	}
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	a, _ := tape.Param(mustTo(t, v, device))
	b, _ := tape.Param(mustTo(t, v, device))
	gradient, err := tape.Add(a.Value(), b.Value())
	if err != nil {
		t.Fatal(err)
	}
	output := mustTo(t, v, device)
	if err := tape.Custom("shared", []*Tensor{a.Value(), b.Value()}, output,
		func(*Tensor) ([]*Tensor, error) { return []*Tensor{gradient, gradient}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(output, output); err != nil {
		t.Fatal(err)
	}
	tape.SetClearOptions(ClearOptions{KeepGradients: true})
	tape.Clear()
	return tape, a, b, gradient
}

func TestRetainedGradientCanBecomeParameter(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	tape, _, _, gradient := retainedOutputGradient(t)
	parameter, err := tape.Param(gradient)
	if err != nil {
		t.Fatal(err)
	}
	y, err := tape.Add(parameter.Value(), parameter.Value())
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, y); err != nil {
		t.Fatal(err)
	}
	if _, err := parameter.Value().ToHost(); err != nil {
		t.Fatalf("Backward released a retained gradient promoted to parameter: %v", err)
	}
	tape.Close()
	if accel.ResidentBytes(device) != baseline {
		t.Fatal("promoted parameter or gradients leaked")
	}
}

func TestRetainedGradientCanBeReusedByNextBackward(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	tape, a, b, gradient := retainedOutputGradient(t)
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	output := mustTo(t, v, device)
	if err := tape.Custom("reuse", []*Tensor{a.Value(), b.Value()}, output,
		func(*Tensor) ([]*Tensor, error) { return []*Tensor{gradient, gradient}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(output, output); err != nil {
		t.Fatal(err)
	}
	if _, err := gradient.ToHost(); err != nil {
		t.Fatalf("Backward released the new pass's reused gradient: %v", err)
	}
	if err := tape.SGD(0.01); err != nil {
		t.Fatal(err)
	}
	tape.SetClearOptions(ClearOptions{})
	tape.Clear()
	if !gradient.device.Released() {
		t.Fatal("discarding gradients must eventually release the reused buffer")
	}
	tape.Close()
	if accel.ResidentBytes(device) != baseline {
		t.Fatal("reused gradient leaked")
	}
}

func TestFailedBackwardKeepsRetainedGradient(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	tape, a, b, gradient := retainedOutputGradient(t)
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	intermediate, output := mustTo(t, v, device), mustTo(t, v, device)
	wantErr := errors.New("reverse rule failed")
	if err := tape.Custom("fail", []*Tensor{a.Value()}, intermediate,
		func(*Tensor) ([]*Tensor, error) { return nil, wantErr }); err != nil {
		t.Fatal(err)
	}
	if err := tape.Custom("reuse", []*Tensor{intermediate, b.Value()}, output,
		func(*Tensor) ([]*Tensor, error) { return []*Tensor{gradient, gradient}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(output, output); !errors.Is(err, wantErr) {
		t.Fatalf("Backward error = %v", err)
	}
	if a.grad != gradient || b.grad != gradient {
		t.Fatal("failed pass replaced old parameter gradients")
	}
	if _, err := gradient.ToHost(); err != nil {
		t.Fatalf("failed Backward released a previous successful gradient: %v", err)
	}
	tape.Close()
	if accel.ResidentBytes(device) != baseline {
		t.Fatal("failed backward cleanup leaked buffers")
	}
}
