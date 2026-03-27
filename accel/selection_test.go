package accel

import "testing"

func TestSelectPrimaryDevicePrefersHigherScoreWithinBackend(t *testing.T) {
	devices := []Device{
		{ID: "cuda:1", Backend: BackendCUDA, Type: DeviceTypeIntegrated, Score: 30},
		{ID: "cuda:0", Backend: BackendCUDA, Type: DeviceTypeDiscrete, Score: 80},
		{ID: "webgpu:0", Backend: BackendWebGPU, Type: DeviceTypeIntegrated, Score: 40},
	}

	device, ok := selectPrimaryDevice(devices, []Backend{BackendCUDA, BackendMetal, BackendWebGPU}, nil)
	if !ok {
		t.Fatal("expected a primary device")
	}
	if device.ID != "cuda:0" {
		t.Fatalf("expected cuda:0 to win by score, got %q", device.ID)
	}
}

func TestSelectPrimaryDeviceHonorsPreferredDeviceIDs(t *testing.T) {
	devices := []Device{
		{ID: "cuda:0", Backend: BackendCUDA, Type: DeviceTypeDiscrete, Score: 80},
		{ID: "webgpu:0", Backend: BackendWebGPU, Type: DeviceTypeIntegrated, Score: 40},
	}

	device, ok := selectPrimaryDevice(devices, []Backend{BackendCUDA, BackendWebGPU}, []string{"webgpu:0"})
	if !ok {
		t.Fatal("expected a primary device")
	}
	if device.ID != "webgpu:0" {
		t.Fatalf("expected preferred device webgpu:0, got %q", device.ID)
	}
}

func TestNormalizeDeviceAppliesDefaultScore(t *testing.T) {
	device := normalizeDiscoveredDevice(Device{
		ID:          "webgpu:0",
		Backend:     BackendWebGPU,
		Type:        DeviceTypeIntegrated,
		MemoryClass: MemoryClassShared,
	})

	if device.Score <= 0 {
		t.Fatalf("expected positive normalized score, got %v", device.Score)
	}
	if !device.SharedMemory {
		t.Fatal("expected shared-memory devices to be flagged")
	}
}
