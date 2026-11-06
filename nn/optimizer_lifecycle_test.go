package nn

import (
	"math"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func lifecycleStep(t *testing.T, tape *Tape, p *Parameter) {
	t.Helper()
	y, err := tape.Add(p.Value(), p.Value())
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, y); err != nil {
		t.Fatal(err)
	}
}

func TestOptimizerClearRetainsOnlyParameterGradients(t *testing.T) {
	tape := NewTape()
	defer tape.Close()
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	p, _ := tape.Param(v)
	lifecycleStep(t, tape, p)
	want := p.Grad().data
	tape.Clear(ClearOptions{KeepGradients: true})
	if len(tape.ops) != 0 || len(tape.grads) != 1 || p.Grad() == nil {
		t.Fatal("Clear must release graph and intermediate gradients but retain parameter gradients")
	}
	sameBitsSlice(t, "gradient after Clear", p.Grad().data, want)
	if err := tape.AdamW(0.01, 0.1); err != nil {
		t.Fatal(err)
	}
	if p.adamStep != 1 {
		t.Fatal("retained gradient was not used by AdamW")
	}
	lifecycleStep(t, tape, p)
	for i, got := range p.Grad().data {
		if math.Float32bits(got) != math.Float32bits(4*p.Value().data[i]) {
			t.Fatal("Backward accumulated rather than replaced retained gradients")
		}
	}
	tape.Clear()
	if p.Grad() != nil || len(tape.grads) != 0 {
		t.Fatal("default Clear must discard gradients")
	}
}

func TestOptimizerClearCyclesAndFreeze(t *testing.T) {
	for _, precision := range []accel.Precision{accel.PrecisionExact, accel.PrecisionFloat32Rounded} {
		for _, optimizer := range []string{"sgd", "momentum", "adam", "adamw"} {
			t.Run(string(precision)+"/"+optimizer, func(t *testing.T) {
				tape, reference := NewTape(), NewTape()
				defer tape.Close()
				defer reference.Close()
				for _, candidate := range []*Tape{tape, reference} {
					if err := candidate.SetPrecision(precision); err != nil {
						t.Fatal(err)
					}
				}
				v, _ := NewTensor([]int{2}, []float32{1, -2})
				r, _ := NewTensor([]int{2}, []float32{1, -2})
				p, _ := tape.Param(v)
				q, _ := reference.Param(r)
				update := func(tape *Tape) {
					var err error
					switch optimizer {
					case "sgd":
						err = tape.SGD(0.01)
					case "momentum":
						err = tape.SGDMomentum(0.01, 0.9)
					case "adam":
						err = tape.Adam(0.01)
					case "adamw":
						err = tape.AdamW(0.01, 0.2)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				for step := 0; step < 25; step++ {
					lifecycleStep(t, tape, p)
					lifecycleStep(t, reference, q)
					update(tape)
					update(reference)
					p.SetFrozen(true)
					if !p.GetFrozen() {
						t.Fatal("freeze was not retained")
					}
					for repeat := 0; repeat < 3; repeat++ {
						update(tape) // still has the old gradient and initialized optimizer state
					}
					sameBitsSlice(t, "frozen values", p.value.data, q.value.data)
					sameBitsSlice(t, "frozen first moment", p.adamM, q.adamM)
					sameBitsSlice(t, "frozen second moment", p.adamV, q.adamV)
					sameBitsSlice(t, "frozen velocity", p.velocity, q.velocity)
					if p.adamStep != q.adamStep {
						t.Fatal("freeze advanced step counter")
					}
					p.SetFrozen(false)
					tape.Clear()
					if tape.ops != nil || len(tape.grads) != 0 {
						t.Fatal("graph history retained")
					}
					// Independent fresh graph, with the same Parameter and optimizer slots.
					reference.ops = nil
					reference.grads = make(map[*Tensor]*Tensor)
					q.grad = nil
				}
				lifecycleStep(t, tape, p)
				lifecycleStep(t, reference, q)
				update(tape)
				update(reference)
				sameBitsSlice(t, "next update", p.value.data, q.value.data)
			})
		}
	}
}

func TestOptimizerClearResidentOwnership(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	p, err := tape.Param(mustTo(t, v, device))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		y, err := tape.Add(p.Value(), p.Value())
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, y); err != nil {
			t.Fatal(err)
		}
		if err := tape.AdamW(0.01, 0.2); err != nil {
			t.Fatal(err)
		}
		p.SetFrozen(true)
		step := p.adamStep
		before, _ := p.Value().ToHost()
		if err := tape.AdamW(0.01, 0.2); err != nil {
			t.Fatal(err)
		}
		after, _ := p.Value().ToHost()
		if p.adamStep != step || !reflect.DeepEqual(before.data, after.data) {
			t.Fatal("frozen device parameter was updated")
		}
		p.SetFrozen(false)
		tape.Clear(ClearOptions{KeepGradients: true})
		if len(tape.ops) != 0 || p.grad == nil || p.grad.device.Released() {
			t.Fatal("Clear released retained device gradients or kept operations")
		}
		// Two-element parameter + two moments + one retained gradient.
		if got := accel.ResidentBytes(device) - baseline; got != 32 {
			t.Fatalf("Clear retained %d bytes, want 32", got)
		}
	}
	tape.Close()
	if got := accel.ResidentBytes(device); got != baseline {
		t.Fatalf("Close leaked retained gradients: %d bytes, baseline %d", got, baseline)
	}
}

func TestOptimizerClearRetainsAliasedResidentOutputGradient(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32); err != nil {
		t.Fatal(err)
	}
	v, _ := NewTensor([]int{2}, []float32{1, -2})
	a, err := tape.Param(mustTo(t, v, device))
	if err != nil {
		t.Fatal(err)
	}
	b, err := tape.Param(mustTo(t, v, device))
	if err != nil {
		t.Fatal(err)
	}
	gradient, err := tape.Add(a.Value(), b.Value())
	if err != nil {
		t.Fatal(err)
	}
	output := mustTo(t, v, device)
	if err := tape.Custom("shared-output-gradient", []*Tensor{a.Value(), b.Value()}, output,
		func(up *Tensor) ([]*Tensor, error) { return []*Tensor{gradient, gradient}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(output, output); err != nil {
		t.Fatal(err)
	}
	tape.Clear(ClearOptions{KeepGradients: true})
	if gradient.device.Released() || len(tape.gradOwned) != 1 || a.grad != b.grad {
		t.Fatal("shared parameter gradient needs one retained owner across both lists")
	}
	if got := accel.ResidentBytes(device) - baseline; got != 24 {
		t.Fatalf("retained %d device bytes, want 24", got)
	}
	if err := tape.SGD(0.01); err != nil {
		t.Fatal(err)
	}
	tape.Close()
	if accel.ResidentBytes(device) != baseline {
		t.Fatal("Close leaked an aliased output-gradient")
	}
}

func TestFrozenParameterAllowsExplicitUpdate(t *testing.T) {
	tape := NewTape()
	defer tape.Close()
	v, _ := NewTensor(nil, []float32{1})
	p, _ := tape.Param(v)
	p.SetFrozen(true)
	if err := p.Update(func(values, gradient []float32) { values[0] = 2 }); err != nil {
		t.Fatal(err)
	}
	if p.Value().data[0] != 2 {
		t.Fatal("freeze must not block an explicit caller-managed update")
	}
}
