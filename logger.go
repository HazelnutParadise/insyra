package insyra

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// LogFatal records a failure the library cannot work around.
//
// It does NOT end the process: a library must never terminate its caller. The
// message goes to the global diagnostic buffer and to the log, and the call
// returns so the caller can handle the situation. Set
// Config.SetPanicOnError(true) to make it panic instead (a panic, never
// os.Exit, so it can be recovered).
func LogFatal(packageName, funcName, msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	pushError(LogLevelFatal, packageName, funcName, formatted)
	emit(1, LogLevelFatal, packageName, funcName, msg, args...)
	panicIfConfigured(LogLevelFatal, packageName, funcName, formatted)
}

// LogError records a call that could not do what it was asked. This is the
// level every instance-level Err() uses. Like LogFatal it never ends the
// process; Config.SetPanicOnError(true) turns it into a recoverable panic.
func LogError(packageName, funcName, msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	pushError(LogLevelError, packageName, funcName, formatted)
	if Config.GetLogLevel() <= LogLevelError {
		emit(1, LogLevelError, packageName, funcName, msg, args...)
	}
	panicIfConfigured(LogLevelError, packageName, funcName, formatted)
}

// panicIfConfigured turns a recorded error into a panic when the caller opted
// in with Config.SetPanicOnError(true). The panic value is an *ErrorInfo,
// which implements error, so a recovering caller can type-assert it.
func panicIfConfigured(level LogLevel, packageName, funcName, message string) {
	if !Config.GetPanicOnError() {
		return
	}
	panic(&ErrorInfo{
		Level:       level,
		PackageName: packageName,
		FuncName:    funcName,
		Message:     message,
		Timestamp:   time.Now(),
	})
}

func LogWarning(packageName, funcName, msg string, args ...any) {
	pushError(LogLevelWarning, packageName, funcName, fmt.Sprintf(msg, args...))
	if Config.GetLogLevel() > LogLevelWarning {
		return
	}
	emit(1, LogLevelWarning, packageName, funcName, msg, args...)
}

func LogDebug(packageName, funcName, msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelDebug {
		return
	}
	emit(1, LogLevelDebug, packageName, funcName, msg, args...)
}

func LogInfo(packageName, funcName, msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelInfo {
		return
	}
	emit(1, LogLevelInfo, packageName, funcName, msg, args...)
}

// emit writes one line that has already passed the log level. With a logger
// set (Config.SetLogger) it becomes a slog record; otherwise it goes to the
// standard log package in insyra's text format. skip is the number of frames
// between emit's caller and the function the record's source should name:
// 1 when a Log function calls emit, so the source is whoever called it.
func emit(skip int, level LogLevel, packageName, funcName, msg string, args ...any) {
	if l := Config.GetLogger(); l != nil {
		emitRecord(l, skip+1, level, packageName, funcName, fmt.Sprintf(msg, args...))
		return
	}
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:1]) + msg[1:]
	fullMsg := textPrefix(level)
	if packageName != "" {
		fullMsg += packageName + "." + funcName + ": "
	}
	fullMsg += msg
	if level >= LogLevelError {
		// The reset goes before the newline, so the next line starts plain.
		fullMsg = colorText(log.Writer(), "31", strings.TrimSuffix(fullMsg, "\n")) + "\n"
	}
	log.Printf(fullMsg, args...)
}

func textPrefix(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "<insyra - Debug> "
	case LogLevelInfo:
		return "[insyra - Info] "
	case LogLevelWarning:
		return "[insyra - Warning] "
	case LogLevelError:
		return "[insyra - Error] "
	default:
		return "<{[insyra - FATAL!]}> "
	}
}

// slogLevelFatal is the slog level a Fatal line is recorded at: above Error,
// since slog has no level of its own for it.
const slogLevelFatal = slog.LevelError + 4

func slogLevel(level LogLevel) slog.Level {
	switch level {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarning:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slogLevelFatal
	}
}

// emitRecord builds the record by hand rather than calling l.Log, so its
// source is the function that logged instead of this file (the pattern the
// log/slog documentation gives for wrapping its output methods).
func emitRecord(l *slog.Logger, skip int, level LogLevel, packageName, funcName, message string) {
	ctx := context.Background()
	sl := slogLevel(level)
	if !l.Enabled(ctx, sl) {
		return
	}
	var pcs [1]uintptr
	// Skip runtime.Callers, emitRecord, and then skip frames above it.
	runtime.Callers(2+skip, pcs[:])
	record := slog.NewRecord(time.Now(), sl, strings.TrimSuffix(message, "\n"), pcs[0])
	if packageName != "" {
		record.AddAttrs(slog.String("package", packageName))
	}
	if funcName != "" {
		record.AddAttrs(slog.String("func", funcName))
	}
	_ = l.Handler().Handle(ctx, record)
}
