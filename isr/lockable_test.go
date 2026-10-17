package isr

import "github.com/HazelnutParadise/insyra"

// The isr wrappers embed *insyra.DataList and *insyra.DataTable, so they carry
// the unexported lock method and can be passed to insyra.AtomicDoAll directly.
// This fails to compile if that ever stops being true.
var (
	_ insyra.Lockable = (*DataList)(nil)
	_ insyra.Lockable = (*DataTable)(nil)
)
