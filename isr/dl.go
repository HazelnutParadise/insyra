package isr

import "github.com/HazelnutParadise/insyra"

type DL struct {
	*insyra.DataList
}

func (dl DL) From(data ...any) *DL {
	dl.DataList = insyra.NewDataList(data)
	return &dl
}

func (dl *DL) At(index int) any {
	return dl.Get(index)
}
