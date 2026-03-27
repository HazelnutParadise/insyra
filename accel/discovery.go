package accel

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Discoverer interface {
	Name() string
	Discover(cfg Config) ([]Device, error)
}

var (
	discoverersMu sync.RWMutex
	discoverers   []Discoverer
)

func RegisterDiscoverer(d Discoverer) {
	discoverersMu.Lock()
	defer discoverersMu.Unlock()
	discoverers = append(discoverers, d)
}

func ResetDiscoverersForTest() {
	discoverersMu.Lock()
	defer discoverersMu.Unlock()
	discoverers = nil
}

func currentDiscoverers() []Discoverer {
	discoverersMu.RLock()
	defer discoverersMu.RUnlock()
	combined := append([]Discoverer(nil), builtinDiscoverers()...)
	combined = append(combined, discoverers...)
	return combined
}

func (s *Session) Discover() error {
	if s.closed {
		return errors.New("accel: session closed")
	}
	if s.cfg.Mode == ModeCPU {
		s.setDiscoveryResult(nil)
		return nil
	}

	var found []Device
	var errs []error
	for _, discoverer := range currentDiscoverers() {
		devices, err := runDiscovererWithTimeout(discoverer, s.cfg)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, device := range devices {
			found = append(found, normalizeDiscoveredDevice(device, s.cfg))
		}
	}
	found = dedupeDiscoveredDevices(found)

	s.setDiscoveryResult(found)
	joinedErr := errors.Join(errs...)
	if joinedErr != nil && !s.Report().Accelerated && !strictGPURequired(s.cfg) {
		report := s.Report()
		report.FallbackReason = FallbackReasonDiscoveryError
		report.Metrics = discoveryMetrics(s.devices, report, len(errs))
		if len(s.reports) == 0 {
			s.reports = []Report{cloneReport(report)}
		} else {
			s.reports[len(s.reports)-1] = cloneReport(report)
		}
	}
	if strictGPURequired(s.cfg) && !s.Report().Accelerated {
		strictErr := fmt.Errorf("accel: strict-gpu mode requires an accelerator (%s)", s.Report().FallbackReason)
		if joinedErr != nil {
			return errors.Join(joinedErr, strictErr)
		}
		return strictErr
	}
	return joinedErr
}

func (s *Session) setDiscoveryResult(devices []Device) {
	s.devices = append([]Device(nil), devices...)

	report := s.Report()
	now := time.Now()
	report.GeneratedAt = now
	report.StartedAt = now
	report.FinishedAt = now
	report.SelectedBackend = BackendUnknown
	report.DiscoveredDeviceIDs = deviceIDs(devices)
	report.SelectedDeviceIDs = nil
	report.SelectedDevices = nil
	report.Accelerated = false
	report.FallbackReason = initialFallbackReason(s.cfg.Mode)

	if primary, ok := selectPrimaryDevice(devices, s.cfg.PreferredBackends, s.cfg.PreferredDevices); ok {
		report.Accelerated = true
		report.SelectedBackend = primary.Backend
		report.SelectedDeviceIDs = []string{primary.ID}
		report.SelectedDevices = []string{primary.ID}
		report.FallbackReason = FallbackReasonNone
	}
	if strictGPURequired(s.cfg) && !report.Accelerated {
		report.FallbackReason = FallbackReasonStrictGPUUnavailable
	}
	report.Metrics = discoveryMetrics(devices, report, 0)

	if len(s.reports) == 0 {
		s.reports = []Report{cloneReport(report)}
		return
	}
	s.reports[len(s.reports)-1] = cloneReport(report)
}

func selectPrimaryDevice(devices []Device, preferred []Backend, preferredDevices []string) (Device, bool) {
	for _, preferredID := range preferredDevices {
		for _, device := range devices {
			if device.ID == preferredID {
				return device, true
			}
		}
	}

	for _, backend := range preferred {
		var selected Device
		found := false
		for _, device := range devices {
			if device.Backend == backend {
				if !found || device.Score > selected.Score {
					selected = device
					found = true
				}
			}
		}
		if found {
			return selected, true
		}
	}
	if len(devices) == 0 {
		return Device{}, false
	}
	selected := devices[0]
	for _, device := range devices[1:] {
		if device.Score > selected.Score {
			selected = device
		}
	}
	return selected, true
}

func normalizeDiscoveredDevice(device Device, cfg Config) Device {
	cloned := cloneDevice(device)
	cloned.BudgetBytes = normalizeBudgetBytes(cloned, cfg)
	cloned.CapabilitySummary = normalizeCapabilitySummary(cloned)
	if cloned.Score <= 0 {
		cloned.Score = defaultDeviceScore(cloned)
	}
	return cloned
}

func normalizeCapabilitySummary(device Device) map[string]bool {
	caps := cloneCaps(device.CapabilitySummary)
	if caps == nil {
		caps = map[string]bool{}
	}
	caps["shardable"] = true
	caps["validity_bitmap"] = true
	caps["encoded_strings"] = true
	caps["heterogeneous_planning"] = true
	caps["native_probe"] = device.ProbeSource == ProbeSourceNative
	caps["env_stub"] = device.ProbeSource == ProbeSourceEnvStub

	if device.MemoryClass == MemoryClassShared || device.SharedMemory {
		caps["shared_memory"] = true
		delete(caps, "device_local_memory")
	} else if device.MemoryClass == MemoryClassDevice {
		caps["device_local_memory"] = true
		delete(caps, "shared_memory")
	}
	return caps
}

