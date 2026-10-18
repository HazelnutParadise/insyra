package isr

import "github.com/HazelnutParadise/insyra"

// UseDL converts a DataList or DL to a *DL.
// It is the alias of PtrDL.
// Recommended to use this function instead of PtrDL.
func UseDL[T *insyra.DataList | DataList](l T) *DataList {
	switch concrete := any(l).(type) {
	case *insyra.DataList:
		return &DataList{concrete}
	case DataList:
		return &concrete
	default:
		insyra.LogWarning("isr", "UseDL", "got unexpected type %T", l)
		return nil
	}
}

// UseDT converts a DataTable or DT to a *DT.
// It is the alias of PtrDT.
// Recommended to use this function instead of PtrDT.
func UseDT[T *insyra.DataTable | DataTable](t T) *DataTable {
	switch concrete := any(t).(type) {
	case *insyra.DataTable:
		return &DataTable{concrete}
	case DataTable:
		return &concrete
	default:
		insyra.LogWarning("isr", "UseDT", "got unexpected type %T", t)
		return nil
	}
}
