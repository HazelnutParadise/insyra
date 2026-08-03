package accel

import "testing"

func TestStrictGPUModeStillFailsWithoutDevice(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	t.Setenv("INSYRA_ACCEL_DISABLE_WGPU", "1")
	session, err := Open(Config{Mode: ModeStrictGPU})
	if err == nil {
		t.Fatal("strict GPU mode returned nil error without a device")
	}
	if session == nil {
		t.Fatal("strict GPU mode did not return its session report")
	}
	if got := session.Report().FallbackReason; got != FallbackReasonStrictGPUUnavailable {
		t.Fatalf("strict GPU fallback reason = %q, want %q", got, FallbackReasonStrictGPUUnavailable)
	}
}
