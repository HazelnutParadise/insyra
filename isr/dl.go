package isr

import "github.com/HazelnutParadise/insyra"

// Use `DL.From` to create a new DataList from a slice or multiple elements of any type.
var DL = dl{}

// DLs is a type alias for []*DL.
// It is used to represent a list of DataList.
type DLs = []insyra.IDataList

// DL is a type alias for *insyra.DataList.
type dl struct {
	*insyra.DataList
}

// Deprecated: Use UseDL instead.
// PtrDL converts a DataList or DL to a *DL.
// You should no longer use this function, use UseDL instead.
func PtrDL[T *insyra.DataList | dl](l T) *dl {
	return UseDL(l)
}

// From is equivalent to insyra.NewDataList(data...).
// nolint:govet
func (_ dl) From(data ...any) *dl {
	newDL := dl{}
	newDL.DataList = insyra.NewDataList(data)
	return &newDL
}

// At is equivalent to dl.Get(index).
func (l *dl) At(index int) any {
	return l.Get(index)
}

// Push is equivalent to dl.DataList.Append(data...).
func (l *dl) Push(data ...any) *dl {
	l.DataList.Append(data...)
	return l
}
