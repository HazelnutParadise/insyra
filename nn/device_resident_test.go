package nn

import (
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func residentTestDevice(t *testing.T) accel.Device {
	t.Helper()
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	device := accel.DefaultDevice()
	if device.ID == "" {
		t.Skip("no usable device")
	}
	return device
}

func randomTensor(t *testing.T, r *rand.Rand, shape ...int) *Tensor {
	t.Helper()
	count := 1
	for _, d := range shape {
		count *= d
	}
	data := make([]float32, count)
	for i := range data {
		data[i] = float32(r.NormFloat64()) * float32(math.Ldexp(1, r.Intn(12)-6))
	}
	tensor, err := NewTensor(shape, data)
	if err != nil {
		t.Fatal(err)
	}
	return tensor
}

type stepResult struct {
	output, e, r, bias, gradE []float32
}

// runRoundedSteps trains two SGD steps of MatMul, MatMul, Add and returns the
// final numbers. device is the zero Device for the host run.
func runRoundedSteps(t *testing.T, device accel.Device, seed int64) stepResult {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	X, E, R, bias := randomTensor(t, rng, 5, 8), randomTensor(t, rng, 8, 6), randomTensor(t, rng, 6, 4), randomTensor(t, rng, 4)
	upstream := randomTensor(t, rng, 5, 4)
	move := func(x *Tensor) *Tensor {
		moved, err := x.To(device)
		if err != nil {
			t.Fatal(err)
		}
		return moved
	}
	X, E, R, bias, upstream = move(X), move(E), move(R), move(bias), move(upstream)
	defer X.Release()
	defer upstream.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Tensor{E, R, bias} {
		if _, err := tape.Param(p); err != nil {
			t.Fatal(err)
		}
	}
	var out *Tensor
	for step := 0; step < 2; step++ {
		tape.Clear()
		h, err := tape.MatMul(X, E)
		if err != nil {
			t.Fatal(err)
		}
		y, err := tape.MatMul(h, R)
		if err != nil {
			t.Fatal(err)
		}
		out, err = tape.Add(y, bias)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(out, upstream); err != nil {
			t.Fatal(err)
		}
		if err := tape.SGD(0.05); err != nil {
			t.Fatal(err)
		}
	}
	host := func(x *Tensor) []float32 {
		h, err := x.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		return h.data
	}
	grad, err := tape.Grad(E)
	if err != nil {
		t.Fatal(err)
	}
	result := stepResult{output: host(out), e: host(E), r: host(R), bias: host(bias), gradE: host(grad)}
	grad.Release()
	return result
}

func sameBitsSlice(t *testing.T, name string, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
			t.Fatalf("%s[%d] = %g (%#x), want %g (%#x)", name, i, got[i], math.Float32bits(got[i]), want[i], math.Float32bits(want[i]))
		}
	}
}

func TestRoundedStepOnTheDeviceEqualsTheHostBitForBit(t *testing.T) {
	device := residentTestDevice(t)
	for seed := int64(1); seed <= 3; seed++ {
		host := runRoundedSteps(t, accel.Device{}, seed)
		resident := runRoundedSteps(t, device, seed)
		sameBitsSlice(t, "output", resident.output, host.output)
		sameBitsSlice(t, "E", resident.e, host.e)
		sameBitsSlice(t, "R", resident.r, host.r)
		sameBitsSlice(t, "bias", resident.bias, host.bias)
		sameBitsSlice(t, "grad E", resident.gradE, host.gradE)
	}
}

func TestRoundedHostStepMatchesTheOracle(t *testing.T) {
	// One MatMul + Add + MatMul-gradient under PrecisionFloat32Rounded on the
	// host equals the rounded reference functions.
	rng := rand.New(rand.NewSource(9))
	X, E := randomTensor(t, rng, 4, 7), randomTensor(t, rng, 7, 3)
	up := randomTensor(t, rng, 4, 3)
	tape := NewTape()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(E); err != nil {
		t.Fatal(err)
	}
	y, err := tape.MatMul(X, E)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, up); err != nil {
		t.Fatal(err)
	}
	wantY := matMulRoundedOracle(X, E, false, false)
	sameBitsSlice(t, "y", y.data, wantY)
	grad, err := tape.Grad(E)
	if err != nil {
		t.Fatal(err)
	}
	sameBitsSlice(t, "dE", grad.data, matMulRoundedOracle(X, up, true, false))
}

func TestToTheZeroDeviceIsAHostCopy(t *testing.T) {
	x, _ := NewTensor([]int{2}, []float32{1, 2})
	moved, err := x.To(accel.Device{})
	if err != nil {
		t.Fatal(err)
	}
	if moved == x || moved.Device().ID != "" || moved.onDevice() {
		t.Fatal("To(zero Device) must return a separate host tensor")
	}
	moved.data[0] = 9
	if x.data[0] != 1 {
		t.Fatal("To changed its source")
	}
}

