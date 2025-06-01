// config.go

package insyra

type configStruct struct {
	logLevel               LogLevel
	dontPanic              bool
	defaultErrHandlingFunc func(errType LogLevel, packageName string, funcName string, errMsg string)
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
	c.logLevel = level
}

func (c *configStruct) GetLogLevel() LogLevel {
	return LogLevel(c.logLevel)
}

func (c *configStruct) SetDontPanic(dontPanic bool) {
	c.dontPanic = dontPanic
}

func (c *configStruct) GetDontPanicStatus() bool {
	return c.dontPanic
}

func (c *configStruct) SetDefaultErrHandlingFunc(fn func(errType LogLevel, packageName string, funcName string, errMsg string)) {
	c.defaultErrHandlingFunc = fn
}

func (c *configStruct) GetDefaultErrHandlingFunc() func(errType LogLevel, packageName string, funcName string, errMsg string) {
	return c.defaultErrHandlingFunc
}

// ======================== Configs ========================

// DefaultConfig returns a Config with default values.
func SetDefaultConfig() {
	Config.logLevel = LogLevelInfo
	Config.dontPanic = false
}
