package nn

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

const scaleWGSL = `
struct Params { n: u32, factor: f32, p2: u32, p3: u32, }
@group(0) @binding(0) var<storage, read> a: array<f32>;
@group(0) @binding(1) var<storage, read_write> b: array<f32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    if (gid.x < p.n) {
        b[gid.x] = a[gid.x] * p.factor;
    }
}
`

// scale returns a new resident tensor holding x times two, computed by a
// kernel of the caller's own.
func scaleOnDevice(t *testing.T, k *accel.Kernel, x *Tensor) *Tensor {
	t.Helper()
	in, err := x.DeviceBuffer()
	if err != nil {
		t.Fatal(err)
	}
	out, err := accel.NewDeviceBuffer(in.Device(), make([]float32, in.Len()))
	if err != nil {
		t.Fatal(err)
	}
	groups := uint32((in.Len() + 63) / 64)
	err = accel.Submit(accel.Dispatch{
		Kernel: k, Reads: []*accel.DeviceBuffer{in}, Writes: []*accel.DeviceBuffer{out},
		Params: [16]uint32{uint32(in.Len()), 0x40000000}, Workgroups: [3]uint32{groups, 1, 1},
	})
	if err != nil {
		out.Release()
		t.Fatal(err)
	}
	tensor, err := NewDeviceTensor(x.Shape(), out)
	if err != nil {
		out.Release()
		t.Fatal(err)
	}
	return tensor
}

func newScaleKernel(t *testing.T) *accel.Kernel {
	t.Helper()
	k, err := accel.NewKernel(accel.KernelConfig{Name: "scale2", WGSL: scaleWGSL, ReadOnly: 1, ReadWrite: 1})
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// A kernel of the caller's own sits between two built-in resident operations
// and takes part in the reverse pass; nothing leaves the device until the
// final reads.
func TestCustomRunsACallerKernelOnResidentTensors(t *testing.T) {
	device := residentTestDevice(t)
	k := newScaleKernel(t)
	rng := rand.New(rand.NewSource(546))
	X, W := randomTensor(t, rng, 4, 7), randomTensor(t, rng, 7, 3)
	up := randomTensor(t, rng, 4, 3)
	rx, _ := X.To(device)
	rw, _ := W.To(device)
	rup, _ := up.To(device)
	defer rx.Release()
	defer rup.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(rw); err != nil {
		t.Fatal(err)
	}
	h, err := tape.MatMul(rx, rw)
	if err != nil {
		t.Fatal(err)
	}
	y := scaleOnDevice(t, k, h)
	err = tape.Custom("scale2", []*Tensor{h}, y, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{scaleOnDevice(t, k, upstream)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(y, rup); err != nil {
		t.Fatal(err)
	}
	hHost, err := h.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	yHost, err := y.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range hHost.data {
		if yHost.data[i] != 2*v {
			t.Fatalf("y[%d] = %v, want %v", i, yHost.data[i], 2*v)
		}
	}
	grad, err := tape.Grad(rw)
	if err != nil {
		t.Fatal(err)
	}
	defer grad.Release()
	gradHost, err := grad.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	doubled := make([]float32, len(up.data))
	for i, v := range up.data {
		doubled[i] = 2 * v
	}
	upTwice, _ := NewTensor(up.Shape(), doubled)
	sameBitsSlice(t, "dW", gradHost.data, matMulRoundedOracle(X, upTwice, true, false))
}

func TestCustomOnResidentTensorsNeedsPrecisionFloat32(t *testing.T) {
	device := residentTestDevice(t)
	x, _ := NewTensor([]int{2}, []float32{1, 2})
	y, _ := NewTensor([]int{2}, []float32{2, 4})
	rx, _ := x.To(device)
	ry, _ := y.To(device)
	defer rx.Release()
	defer ry.Release()
	vjp := func(upstream *Tensor) ([]*Tensor, error) { return []*Tensor{nil}, nil }
	for _, precision := range []accel.Precision{accel.PrecisionExact, accel.PrecisionFloat32Rounded} {
		tape := NewTape()
		if err := tape.SetPrecision(precision); err != nil {
			t.Fatal(err)
		}
		err := tape.Custom("double", []*Tensor{rx}, ry, vjp)
		if err == nil || !strings.Contains(err.Error(), "PrecisionFloat32") {
			t.Fatalf("Custom on a %q tape = %v, want a refusal naming PrecisionFloat32", precision, err)
		}
	}
}

func TestCustomRefusesMixedPlacesAndGradientsOnTheHost(t *testing.T) {
	device := residentTestDevice(t)
	host, _ := NewTensor([]int{2}, []float32{1, 2})
	resident, _ := host.To(device)
	out, _ := host.To(device)
	defer resident.Release()
	defer out.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32); err != nil {
		t.Fatal(err)
	}
	vjp := func(upstream *Tensor) ([]*Tensor, error) { return []*Tensor{host}, nil }
	if err := tape.Custom("mixed", []*Tensor{host}, out, vjp); err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("Custom with a host input and a device output = %v", err)
	}
	if err := tape.Custom("hostgrad", []*Tensor{resident}, out, vjp); err != nil {
		t.Fatal(err)
	}
	up, _ := host.To(device)
	defer up.Release()
	if err := tape.BackwardFrom(out, up); err == nil || !strings.Contains(err.Error(), "not on the device") {
		t.Fatalf("a gradient on the host from a device operation = %v", err)
	}
}

func TestClearReleasesWhatCustomAdopted(t *testing.T) {
	device := residentTestDevice(t)
	x, _ := NewTensor([]int{3}, []float32{1, 2, 3})
	rx, _ := x.To(device)
	defer rx.Release()
	before := accel.ResidentBytes(device)
	out, _ := x.To(device)
	tape := NewTape()
	if err := tape.SetPrecision(accel.PrecisionFloat32); err != nil {
		t.Fatal(err)
	}
	err := tape.Custom("copy", []*Tensor{rx}, out, func(u *Tensor) ([]*Tensor, error) { return []*Tensor{nil}, nil })
	if err != nil {
		t.Fatal(err)
	}
	tape.Clear()
	if got := accel.ResidentBytes(device); got != before {
		t.Fatalf("resident bytes after Clear = %d, want %d: the output was not released", got, before)
	}
}

func TestDeviceBufferAndNewDeviceTensorGuardTheirArguments(t *testing.T) {
	host, _ := NewTensor([]int{2}, []float32{1, 2})
	if _, err := host.DeviceBuffer(); err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("DeviceBuffer of a host tensor = %v", err)
	}
	if _, err := NewDeviceTensor([]int{2}, nil); err == nil {
		t.Fatal("NewDeviceTensor of a nil buffer must fail")
	}
}

