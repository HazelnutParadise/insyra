package insyra

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/ccl"
)

// capturedRecord is what recordingHandler keeps of one slog record.
type capturedRecord struct {
	level   slog.Level
	message string
	attrs   map[string]string
	pc      uintptr
}

// recordingHandler is a slog.Handler that accepts every level and keeps each
// record it is given.
type recordingHandler struct {
	mu      sync.Mutex
	records []capturedRecord
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]string{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, capturedRecord{level: r.Level, message: r.Message, attrs: attrs, pc: r.PC})
	h.mu.Unlock()
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func (h *recordingHandler) snapshot() []capturedRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]capturedRecord(nil), h.records...)
}

// useRecordingLogger installs a recording logger at the given insyra level,
// captures the standard log package, and restores both when the test ends.
func useRecordingLogger(t *testing.T, level LogLevel) (*recordingHandler, *bytes.Buffer) {
	t.Helper()
	var std bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	previousLevel := Config.GetLogLevel()
	previousLogger := Config.GetLogger()
	log.SetOutput(&std)
	log.SetFlags(0)
	log.SetPrefix("")
	Config.SetLogLevel(level)
	handler := &recordingHandler{}
	Config.SetLogger(slog.New(handler))
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
		Config.SetLogLevel(previousLevel)
		Config.SetLogger(previousLogger)
	})
	return handler, &std
}

func TestSetLoggerReceivesEachLevel(t *testing.T) {
	handler, std := useRecordingLogger(t, LogLevelDebug)
	// LogFatal ends the program unless Config.SetDontPanic is on.
	previousDontPanic := Config.GetDontPanicStatus()
	Config.SetDontPanic(true)
	defer Config.SetDontPanic(previousDontPanic)

	LogDebug("pkg", "Fn", "value %d", 7)
	LogInfo("pkg", "Fn", "value %d", 7)
	LogWarning("pkg", "Fn", "value %d", 7)
	LogFatal("pkg", "Fn", "value %d", 7)

	records := handler.snapshot()
	want := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError + 4}
	if len(records) != len(want) {
		t.Fatalf("got %d records, want %d: %+v", len(records), len(want), records)
	}
	for i, r := range records {
		if r.level != want[i] {
			t.Errorf("record %d level = %v, want %v", i, r.level, want[i])
		}
		if r.message != "value 7" {
			t.Errorf("record %d message = %q, want %q", i, r.message, "value 7")
		}
		if r.attrs["package"] != "pkg" || r.attrs["func"] != "Fn" || len(r.attrs) != 2 {
			t.Errorf("record %d attrs = %v, want package=pkg func=Fn", i, r.attrs)
		}
	}
	if std.Len() != 0 {
		t.Fatalf("the standard log received output while a logger was set:\n%s", std.String())
	}
}

func TestSetLoggerReceivesARecordedErrorAtWarningLevel(t *testing.T) {
	handler, _ := useRecordingLogger(t, LogLevelInfo)

	NewDataTable(NewDataList(1)).GetCol("Z")

	var found bool
	for _, r := range handler.snapshot() {
		if r.attrs["package"] == "DataTable" && r.attrs["func"] == "GetCol" {
			found = true
			if r.level != slog.LevelWarn {
				t.Errorf("GetCol's failure arrived at %v, want WARN", r.level)
			}
			if strings.Contains(r.message, "\x1b") || strings.Contains(r.message, "[insyra") {
				t.Errorf("message carries the text format: %q", r.message)
			}
		}
	}
	if !found {
		t.Fatalf("no record for DataTable.GetCol: %+v", handler.snapshot())
	}
}

func TestSetLoggerOmitsEmptyNames(t *testing.T) {
	handler, _ := useRecordingLogger(t, LogLevelInfo)

	LogInfo("", "", "hello")

	records := handler.snapshot()
	if len(records) != 1 || records[0].message != "hello" || len(records[0].attrs) != 0 {
		t.Fatalf("got %+v, want one record \"hello\" without attributes", records)
	}
}

