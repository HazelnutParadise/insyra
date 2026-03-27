package accel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

var runProbeCommand = func(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

var runProbeCommandContext = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

func resolveNativeProbe(d builtinDiscoverer) builtinProbeFunc {
	if nativeProbesDisabled() {
		return func(cfg Config) ([]Device, error) {
			return nil, ErrNativeProbeUnavailable
		}
	}
	if override, ok := builtinProbeOverrides[d.backend]; ok && override.native != nil {
		return override.native
	}
	switch d.backend {
	case BackendCUDA:
		return probeNativeCUDA
	case BackendMetal:
		return probeNativeMetal
	case BackendWebGPU:
		return probeNativePortableGPU
	default:
		return func(cfg Config) ([]Device, error) {
			return nil, ErrNativeProbeUnavailable
		}
	}
}

func nativeProbesDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func probeNativeCUDA(cfg Config) ([]Device, error) {
	output, err := commandOutputWithTimeout(
		cfg,
		"nvidia-smi",
		"--query-gpu=index,name,memory.total",
		"--format=csv,noheader,nounits",
	)
	if err != nil {
		return nil, ErrNativeProbeUnavailable
	}
	devices, parseErr := parseNvidiaSMIOutput(output)
	if parseErr != nil {
		return nil, parseErr
	}
	if len(devices) == 0 {
		return nil, ErrNativeProbeUnavailable
	}
	return devices, nil
}

func probeNativeMetal(cfg Config) ([]Device, error) {
	if runtime.GOOS != "darwin" {
		return nil, ErrNativeProbeUnavailable
	}
	output, err := commandOutputWithTimeout(cfg, "system_profiler", "SPDisplaysDataType", "-json")
	if err != nil {
		return nil, ErrNativeProbeUnavailable
	}
	devices, parseErr := parseMetalSystemProfilerJSON(output)
	if parseErr != nil {
		return nil, parseErr
	}
	if len(devices) == 0 {
		return nil, ErrNativeProbeUnavailable
	}
	return devices, nil
}

func probeNativePortableGPU(cfg Config) ([]Device, error) {
	switch runtime.GOOS {
	case "windows":
		output, err := commandOutputWithTimeout(
			cfg,
			"powershell",
			"-NoProfile",
			"-Command",
			"Get-CimInstance Win32_VideoController | Select-Object Name,AdapterCompatibility,AdapterRAM | ConvertTo-Json -Compress",
		)
		if err != nil {
			return nil, ErrNativeProbeUnavailable
		}
		devices, parseErr := parseWindowsVideoControllerJSON(output, BackendWebGPU)
		if parseErr != nil {
			return nil, parseErr
		}
		if len(devices) == 0 {
			return nil, ErrNativeProbeUnavailable
		}
		return devices, nil
	case "linux":
		output, err := commandOutputWithTimeout(cfg, "lspci", "-mm")
		if err != nil {
			return nil, ErrNativeProbeUnavailable
		}
		devices, parseErr := parseLSPCIOutput(output, BackendWebGPU)
		if parseErr != nil {
			return nil, parseErr
		}
		if len(devices) == 0 {
			return nil, ErrNativeProbeUnavailable
		}
		return devices, nil
	default:
		return nil, ErrNativeProbeUnavailable
	}
}

func commandOutputWithTimeout(cfg Config, name string, args ...string) ([]byte, error) {
	timeout := cfg.DiscoveryTimeout
	if timeout <= 0 {
		return runProbeCommand(name, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := runProbeCommandContext(ctx, name, args...)
	if err != nil {
		return nil, ErrNativeProbeUnavailable
	}
	return output, nil
}

func parseWindowsVideoControllersJSON(raw string) ([]Device, error) {
	return parseWindowsVideoControllerJSON([]byte(raw), BackendWebGPU)
}

func parseLSPCIOutput(output []byte, backend Backend) ([]Device, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	devices := make([]Device, 0, len(lines))
	index := 0
	for _, line := range lines {
		parts := extractQuotedFields(line)
		if len(parts) < 3 {
			continue
		}
		className := strings.ToLower(parts[0])
		if !strings.Contains(className, "vga") && !strings.Contains(className, "3d") && !strings.Contains(className, "display") {
			continue
		}
		vendor := normalizeVendor(parts[1])
		name := parts[2]
		deviceType, memoryClass := classifyPortableDevice(vendor, name)
		devices = append(devices, Device{
			ID:           fmt.Sprintf("%s:native:%d", backend, index),
			Name:         name,
			Vendor:       vendor,
			Backend:      backend,
			ProbeSource:  ProbeSourceNative,
			Type:         deviceType,
			MemoryClass:  memoryClass,
			SharedMemory: memoryClass == MemoryClassShared,
			CapabilitySummary: map[string]bool{
				"portable":        true,
				"heuristic_probe": true,
			},
		})
		index++
	}
	return devices, nil
}

func parseNvidiaSMIOutput(output []byte) ([]Device, error) {
	lines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
	devices := make([]Device, 0, len(lines))
	fallbackIndex := 0
	for _, line := range lines {
		text := strings.TrimSpace(string(line))
		if text == "" {
			continue
		}
		parts := strings.Split(text, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("accel: invalid nvidia-smi line %q", text)
		}
		index := strconv.Itoa(fallbackIndex)
		name := ""
		memoryToken := ""
		if len(parts) >= 3 {
			index = strings.TrimSpace(parts[0])
			name = strings.TrimSpace(parts[1])
			memoryToken = strings.TrimSpace(parts[2])
		} else {
			name = strings.TrimSpace(parts[0])
			memoryToken = strings.TrimSpace(parts[1])
		}
		memoryMiB, err := strconv.ParseUint(memoryToken, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("accel: invalid nvidia-smi memory %q", memoryToken)
		}
		devices = append(devices, Device{
			ID:          "cuda:native:" + index,
			Name:        name,
			Vendor:      "nvidia",
			Backend:     BackendCUDA,
			ProbeSource: ProbeSourceNative,
			Type:        DeviceTypeDiscrete,
			MemoryClass: MemoryClassDevice,
			BudgetBytes: memoryMiB * 1024 * 1024,
			CapabilitySummary: map[string]bool{
				"multi_gpu":           true,
				"device_local_memory": true,
			},
		})
		fallbackIndex++
	}
	return devices, nil
}

func parseMetalSystemProfilerJSON(output []byte) ([]Device, error) {
	var payload struct {
		Displays []map[string]any `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, err
	}
	devices := make([]Device, 0, len(payload.Displays))
	for idx, display := range payload.Displays {
		if !supportsMetal(display) {
			continue
		}
		devices = append(devices, Device{
			ID:           fmt.Sprintf("metal:native:%d", idx),
			Name:         stringValue(display, "sppci_model", "spdisplays_model"),
			Vendor:       normalizeVendor(stringValue(display, "spdisplays_vendor")),
			Backend:      BackendMetal,
			ProbeSource:  ProbeSourceNative,
			Type:         DeviceTypeIntegrated,
			MemoryClass:  MemoryClassShared,
			SharedMemory: true,
			BudgetBytes:  parseMemoryString(stringValue(display, "spdisplays_vram_shared")),
			CapabilitySummary: map[string]bool{
				"shared_memory":   true,
				"unified_mem":     true,
				"heuristic_probe": true,
			},
		})
	}
	return devices, nil
}

func parseWindowsVideoControllerJSON(output []byte, backend Backend) ([]Device, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, nil
	}

	var many []map[string]any
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &many); err != nil {
			return nil, err
		}
	} else {
		var one map[string]any
		if err := json.Unmarshal(trimmed, &one); err != nil {
			return nil, err
		}
		many = []map[string]any{one}
	}

	devices := make([]Device, 0, len(many))
	for idx, entry := range many {
		vendor := normalizeVendor(stringValue(entry, "AdapterCompatibility"))
		if vendor == "" {
			continue
		}
		name := stringValue(entry, "Name")
		if name == "" {
			continue
		}
		deviceType, memoryClass := classifyPortableDevice(vendor, name)
		devices = append(devices, Device{
			ID:           fmt.Sprintf("%s:native:%d", backend, idx),
			Name:         name,
			Vendor:       vendor,
			Backend:      backend,
			ProbeSource:  ProbeSourceNative,
			Type:         deviceType,
			MemoryClass:  memoryClass,
			SharedMemory: memoryClass == MemoryClassShared,
			BudgetBytes:  uint64Value(entry["AdapterRAM"]),
			CapabilitySummary: map[string]bool{
				"portable":        true,
				"heuristic_probe": true,
			},
		})
	}
	return devices, nil
}

func supportsMetal(display map[string]any) bool {
	marker := strings.ToLower(stringValue(display, "spdisplays_mtlgpufamilysupport", "spdisplays_metal"))
	return strings.Contains(marker, "support") || strings.Contains(marker, "metal")
}

func stringValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch typed := value.(type) {
			case string:
				return strings.TrimSpace(typed)
			}
		}
	}
	return ""
}

func uint64Value(value any) uint64 {
	switch typed := value.(type) {
	case float64:
		return uint64(typed)
	case int:
		return uint64(typed)
	case int64:
		return uint64(typed)
	case json.Number:
		v, _ := typed.Int64()
		return uint64(v)
	default:
		return 0
	}
}

func parseMemoryString(value string) uint64 {
	text := strings.TrimSpace(strings.ToUpper(value))
	if text == "" {
		return 0
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0
	}
	number, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0
	}
	unit := "B"
	if len(fields) > 1 {
		unit = fields[1]
	}
	switch unit {
	case "GB":
		return number * 1024 * 1024 * 1024
	case "MB":
		return number * 1024 * 1024
	default:
		return number
	}
}

func normalizeVendor(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(text, "nvidia"):
		return "nvidia"
	case strings.Contains(text, "intel"):
		return "intel"
	case strings.Contains(text, "advanced micro devices"), strings.Contains(text, "amd"), strings.Contains(text, "radeon"):
		return "amd"
	case strings.Contains(text, "apple"):
		return "apple"
	default:
		return text
	}
}

func classifyPortableDevice(vendor, name string) (DeviceType, MemoryClass) {
	text := strings.ToLower(name)
	switch vendor {
	case "intel":
		return DeviceTypeIntegrated, MemoryClassShared
	case "amd":
		if strings.Contains(text, "radeon(tm) graphics") || strings.Contains(text, "vega") || strings.Contains(text, "apu") {
			return DeviceTypeIntegrated, MemoryClassShared
		}
		return DeviceTypeDiscrete, MemoryClassDevice
	case "nvidia":
		return DeviceTypeDiscrete, MemoryClassDevice
	default:
		return DeviceTypeIntegrated, MemoryClassShared
	}
}

func extractQuotedFields(line string) []string {
	fields := []string{}
	inQuote := false
	start := 0
	for idx, r := range line {
		if r != '"' {
			continue
		}
		if !inQuote {
			inQuote = true
			start = idx + 1
			continue
		}
		fields = append(fields, line[start:idx])
		inQuote = false
	}
	return fields
}
