package wgpu

import (
	"context"
	"strings"
	"testing"
)

const testIncrementWGSL = `
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

func TestNewKernelCompilesItsSourceAsWritten(t *testing.T) {
	k, err := NewKernel("plain", "fn main() {}", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := k.k.source(); got != "fn main() {}" {
		t.Fatalf("a caller's kernel got extra source: %d bytes", len(got))
	}
}

func TestSubmitRefusesAZeroWorkgroupCount(t *testing.T) {
	k, err := NewKernel("increment", testIncrementWGSL, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := Submit([]Dispatch{{Kernel: k, Writes: []*Buffer{{}}, X: 1, Y: 0, Z: 1}}); err == nil || !strings.Contains(err.Error(), "zero workgroup") {
		t.Fatalf("Run with a zero count = %v", err)
	}
}

// A Run that would take the per-dispatch objects past maxPendingObjects
// waits for the queue first, freeing what earlier work kept alive, and still
// computes the right result.
func TestSubmitPastThePendingBoundSynchronisesFirst(t *testing.T) {
	requireResidentDevice(t)
	k, err := NewKernel("increment", testIncrementWGSL, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Upload([]float32{0})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Release()
	one := Dispatch{Kernel: k, Writes: []*Buffer{b}, Params: [16]uint32{1}, X: 1, Y: 1, Z: 1}
	if err := Submit([]Dispatch{one}); err != nil {
		t.Fatal(err)
	}
	submitMu.Lock()
	before := len(pending)
	submitMu.Unlock()
	if before == 0 {
		t.Fatal("the first Run left nothing pending; the test cannot tell a synchronisation apart")
	}
	many := (maxPendingObjects-before)/2 + 1
	dispatches := make([]Dispatch, many)
	for i := range dispatches {
		dispatches[i] = one
	}
	if err := Submit(dispatches); err != nil {
		t.Fatal(err)
	}
	submitMu.Lock()
	after := len(pending)
	submitMu.Unlock()
	if after != 2*many {
		t.Fatalf("pending after the large Run = %d, want %d: the earlier %d objects were not freed by a synchronisation", after, 2*many, before)
	}
	got, err := b.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := float32(many + 1); got[0] != want {
		t.Fatalf("after %d increments = %v, want %v", many+1, got[0], want)
	}
}
