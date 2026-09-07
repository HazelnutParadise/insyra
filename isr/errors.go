package isr

import "github.com/HazelnutParadise/insyra"

// isr is a block-syntax API: every method has to return something the next
// method in the chain can be called on. A failure therefore never ends the
// process and never returns nil — it records the error on the object it hands
// back, and the caller checks Err() (or PopErr()) once at the end of the
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

// The embedded *insyra.DataTable / *insyra.DataList already provide Err() and
// PopErr(), which return *insyra.ErrorInfo and so work unchanged. ClearErr and
// SetErr, however, return the *embedded* type, which would end the isr chain:
//
//	isr.DT.From(x).ClearErr().Push(row)   // *insyra.DataTable has no Push
//
// These overrides keep the block syntax intact.

// ClearErr clears the recorded error and returns the DT for further chaining.
func (t *dt) ClearErr() *dt {
	t.DataTable.ClearErr()
	return t
}

// SetErr records an error on the DT and returns it for further chaining.
func (t *dt) SetErr(packageName, funcName, msg string, args ...any) *dt {
	t.DataTable.SetErr(packageName, funcName, msg, args...)
	return t
}

// ClearErr clears the recorded error and returns the DL for further chaining.
func (l *dl) ClearErr() *dl {
	l.DataList.ClearErr()
	return l
}

// SetErr records an error on the DL and returns it for further chaining.
func (l *dl) SetErr(packageName, funcName, msg string, args ...any) *dl {
	l.DataList.SetErr(packageName, funcName, msg, args...)
	return l
}
