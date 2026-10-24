package isr

import "github.com/HazelnutParadise/insyra"

// Use `DL.From` to create a new DataList from a slice or multiple elements of any type.
var DL = DataList{}

// DLs is a type alias for []insyra.IDataList.
// It is used to represent a list of DataList.
type DLs = []insyra.IDataList

// DataList is isr's list: an *insyra.DataList whose methods return the isr
// type, so short-syntax chains keep going. Name it in your own signatures as
// *isr.DataList; build one with DL.From, DL.Of or UseDL.
type DataList struct {
	*insyra.DataList
}

// Deprecated: Use UseDL instead.
// PtrDL converts a DataList or DL to a *DL.
// You should no longer use this function, use UseDL instead.
func PtrDL[T *insyra.DataList | DataList](l T) *DataList {
	return UseDL(l)
}

// From is equivalent to insyra.NewDataList(data...).
// nolint:govet
func (d DataList) From(data ...any) *DataList {
	newDL := DataList{}
	newDL.DataList = insyra.NewDataList(data)
	return &newDL
}

// Of is an alias for From.
// It creates a new DataList from a slice or multiple elements of any type.
func (d DataList) Of(data ...any) *DataList {
	return d.From(data...)
}

// At is equivalent to dl.Get(index).
func (l *DataList) At(index int) any {
	return l.Get(index)
}

// Push is equivalent to dl.DataList.Append(data...).
func (l *DataList) Push(data ...any) *DataList {
	l.Append(data...)
	return l
}
