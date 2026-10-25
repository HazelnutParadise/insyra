package accel

import (
	"errors"
	"os"
	"testing"
)

const spinWGSL = `
struct Params { n: u32, p1: u32, p2: u32, p3: u32, }
@group(0) @binding(0) var<storage, read_write> b: array<f32>;
@group(0) @binding(1) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    var x = gid.x + 1u;
    for (var i = 0u; i < p.n; i = i + 1u) {
        x = x * 1664525u + 1013904223u;
    }
    b[gid.x] = f32(x & 255u);
}
`

// A dispatch that runs long enough for the operating system to stop it loses
// the device. Every buffer from before is gone and reports ErrDeviceLost, and
// the next call that needs a device opens a new one: the same kernel binds to
// a new buffer and computes correctly. It provokes a real GPU reset (on
// Windows the driver recovers; a display can flicker), so it runs only when
// INSYRA_ACCEL_DEVICE_LOSS_TESTS=1 and a GPU is available.
func TestDeviceLossIsReportedAndTheNextCallGetsANewDevice(t *testing.T) {
	requireGPU(t)
	if os.Getenv("INSYRA_ACCEL_DEVICE_LOSS_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_DEVICE_LOSS_TESTS=1: this test makes the driver reset the GPU")
	}
	device := residentDeviceForTest(t)
	spin := mustKernel(t, KernelConfig{Name: "spin", WGSL: spinWGSL, ReadWrite: 1})
	double := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})

	const elements = 4096
	victim := mustBuffer(t, device, make([]float32, elements))
	bound := mustBind(t, spin, nil, bufs(victim))

	var lost error
	for iterations := uint32(1 << 26); iterations <= 1<<31 && lost == nil; iterations <<= 1 {
		err := Submit(Dispatch{Binding: bound, Params: [16]uint32{iterations}, Workgroups: workgroups(elements)})
		if err == nil {
			_, err = victim.ToHost()
		}
		if err != nil {
			lost = err
		}
	}
	if lost == nil {
		t.Skip("could not provoke a device loss on this GPU within the iteration budget")
	}
	if !errors.Is(lost, ErrDeviceLost) {
		t.Fatalf("the failed call reported %v, want ErrDeviceLost", lost)
	}
	var resident *ResidentError
	if !errors.As(lost, &resident) || resident.Reason != FallbackReasonDeviceLost {
		t.Fatalf("the failed call reported %v, want a ResidentError with reason %q", lost, FallbackReasonDeviceLost)
	}

	if _, err := victim.ToHost(); !errors.Is(err, ErrDeviceLost) {
		t.Fatalf("reading a buffer from before the loss = %v, want ErrDeviceLost", err)
	}
	if err := Submit(Dispatch{Binding: bound, Params: [16]uint32{1}, Workgroups: workgroups(elements)}); !errors.Is(err, ErrDeviceLost) {
		t.Fatalf("submitting a binding from before the loss = %v, want ErrDeviceLost", err)
	}

	in := mustBuffer(t, device, []float32{1, 2, 3})
	out := mustBuffer(t, device, make([]float32, 3))
	again := mustBind(t, double, bufs(in), bufs(out))
	if err := Submit(Dispatch{Binding: again, Params: [16]uint32{3}, Workgroups: workgroups(3)}); err != nil {
		t.Fatalf("a new buffer and binding after the loss: %v", err)
	}
	if got := mustHost(t, out); got[0] != 2 || got[1] != 4 || got[2] != 6 {
		t.Fatalf("after the loss the kernel computed %v, want [2 4 6]", got)
	}
}
