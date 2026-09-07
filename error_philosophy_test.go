package insyra

import (
	"errors"
	"testing"
)

// restoreConfig snapshots the global config knobs this file touches and puts
// them back when the test ends.
func restoreConfig(t *testing.T) {
	t.Helper()
	level := Config.GetLogLevel()
	panicOnError := Config.GetPanicOnError()
	t.Cleanup(func() {
		Config.SetLogLevel(level)
		Config.SetPanicOnError(panicOnError)
	})
}

// K-1: a library must not end its caller's process. With the default config
// a fatal is recorded and execution continues.
func TestLogFatalDoesNotTerminate(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)
	ClearErrors()

	reached := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("LogFatal panicked with the default config: %v", r)
			}
		}()
		LogFatal("test", "TestLogFatalDoesNotTerminate", "boom")
		reached = true
	}()
	if !reached {
		t.Fatal("execution did not continue past LogFatal")
	}
	if !HasError() {
		t.Fatal("LogFatal did not record the error in the global buffer")
	}
}

// K-1: panicking is opt-in, and it is a panic (recoverable), never os.Exit.
func TestPanicOnErrorIsOptInAndRecoverable(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(true)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		LogFatal("test", "TestPanicOnErrorIsOptInAndRecoverable", "boom")
	}()
	if recovered == nil {
		t.Fatal("SetPanicOnError(true) did not panic on a fatal")
	}
	var asErr error
	if e, ok := recovered.(error); ok {
		asErr = e
	}
	if asErr == nil {
		t.Fatalf("panic value %T does not implement error", recovered)
	}

	// An error recorded on an instance panics under the same switch.
	dl := NewDataList(1.0, 2.0)
	func() {
		defer func() { recovered = recover() }()
		recovered = nil
		dl.MovingAverage(0)
	}()
	if recovered == nil {
		t.Fatal("SetPanicOnError(true) did not panic on an instance error")
	}
}

// K-3: Error sits between Warning and Fatal, and LogError uses it.
func TestErrorLevelOrdering(t *testing.T) {
	restoreConfig(t)
	if LogLevelDebug >= LogLevelInfo || LogLevelInfo >= LogLevelWarning ||
		LogLevelWarning >= LogLevelError || LogLevelError >= LogLevelFatal {
		t.Fatalf("levels out of order: debug=%d info=%d warning=%d error=%d fatal=%d",
			LogLevelDebug, LogLevelInfo, LogLevelWarning, LogLevelError, LogLevelFatal)
	}
	if got := LogLevelError.String(); got != "ERROR" {
		t.Fatalf("LogLevelError.String() = %q, want %q", got, "ERROR")
	}

	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)
	ClearErrors()
	LogError("test", "TestErrorLevelOrdering", "recorded")
	errs := GetAllErrors()
	if len(errs) == 0 || errs[len(errs)-1].Level != LogLevelError {
		t.Fatalf("LogError did not record at Error level: %v", errs)
	}

	// An instance failure is an Error, not a Warning.
	dl := NewDataList(1.0, "x")
	dl.Normalize()
	err := dl.Err()
	if err == nil {
		t.Fatal("Normalize over a non-numeric cell recorded nothing")
	}
	if err.Level != LogLevelError {
		t.Fatalf("instance error level = %v, want Error", err.Level)
	}
	if !errors.Is(error(*err), error(*err)) { // ErrorInfo satisfies error
		t.Fatal("ErrorInfo does not behave as an error")
	}
}

// SetDontPanic stays one release as the inverse of SetPanicOnError.
func TestSetDontPanicIsDeprecatedAlias(t *testing.T) {
	restoreConfig(t)
	Config.SetDontPanic(false)
	if !Config.GetPanicOnError() {
		t.Fatal("SetDontPanic(false) should mean panic on error")
	}
	if Config.GetDontPanicStatus() {
		t.Fatal("GetDontPanicStatus should mirror SetDontPanic(false)")
	}
	Config.SetDontPanic(true)
	if Config.GetPanicOnError() {
		t.Fatal("SetDontPanic(true) should mean do not panic")
	}
	if !Config.GetDontPanicStatus() {
		t.Fatal("GetDontPanicStatus should mirror SetDontPanic(true)")
	}
}
