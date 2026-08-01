package accel

import "github.com/HazelnutParadise/insyra/accel/internal/wgpu"

func wgpuInfoForTest(name, vendor string, isMetal, unified bool) wgpu.Info {
	return wgpu.Info{
		Name:           name,
		Vendor:         vendor,
		Driver:         "test",
		IsMetal:        isMetal,
		UnifiedMemory:  unified,
		MaxBufferBytes: 1 << 30,
	}
}
