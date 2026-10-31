package wgpu

import (
	"context"
	"math/bits"
	"testing"
)

// Kernels may use countLeadingZeros, countTrailingZeros and a scalar select()
// as an operand. gogpu/naga translated them wrongly before: until v0.20.4
// countLeadingZeros became the bit index of the highest set bit on Vulkan and
// countTrailingZeros(0) gave 0xffffffff (gogpu/naga#133), and until v0.19.1 a
// select() used as an operand lost its parentheses in Metal (gogpu/naga#95).
const testBuiltinsWGSL = `
struct Params { n: u32, p1: u32, p2: u32, p3: u32, }
@group(0) @binding(0) var<storage, read> x: array<u32>;
@group(0) @binding(1) var<storage, read_write> out: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x;
    if (i < p.n) {
        let v = x[i];
        out[3u * i] = countLeadingZeros(v);
        out[3u * i + 1u] = countTrailingZeros(v);
        out[3u * i + 2u] = (v >> 24u) | select(0u, 0xffffff00u, (v & 0x80000000u) != 0u);
    }
}
`

func TestWGSLBuiltinsMatchTheHost(t *testing.T) {
	requireResidentDevice(t)
	inputs := []uint32{0, 1, 2, 3, 0x7fffff, 0x800000, 0x12345678, 0x7fffffff, 0x80000000, 0xfffffffe, 0xffffffff}
	k, err := NewKernel("builtins", testBuiltinsWGSL, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	in, err := UploadBits(inputs)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Release()
	out, err := UploadBits(make([]uint32, 3*len(inputs)))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Release()
	bound, err := k.Bind([]*Buffer{in}, []*Buffer{out})
	if err != nil {
		t.Fatal(err)
	}
	defer bound.Release()
	if err := Submit([]Dispatch{{Binding: bound, Params: [16]uint32{uint32(len(inputs))}, X: 1, Y: 1, Z: 1}}); err != nil {
		t.Fatal(err)
	}
	got, err := out.DownloadBits(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range inputs {
		carry := v >> 24
		if v&0x80000000 != 0 {
			carry |= 0xffffff00
		}
		want := [3]uint32{uint32(bits.LeadingZeros32(v)), uint32(bits.TrailingZeros32(v)), carry}
		if have := [3]uint32(got[3*i : 3*i+3]); have != want {
			t.Errorf("x = %#x: countLeadingZeros, countTrailingZeros, select operand = %v, want %v", v, have, want)
		}
	}
}
