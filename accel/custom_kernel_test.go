package accel

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

const doubleWGSL = `
struct Params { n: u32, p1: u32, p2: u32, p3: u32, }
@group(0) @binding(0) var<storage, read> a: array<f32>;
@group(0) @binding(1) var<storage, read_write> b: array<f32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    if (gid.x < p.n) {
        b[gid.x] = a[gid.x] * 2.0;
    }
}
`

const incrementWGSL = `
struct Params { n: u32, p1: u32, p2: u32, p3: u32, }
@group(0) @binding(0) var<storage, read_write> b: array<f32>;
@group(0) @binding(1) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    if (gid.x < p.n) {
        b[gid.x] = b[gid.x] + 1.0;
    }
}
`

func mustKernel(t *testing.T, config KernelConfig) *Kernel {
	t.Helper()
	k, err := NewKernel(config)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func mustBind(t *testing.T, k *Kernel, reads, writes []*DeviceBuffer) *Binding {
	t.Helper()
	b, err := k.Bind(reads, writes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Release)
	return b
}

func workgroups(n int) [3]uint32 { return [3]uint32{uint32((n + 63) / 64), 1, 1} }

func bufs(b ...*DeviceBuffer) []*DeviceBuffer { return b }

func TestNewKernelRefusesBadConfigs(t *testing.T) {
	cases := []struct {
		name   string
		config KernelConfig
		want   string
	}{
		{"no name", KernelConfig{WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1}, "needs a name"},
		{"no source", KernelConfig{Name: "x", ReadOnly: 1, ReadWrite: 1}, "no source"},
		{"negative", KernelConfig{Name: "x", WGSL: doubleWGSL, ReadOnly: -1, ReadWrite: 1}, "negative"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewKernel(c.config); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("NewKernel = %v, want an error containing %q", err, c.want)
			}
		})
	}
}

func TestBindAndSubmitRefuseWhatTheyCannotUse(t *testing.T) {
	k := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})
	if _, err := k.Bind(nil, nil); err == nil || !strings.Contains(err.Error(), "takes 1 read-only and 1 read-write") {
		t.Fatalf("Bind with no buffers = %v", err)
	}
	if _, err := k.Bind(bufs(nil), bufs(nil)); err == nil || !strings.Contains(err.Error(), "is nil") {
		t.Fatalf("Bind with nil buffers = %v", err)
	}
	none := mustKernel(t, KernelConfig{Name: "none", WGSL: doubleWGSL})
	if _, err := none.Bind(nil, nil); err == nil || !strings.Contains(err.Error(), "binds no buffer") {
		t.Fatalf("Bind of a kernel with no buffers = %v", err)
	}
	if err := Submit(Dispatch{}); err == nil || !strings.Contains(err.Error(), "no binding") {
		t.Fatalf("Submit without a binding = %v", err)
	}
	if err := Submit(); err != nil {
		t.Fatalf("Submit of nothing = %v", err)
	}
}

func TestNewBitsBufferRefusesTheHost(t *testing.T) {
	if _, err := NewBitsBuffer(Device{}, []uint32{1}); err == nil || !strings.Contains(err.Error(), "needs a device") {
		t.Fatalf("NewBitsBuffer on the host = %v", err)
	}
}

func residentDeviceForTest(t *testing.T) Device {
	t.Helper()
	requireGPU(t)
	device, err := ResidentDevice()
	if err != nil {
		t.Skipf("no resident device: %v", err)
	}
	return device
}

func mustBuffer(t *testing.T, device Device, values []float32) *DeviceBuffer {
	t.Helper()
	b, err := NewDeviceBuffer(device, values)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Release)
	return b
}

func mustHost(t *testing.T, b *DeviceBuffer) []float32 {
	t.Helper()
	values, err := b.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	return values
}

func TestSubmitWritesAResidentBuffer(t *testing.T) {
	device := residentDeviceForTest(t)
	k := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})
	in := []float32{1, -2.5, 0, 3e30, float32(math.Copysign(0, -1))}
	a := mustBuffer(t, device, in)
	b := mustBuffer(t, device, make([]float32, len(in)))
	bound := mustBind(t, k, bufs(a), bufs(b))
	if err := Submit(Dispatch{Binding: bound, Params: [16]uint32{uint32(len(in))}, Workgroups: workgroups(len(in))}); err != nil {
		t.Fatal(err)
	}
	got, back := mustHost(t, b), mustHost(t, a)
	for i, v := range in {
		if got[i] != v*2 || math.Float32bits(back[i]) != math.Float32bits(v) {
			t.Fatalf("element %d: out %v (want %v), in %v (want %v)", i, got[i], v*2, back[i], v)
		}
	}
}

