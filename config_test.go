package insyra

import (
	"sync"
	"testing"
)

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

// keepConfig restores the global Config fields these tests change, so the
// rest of the package's tests run with the configuration they started with.
func keepConfig(t *testing.T) {
	t.Helper()
	level := Config.GetLogLevel()
	colored := Config.GetUseColoredOutput()
	dontPanic := Config.GetDontPanicStatus()
	hook := Config.GetDefaultErrHandlingFunc()
	t.Cleanup(func() {
		Config.SetLogLevel(level)
		Config.SetUseColoredOutput(colored)
		Config.SetDontPanic(dontPanic)
		Config.SetDefaultErrHandlingFunc(hook)
	})
}

func TestConfigSettersRaceFree(t *testing.T) {
	keepConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				Config.SetLogLevel(LogLevelWarning)
				Config.SetUseColoredOutput(j%2 == 0)
				Config.SetDontPanic(true)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				LogInfo("t", "t", "x")
				_ = Config.GetLogLevel()
				_ = colorText("1", "y")
				_ = Config.GetDontPanicStatus()
			}
		}()
	}
	wg.Wait()
}
