package isr

import "github.com/HazelnutParadise/insyra"

// isr is a block-syntax API: every method has to return something the next
// method in the chain can be called on. A failure in DT.From, Col, Row or Push
// therefore does not end the process; it records the error on the object it
// hands back, and the caller checks Err() (or PopErr()) at the end of the
// chain:
//
//	t := isr.DT.From(isr.CSV{FilePath: path}).Push(row)
//	if err := t.PopErr(); err != nil { ... }
//
// failDT and failDL are the two helpers that do this. They guarantee the
// wrapper holds a usable underlying value before recording the error, so the
// caller cannot dereference a nil.

func failDT(t *dt, funcName, msg string, args ...any) *dt {
	if t.DataTable == nil {
		t.DataTable = insyra.NewDataTable()
	}
	t.SetErr("isr", funcName, msg, args...)
	return t
}

func failDL(l *dl, funcName, msg string, args ...any) *dl {
	if l.DataList == nil {
		l.DataList = insyra.NewDataList()
	}
	l.SetErr("isr", funcName, msg, args...)
	return l
}
