package insyra

import (
	"bytes"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
)

// The thread-safety switch can be turned back on without resetting the other
// settings, as SetDefaultConfig would.

func TestTurnOnThreadSafetyKeepsTheOtherSettings(t *testing.T) {
	defer SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
	Config.SetUseColoredOutput(false)
	Config.SetAcceleration(false)
	Config.Dangerously_TurnOffThreadSafety()
	Config.TurnOnThreadSafety()
	if !Config.threadSafe.Load() {
		t.Fatal("thread safety is still off after TurnOnThreadSafety")
	}
	if Config.GetLogLevel() != LogLevelFatal || Config.GetUseColoredOutput() || Config.GetAccelerationEnabled() {
		t.Fatalf("TurnOnThreadSafety changed other settings: log level %v, coloured %v, acceleration %v",
			Config.GetLogLevel(), Config.GetUseColoredOutput(), Config.GetAccelerationEnabled())
	}
}

func TestTurnOnThreadSafetyLocksAgain(t *testing.T) {
	defer SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
	Config.Dangerously_TurnOffThreadSafety()
	list := NewDataList()
	Config.TurnOnThreadSafety()

	const goroutines, appends = 64, 100
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < appends; i++ {
				list.Append(i)
			}
		}()
	}
	wg.Wait()
	if got := list.Len(); got != goroutines*appends {
		t.Fatalf("after %d concurrent appends the list holds %d", goroutines*appends, got)
	}
}

func TestTurnOnThreadSafetyLogsOnlyATransition(t *testing.T) {
	defer SetDefaultConfig()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	Config.SetLogLevel(LogLevelFatal)
	Config.Dangerously_TurnOffThreadSafety()
	Config.SetLogLevel(LogLevelInfo)
	buf.Reset()
	Config.TurnOnThreadSafety()
	Config.TurnOnThreadSafety()
	if got := strings.Count(buf.String(), "[insyra - Info] config.TurnOnThreadSafety"); got != 1 {
		t.Fatalf("TurnOnThreadSafety logged %d Info lines over one transition and one repeat; want 1. Log:\n%s", got, buf.String())
	}
}

func TestThreadSafetySwitchSaysWhatItTurnsOff(t *testing.T) {
	doc := rootDocComments(t, "config.go")["Dangerously_TurnOffThreadSafety"]
	for _, want := range []string{"AtomicDo", "AtomicDoAll", "Config, the global error buffer and logging stay safe", "TurnOnThreadSafety"} {
		if !strings.Contains(doc, want) {
			t.Errorf("Dangerously_TurnOffThreadSafety doc is %q; want it to mention %q", doc, want)
		}
	}
}
