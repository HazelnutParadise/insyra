package insyra

import "log"

func LogWarning(msg string, args ...interface{}) {
	if Config.GetLogLevel() > LogLevelWarning {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Printf("[insyra - Warning] "+msg, args...)
}

func LogDebug(msg string, args ...interface{}) {
	if Config.GetLogLevel() > LogLevelDebug {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Printf("<insyra - Debug> "+msg, args...)
}

func LogInfo(msg string, args ...interface{}) {
	if Config.GetLogLevel() > LogLevelInfo {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	log.Printf("[insyra - Info] "+msg, args...)
}
