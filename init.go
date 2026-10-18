// `insyra` main package provides unified interfaces and structures for data manipulation and analysis.
package insyra

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/internal/ccl"
)

func init() {
	SetDefaultConfig()
	initCCLFunctions()
	ccl.SetWarnHook(cclWarning)
}

// cclWarning gives CCL's warnings what LogWarning gives everyone else's: the
// error buffer, the error hook, the log level and the logger. The record's
// source is the code that called ccl.Warn.
func cclWarning(funcName, format string, args ...any) {
	pushError(LogLevelWarning, "ccl", funcName, fmt.Sprintf(format, args...))
	if Config.GetLogLevel() <= LogLevelWarning {
		emit(2, LogLevelWarning, "ccl", funcName, format, args...)
	}
}
