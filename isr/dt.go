package isr

import (
	"github.com/HazelnutParadise/insyra"
)

type DT struct {
	*insyra.DataTable
}

func (dt DT) FromDL(dl insyra.IDataList) *DT {
	switch v := dl.(type) {
	case *insyra.DataList:
		dt.DataTable = insyra.NewDataTable(v)
	case *DL:
		dt.DataTable = insyra.NewDataTable(v.DataList)
	default:
		insyra.LogFatal("DT{}.FromDL(): got unexpected type %T", dl)
	}
	return &dt
}

func (dt *DT) Col(col any) *insyra.DataList {
	var dl *insyra.DataList
	switch v := col.(type) {
	case int:
		dl = dt.DataTable.GetColByNumber(v)
	case string:
		dl = dt.DataTable.GetCol(v)
	default:
		insyra.LogFatal("DT{}.Col(): got unexpected type %T", col)
	}
	return dl
}

// func (dt *DT) Iloc(indices ...interface{}) *DT {
// 	switch len(indices) {
// 	case 1:
// 		switch v := indices[0].(type) {
// 		case int:
// 			DT{}. = dt.GetColByNumber(v)

// 			dl.DataList = dl.DataList.SelectRows(v.Start, v.End) // 切片選取
// 		}
// 	default:
// 		dl.DataList = dl.DataList.SelectRows(indices...) // 多行選取
// 	}
// 	return dl
// }
