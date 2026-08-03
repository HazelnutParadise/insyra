package accel

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

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

func TestDeviceMatMulConfigDisabledBeforeSession(t *testing.T) {
	previous := insyra.Config.GetAccelerationEnabled()
	t.Cleanup(func() { insyra.Config.SetAcceleration(previous) })
	insyra.Config.SetAcceleration(false)

	_, err := DeviceMatMul([]float32{1}, 1, 1, []float32{1}, 1, 1)
	if err == nil {
		t.Fatal("Config-disabled DeviceMatMul returned nil error")
	}
}
