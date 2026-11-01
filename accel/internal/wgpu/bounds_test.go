package wgpu

import (
	"context"
	"testing"

	"github.com/gogpu/naga"
)

// testOverrunWGSL writes 64 floats from the start of a buffer whatever its
// length, so on a short buffer every invocation past its end writes out of
// bounds.
const testOverrunWGSL = `
struct Params { p0: u32, p1: u32, p2: u32, p3: u32, }
@group(0) @binding(0) var<storage, read_write> b: array<f32>;
@group(0) @binding(1) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    b[gid.x] = 7.0;
}
`

func TestBoundedSPIRVAddsBufferChecks(t *testing.T) {
	plain, err := naga.Compile(testOverrunWGSL)
	if err != nil {
		t.Fatal(err)
	}
	bounded, err := boundedSPIRV(testOverrunWGSL)
	if err != nil {
		t.Fatal(err)
	}
	if len(bounded)*4 <= len(plain) {
		t.Errorf("bounded SPIR-V is %d bytes, unchecked %d: no checks were added", len(bounded)*4, len(plain))
	}
	if _, err := boundedSPIRV("not wgsl"); err == nil {
		t.Error("invalid WGSL compiled")
	}
}

// A caller's kernel that writes past its buffer leaves the buffer allocated
// next to it unchanged, on every backend: Metal skips the write itself, and
// on Vulkan Insyra compiles the kernel with bounds checks.
func TestCustomKernelOutOfBoundsWriteIsContained(t *testing.T) {
	requireResidentDevice(t)
	k, err := NewKernel("overrun", testOverrunWGSL, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	short, err := Upload(make([]float32, 4))
	if err != nil {
		t.Fatal(err)
	}
	defer short.Release()
	neighbours := make([]*Buffer, 8)
	for i := range neighbours {
		vals := make([]float32, 64)
		for j := range vals {
			vals[j] = float32(i + 1)
		}
		if neighbours[i], err = Upload(vals); err != nil {
			t.Fatal(err)
		}
		defer neighbours[i].Release()
	}
	bound, err := k.Bind(nil, []*Buffer{short})
	if err != nil {
		t.Fatal(err)
	}
	defer bound.Release()
	if err := Submit([]Dispatch{{Binding: bound, X: 1, Y: 1, Z: 1}}); err != nil {
		t.Fatal(err)
	}
	got, err := short.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range got {
		if v != 7 {
			t.Errorf("in-bounds element %d = %v, want 7", i, v)
		}
	}
	for i, nb := range neighbours {
		vals, err := nb.Download(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for j, v := range vals {
			if v != float32(i+1) {
				t.Fatalf("neighbour %d element %d = %v, want %v: the out-of-bounds write landed in it", i, j, v, i+1)
			}
		}
	}
}
