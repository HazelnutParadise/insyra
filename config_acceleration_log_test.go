package insyra

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// TestSetAccelerationLogsOnlyOnTransition pins the toggle's log contract: a
// real state change writes one info line naming the new state, and repeating
// the same value writes nothing.
func TestSetAccelerationLogsOnlyOnTransition(t *testing.T) {
	var output bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	previousLevel := Config.GetLogLevel()
	previousEnabled := Config.GetAccelerationEnabled()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	Config.SetLogLevel(LogLevelInfo)
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
		Config.SetLogLevel(previousLevel)
		Config.SetAcceleration(previousEnabled)
	})

	Config.SetAcceleration(true) // establish a known state; may or may not log
	output.Reset()

	Config.SetAcceleration(false)
	if got := strings.Count(output.String(), "Acceleration disabled by config"); got != 1 {
		t.Fatalf("disable transition logged %d times, want 1; output:\n%s", got, output.String())
	}

	output.Reset()
	Config.SetAcceleration(false)
	if output.Len() != 0 {
		t.Fatalf("repeated same-value toggle logged; output:\n%s", output.String())
	}

	output.Reset()
	Config.SetAcceleration(true)
	if got := strings.Count(output.String(), "Acceleration enabled by config"); got != 1 {
		t.Fatalf("enable transition logged %d times, want 1; output:\n%s", got, output.String())
	}
}
