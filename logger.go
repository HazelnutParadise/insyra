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
	msg = strings.ToUpper(msg[0:0]) + msg[1:]
	if Config.dontPanic {
		log.Printf("<{[insyra - FATAL!]}> "+packageName+"."+funcName+": "+msg, args...)
		return
	}
	log.Fatalf("<{[insyra - FATAL!]}> "+msg, args...)
}

func LogWarning(packageName, funcName, msg string, args ...any) {
	pushError(LogLevelWarning, packageName, funcName, fmt.Sprintf(msg, args...))
	if Config.GetLogLevel() > LogLevelWarning {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:0]) + msg[1:]
	log.Printf("[insyra - Warning] "+packageName+"."+funcName+": "+msg, args...)
}

func LogDebug(packageName, funcName, msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelDebug {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:0]) + msg[1:]
	log.Printf("<insyra - Debug> "+packageName+"."+funcName+": "+msg, args...)
}

func LogInfo(packageName, funcName, msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelInfo {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	msg = strings.ToUpper(msg[0:0]) + msg[1:]
	log.Printf("[insyra - Info] "+packageName+"."+funcName+": "+msg, args...)
}
