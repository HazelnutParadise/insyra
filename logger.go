package insyra

import (
	"fmt"
	"log"
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
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:1]) + msg[1:]
	var fullMsg = "<{[insyra - FATAL!]}> "
	if packageName != "" {
		fullMsg += packageName + "." + funcName + ": "
	}
	fullMsg += msg
	log.Printf(colorText("31", fullMsg), args...)
	panicIfConfigured(LogLevelFatal, packageName, funcName, formatted)
}

// LogError records a call that could not do what it was asked. This is the
// level every instance-level Err() uses. Like LogFatal it never ends the
// process; Config.SetPanicOnError(true) turns it into a recoverable panic.
func LogError(packageName, funcName, msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	pushError(LogLevelError, packageName, funcName, formatted)
	if Config.GetLogLevel() <= LogLevelError {
		if len(msg) == 0 || msg[len(msg)-1] != '\n' {
			msg += "\n"
		}
		msg = strings.ToUpper(msg[0:1]) + msg[1:]
		var fullMsg = "[insyra - Error] "
		if packageName != "" {
			fullMsg += packageName + "." + funcName + ": "
		}
		fullMsg += msg
		log.Printf(colorText("31", fullMsg), args...)
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
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:1]) + msg[1:]
	var fullMsg = "[insyra - Warning] "
	if packageName != "" {
		fullMsg += packageName + "." + funcName + ": "
	}
	fullMsg += msg
	log.Printf(fullMsg, args...)
}

func LogDebug(packageName, funcName, msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelDebug {
		return
	}
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:1]) + msg[1:]
	var fullMsg = "<insyra - Debug> "
	if packageName != "" {
		fullMsg += packageName + "." + funcName + ": "
	}
	fullMsg += msg
	log.Printf(fullMsg, args...)
}

func LogInfo(packageName, funcName, msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelInfo {
		return
	}
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:1]) + msg[1:]
	var fullMsg = "[insyra - Info] "
	if packageName != "" {
		fullMsg += packageName + "." + funcName + ": "
	}
	fullMsg += msg
	log.Printf(fullMsg, args...)
}
