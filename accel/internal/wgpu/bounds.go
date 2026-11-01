package wgpu

import (
	"encoding/binary"
	"fmt"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/naga"
	"github.com/gogpu/naga/spirv"
	gowgpu "github.com/gogpu/wgpu"
)

// shaderModuleDescriptor describes the shader module of a kernel. A bounded
// kernel on Vulkan is compiled here, with naga's bounds checks on, and handed
// over as SPIR-V: gogpu/wgpu v0.34.5 compiles WGSL for Vulkan with no bounds
// checks and does not enable robustBufferAccess, so a caller's kernel that
// indexed past a buffer could write into another one (gogpu/wgpu#370). With
// the checks, an out-of-bounds read of a storage or uniform buffer returns
// zero and an out-of-bounds write does nothing, as Metal does; other
// out-of-bounds indices are clamped, as WebGPU requires. Every other kernel
// and backend passes its WGSL through unchanged.
func shaderModuleDescriptor(h *handle, label, wgsl string, bounded bool) (*gowgpu.ShaderModuleDescriptor, error) {
	if !bounded || h.info.Backend != gputypes.BackendVulkan {
		return &gowgpu.ShaderModuleDescriptor{Label: label, WGSL: wgsl}, nil
	}
	code, err := boundedSPIRV(wgsl)
	if err != nil {
		return nil, err
	}
	return &gowgpu.ShaderModuleDescriptor{Label: label, SPIRV: code}, nil
}

// boundedSPIRV compiles WGSL to SPIR-V with naga's default options plus the
// bounds checks shaderModuleDescriptor describes.
func boundedSPIRV(wgsl string) ([]uint32, error) {
	opts := naga.DefaultOptions()
	opts.BoundsCheckPolicies = spirv.BoundsCheckPolicies{
		Index:  spirv.BoundsCheckRestrict,
		Buffer: spirv.BoundsCheckReadZeroSkipWrite,
	}
	raw, err := naga.CompileWithOptions(wgsl, opts)
	if err != nil {
		return nil, fmt.Errorf("naga WGSL compilation failed: %w", err)
	}
	if len(raw)%4 != 0 {
		return nil, fmt.Errorf("naga output of %d bytes is not whole 32-bit words", len(raw))
	}
	code := make([]uint32, len(raw)/4)
	for i := range code {
		code[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}
	return code, nil
}
