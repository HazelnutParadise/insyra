package nn

import (
	"math"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

type softCEResult struct{ loss, grad []float32 }

// softCERun runs one soft-target or distillation loss and its backward pass
// under PrecisionFloat32Rounded, on the host or on a device.
func softCERun(t *testing.T, device accel.Device, distill bool, first, second *Tensor) softCEResult {
	t.Helper()
	var err error
	if first, err = first.To(device); err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	if second, err = second.To(device); err != nil {
		t.Fatal(err)
	}
	defer second.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(first); err != nil {
		t.Fatal(err)
	}
	var loss *Tensor
	if distill {
		loss, err = tape.DistillationLoss(first, second)
	} else {
		loss, err = tape.SoftTargetCrossEntropy(first, second)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	grad, err := tape.Grad(first)
	if err != nil {
		t.Fatal(err)
	}
	hostLoss, err := loss.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	hostGrad, err := grad.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	return softCEResult{loss: append([]float32(nil), hostLoss.data...), grad: append([]float32(nil), hostGrad.data...)}
}

func softCEDeviceInputs() (*Tensor, *Tensor, *Tensor) {
	rng := rand.New(rand.NewSource(808))
	const classes = 7
	var zs, zt []float32
	add := func(s, te []float32) {
		for len(s) < classes {
			s = append(s, 0)
			te = append(te, 0)
		}
		zs = append(zs, s...)
		zt = append(zt, te...)
	}
	for i := 0; i < 20; i++ {
		s, te := make([]float32, classes), make([]float32, classes)
		for c := range s {
			s[c] = float32(rng.NormFloat64() * 3)
			te[c] = s[c] + float32(rng.NormFloat64()*0.01)
		}
		add(s, te)
	}
	add([]float32{40, 0, -3}, []float32{39, 0.5, -3})
	add([]float32{1e-20, 0, 1}, []float32{0, 0, 1})
	add([]float32{0.5, -1, 2, 0, 1, 2, 3}, []float32{0.5, -1, 2, 0, 1, 2, 3})
	add([]float32{0, 1, 0}, []float32{1000, 0, 0})
	add([]float32{90, 0, 0, 3}, []float32{90.5, 0, 0, 3})
	add([]float32{1e20, 1e20}, []float32{1, 0})
	add([]float32{1, 1, 0, 0, 0, 0, 0}, []float32{1, 0, 1, 0, 0, 0, 0})
	rows := len(zs) / classes
	student, _ := NewTensor([]int{rows, classes}, zs)
	teacher, _ := NewTensor([]int{rows, classes}, zt)
	probs := make([]float32, len(zt))
	for r := 0; r < rows; r++ {
		copy(probs[r*classes:], roundedSoftmaxRowOf(zt[r*classes:(r+1)*classes]).q)
	}
	targets, _ := NewTensor([]int{rows, classes}, probs)
	return student, teacher, targets
}

// The device returns the host's bits for both losses, on rows that include a
// label past the cliff, a perturbation beside a large logit, equal logits and
// a teacher whose probabilities underflow.
func TestRoundedSoftLossesOnTheDeviceEqualTheHost(t *testing.T) {
	device := residentTestDevice(t)
	student, teacher, targets := softCEDeviceInputs()
	for _, c := range []struct {
		name    string
		distill bool
		second  *Tensor
	}{{"soft-target", false, targets}, {"distillation", true, teacher}} {
		host := softCERun(t, accel.Device{}, c.distill, student, c.second)
		dev := softCERun(t, device, c.distill, student, c.second)
		sameBitsSlice(t, c.name+" loss", dev.loss, host.loss)
		sameBitsSlice(t, c.name+" gradient", dev.grad, host.grad)
		for i, v := range host.grad {
			if math.IsNaN(float64(v)) {
				t.Errorf("%s gradient[%d] is NaN", c.name, i)
			}
		}
	}
}
