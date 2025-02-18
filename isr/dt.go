package isr

import (
	"fmt"

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

// From converts a DataList, DL, Row, Col, []Row, []Col, CSV, map[string]any, or map[int]any to a DataTable.
func (dt DT) From(item any) *DT {
	switch val := item.(type) {
	case *insyra.DataList:
		dt.DataTable = insyra.NewDataTable(val)
	case *DL:
		dt.DataTable = insyra.NewDataTable(val.DataList)
	case []*insyra.DataList:
		dt.DataTable = insyra.NewDataTable()
		for _, dl := range val {
			dt.DataTable.AppendCols(dl)
		}
	case []DL:
		dt.DataTable = insyra.NewDataTable()
		for _, dl := range val {
			dt.DataTable.AppendCols(dl.DataList)
		}
	case DLs:
		dt.DataTable = insyra.NewDataTable()
		for _, dl := range val {
			newdl := insyra.NewDataList(dl.Data()...)
			dt.DataTable.AppendCols(newdl)
		}
	case Row:
		dt.DataTable = insyra.NewDataTable()
		err := fromRowToDT(&dt, val)
		if err != nil {
			insyra.LogFatal("DT{}.From(): %v", err)
		}
	case []Row:
		dt.DataTable = insyra.NewDataTable()
		for _, r := range val {
			err := fromRowToDT(&dt, r)
			if err != nil {
				insyra.LogFatal("DT{}.From(): %v", err)
			}
		}
	case Col:
		dt.DataTable = insyra.NewDataTable()
		err := fromRowToDT(&dt, val)
		if err != nil {
			insyra.LogFatal("DT{}.From(): %v", err)
		}
		dt.Transpose()
	case []Col:
		dt.DataTable = insyra.NewDataTable()
		for _, r := range val {
			err := fromRowToDT(&dt, r)
			if err != nil {
				insyra.LogFatal("DT{}.From(): %v", err)
			}
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
		insyra.LogFatal("DT{}.FromDL(): got unexpected type %T", item)
	}
	return &dt
}

// Col returns a DL that contains the column at the specified index.
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

// Row returns a DL that contains the row at the specified index.
func (dt *DT) Row(row int) *DL {
	var dl DL
	dl.DataList = dt.DataTable.GetRow(row)
	return &dl
}

// At returns the element at the specified row and column.
func (dt *DT) At(row int, col any) any {
	switch v := col.(type) {
	case int:
		return dt.DataTable.GetElementByNumberIndex(row, v)
	case string:
		return dt.DataTable.GetElement(row, v)
	default:
		insyra.LogWarning("DT{}.At(): got unexpected type %T. Returning nil.", col)
	}
	return nil
}

func (dt *DT) Push(data any) *DT {
	switch val := data.(type) {
	case *insyra.DataList:
		dt.DataTable.AppendCols(val)
	case *DL:
		dt.DataTable.AppendCols(val.DataList)
	case []*insyra.DataList:
		for _, dl := range val {
			dt.DataTable.AppendCols(dl)
		}
	case []DL:
		for _, dl := range val {
			dt.DataTable.AppendCols(dl.DataList)
		}
	case DLs:
		for _, dl := range val {
			newdl := insyra.NewDataList(dl.Data()...)
			dt.DataTable.AppendCols(newdl)
		}
	case Row:
		// TODO: insyra包處理數字索引的問題
		err := fromRowToDT(dt, val)
		if err != nil {
			insyra.LogFatal("DT{}.Push(): %v", err)
		}
	case []Row:
		for _, r := range val {
			// TODO
			err := fromRowToDT(dt, r)
			if err != nil {
				insyra.LogFatal("DT{}.Push(): %v", err)
			}
		}
	case Col:
		// TODO
	case []Col:
		// for _, r := range val {
		// 	// TODO
		// }
	default:
		insyra.LogFatal("DT{}.Push(): got unexpected type %T", data)
	}
	return dt
}

// func (dt *DT) Iloc(indices ...any) *DT {
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

func fromRowToDT(dt *DT, val map[any]any) error {
	strMap := make(map[string]any)
	if isIntKey(val) || isStrKey(val) {
		for k, v := range val {
			if isInt(k) || isStr(k) {
				strMap[conv.ToString(k)] = v
			}
		}
		dt.AppendRowsByColIndex(strMap)
	} else if isNameKey(val) {
		for k, v := range val {
			strMap[k.(name).value] = v
		}
		dt.AppendRowsByColName(strMap)
	} else {
		return fmt.Errorf("got unexpected type %T", val)
	}
	return nil
}

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

// func isName(m any) bool {
// 	_, ok := m.(name)
// 	return ok
// }