func TestSubmitOrdersItsDispatches(t *testing.T) {
	device := residentDeviceForTest(t)
	k := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})
	a := mustBuffer(t, device, []float32{1, 2, 3})
	b := mustBuffer(t, device, make([]float32, 3))
	c := mustBuffer(t, device, make([]float32, 3))
	n := [16]uint32{3}
	if err := Submit(
		Dispatch{Binding: mustBind(t, k, bufs(a), bufs(b)), Params: n, Workgroups: workgroups(3)},
		Dispatch{Binding: mustBind(t, k, bufs(b), bufs(c)), Params: n, Workgroups: workgroups(3)},
	); err != nil {
		t.Fatal(err)
	}
	if got := mustHost(t, c); got[0] != 4 || got[1] != 8 || got[2] != 12 {
		t.Fatalf("chained Submit = %v, want [4 8 12]", got)
	}
}

// The same binding launched with different parameters in one Submit reads
// each launch's own parameters.
func TestSubmitGivesEachLaunchItsOwnParameters(t *testing.T) {
	device := residentDeviceForTest(t)
	k := mustKernel(t, KernelConfig{Name: "increment", WGSL: incrementWGSL, ReadWrite: 1})
	b := mustBuffer(t, device, []float32{0, 0, 0, 0})
	bound := mustBind(t, k, nil, bufs(b))
	// n = 1, 2, 3, 4 elements: element i is incremented by the launches with n > i.
	var launches []Dispatch
	for n := 1; n <= 4; n++ {
		launches = append(launches, Dispatch{Binding: bound, Params: [16]uint32{uint32(n)}, Workgroups: workgroups(4)})
	}
	if err := Submit(launches...); err != nil {
		t.Fatal(err)
	}
	if got := mustHost(t, b); got[0] != 4 || got[1] != 3 || got[2] != 2 || got[3] != 1 {
		t.Fatalf("after launches with n = 1..4: %v, want [4 3 2 1]", got)
	}
}

func TestSubmitSendsNothingWhenADispatchCannotRun(t *testing.T) {
	device := residentDeviceForTest(t)
	inc := mustKernel(t, KernelConfig{Name: "increment", WGSL: incrementWGSL, ReadWrite: 1})
	a := mustBuffer(t, device, []float32{5})
	doomed, err := NewDeviceBuffer(device, []float32{0})
	if err != nil {
		t.Fatal(err)
	}
	good := mustBind(t, inc, nil, bufs(a))
	bad := mustBind(t, inc, nil, bufs(doomed))
	doomed.Release()
	err = Submit(
		Dispatch{Binding: good, Params: [16]uint32{1}, Workgroups: workgroups(1)},
		Dispatch{Binding: bad, Params: [16]uint32{1}, Workgroups: workgroups(1)},
	)
	if err == nil || !strings.Contains(err.Error(), "dispatch 1") || !strings.Contains(err.Error(), "released") {
		t.Fatalf("Submit with a released buffer = %v", err)
	}
	if got := mustHost(t, a); got[0] != 5 {
		t.Fatalf("the first dispatch ran: %v", got)
	}
	host := &DeviceBuffer{inner: a.inner}
	if _, err := inc.Bind(nil, bufs(host)); err == nil || !strings.Contains(err.Error(), "on the host") {
		t.Fatalf("Bind with a host buffer = %v", err)
	}
	released := mustBind(t, inc, nil, bufs(a))
	released.Release()
	if err := Submit(Dispatch{Binding: released, Params: [16]uint32{1}, Workgroups: workgroups(1)}); err == nil || !strings.Contains(err.Error(), "released") {
		t.Fatalf("Submit of a released binding = %v", err)
	}
}

func TestDeviceBufferBitsRoundTrip(t *testing.T) {
	device := residentDeviceForTest(t)
	bits := []uint32{0, 1, 0xFFFFFFFF, 0x7fc00001, 0xffc12345, 0x80000000}
	b, err := NewBitsBuffer(device, bits)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Release()
	got, err := b.BitsToHost()
	if err != nil {
		t.Fatal(err)
	}
	for i := range bits {
		if got[i] != bits[i] {
			t.Fatalf("bits %d: got %#x, want %#x", i, got[i], bits[i])
		}
	}
}

