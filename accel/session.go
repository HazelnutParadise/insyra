package accel

import (
	"fmt"
	"time"
)

type Session struct {
	cfg     Config
	devices []Device
	reports []Report
	closed  bool
}

func Open(cfg Config) (*Session, error) {
	session := NewSession(cfg)
	if err := session.Discover(); err != nil {
		return session, err
	}
	return session, nil
}

func NewSession(cfgs ...Config) *Session {
	cfg := DefaultConfig()
	if len(cfgs) > 0 {
		cfg = normalizeConfig(cfgs[0])
	}

	baseReport := Report{
		Mode:              cfg.Mode,
		SelectedBackend:   BackendUnknown,
		FallbackReason:    initialFallbackReason(cfg.Mode),
		GeneratedAt:       time.Now(),
		StartedAt:         time.Now(),
		FinishedAt:        time.Now(),
		DiscoveredDeviceIDs: nil,
		SelectedDeviceIDs: nil,
	}

	return &Session{
		cfg:     cfg,
		reports: []Report{baseReport},
	}
}

func (s *Session) Config() Config {
	cfg := s.cfg
	cfg.PreferredBackends = append([]Backend(nil), s.cfg.PreferredBackends...)
	cfg.PreferredDevices = append([]string(nil), s.cfg.PreferredDevices...)
	return cfg
}

func (s *Session) Devices() []Device {
	cloned := make([]Device, len(s.devices))
	for i, device := range s.devices {
		cloned[i] = cloneDevice(device)
	}
	return cloned
}

func (s *Session) Report() Report {
	if len(s.reports) == 0 {
		return Report{}
	}
	return cloneReport(s.reports[len(s.reports)-1])
}

func (s *Session) Reports() []Report {
	cloned := make([]Report, len(s.reports))
	for i, report := range s.reports {
		cloned[i] = cloneReport(report)
	}
	return cloned
}

func (s *Session) LastReport() *Report {
	if len(s.reports) == 0 {
		return nil
	}
	report := s.Report()
	return &report
}

func (s *Session) RegisterDevice(device Device) error {
	if s.closed {
		return fmt.Errorf("accel: session closed")
	}
	if device.ID == "" {
		return fmt.Errorf("accel: device id is required")
	}
	s.devices = append(s.devices, cloneDevice(device))
	return nil
}

func (s *Session) RecordReport(report Report) error {
	if s.closed {
		return fmt.Errorf("accel: session closed")
	}
	cloned := cloneReport(report)
	if cloned.Mode == "" {
		cloned.Mode = s.cfg.Mode
	}
	if len(cloned.SelectedDevices) == 0 && len(cloned.SelectedDeviceIDs) > 0 {
		cloned.SelectedDevices = append([]string(nil), cloned.SelectedDeviceIDs...)
	}
	if len(cloned.SelectedDeviceIDs) == 0 && len(cloned.SelectedDevices) > 0 {
		cloned.SelectedDeviceIDs = append([]string(nil), cloned.SelectedDevices...)
	}
	if cloned.GeneratedAt.IsZero() {
		cloned.GeneratedAt = time.Now()
	}
	if cloned.StartedAt.IsZero() {
		cloned.StartedAt = time.Now()
	}
	if cloned.FinishedAt.IsZero() {
		cloned.FinishedAt = cloned.StartedAt
	}
	s.reports = append(s.reports, cloned)
	limit := s.cfg.ReportHistorySize
	if limit <= 0 {
		limit = 1
	}
	if len(s.reports) > limit {
		s.reports = append([]Report(nil), s.reports[len(s.reports)-limit:]...)
	}
	return nil
}

func (s *Session) Close() error {
	s.closed = true
	return nil
}

func (s *Session) Closed() bool {
	return s.closed
}

func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()

	if cfg.Mode == "" {
		cfg.Mode = defaults.Mode
	}
	if cfg.MemoryBudget.DeviceFraction <= 0 {
		cfg.MemoryBudget.DeviceFraction = defaults.MemoryBudget.DeviceFraction
	}
	if cfg.MemoryBudget.SharedFraction <= 0 {
		cfg.MemoryBudget.SharedFraction = defaults.MemoryBudget.SharedFraction
	}
	if len(cfg.PreferredBackends) == 0 {
		cfg.PreferredBackends = append([]Backend(nil), defaults.PreferredBackends...)
	} else {
		cfg.PreferredBackends = append([]Backend(nil), cfg.PreferredBackends...)
	}
	cfg.PreferredDevices = append([]string(nil), cfg.PreferredDevices...)
	if cfg.ReportHistorySize <= 0 {
		cfg.ReportHistorySize = defaults.ReportHistorySize
	}
	if cfg.DiscoveryTimeout <= 0 {
		cfg.DiscoveryTimeout = defaults.DiscoveryTimeout
	}
	if !cfg.EnableFallback && !cfg.Strict && cfg.Mode != ModeStrictGPU {
		cfg.EnableFallback = defaults.EnableFallback
	}
	return cfg
}

func initialFallbackReason(mode Mode) FallbackReason {
	if mode == ModeCPU {
		return FallbackReasonCPUOnly
	}
	return FallbackReasonNoAccelerator
}

func cloneDevice(device Device) Device {
	cloned := device
	if device.CapabilitySummary != nil {
		cloned.CapabilitySummary = make(map[string]bool, len(device.CapabilitySummary))
		for key, value := range device.CapabilitySummary {
			cloned.CapabilitySummary[key] = value
		}
	}
	if cloned.MemoryClass == "" {
		cloned.MemoryClass = MemoryClassUnknown
	}
	if cloned.MemoryClass == MemoryClassShared {
		cloned.SharedMemory = true
	}
	return cloned
}

func cloneReport(report Report) Report {
	cloned := report
	cloned.DiscoveredDeviceIDs = append([]string(nil), report.DiscoveredDeviceIDs...)
	cloned.SelectedDeviceIDs = append([]string(nil), report.SelectedDeviceIDs...)
	cloned.SelectedDevices = append([]string(nil), report.SelectedDevices...)
	if report.Metrics != nil {
		cloned.Metrics = make(map[string]float64, len(report.Metrics))
		for key, value := range report.Metrics {
			cloned.Metrics[key] = value
		}
	}
	return cloned
}
