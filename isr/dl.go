package isr

import "github.com/HazelnutParadise/insyra"

// DLs is a type alias for []*DL.
// It is used to represent a list of DataList.
type DLs = []insyra.IDataList

// DL is a type alias for *insyra.DataList.
type DL struct {
	*insyra.DataList
}

// PtrDL converts a DataList or DL to a *DL.
func PtrDL[T *insyra.DataList | DL](dl T) *DL {
	switch concrete := any(dl).(type) {
	case *insyra.DataList:
		return &DL{concrete}
	case DL:
		return &concrete
	default:
		insyra.LogFatal("isr.PtrDL(): got unexpected type %T", dl)
		return nil
	}
}

// From is equivalent to insyra.NewDataList(data...).
func (dl DL) From(data ...any) *DL {
	dl.DataList = insyra.NewDataList(data)
	return &dl
}

// At is equivalent to dl.Get(index).
func (dl *DL) At(index int) any {
	return dl.Get(index)
}