func TestNewDeviceTensorLeavesTheBufferWithTheCallerOnAnError(t *testing.T) {
	device := residentTestDevice(t)
	buffer, err := accel.NewDeviceBuffer(device, []float32{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Release()
	if _, err := NewDeviceTensor([]int{2, 2}, buffer); err == nil || !strings.Contains(err.Error(), "holds 3") {
		t.Fatalf("NewDeviceTensor with the wrong shape = %v", err)
	}
	if buffer.Released() {
		t.Fatal("a refused NewDeviceTensor released the caller's buffer")
	}
	tensor, err := NewDeviceTensor([]int{3}, buffer)
	if err != nil {
		t.Fatal(err)
	}
	lent, err := tensor.DeviceBuffer()
	if err != nil || lent != buffer {
		t.Fatalf("DeviceBuffer = %v, %v; want the adopted buffer", lent, err)
	}
}

// A host operation whose reverse rule returns a resident gradient is refused,
// as it was before Custom accepted resident tensors.
func TestCustomRefusesAResidentGradientForAHostInput(t *testing.T) {
	device := residentTestDevice(t)
	x, _ := NewTensor([]int{2}, []float32{1, 2})
	y, _ := NewTensor([]int{2}, []float32{2, 4})
	resident, _ := x.To(device)
	defer resident.Release()
	tape := NewTape()
	if _, err := tape.Param(x); err != nil {
		t.Fatal(err)
	}
	err := tape.Custom("hostop", []*Tensor{x}, y, func(upstream *Tensor) ([]*Tensor, error) { return []*Tensor{resident}, nil })
	if err != nil {
		t.Fatal(err)
	}
	up, _ := NewTensor([]int{2}, []float32{1, 1})
	if err := tape.BackwardFrom(y, up); err == nil || !strings.Contains(err.Error(), "both be on the host") {
		t.Fatalf("a resident gradient for a host input = %v", err)
	}
}
