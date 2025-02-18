package isr

import "github.com/HazelnutParadise/insyra"

// DLs is a type alias for []*DL.
// It is used to represent a list of DataList.
type DLs = []insyra.IDataList

// DL is a type alias for *insyra.DataList.
type DL struct {
	*insyra.DataList
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

// Push is equivalent to dl.DataList.Append(data...).
func (dl *DL) Push(data ...any) *DL {
	dl.DataList.Append(data...)
	return dl
}
