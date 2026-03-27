package accel

import (
	"errors"
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
	return append([]Discoverer(nil), discoverers...)
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
			found = append(found, normalizeDiscoveredDevice(device))
		}
	}

	s.setDiscoveryResult(found)
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (s *Session) setDiscoveryResult(devices []Device) {
	s.devices = append([]Device(nil), devices...)

	report := s.Report()
	now := time.Now()
	report.GeneratedAt = now
	report.StartedAt = now
	report.FinishedAt = now
	report.SelectedBackend = BackendUnknown
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

func normalizeDiscoveredDevice(device Device) Device {
	cloned := cloneDevice(device)
	if cloned.Score <= 0 {
		cloned.Score = defaultDeviceScore(cloned)
	}
	return cloned
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
