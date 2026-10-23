package nn

import (
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// A parameter initialised through a tape operation is marked after the
// operation produced it. Clear must not release it, and Close must.
func TestClearKeepsAParameterProducedByAnOperation(t *testing.T) {
	device := residentTestDevice(t)
	a, _ := NewTensor([]int{2, 2}, []float32{1, 2, 3, 4})
	b, _ := NewTensor([]int{2, 2}, []float32{1, 0, 0, 1})
	a, b = mustTo(t, a, device), mustTo(t, b, device)
	defer a.Release()
	defer b.Release()
	tape := NewTape()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	w, err := tape.MatMul(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(w); err != nil {
		t.Fatal(err)
	}
	tape.Clear()
	host, err := w.ToHost()
	if err != nil {
		t.Fatalf("parameter was released by Clear: %v", err)
	}
	want := []float32{1, 2, 3, 4}
	got, _ := host.Float32Data()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parameter after Clear = %v, want %v", got, want)
		}
	}
	// The next step still trains with it.
	y, err := tape.MatMul(a, w)
	if err != nil {
		t.Fatalf("step after Clear: %v", err)
	}
	if err := tape.BackwardFrom(y, y); err != nil {
		t.Fatalf("backward after Clear: %v", err)
	}
	tape.Close()
	if _, err := w.ToHost(); err == nil {
		t.Error("Close left the parameter on the device")
	}
}
