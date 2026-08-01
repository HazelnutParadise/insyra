package accel

import (
	"context"
	"testing"
	"time"
)

func TestParseNvidiaSMIOutput(t *testing.T) {
	output := []byte("0, NVIDIA GeForce RTX 4070, 12282\n1, NVIDIA RTX A2000, 6144\n")

	devices, err := parseNvidiaSMIOutput(output)
	if err != nil {
		t.Fatalf("parseNvidiaSMIOutput failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].ID != "cuda:native:0" {
		t.Fatalf("expected first cuda native id, got %q", devices[0].ID)
	}
	if devices[0].ProbeSource != ProbeSourceNative {
		t.Fatalf("expected native probe source, got %q", devices[0].ProbeSource)
	}
	if devices[0].Vendor != "nvidia" {
		t.Fatalf("expected nvidia vendor, got %q", devices[0].Vendor)
	}
	if devices[0].BudgetBytes != 12282*1024*1024 {
		t.Fatalf("unexpected budget bytes: %d", devices[0].BudgetBytes)
	}
}

func TestParseWindowsVideoControllerJSONForWebGPU(t *testing.T) {
	jsonText := []byte(`[
  {"Name":"Intel(R) Iris(R) Xe Graphics","AdapterCompatibility":"Intel Corporation","AdapterRAM":2147483648},
  {"Name":"NVIDIA GeForce RTX 4070","AdapterCompatibility":"NVIDIA","AdapterRAM":12884901888},
  {"Name":"AMD Radeon(TM) Graphics","AdapterCompatibility":"Advanced Micro Devices, Inc.","AdapterRAM":2147483648}
]`)

	devices, err := parseWindowsVideoControllerJSON(jsonText, BackendWebGPU)
	if err != nil {
		t.Fatalf("parseWindowsVideoControllerJSON failed: %v", err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 portable devices, got %d", len(devices))
	}
	if devices[0].Vendor != "intel" {
		t.Fatalf("expected intel vendor first, got %q", devices[0].Vendor)
	}
	if devices[0].MemoryClass != MemoryClassShared {
		t.Fatalf("expected intel gpu shared memory, got %q", devices[0].MemoryClass)
	}
	if devices[1].Vendor != "nvidia" {
		t.Fatalf("expected nvidia vendor second, got %q", devices[1].Vendor)
	}
	if devices[2].Vendor != "amd" {
		t.Fatalf("expected amd vendor third, got %q", devices[2].Vendor)
	}
	if devices[1].ProbeSource != ProbeSourceNative || devices[2].ProbeSource != ProbeSourceNative {
		t.Fatal("expected native probe source for portable devices")
	}
}

func TestParseWindowsVideoControllerCSVForWebGPU(t *testing.T) {
	csvText := []byte("Node,AdapterCompatibility,AdapterRAM,Name\r\nHOST,Intel Corporation,2147483648,Intel(R) Iris(R) Xe Graphics\r\nHOST,Advanced Micro Devices, Inc.,2147483648,AMD Radeon(TM) Graphics\r\n")

	devices, err := parseWindowsVideoControllerCSV(csvText, BackendWebGPU)
	if err != nil {
		t.Fatalf("parseWindowsVideoControllerCSV failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].Vendor != "intel" {
		t.Fatalf("expected intel vendor first, got %q", devices[0].Vendor)
	}
	if devices[1].Vendor != "amd" {
		t.Fatalf("expected amd vendor second, got %q", devices[1].Vendor)
	}
	if devices[0].ProbeSource != ProbeSourceNative || devices[1].ProbeSource != ProbeSourceNative {
		t.Fatal("expected native probe source for windows csv parsing")
	}
}

func TestParseMetalSystemProfilerJSON(t *testing.T) {
	jsonText := []byte(`{
  "SPDisplaysDataType": [
    {
      "spdisplays_vendor":"Apple",
      "sppci_model":"Apple M3",
      "spdisplays_mtlgpufamilysupport":"spdisplays_supported",
      "spdisplays_vram_shared":"8192 MB"
    }
  ]
}`)

	devices, err := parseMetalSystemProfilerJSON(jsonText)
	if err != nil {
		t.Fatalf("parseMetalSystemProfilerJSON failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 metal device, got %d", len(devices))
	}
	if devices[0].Backend != BackendMetal {
		t.Fatalf("expected metal backend, got %q", devices[0].Backend)
	}
	if devices[0].MemoryClass != MemoryClassShared {
		t.Fatalf("expected shared memory, got %q", devices[0].MemoryClass)
	}
	if devices[0].Vendor != "apple" {
		t.Fatalf("expected apple vendor, got %q", devices[0].Vendor)
	}
}

func TestParseLSPCIOutputForWebGPU(t *testing.T) {
	output := []byte(`00:02.0 "VGA compatible controller" "Intel Corporation" "Iris Xe Graphics" -r01 "Dell" "Device 0001"
01:00.0 "VGA compatible controller" "NVIDIA Corporation" "AD104 [GeForce RTX 4070]" -r01 "Dell" "Device 0002"
02:00.0 "VGA compatible controller" "Advanced Micro Devices, Inc. [AMD/ATI]" "Radeon(TM) Graphics" -r01 "Dell" "Device 0003"
`)

	devices, err := parseLSPCIOutput(output, BackendWebGPU)
	if err != nil {
		t.Fatalf("parseLSPCIOutput failed: %v", err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 portable devices, got %d", len(devices))
	}
	if devices[0].Vendor != "intel" {
		t.Fatalf("expected intel device first, got %q", devices[0].Vendor)
	}
	if devices[1].Vendor != "nvidia" {
		t.Fatalf("expected nvidia device second, got %q", devices[1].Vendor)
	}
	if devices[2].Vendor != "amd" {
		t.Fatalf("expected amd device third, got %q", devices[2].Vendor)
	}
	if devices[0].ProbeSource != ProbeSourceNative || devices[1].ProbeSource != ProbeSourceNative || devices[2].ProbeSource != ProbeSourceNative {
		t.Fatal("expected native probe source for lspci devices")
	}
}

func TestParseLSHWDisplayJSONForWebGPU(t *testing.T) {
	jsonText := []byte(`[
  {
    "id":"display",
    "class":"display",
    "product":"Iris Xe Graphics",
    "vendor":"Intel Corporation",
    "size":2147483648
  },
  {
    "id":"display:1",
    "class":"display",
    "product":"Radeon(TM) Graphics",
    "vendor":"Advanced Micro Devices, Inc. [AMD/ATI]",
    "size":2147483648
  }
]`)

	devices, err := parseLSHWDisplayJSON(jsonText, BackendWebGPU)
	if err != nil {
		t.Fatalf("parseLSHWDisplayJSON failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].Vendor != "intel" {
		t.Fatalf("expected intel vendor first, got %q", devices[0].Vendor)
	}
	if devices[1].Vendor != "amd" {
		t.Fatalf("expected amd vendor second, got %q", devices[1].Vendor)
	}
	if devices[0].BudgetBytes != 2147483648 {
		t.Fatalf("expected lshw size to map to budget bytes, got %d", devices[0].BudgetBytes)
	}
}

func TestProbeNativeCUDARespectsDiscoveryTimeout(t *testing.T) {
	previous := runProbeCommandContext
	runProbeCommandContext = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer func() {
		runProbeCommandContext = previous
	}()

	start := time.Now()
	_, err := probeNativeCUDA(Config{DiscoveryTimeout: 10 * time.Millisecond})
	if err == nil {
		t.Fatal("expected timeout to surface as unavailable probe")
	}
	if err != ErrNativeProbeUnavailable {
		t.Fatalf("expected native probe unavailable on timeout, got %v", err)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("expected probe timeout to return promptly, took %s", time.Since(start))
	}
}
