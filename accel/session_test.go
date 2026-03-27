package accel

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Mode != ModeAuto {
		t.Fatalf("expected ModeAuto, got %q", cfg.Mode)
	}
	if cfg.MemoryBudget.DeviceFraction != 0.60 {
		t.Fatalf("expected device memory fraction 0.60, got %.2f", cfg.MemoryBudget.DeviceFraction)
	}
	if cfg.MemoryBudget.SharedFraction != 0.35 {
		t.Fatalf("expected shared memory fraction 0.35, got %.2f", cfg.MemoryBudget.SharedFraction)
	}
}

func TestNewSessionUsesDefaults(t *testing.T) {
	s := NewSession()
	cfg := s.Config()

	if cfg.Mode != ModeAuto {
		t.Fatalf("expected default mode auto, got %q", cfg.Mode)
	}
	if s.Closed() {
		t.Fatal("session should start open")
	}
}

func TestRegisterDeviceAndDefensiveCopy(t *testing.T) {
	s := NewSession()
	err := s.RegisterDevice(Device{
		ID:      "cuda:0",
		Name:    "GPU 0",
		Backend: BackendCUDA,
		Type:    DeviceTypeDiscrete,
	})
	if err != nil {
		t.Fatalf("register device failed: %v", err)
	}

	devices := s.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	devices[0].Name = "mutated"

	again := s.Devices()
	if again[0].Name != "GPU 0" {
		t.Fatal("devices should be defensively copied")
	}
}

func TestRecordReportAppendsHistory(t *testing.T) {
	s := NewSession(Config{
		Mode:              ModeAuto,
		PreferredBackends: []Backend{BackendCUDA},
		MemoryBudget:      MemoryBudgetPolicy{DeviceFraction: 0.60, SharedFraction: 0.35},
	})

	report := Report{
		Mode:            ModeAuto,
		Accelerated:     true,
		SelectedBackend: BackendCUDA,
		SelectedDevices: []string{"cuda:0"},
		StartedAt:       time.Unix(0, 0),
		FinishedAt:      time.Unix(1, 0),
	}
	if err := s.RecordReport(report); err != nil {
		t.Fatalf("record report failed: %v", err)
	}

	reports := s.Reports()
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports including initial report, got %d", len(reports))
	}
	if reports[1].SelectedDevices[0] != "cuda:0" {
		t.Fatalf("expected recorded device id cuda:0, got %v", reports[1].SelectedDevices)
	}
}

func TestClosePreventsMutation(t *testing.T) {
	s := NewSession()
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if !s.Closed() {
		t.Fatal("session should report closed")
	}
	if err := s.RegisterDevice(Device{ID: "cpu:0"}); err == nil {
		t.Fatal("expected register device to fail after close")
	}
	if err := s.RecordReport(Report{}); err == nil {
		t.Fatal("expected record report to fail after close")
	}
}

func TestReportDuration(t *testing.T) {
	r := Report{
		StartedAt:  time.Unix(0, 0),
		FinishedAt: time.Unix(5, 0),
	}
	if got := r.Duration(); got != 5*time.Second {
		t.Fatalf("expected 5s duration, got %v", got)
	}
}
