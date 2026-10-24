package wgpu

import (
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

func TestKernelBindRefusesTheWrongBufferCount(t *testing.T) {
	k, err := NewKernel("increment", testIncrementWGSL, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.Bind(nil, nil); err == nil || !strings.Contains(err.Error(), "takes 0 read-only and 1 read-write") {
		t.Fatalf("Bind with no buffers = %v", err)
	}
}
