// config.go

package insyra

import (
	"log/slog"
	"sync/atomic"
)

// errHandlingFunc is the signature of the hook installed by
// SetDefaultErrHandlingFunc.
type errHandlingFunc = func(errType LogLevel, packageName string, funcName string, errMsg string)

type configStruct struct {
	// Every field below is read on logging hot paths and written by setters
	// that a program may call at any time, so all of them are atomic.
	logLevel               atomic.Int32
	coloredOutput          atomic.Bool
	dontPanic              atomic.Bool
	defaultErrHandlingFunc atomic.Pointer[errHandlingFunc]
	// threadSafe is read on every AtomicDo (hot path) and written by
	// Dangerously_TurnOffThreadSafety / SetDefaultConfig; use an atomic to avoid
	// a data race between those.
	threadSafe atomic.Bool
	// acceleration is read by device dispatch hot paths and written by
	// SetAcceleration / SetDefaultConfig; use an atomic to avoid a data race.
	acceleration atomic.Bool
	// logger, when set, receives every line insyra logs instead of the
	// standard log package; see SetLogger.
	logger atomic.Pointer[slog.Logger]
}

var Config *configStruct = &configStruct{}

type LogLevel int

const (
	// LogLevelDebug is the log level for debug messages.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is the log level for info messages.
	LogLevelInfo
	// LogLevelWarning is the log level for warning messages.
	LogLevelWarning
	// LogLevelFatal is the log level for fatal messages.
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

// GetUseColoredOutput reports whether output is coloured; see
// SetUseColoredOutput. It is true by default.
func (c *configStruct) GetUseColoredOutput() bool {
	return c.coloredOutput.Load()
}

// GetDoesUseColoredOutput reports whether output is coloured.
//
// Deprecated: use GetUseColoredOutput, which is the same method. Removed in 0.4.
func (c *configStruct) GetDoesUseColoredOutput() bool {
	return c.GetUseColoredOutput()
}

func (c *configStruct) SetDontPanic(dontPanic bool) {
	c.dontPanic.Store(dontPanic)
}

func (c *configStruct) GetDontPanicStatus() bool {
	return c.dontPanic.Load()
}

// SetDefaultErrHandlingFunc installs a hook that receives every warning and
// fatal message. The hook runs on one dedicated goroutine, in the order the
// messages were produced, behind a queue of 1,024 calls. A call that finds the
// queue full is not dropped: it is delivered on its own goroutine, so during a
// burst the hook can receive messages out of order. Pass nil to remove the
// hook.
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
// Dangerously_TurnOffThreadSafety turns off the per-instance locking of
// DataList and DataTable. DataList.AtomicDo, DataTable.AtomicDo and AtomicDoAll
// then run their callback without taking any lock, so every DataList and
// DataTable method, and every type that embeds one (isr's included), is no
// longer safe to use from more than one goroutine at a time. You can enjoy an
// extreme performance boost, but data consistency is NOT guaranteed.
// Config, the global error buffer and logging stay safe, because they do not
// use those locks.
//
// Change the switch only while no other goroutine is using a DataList or a
// DataTable. TurnOnThreadSafety turns the locking back on and leaves every
// other setting as it is.
//
// The underscore is deliberate. Go names do not use one, so the call looks
// wrong wherever it is written, and that is the warning. Naming linters report
// only this declaration, never the code that calls it.
func (c *configStruct) Dangerously_TurnOffThreadSafety() {
	c.threadSafe.Store(false)
	LogWarning("config", "Dangerously_TurnOffThreadSafety", "Thread safety is turned off. Data consistency is NOT guaranteed!\nIt may be a mistake. Remove `Dangerously_TurnOffThreadSafety()` in your code, or call `Config.TurnOnThreadSafety()`, to restore thread safety.")
}

// TurnOnThreadSafety turns the per-instance locking of DataList and DataTable
// back on after Dangerously_TurnOffThreadSafety, and changes no other setting.
// Thread safety is on by default, so a program that never turned it off never
// needs this. As with turning it off, call it only while no other goroutine is
// using a DataList or a DataTable. It logs at Info level when it turns locking
// back on, and does nothing when locking is already on.
func (c *configStruct) TurnOnThreadSafety() {
	if previous := c.threadSafe.Swap(true); !previous {
		LogInfo("config", "TurnOnThreadSafety", "thread safety turned back on")
	}
}

// SetLogger sends every line insyra logs to l instead of the standard log
// package. Each line becomes one record at its own level (Debug, Info, Warn,
// and Fatal as slog.LevelError+4), with the formatted message and, when the
// call names them, a "package" and a "func" attribute. The record's source is
// the function that logged. The message carries no "[insyra - …]" prefix and
// no colour codes.
//
// SetLogLevel still decides which lines insyra emits; the handler then keeps
// the ones its own level allows. Pass nil to go back to the standard log
// package, which is the default. A plain io.Writer works through
// slog.New(slog.NewTextHandler(w, nil)).
func (c *configStruct) SetLogger(l *slog.Logger) {
	c.logger.Store(l)
}

// GetLogger returns the logger set with SetLogger, or nil when insyra writes
// to the standard log package.
func (c *configStruct) GetLogger() *slog.Logger {
	return c.logger.Load()
}

// ======================== Configs ========================

// SetDefaultConfig resets every Config field to its default value: log level
// Info, coloured output on, dontPanic off, no error hook, no logger (the
// standard log package), thread safety on, acceleration on.
func SetDefaultConfig() {
	Config.logLevel.Store(int32(LogLevelInfo))
	Config.coloredOutput.Store(true)
	Config.dontPanic.Store(false)
	Config.defaultErrHandlingFunc.Store(nil)
	Config.logger.Store(nil)
	Config.threadSafe.Store(true)
	Config.acceleration.Store(true)
}
