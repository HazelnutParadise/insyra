package isr

import "github.com/HazelnutParadise/insyra"

// UseDL converts a DataList or DL to a *DL.
// It is the alias of PtrDL.
// Recommended to use this function instead of PtrDL.
func UseDL[T *insyra.DataList | dl](l T) *dl {
	switch concrete := any(l).(type) {
	case *insyra.DataList:
		return &dl{concrete}
	case dl:
		return &concrete
	default:
		insyra.LogFatal("isr", "PtrDL", "got unexpected type %T", l)
		return nil
	}
}

// UseDT converts a DataTable or DT to a *DT.
// It is the alias of PtrDT.
// Recommended to use this function instead of PtrDT.
func UseDT[T *insyra.DataTable | dt](t T) *dt {
	switch concrete := any(t).(type) {
	case *insyra.DataTable:
		return &dt{concrete}
	case dt:
		return &concrete
	default:
		insyra.LogFatal("isr", "PtrDT", "got unexpected type %T", t)
		return nil
	}
}
