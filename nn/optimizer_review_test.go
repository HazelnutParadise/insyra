package nn

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// Each Clear call decides for itself: an earlier call that kept gradients does
// not make a later plain Clear keep them.
func TestClearOptionsApplyToOneCall(t *testing.T) {
	tape := NewTape()
	defer tape.Close()
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	p, _ := tape.Param(v)
	lifecycleStep(t, tape, p)
	tape.Clear(ClearOptions{KeepGradients: true})
	if p.Grad() == nil {
		t.Fatal("Clear(KeepGradients) discarded the parameter gradient")
	}
	tape.Clear()
	if p.Grad() != nil {
		t.Fatal("a plain Clear after Clear(KeepGradients) kept the gradient")
	}
}

func TestClearWithTwoOptionsIsAnError(t *testing.T) {
	tape := NewTape()
	defer tape.Close()
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	p, _ := tape.Param(v)
	lifecycleStep(t, tape, p)
	tape.Clear(ClearOptions{KeepGradients: true}, ClearOptions{})
	if p.Grad() != nil {
		t.Fatal("Clear with two options kept a gradient")
	}
	y, err := tape.Add(p.Value(), p.Value())
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, y); err == nil || !strings.Contains(err.Error(), "ClearOptions") {
		t.Fatalf("BackwardFrom after Clear with two options = %v, want the ClearOptions error", err)
	}
}

// A frozen parameter is not trained, so it takes no part in clipping:
// a = [3, 4] (norm 5) trainable and b = [30, 40] frozen give a norm of 5.
func TestClipGradNormLeavesFrozenParametersOut(t *testing.T) {
	tape := NewTape()
	defer tape.Close()
	av, _ := NewTensor([]int{2}, []float32{1.5, 2})
	bv, _ := NewTensor([]int{2}, []float32{15, 20})
	a, _ := tape.Param(av)
	b, _ := tape.Param(bv)
	b.SetFrozen(true)
	ya, err := tape.Mul(a.Value(), a.Value())
	if err != nil {
		t.Fatal(err)
	}
	yb, err := tape.Mul(b.Value(), b.Value())
	if err != nil {
		t.Fatal(err)
	}
	y, err := tape.Add(ya, yb)
	if err != nil {
		t.Fatal(err)
	}
	ones, _ := NewTensor([]int{2}, []float32{1, 1})
	if err := tape.BackwardFrom(y, ones); err != nil {
		t.Fatal(err)
	}
	norm, err := tape.ClipGradNorm(1)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(norm)-5) > 1e-6 {
		t.Fatalf("norm = %v, want 5 from the trainable parameter alone", norm)
	}
	sameBitsSlice(t, "trainable gradient", a.Grad().data, []float32{3.0 / 5, 4.0 / 5})
	sameBitsSlice(t, "frozen gradient", b.Grad().data, []float32{30, 40})
}

// A file from a newer release names its version, not the first tensor this
// release does not know.
func TestOptimizerStateReportsANewerVersionFirst(t *testing.T) {
	source, named := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, source, named, "adam", 2)
	newer := optimizerStateMutateFile(t, optimizerStateExport(t, source, named), func(tensors map[string]*Tensor) {
		tensors[optimizerStateKey(optimizerStateMatrixName, "future_slot")] = tensors[optimizerStateKey(optimizerStateMatrixName, "adam_m")]
	}, func(metadata map[string]string) { metadata[optimizerStateVersionKey] = "3" })
	target, targetNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	err := target.LoadOptimizerState(bytes.NewReader(newer), targetNamed)
	if err == nil || !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("error = %v, want the unsupported version", err)
	}
}

// Float32 and Float32Rounded run the same Adam, so each reads the other's file.
func TestOptimizerStateFloat32AndRoundedShareCheckpoints(t *testing.T) {
	for _, pair := range [][2]accel.Precision{
		{accel.PrecisionFloat32, accel.PrecisionFloat32Rounded},
		{accel.PrecisionFloat32Rounded, accel.PrecisionFloat32},
	} {
		source, named := newOptimizerStateTape(t, pair[0], nil)
		optimizerStateRunSteps(t, source, named, "adam", 3)
		encoded := optimizerStateExport(t, source, named)
		target, targetNamed := newOptimizerStateTape(t, pair[1], nil)
		if err := target.LoadOptimizerState(bytes.NewReader(encoded), targetNamed); err != nil {
			t.Fatalf("%s file into a %s tape: %v", pair[0], pair[1], err)
		}
		if got := optimizerStateExport(t, target, targetNamed); !bytes.Equal(got, encoded) {
			t.Fatalf("%s file into a %s tape did not round-trip", pair[0], pair[1])
		}
		exact, exactNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
		if err := exact.LoadOptimizerState(bytes.NewReader(encoded), exactNamed); err == nil {
			t.Fatalf("an exact tape loaded a %s file", pair[0])
		}
	}
}

func TestOptimizerStateRefusesAStepTheFileCannotHold(t *testing.T) {
	source, named := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, source, named, "adam", 1)
	named[optimizerStateMatrixName].adamStep = math.MaxInt64 + 1
	var out bytes.Buffer
	if err := source.SaveOptimizerState(&out, named); err == nil || out.Len() != 0 {
		t.Fatalf("saved a step above MaxInt64: err %v, %d bytes", err, out.Len())
	}
}

// When the first resident Adam step fails, the moments it just made are
// released instead of being left as zeros the step never used.
func TestRoundedAdamReleasesNewMomentsWhenTheFirstStepFails(t *testing.T) {
	device := residentTestDevice(t)
	tape, named := stateDeviceTape(t, device, nil)
	p := named["device/weight"]
	y, err := tape.Add(p.Value(), p.Value())
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, y); err != nil {
		t.Fatal(err)
	}
	tape.grads[p.value].device.Release()
	if err := tape.Adam(0.01); err == nil {
		t.Fatal("Adam with a released gradient succeeded")
	}
	if p.adamMDev != nil || p.adamVDev != nil || p.adamStep != 0 {
		t.Fatalf("failed first step left moments %v %v at step %d", p.adamMDev, p.adamVDev, p.adamStep)
	}
}