func TestSetLoggerSourceIsTheCaller(t *testing.T) {
	handler, _ := useRecordingLogger(t, LogLevelInfo)

	LogInfo("pkg", "Fn", "where")

	records := handler.snapshot()
	if len(records) != 1 || records[0].pc == 0 {
		t.Fatalf("got %+v, want one record with a program counter", records)
	}
	frame, _ := runtime.CallersFrames([]uintptr{records[0].pc}).Next()
	if !strings.HasSuffix(frame.Function, "TestSetLoggerSourceIsTheCaller") {
		t.Fatalf("record source is %s, want the test function", frame.Function)
	}
}

func TestSetLoggerKeepsInsyrasLevel(t *testing.T) {
	handler, _ := useRecordingLogger(t, LogLevelWarning)

	LogInfo("pkg", "Fn", "quiet")
	LogWarning("pkg", "Fn", "loud")

	records := handler.snapshot()
	if len(records) != 1 || records[0].level != slog.LevelWarn || records[0].message != "loud" {
		t.Fatalf("got %+v, want only the warning", records)
	}
}

func TestSetLoggerNilRestoresTheDefaultOutput(t *testing.T) {
	handler, std := useRecordingLogger(t, LogLevelInfo)

	Config.SetLogger(nil)
	if Config.GetLogger() != nil {
		t.Fatal("GetLogger is not nil after SetLogger(nil)")
	}
	LogWarning("pkg", "Fn", "careful")

	if len(handler.snapshot()) != 0 {
		t.Fatalf("the cleared logger still received records: %+v", handler.snapshot())
	}
	if got, want := std.String(), "[insyra - Warning] pkg.Fn: Careful\n"; got != want {
		t.Fatalf("standard log got %q, want %q", got, want)
	}
}

func TestGetLoggerReturnsTheLoggerSet(t *testing.T) {
	useRecordingLogger(t, LogLevelInfo)
	l := slog.New(&recordingHandler{})
	Config.SetLogger(l)
	if Config.GetLogger() != l {
		t.Fatal("GetLogger does not return the logger passed to SetLogger")
	}
}

func TestSetDefaultConfigClearsTheLogger(t *testing.T) {
	previous := struct {
		level       LogLevel
		colored     bool
		panicOnErr  bool
		threadSafe  bool
		accelerated bool
	}{Config.GetLogLevel(), Config.GetDoesUseColoredOutput(), Config.GetDontPanicStatus(), Config.threadSafe.Load(), Config.GetAccelerationEnabled()}
	useRecordingLogger(t, LogLevelInfo)
	t.Cleanup(func() {
		Config.SetLogLevel(previous.level)
		Config.SetUseColoredOutput(previous.colored)
		Config.SetDontPanic(previous.panicOnErr)
		Config.threadSafe.Store(previous.threadSafe)
		Config.acceleration.Store(previous.accelerated)
	})

	SetDefaultConfig()

	if Config.GetLogger() != nil {
		t.Fatal("SetDefaultConfig left the logger set")
	}
}

func TestCCLWarningsReachTheLogger(t *testing.T) {
	handler, _ := useRecordingLogger(t, LogLevelInfo)

	ccl.Warn("evaluate", "could not restore row %d", 3)

	records := handler.snapshot()
	if len(records) != 1 || records[0].level != slog.LevelWarn || records[0].message != "could not restore row 3" ||
		records[0].attrs["package"] != "ccl" || records[0].attrs["func"] != "evaluate" {
		t.Fatalf("got %+v, want one Warn record from ccl.evaluate", records)
	}
}

func TestSetLoggerIsSafeWhileLogging(t *testing.T) {
	useRecordingLogger(t, LogLevelInfo)
	l := slog.New(&recordingHandler{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			Config.SetLogger(l)
			Config.SetLogger(nil)
			_ = Config.GetLogger()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			LogInfo("pkg", "Fn", "tick %d", i)
		}
	}()
	wg.Wait()
}
