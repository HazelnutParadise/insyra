package isr

import "github.com/HazelnutParadise/insyra"

// UseDL converts a DataList or DL to a *DL.
// It is the alias of PtrDL.
// Recommended to use this function instead of PtrDL.
func UseDL[T *insyra.DataList | dl](l T) *dl {
	switch concrete := any(l).(type) {
	case *insyra.DataList:
		if concrete == nil {
			return &dl{insyra.NewDataList().SetErr("isr", "UseDL", "got a nil *insyra.DataList")}
		}
		return &dl{concrete}
	case dl:
		if concrete.DataList == nil {
			concrete.DataList = insyra.NewDataList().SetErr("isr", "UseDL", "got a DL with no underlying DataList")
		}
		return &concrete
	default:
		return &dl{insyra.NewDataList().SetErr("isr", "UseDL", "got unexpected type %T", l)}
	}
}

// UseDT converts a DataTable or DT to a *DT.
// It is the alias of PtrDT.
// Recommended to use this function instead of PtrDT.
func UseDT[T *insyra.DataTable | dt](t T) *dt {
	switch concrete := any(t).(type) {
	case *insyra.DataTable:
		if concrete == nil {
			return &dt{insyra.NewDataTable().SetErr("isr", "UseDT", "got a nil *insyra.DataTable")}
		}
		return &dt{concrete}
	case dt:
		if concrete.DataTable == nil {
			concrete.DataTable = insyra.NewDataTable().SetErr("isr", "UseDT", "got a DT with no underlying DataTable")
		}
		return &concrete
	default:
		return &dt{insyra.NewDataTable().SetErr("isr", "UseDT", "got unexpected type %T", t)}
	}
}
