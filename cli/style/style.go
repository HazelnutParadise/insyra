// Package style formats the error and warning lines the insyra command prints.
package style

import "github.com/HazelnutParadise/insyra/internal/dsl/style"

// ErrorText returns message as an error line, "error: <message>", in red unless
// insyra's colored output is off.
func ErrorText(message string) string { return style.ErrorText(message) }

// WarningText returns message as a warning line, "warn: <message>", in yellow
// unless insyra's colored output is off.
func WarningText(message string) string { return style.WarningText(message) }
