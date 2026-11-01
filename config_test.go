package insyra

import (
	"os"
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

func TestConfigSettersRaceFree(t *testing.T) {
	restoreConfig(t)
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
				_ = colorText(os.Stdout, "1", "y")
				_ = Config.GetDontPanicStatus()
			}
		}()
	}
	wg.Wait()
	Config.SetLogLevel(LogLevelFatal)
}
