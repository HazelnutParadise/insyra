package insyra

import (
	"fmt"
	"log"
	"strings"
)

func LogFatal(packageName, funcName, msg string, args ...any) {
	pushError(LogLevelFatal, packageName, funcName, fmt.Sprintf(msg, args...))
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:1]) + msg[1:]
	var fullMsg = "<{[insyra - FATAL!]}> "
	if packageName != "" {
		fullMsg += packageName + "." + funcName + ": "
	}
	fullMsg += msg
	if Config.dontPanic {
		log.Printf(fullMsg, args...)
		return
	}
	log.Fatalf(colorText("31", fullMsg), args...)
}

func LogWarning(packageName, funcName, msg string, args ...any) {
	pushError(LogLevelWarning, packageName, funcName, fmt.Sprintf(msg, args...))
	if Config.GetLogLevel() > LogLevelWarning {
		return
	}
	if msg[len(msg)-1] != '\n' {
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
	if msg[len(msg)-1] != '\n' {
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
	if msg[len(msg)-1] != '\n' {
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