func TestSetPrecisionRefusesTheToleranceMode(t *testing.T) {
	tape := NewTape()
	if err := tape.SetPrecision(accel.PrecisionFloat32); err == nil {
		t.Fatal("PrecisionFloat32 must be refused")
	}
	if tape.GetPrecision() != accel.PrecisionExact {
		t.Fatalf("default precision = %q", tape.GetPrecision())
	}
}

func TestResidentTensorsAreRefusedWhereTheyCannotRun(t *testing.T) {
	device := residentTestDevice(t)
	x, _ := NewTensor([]int{2, 2}, []float32{1, 2, 3, 4})
	resident, err := x.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer resident.Release()
	if _, err := resident.Float32Data(); err == nil || !strings.Contains(err.Error(), "ToHost") {
		t.Fatalf("Float32Data error = %v, want one naming ToHost", err)
	}
	if _, err := Relu(resident); err == nil || !strings.Contains(err.Error(), "ToHost") {
		t.Fatalf("Relu error = %v, want one naming ToHost", err)
	}
	tape := NewTape() // default precision
	defer tape.Close()
	if _, err := tape.MatMul(resident, resident); err == nil || !strings.Contains(err.Error(), "PrecisionFloat32Rounded") {
		t.Fatalf("MatMul error = %v, want one naming PrecisionFloat32Rounded", err)
	}
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.MatMul(resident, x); err == nil {
		t.Fatal("mixed host/device operands must be refused")
	}
	if _, err := tape.Relu(resident); err == nil || !strings.Contains(err.Error(), "ToHost") {
		t.Fatalf("Tape.Relu error = %v", err)
	}
}

func TestClearReleasesStepTensorsAndCloseReleasesParameters(t *testing.T) {
	device := residentTestDevice(t)
	before := accel.ResidentBytes(device)
	rng := rand.New(rand.NewSource(3))
	X, E := randomTensor(t, rng, 4, 4), randomTensor(t, rng, 4, 4)
	rx, _ := X.To(device)
	re, _ := E.To(device)
	up, _ := randomTensor(t, rng, 4, 4).To(device)
	baseline := before + 3*64
	tape := NewTape()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(re); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		y, err := tape.MatMul(rx, re)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, up); err != nil {
			t.Fatal(err)
		}
		tape.Clear()
	}
	if got := accel.ResidentBytes(device); got != baseline {
		t.Fatalf("resident bytes after Clear = %d, want %d (the parameter and input)", got, baseline)
	}
	tape.Close()
	rx.Release()
	up.Release()
	if got := accel.ResidentBytes(device); got != before {
		t.Fatalf("resident bytes after Close and releasing the inputs = %d, want %d", got, before)
	}
	if _, err := tape.MatMul(rx, rx); err == nil {
		t.Fatal("a closed tape must refuse operations")
	}
}

// The Codex review of the first pull request found three holes in the resident
// representation; each test below failed before its fix.

func TestResidentTensorKeepsItsLength(t *testing.T) {
	device := residentTestDevice(t)
	x, _ := NewTensor([]int{3, 4}, make([]float32, 12))
	moved, err := x.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer moved.Release()
	if moved.Len() != 12 {
		t.Fatalf("Len() = %d on a device, want 12", moved.Len())
	}
}

func TestSaveSafeTensorsRefusesAResidentTensor(t *testing.T) {
	device := residentTestDevice(t)
	x, _ := NewTensor([]int{2}, []float32{1, 2})
	moved, err := x.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer moved.Release()
	var out strings.Builder
	err = SaveSafeTensors(&out, map[string]*Tensor{"w": moved})
	if err == nil || !strings.Contains(err.Error(), "ToHost") {
		t.Fatalf("SaveSafeTensors error = %v, want one naming ToHost", err)
	}
	if out.Len() != 0 {
		t.Fatal("SaveSafeTensors wrote bytes before refusing")
	}
}

func TestResidentArithmeticRefusesLabelTensors(t *testing.T) {
	device := residentTestDevice(t)
	labels, _ := NewInt64Tensor([]int{4}, []int64{1, 1, 1, 1})
	moved, err := labels.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer moved.Release()
	tape := NewTape()
	defer tape.Close()
	_ = tape.SetPrecision(accel.PrecisionFloat32Rounded)
	if _, err := tape.Add(moved, moved); err == nil || !strings.Contains(err.Error(), "float32") {
		t.Fatalf("Add on int64 labels error = %v", err)
	}
	square, _ := NewInt64Tensor([]int{2, 2}, []int64{1, 0, 0, 1})
	sq, err := square.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer sq.Release()
	if _, err := tape.MatMul(sq, sq); err == nil || !strings.Contains(err.Error(), "float32") {
		t.Fatalf("MatMul on int64 labels error = %v", err)
	}
}
