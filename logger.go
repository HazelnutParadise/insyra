package insyra

import "log"

func LogFatal(msg string, args ...any) {
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Fatalf("<{[insyra - FATAL!]}> "+msg, args...)
}

func LogWarning(msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelWarning {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Printf("[insyra - Warning] "+msg, args...)
}

func LogDebug(msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelDebug {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Printf("<insyra - Debug> "+msg, args...)
}

func LogInfo(msg string, args ...any) {
	if Config.GetLogLevel() > LogLevelInfo {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Printf("[insyra - Info] "+msg, args...)
}
