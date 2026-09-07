// config.go

package insyra

import "sync/atomic"

// errHandlingFunc is the signature of the hook installed by
// SetDefaultErrHandlingFunc.
type errHandlingFunc = func(errType LogLevel, packageName string, funcName string, errMsg string)

type configStruct struct {
	// Every field below is read on logging hot paths and written by setters
	// that a program may call at any time, so all of them are atomic.
	logLevel               atomic.Int32
	coloredOutput          atomic.Bool
	panicOnError           atomic.Bool
	defaultErrHandlingFunc atomic.Pointer[errHandlingFunc]
	// threadSafe is read on every AtomicDo (hot path) and written by
	// Dangerously_TurnOffThreadSafety / SetDefaultConfig; use an atomic to avoid
	// a data race between those.
	threadSafe atomic.Bool
	// acceleration is read by device dispatch hot paths and written by
	// SetAcceleration / SetDefaultConfig; use an atomic to avoid a data race.
	acceleration atomic.Bool
}

var Config *configStruct = &configStruct{}

type LogLevel int

const (
	// LogLevelDebug is the log level for debug messages.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is the log level for info messages.
	LogLevelInfo
	// LogLevelWarning is the log level for messages about something that was
	// done but is worth noticing. A warning never reaches Err().
	LogLevelWarning
	// LogLevelError is the log level for a call that could not do what it was
	// asked. Everything recorded on an instance's Err() uses this level.
	LogLevelError
	// LogLevelFatal is the log level for a failure the library cannot work
	// around. It still does not end the process: see Config.SetPanicOnError.
	LogLevelFatal
)

func (c *configStruct) SetLogLevel(level LogLevel) {
	c.logLevel.Store(int32(level))
}

func (c *configStruct) GetLogLevel() LogLevel {
	return LogLevel(c.logLevel.Load())
}

func (c *configStruct) SetUseColoredOutput(colored bool) {
	c.coloredOutput.Store(colored)
}

func (c *configStruct) GetDoesUseColoredOutput() bool {
	return c.coloredOutput.Load()
}

// SetPanicOnError controls what happens when the library records an error.
//
// The default is false: insyra never ends or interrupts your program. An
// error is written to the instance's Err() (for DataList, DataTable and isr)
// and to the global diagnostic buffer, and the call returns something usable.
//
// Set it to true to fail fast instead: every recorded error, including the
// ones LogFatal produces, panics immediately with an *ErrorInfo (which
// implements error). It is a panic, never os.Exit, so a caller can recover.
// This suits a script or a notebook, where stopping at the first mistake beats
// carrying on with empty data.
func (c *configStruct) SetPanicOnError(panicOnError bool) {
	c.panicOnError.Store(panicOnError)
}

// GetPanicOnError reports whether a recorded error panics; see SetPanicOnError.
func (c *configStruct) GetPanicOnError() bool {
	return c.panicOnError.Load()
}

// SetDontPanic is the inverse of SetPanicOnError.
//
// Deprecated: use SetPanicOnError. insyra no longer terminates the process on
// a fatal, so "don't panic" is the default and this setter only remains so
// existing code keeps compiling. It will be removed in the release after next.
func (c *configStruct) SetDontPanic(dontPanic bool) {
	c.panicOnError.Store(!dontPanic)
}

// GetDontPanicStatus reports the inverse of GetPanicOnError.
//
// Deprecated: use GetPanicOnError. It will be removed in the release after
// next.
func (c *configStruct) GetDontPanicStatus() bool {
	return !c.panicOnError.Load()
}

// SetDefaultErrHandlingFunc installs a hook that receives every warning and
// fatal message. The hook runs on one dedicated goroutine, in the order the
// messages were produced, behind a bounded queue: if the queue is full the
// message is still kept in the error buffer but the hook call is dropped.
// Pass nil to remove the hook.
func (c *configStruct) SetDefaultErrHandlingFunc(fn func(errType LogLevel, packageName string, funcName string, errMsg string)) {
	if fn == nil {
		c.defaultErrHandlingFunc.Store(nil)
		return
	}
	c.defaultErrHandlingFunc.Store(&fn)
}

func (c *configStruct) GetDefaultErrHandlingFunc() func(errType LogLevel, packageName string, funcName string, errMsg string) {
	p := c.defaultErrHandlingFunc.Load()
	if p == nil {
		return nil
	}
	return *p
}

// SetAcceleration controls whether device acceleration may be used.
func (c *configStruct) SetAcceleration(enabled bool) {
	// Log only on an actual transition: the toggle is the event worth a line,
	// and repeated same-value calls should stay silent.
	if previous := c.acceleration.Swap(enabled); previous != enabled {
		if enabled {
			LogInfo("insyra", "SetAcceleration", "acceleration enabled by config")
		} else {
			LogInfo("insyra", "SetAcceleration", "acceleration disabled by config")
		}
	}
}

// GetAccelerationEnabled reports whether device acceleration is enabled.
func (c *configStruct) GetAccelerationEnabled() bool {
	return c.acceleration.Load()
}

// # NOT RECOMMENDED!
//
// Dangerously_TurnOffThreadSafety turns off thread safety for all data structures.
// You can enjoy extreme performance boost, but data consistency is NOT guaranteed.
func (c *configStruct) Dangerously_TurnOffThreadSafety() {
	c.threadSafe.Store(false)
	LogWarning("config", "Dangerously_TurnOffThreadSafety", "Thread safety is turned off. Data consistency is NOT guaranteed!\nIt may be a mistake. Remove `Dangerously_TurnOffThreadSafety()` in your code to restore thread safety.")
}

// ======================== Configs ========================

// SetDefaultConfig resets every Config field to its default value: log level
// Info, coloured output on, panic-on-error off, no error hook, thread safety
// on, acceleration on.
func SetDefaultConfig() {
	Config.logLevel.Store(int32(LogLevelInfo))
	Config.coloredOutput.Store(true)
	Config.panicOnError.Store(false)
	Config.defaultErrHandlingFunc.Store(nil)
	Config.threadSafe.Store(true)
	Config.acceleration.Store(true)
}
