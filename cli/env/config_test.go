package env

import "testing"

func TestGlobalConfigDefaultsAndUpdatesAccelMode(t *testing.T) {
	setupTempHome(t)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load global config failed: %v", err)
	}
	if cfg.AccelMode != "auto" {
		t.Fatalf("expected default accel mode auto, got %q", cfg.AccelMode)
	}

	updated, err := UpdateGlobalConfig("accel.mode", "strict-gpu")
	if err != nil {
		t.Fatalf("update global config failed: %v", err)
	}
	if updated.AccelMode != "strict-gpu" {
		t.Fatalf("expected updated accel mode strict-gpu, got %q", updated.AccelMode)
	}
}
