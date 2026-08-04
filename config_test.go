package insyra

import "testing"

func TestAccelerationConfigDefaultsEnabled(t *testing.T) {
	previous := Config.GetAccelerationEnabled()
	t.Cleanup(func() {
		Config.SetAcceleration(previous)
	})
	if !Config.GetAccelerationEnabled() {
		t.Fatal("default acceleration is disabled")
	}

	Config.SetAcceleration(false)
	if Config.GetAccelerationEnabled() {
		t.Fatal("SetAcceleration(false) did not disable acceleration")
	}
	Config.SetAcceleration(true)
	if !Config.GetAccelerationEnabled() {
		t.Fatal("SetAcceleration(true) did not re-enable acceleration")
	}
}