// A built-in resident operation and a caller's Submit share one queue, so each
// sees the other's writes without a synchronisation, either way round.
func TestSubmitAndBuiltinsShareOneQueue(t *testing.T) {
	device := residentDeviceForTest(t)
	double := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})
	a := mustBuffer(t, device, []float32{1, 2, 3, 4})
	identity := mustBuffer(t, device, []float32{1, 0, 0, 1})

	product, err := RoundedMatMul(a, identity, 2, 2, 2, false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer product.Release()
	doubled := mustBuffer(t, device, make([]float32, 4))
	if err := Submit(Dispatch{Binding: mustBind(t, double, bufs(product), bufs(doubled)), Params: [16]uint32{4}, Workgroups: workgroups(4)}); err != nil {
		t.Fatal(err)
	}
	again, err := RoundedMatMul(doubled, identity, 2, 2, 2, false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Release()
	if got := mustHost(t, again); got[0] != 2 || got[1] != 4 || got[2] != 6 || got[3] != 8 {
		t.Fatalf("built-in -> Submit -> built-in = %v, want [2 4 6 8]", got)
	}
}

// More launches than the parameter ring holds still return the right result:
// the ring drains the queue and starts over when it is used up.
func TestSubmitPastTheParameterRing(t *testing.T) {
	device := residentDeviceForTest(t)
	inc := mustKernel(t, KernelConfig{Name: "increment", WGSL: incrementWGSL, ReadWrite: 1})
	b := mustBuffer(t, device, []float32{0, 10})
	bound := mustBind(t, inc, nil, bufs(b))
	one := Dispatch{Binding: bound, Params: [16]uint32{2}, Workgroups: workgroups(2)}
	total := 0
	for _, n := range []int{3000, 3000, 1200, 4096} {
		batch := make([]Dispatch, n)
		for i := range batch {
			batch[i] = one
		}
		if err := Submit(batch...); err != nil {
			t.Fatal(err)
		}
		total += n
	}
	if got := mustHost(t, b); got[0] != float32(total) || got[1] != float32(total+10) {
		t.Fatalf("after %d increments = %v, want [%d %d]", total, got, total, total+10)
	}
	tooMany := make([]Dispatch, 4097)
	for i := range tooMany {
		tooMany[i] = one
	}
	if err := Submit(tooMany...); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("Submit of more dispatches than the ring holds = %v", err)
	}
}

func TestBindRefusesABufferReadAndWritten(t *testing.T) {
	device := residentDeviceForTest(t)
	k := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})
	a := mustBuffer(t, device, []float32{1, 2})
	if _, err := k.Bind(bufs(a), bufs(a)); err == nil || !strings.Contains(err.Error(), "also bound for reading") {
		t.Fatalf("Bind with one buffer read and written = %v", err)
	}
}

func TestSubmitRefusesAZeroOrOversizedWorkgroupCount(t *testing.T) {
	device := residentDeviceForTest(t)
	limits, err := ResidentLimits()
	if err != nil {
		t.Fatal(err)
	}
	k := mustKernel(t, KernelConfig{Name: "increment", WGSL: incrementWGSL, ReadWrite: 1})
	b := mustBuffer(t, device, []float32{0})
	bound := mustBind(t, k, nil, bufs(b))
	err = Submit(Dispatch{Binding: bound, Workgroups: [3]uint32{1, 0, 1}})
	if err == nil || !strings.Contains(err.Error(), "zero workgroup") {
		t.Fatalf("Submit with a zero count = %v", err)
	}
	err = Submit(Dispatch{Binding: bound, Params: [16]uint32{1}, Workgroups: [3]uint32{limits.MaxWorkgroupsPerDimension + 1, 1, 1}})
	if err == nil || !strings.Contains(err.Error(), "dispatch 0") || !strings.Contains(err.Error(), "workgroups") {
		t.Fatalf("Submit past the workgroup limit = %v", err)
	}
}

func TestBindReportsAShaderThatDoesNotCompile(t *testing.T) {
	device := residentDeviceForTest(t)
	k := mustKernel(t, KernelConfig{Name: "broken", WGSL: "fn main() { this is not wgsl }", ReadWrite: 1})
	b := mustBuffer(t, device, []float32{0})
	if _, err := k.Bind(nil, bufs(b)); err == nil || !strings.Contains(err.Error(), "shader") {
		t.Fatalf("Bind of a broken shader = %v", err)
	}
}