func strictGPURequired(cfg Config) bool {
	return cfg.Mode == ModeStrictGPU || cfg.Strict
}

func deviceIDs(devices []Device) []string {
	ids := make([]string, 0, len(devices))
	for _, device := range devices {
		ids = append(ids, device.ID)
	}
	return ids
}

func discoveryMetrics(devices []Device, report Report, discoveryErrors int) map[string]float64 {
	metrics := map[string]float64{
		"devices.discovered":           float64(len(devices)),
		"devices.selected":             float64(len(report.SelectedDeviceIDs)),
		"devices.native":               0,
		"devices.env_stub":             0,
		"devices.shared_memory":        0,
		"devices.device_local":         0,
		"fallback.occurred":            1,
		"discovery.errors":             float64(discoveryErrors),
		"memory.budget_bytes_total":    0,
		"memory.budget_bytes_selected": 0,
		"cache.resident_buffers":       0,
		"cache.resident_bytes":         0,
	}
	if report.FallbackReason == FallbackReasonNone {
		metrics["fallback.occurred"] = 0
	}

	selectedSet := make(map[string]struct{}, len(report.SelectedDeviceIDs))
	for _, id := range report.SelectedDeviceIDs {
		selectedSet[id] = struct{}{}
	}
	for _, device := range devices {
		switch device.ProbeSource {
		case ProbeSourceNative:
			metrics["devices.native"]++
		case ProbeSourceEnvStub:
			metrics["devices.env_stub"]++
		}
		switch {
		case device.MemoryClass == MemoryClassShared || device.SharedMemory:
			metrics["devices.shared_memory"]++
		case device.MemoryClass == MemoryClassDevice:
			metrics["devices.device_local"]++
		}
		metrics["memory.budget_bytes_total"] += float64(device.BudgetBytes)
		if _, ok := selectedSet[device.ID]; ok {
			metrics["memory.budget_bytes_selected"] += float64(device.BudgetBytes)
		}
	}
	return metrics
}

func normalizeBudgetBytes(device Device, cfg Config) uint64 {
	baseBudget := inferredDeviceBudgetBytes(device)
	if baseBudget == 0 {
		return 0
	}

	fraction := cfg.MemoryBudget.DeviceFraction
	if device.MemoryClass == MemoryClassShared || device.SharedMemory {
		fraction = cfg.MemoryBudget.SharedFraction
	}
	if fraction <= 0 {
		return baseBudget
	}
	return uint64(float64(baseBudget) * fraction)
}

func inferredDeviceBudgetBytes(device Device) uint64 {
	if device.BudgetBytes > 0 {
		return device.BudgetBytes
	}
	switch {
	case device.MemoryClass == MemoryClassShared || device.SharedMemory:
		if hostBytes := hostMemoryBytes(); hostBytes > 0 {
			return hostBytes
		}
		return 4 * 1024 * 1024 * 1024
	case device.MemoryClass == MemoryClassDevice:
		return 2 * 1024 * 1024 * 1024
	default:
		return 0
	}
}

func defaultDeviceScore(device Device) float64 {
	score := 10.0
	switch device.Backend {
	case BackendCUDA:
		score += 30
	case BackendMetal:
		score += 25
	case BackendWebGPU:
		score += 20
	}

	switch device.Type {
	case DeviceTypeDiscrete:
		score += 30
	case DeviceTypeIntegrated:
		score += 15
	case DeviceTypeCPU:
		score -= 10
	}

	if device.MemoryClass == MemoryClassShared {
		score -= 5
	}

	return score
}

func runDiscovererWithTimeout(discoverer Discoverer, cfg Config) ([]Device, error) {
	timeout := cfg.DiscoveryTimeout
	if timeout <= 0 {
		return discoverer.Discover(cfg)
	}

	type result struct {
		devices []Device
		err     error
	}
	done := make(chan result, 1)
	go func() {
		devices, err := discoverer.Discover(cfg)
		done <- result{devices: devices, err: err}
	}()

	select {
	case res := <-done:
		return res.devices, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("accel: discoverer %s timed out after %s", discoverer.Name(), timeout)
	}
}

func dedupeDiscoveredDevices(devices []Device) []Device {
	if len(devices) < 2 {
		return devices
	}

	bestIndex := map[string]int{}
	for idx, device := range devices {
		key := canonicalDeviceKey(device)
		current, ok := bestIndex[key]
		if !ok || backendPriority(device.Backend) < backendPriority(devices[current].Backend) {
			bestIndex[key] = idx
		}
	}

	deduped := make([]Device, 0, len(bestIndex))
	for idx, device := range devices {
		if bestIndex[canonicalDeviceKey(device)] == idx {
			deduped = append(deduped, device)
		}
	}
	return deduped
}

func canonicalDeviceKey(device Device) string {
	name := device.Name
	if name == "" {
		name = device.ID
	}
	return string(normalizeVendor(device.Vendor)) + ":" + name
}

func backendPriority(backend Backend) int {
	switch backend {
	case BackendCUDA:
		return 0
	case BackendMetal:
		return 1
	case BackendWebGPU:
		return 2
	default:
		return 3
	}
}
