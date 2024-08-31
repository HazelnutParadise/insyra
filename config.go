package insyra

type configStruct struct {
	logLevel LogLevel
}

var Config *configStruct

type LogLevel int

const (
	// LogLevelDebug is the log level for debug messages.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is the log level for info messages.
	LogLevelInfo
	// LogLevelWarning is the log level for warning messages.
	LogLevelWarning
)

func (c *configStruct) SetLogLevel(level LogLevel) {
	c.logLevel = level
}

func (c *configStruct) GetLogLevel() LogLevel {
	return LogLevel(c.logLevel)
}

// ======================== Configs ========================
// DefaultConfig returns a Config with default values.
func SetDefaultConfig() {
	Config = &configStruct{
		logLevel: LogLevelInfo,
	}
}
