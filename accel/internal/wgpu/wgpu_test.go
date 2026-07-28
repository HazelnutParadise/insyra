package wgpu

import (
	"testing"

	"github.com/gogpu/gputypes"
)

func TestSoftwareAdapterIsNotAcceptable(t *testing.T) {
	cases := []struct {
		name string
		info gputypes.AdapterInfo
		want bool
	}{
		{"cpu device type", gputypes.AdapterInfo{Name: "Whatever", DeviceType: gputypes.DeviceTypeCPU}, true},
		{"software renderer by name", gputypes.AdapterInfo{Name: "Software Renderer", DeviceType: gputypes.DeviceTypeOther}, true},
		{"real gpu", gputypes.AdapterInfo{Name: "Apple M3", DeviceType: gputypes.DeviceTypeDiscreteGPU}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isSoftwareAdapter(testCase.info); got != testCase.want {
				t.Fatalf("isSoftwareAdapter(%+v) = %v, want %v", testCase.info, got, testCase.want)
			}
		})
	}
}

func TestUnifiedMemoryIgnoresTheReportedDeviceType(t *testing.T) {
	// gogpu reports an Apple M3 as DiscreteGPU, which would put a unified-memory
	// device in the discrete VRAM class.
	m3 := gputypes.AdapterInfo{
		Name: "Apple M3", Vendor: "Apple",
		Backend: gputypes.BackendMetal, DeviceType: gputypes.DeviceTypeDiscreteGPU,
	}
	if !unifiedMemory(m3) {
		t.Fatal("Apple Silicon has unified memory and must be classified as shared")
	}
	nvidia := gputypes.AdapterInfo{
		Name: "NVIDIA GeForce RTX 4090", Vendor: "NVIDIA",
		Backend: gputypes.BackendVulkan, DeviceType: gputypes.DeviceTypeDiscreteGPU,
	}
	if unifiedMemory(nvidia) {
		t.Fatal("a discrete NVIDIA GPU must not be classified as shared memory")
	}
}

func TestChunkSizeStaysWithinDispatchLimits(t *testing.T) {
	if _, err := Probe(); err != nil {
		t.Skipf("no usable GPU on this host: %v", err)
	}
	h, err := acquire()
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	chunk := h.maxElementsPerChunk()
	if chunk%elemsPerGroup != 0 {
		t.Fatalf("chunk size %d must be a whole number of workgroups", chunk)
	}
	if chunk > maxElemsPerDispatch {
		t.Fatalf("chunk size %d exceeds the workgroup-count limit %d", chunk, maxElemsPerDispatch)
	}
}