func TestResidentLimitsAreUsable(t *testing.T) {
	residentDeviceForTest(t)
	limits, err := ResidentLimits()
	if err != nil {
		t.Fatal(err)
	}
	if limits.MaxStorageBuffers < 4 || limits.MaxBufferBytes < 1<<20 || limits.MaxWorkgroupsPerDimension < 1 || limits.MaxInvocationsPerWorkgroup < 1 {
		t.Fatalf("implausible limits %+v", limits)
	}
}

// Write waits for the kernels already submitted, so a kernel reading the
// buffer finishes with the old words and the next one sees the new ones.
func TestWriteWaitsForEarlierKernelsAndZeroClears(t *testing.T) {
	device := residentDeviceForTest(t)
	double := mustKernel(t, KernelConfig{Name: "double", WGSL: doubleWGSL, ReadOnly: 1, ReadWrite: 1})
	in := mustBuffer(t, device, []float32{1, 2, 3})
	out := mustBuffer(t, device, make([]float32, 3))
	bound := mustBind(t, double, bufs(in), bufs(out))
	run := func() {
		t.Helper()
		if err := Submit(Dispatch{Binding: bound, Params: [16]uint32{3}, Workgroups: workgroups(3)}); err != nil {
			t.Fatal(err)
		}
	}
	run()
	if err := in.Write([]float32{10, 20, 30}); err != nil {
		t.Fatal(err)
	}
	if got := mustHost(t, out); got[0] != 2 || got[1] != 4 || got[2] != 6 {
		t.Fatalf("the kernel before Write saw %v, want the old input doubled", got)
	}
	run()
	if got := mustHost(t, out); got[0] != 20 || got[1] != 40 || got[2] != 60 {
		t.Fatalf("the kernel after Write saw %v, want the new input doubled", got)
	}
	if err := out.Zero(); err != nil {
		t.Fatal(err)
	}
	if got := mustHost(t, out); got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Fatalf("Zero left %v", got)
	}
	if err := in.Write([]float32{1}); err == nil || !strings.Contains(err.Error(), "holds 3") {
		t.Fatalf("Write of the wrong length = %v", err)
	}
	bits, err := NewBitsBuffer(device, []uint32{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	defer bits.Release()
	if err := bits.WriteBits([]uint32{0x7fc00001, 0xffc12345}); err != nil {
		t.Fatal(err)
	}
	if got, _ := bits.BitsToHost(); got[0] != 0x7fc00001 || got[1] != 0xffc12345 {
		t.Fatalf("WriteBits round trip = %#x", got)
	}
}

// BenchmarkSubmitDispatches compares one Submit of many dispatches, one
// submission, with one Submit per dispatch, a submission each, the way 28
// steps of a recurrent core would otherwise be launched.
func BenchmarkSubmitDispatches(b *testing.B) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		b.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	device, err := ResidentDevice()
	if err != nil {
		b.Skipf("no resident device: %v", err)
	}
	inc, err := NewKernel(KernelConfig{Name: "increment", WGSL: incrementWGSL, ReadWrite: 1})
	if err != nil {
		b.Fatal(err)
	}
	buffer, err := NewDeviceBuffer(device, make([]float32, 4096))
	if err != nil {
		b.Fatal(err)
	}
	defer buffer.Release()
	bound, err := inc.Bind(nil, []*DeviceBuffer{buffer})
	if err != nil {
		b.Fatal(err)
	}
	defer bound.Release()
	one := Dispatch{Binding: bound, Params: [16]uint32{4096}, Workgroups: [3]uint32{64, 1, 1}}
	for _, n := range []int{28, 168} {
		batch := make([]Dispatch, n)
		for i := range batch {
			batch[i] = one
		}
		b.Run(fmt.Sprintf("dispatches=%d/one-submit", n), func(b *testing.B) {
			for b.Loop() {
				if err := Submit(batch...); err != nil {
					b.Fatal(err)
				}
				if err := SynchronizeResident(); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("dispatches=%d/submit-each", n), func(b *testing.B) {
			for b.Loop() {
				for range n {
					if err := Submit(one); err != nil {
						b.Fatal(err)
					}
				}
				if err := SynchronizeResident(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
