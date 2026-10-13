package ccl

import (
	"log"
	"sync/atomic"
)

// WarnFunc receives a warning about something CCL worked around.
type WarnFunc func(funcName, format string, args ...any)

var warnHook atomic.Pointer[WarnFunc]

// SetWarnHook routes CCL's warnings to fn. The root package sets it at init,
// so the warning reaches insyra's logger and log level; nil restores the
// standard log package, which is what a program using CCL without the root
// package gets.
func SetWarnHook(fn WarnFunc) {
	if fn == nil {
		warnHook.Store(nil)
		return
	}
	warnHook.Store(&fn)
}

// Warn reports a warning through the hook set with SetWarnHook.
func Warn(funcName, format string, args ...any) {
	if fn := warnHook.Load(); fn != nil {
		(*fn)(funcName, format, args...)
		return
	}
	log.Printf("ccl: "+format, args...)
}
