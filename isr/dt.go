package isr

import (
	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
)

type DT struct {
	*insyra.DataTable
}

// Row is a type alias for map[any]any.
// It is used to represent a row in a DataTable.
// Every key in the map represents a column Index or Number Index.
type Row map[any]any

// From converts a DataList, DL, Row, map[string]any, or map[int]any to a DataTable.
func (dt DT) From(dl any) *DT {
	switch v := dl.(type) {
	case *insyra.DataList:
		dt.DataTable = insyra.NewDataTable(v)
	case *DL:
		dt.DataTable = insyra.NewDataTable(v.DataList)
	case Row:
		if isIntKey(v) || isStrKey(v) {
			strMap := make(map[string]any)
			for k, v := range v {
				strMap[conv.ToString(k)] = v
			}
			dt.DataTable = insyra.NewDataTable().AppendRowsByColIndex(strMap)
		} else if isNameKey(v) {
			strMap := make(map[string]any)
			for k, v := range v {
				strMap[k.(name).value] = v
			}
			dt.DataTable = insyra.NewDataTable().AppendRowsByColName(strMap)
		} else {
			insyra.LogFatal("DT{}.FromDL(): got unexpected type %T", dl)
		}
	case map[string]any:
		dt.DataTable = insyra.NewDataTable().AppendRowsByColIndex(v)
	case map[int]any:
		strV := make(map[string]any)
		for k, v := range v {
			strV[conv.ToString(k)] = v
		}
		dt.DataTable = insyra.NewDataTable().AppendRowsByColIndex(strV)
	default:
		insyra.LogFatal("DT{}.FromDL(): got unexpected type %T", dl)
	}
	return &dt
}

func (dt *DT) Col(col any) *DL {
	var dl DL
	switch v := col.(type) {
	case int:
		dl.DataList = dt.DataTable.GetColByNumber(v)
	case string:
		dl.DataList = dt.DataTable.GetCol(v)
	default:
		insyra.LogFatal("DT{}.Col(): got unexpected type %T", col)
	}
	return &dl
}

func (dt *DT) Row(row int) *DL {
	var dl DL
	dl.DataList = dt.DataTable.GetRow(row)
	return &dl
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

func isStrKey(m map[any]any) bool {
	for k := range m {
		_, ok := k.(string)
		if !ok {
			return false
		}
	}
	return true
}

func isIntKey(m map[any]any) bool {
	for k := range m {
		_, ok := k.(int)
		if !ok {
			return false
		}
	}
	return true
}

func isNameKey(m map[any]any) bool {
	for k := range m {
		_, ok := k.(name)
		if !ok {
			return false
		}
	}
	return true
}
