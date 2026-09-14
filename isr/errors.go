package isr

import "github.com/HazelnutParadise/insyra"

// A failure in DT.From, Col, Row or Push does not end the program. It logs a
// warning, and the call returns what it returned on v0.3.2 with
// Config.SetDontPanic(true), where LogFatal only logged and the code went on:
// a source that cannot be read leaves a nil DataTable, an unsupported selector
// a nil DataList, and a row or column that cannot be added is skipped. Both
// configurations return that value.
//
// recordDT records the error on the table as well, but only when there is a
// table to record it on, so nothing that was nil becomes non-nil.
func recordDT(t *dt, funcName, msg string, args ...any) {
	if t.DataTable == nil {
		insyra.LogWarning("isr", funcName, msg, args...)
		return
	}
	t.SetErr("isr", funcName, msg, args...)
}
