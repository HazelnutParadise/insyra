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
		devices, err := discoverer.Discover(s.cfg)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, device := range devices {
			found = append(found, normalizeDiscoveredDevice(device, s.cfg))
		}
	}

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
	if cloned.Score <= 0 {
		cloned.Score = defaultDeviceScore(cloned)
	}
	return cloned
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
		metrics["memory.budget_bytes_total"] += float64(device.BudgetBytes)
		if _, ok := selectedSet[device.ID]; ok {
			metrics["memory.budget_bytes_selected"] += float64(device.BudgetBytes)
		}
	}
	return metrics
}

func normalizeBudgetBytes(device Device, cfg Config) uint64 {
	if device.BudgetBytes == 0 {
		return 0
	}

	fraction := cfg.MemoryBudget.DeviceFraction
	if device.MemoryClass == MemoryClassShared || device.SharedMemory {
		fraction = cfg.MemoryBudget.SharedFraction
	}
	if fraction <= 0 {
		return device.BudgetBytes
	}
	return uint64(float64(device.BudgetBytes) * fraction)
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
