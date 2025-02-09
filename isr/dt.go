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

// DL is a type alias for map[any]any.
// It is used to represent a column in a DataTable.
// Every key in the map represents a row Index.
type Col map[any]any

// From converts a DataList, DL, Row, CSV, map[string]any, or map[int]any to a DataTable.
func (dt DT) From(dl any) *DT {
	switch val := dl.(type) {
	case *insyra.DataList:
		dt.DataTable = insyra.NewDataTable(val)
	case *DL:
		dt.DataTable = insyra.NewDataTable(val.DataList)
	case Row:
		strMap := make(map[string]any)
		if isIntKey(val) || isStrKey(val) {
			for k, v := range val {
				if isInt(k) || isStr(k) {
					strMap[conv.ToString(k)] = v
				}
			}
			dt.DataTable = insyra.NewDataTable().AppendRowsByColIndex(strMap)
		} else if isNameKey(val) {
			for k, v := range val {
				strMap[k.(name).value] = v
			}
			dt.DataTable = insyra.NewDataTable().AppendRowsByColName(strMap)
		} else {
			insyra.LogFatal("DT{}.FromDL(): got unexpected type %T", dl)
		}
	case Col:
		dt.DataTable = insyra.NewDataTable()
		if isIntKey(val) {
			strMap := make(map[string]any)
			for k, v := range val {
				strMap[conv.ToString(k)] = v
			}
			dt.DataTable.AppendRowsByColIndex(strMap)
		} else if isNameKey(val) {
			strMap := make(map[string]any)
			for k, v := range val {
				strMap[k.(name).value] = v
			}
			dt.DataTable.AppendRowsByColName(strMap)
		}
		dt.Transpose()
	case CSV:
		dt.DataTable = insyra.NewDataTable()
		err := dt.LoadFromCSV(val.FilePath, val.LoadOpts.FirstCol2RowNames, val.LoadOpts.FirstRow2ColNames)
		if err != nil {
			insyra.LogFatal("DT{}.From(): %v", err)
		}
	case map[string]any:
		dt.DataTable = insyra.NewDataTable().AppendRowsByColIndex(val)
	case map[int]any:
		strV := make(map[string]any)
		for k, v := range val {
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

func isStr(m any) bool {
	_, ok := m.(string)
	return ok
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

func isInt(m any) bool {
	_, ok := m.(int)
	return ok
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

func isName(m any) bool {
	_, ok := m.(name)
	return ok
}
