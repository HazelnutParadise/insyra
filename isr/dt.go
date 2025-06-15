package isr

import (
	"fmt"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
)

// Use `DT.From` to create a new DataTable from a DataList, DL, Row, Col, []Row, []Col, CSV, JSON, map[string]any, or map[int]any.
var DT = dt{}

type dt struct {
	*insyra.DataTable
}

// Row is a type alias for map[any]any.
// It is used to represent a row in a DataTable.
// Every key in the map represents a column Index or Number Index.
type Row map[any]any

// Rows is a type alias for []Row.
type Rows = []Row

// DL is a type alias for map[any]any.
// It is used to represent a column in a DataTable.
// Every key in the map represents a row Index.
type Col map[any]any

// Cols is a type alias for []Col.
type Cols = []Col

// PtrDT converts a DataTable or DT to a *DT.
func PtrDT[T *insyra.DataTable | dt](t T) *dt {
	switch concrete := any(t).(type) {
	case *insyra.DataTable:
		return &dt{concrete}
	case dt:
		return &concrete
	default:
		insyra.LogFatal("isr", "PtrDT", "got unexpected type %T", t)
		return nil
	}
}

// From converts a DataList, DL, Row, Col, []Row, []Col, CSV, map[string]any, or map[int]any to a DataTable.
// nolint:govet
func (_ dt) From(item any) *dt {
	t := dt{}
	switch val := item.(type) {
	case *insyra.DataList:
		t.DataTable = insyra.NewDataTable(val)
	case *dl:
		t.DataTable = insyra.NewDataTable(val.DataList)
	case []*insyra.DataList:
		t.DataTable = insyra.NewDataTable()
		for _, l := range val {
			t.DataTable.AppendCols(l)
		}
	case []dl:
		t.DataTable = insyra.NewDataTable()
		for _, l := range val {
			t.DataTable.AppendCols(l.DataList)
		}
	case DLs:
		t.DataTable = insyra.NewDataTable()
		for _, l := range val {
			newdl := insyra.NewDataList(l.Data()...)
			if l.GetName() != "" {
				newdl.SetName(l.GetName())
			}
			t.DataTable.AppendCols(newdl)
		}
	case Row:
		t.DataTable = insyra.NewDataTable()
		err := fromRowToDT(&t, val)
		if err != nil {
			insyra.LogFatal("DT", "From", "%v", err)
		}
	case Rows:
		t.DataTable = insyra.NewDataTable()
		for _, r := range val {
			err := fromRowToDT(&t, r)
			if err != nil {
				insyra.LogFatal("DT", "From", "%v", err)
			}
		}
	case Col:
		t.DataTable = insyra.NewDataTable()
		err := fromRowToDT(&t, val)
		if err != nil {
			insyra.LogFatal("DT", "From", "%v", err)
		}
		t.Transpose()
	case Cols:
		t.DataTable = insyra.NewDataTable()
		for _, r := range val {
			err := fromRowToDT(&t, r)
			if err != nil {
				insyra.LogFatal("DT", "From", "%v", err)
			}
		}
		t.Transpose()
	case CSV:
		t.DataTable = insyra.NewDataTable()
		var err error
		t.DataTable, err = insyra.ReadCSV(val.FilePath, val.InputOpts.FirstCol2RowNames, val.InputOpts.FirstRow2ColNames)
		if err != nil {
			insyra.LogFatal("DT", "From", "%v", err)
		}
	case JSON:
		t.DataTable = insyra.NewDataTable()
		var err error
		if val.FilePath != "" {
			t.DataTable, err = insyra.ReadJSON(val.FilePath)
			if err != nil {
				insyra.LogFatal("DT", "From", "%v", err)
			}
		}

		t.DataTable, err = insyra.ReadJSON_Bytes(val.Bytes)
		if err != nil {
			insyra.LogFatal("DT", "From", "%v", err)
		}
	case map[string]any:
		t.DataTable = insyra.NewDataTable().AppendRowsByColIndex(val)
	case map[int]any:
		strV := make(map[string]any)
		for k, v := range val {
			strV[conv.ToString(k)] = v
		}
		t.DataTable = insyra.NewDataTable().AppendRowsByColIndex(strV)
	case nil:
		// do nothing, return an empty DataTable
	default:
		insyra.LogFatal("DT", "From", "got unexpected type %T", item)
	}
	return &t
}

// Col returns a DL that contains the column at the specified index.
func (t *dt) Col(col any) *dl {
	var l dl
	switch v := col.(type) {
	case int:
		l.DataList = t.DataTable.GetColByNumber(v)
	case string:
		l.DataList = t.DataTable.GetCol(v)
	case name:
		colDt := t.DataTable.FilterByColNameEqualTo(v.value)
		l.DataList = colDt.GetColByNumber(0)
	default:
		insyra.LogFatal("DT", "Col", "got unexpected type %T", col)
	}
	return &l
}

// Row returns a DL that contains the row at the specified index.
func (t *dt) Row(row any) *dl {
	var l dl
	switch v := row.(type) {
	case int:
		l.DataList = t.DataTable.GetRow(v)
	case name:
		rowDt := t.DataTable.FilterByRowNameEqualTo(v.value)
		l.DataList = rowDt.GetRow(0)
	default:
		insyra.LogFatal("DT", "Row", "got unexpected type %T", row)
	}
	return &l
}

// At returns the element at the specified row and column.
func (t *dt) At(row any, col any) any {
	switch v := col.(type) {
	case int:
		switch r := row.(type) {
		case int:
			return t.DataTable.GetElementByNumberIndex(r, v)
		case name:
			rowDt := t.DataTable.FilterByRowNameEqualTo(r.value)
			return rowDt.GetElementByNumberIndex(0, v)
		default:
			insyra.LogWarning("DT", "At", "got unexpected type %T. Returning nil.", row)
		}
	case string:
		switch r := row.(type) {
		case int:
			return t.DataTable.GetElement(r, v)
		case name:
			rowDt := t.DataTable.FilterByRowNameEqualTo(r.value)
			return rowDt.GetElement(0, v)
		default:
			insyra.LogWarning("DT", "At", "got unexpected type %T. Returning nil.", row)
		}
	case name:
		switch r := row.(type) {
		case int:
			colDt := t.DataTable.FilterByColNameEqualTo(v.value)
			return colDt.GetElementByNumberIndex(r, 0)
		case name:
			rowDt := t.DataTable.FilterByRowNameEqualTo(r.value)
			colDt := rowDt.FilterByColNameEqualTo(v.value)
			return colDt.GetElementByNumberIndex(0, 0)
		default:
			insyra.LogWarning("DT", "At", "got unexpected type %T. Returning nil.", row)
		}
	default:
		insyra.LogWarning("DT", "At", "got unexpected type %T. Returning nil.", col)
	}
	return nil
}

func (t *dt) Push(data any) *dt {
	switch val := data.(type) {
	case *insyra.DataList:
		t.DataTable.AppendCols(val)
	case *dl:
		t.DataTable.AppendCols(val.DataList)
	case []*insyra.DataList:
		for _, l := range val {
			t.DataTable.AppendCols(l)
		}
	case []dl:
		for _, l := range val {
			t.DataTable.AppendCols(l.DataList)
		}
	case DLs:
		for _, l := range val {
			newdl := insyra.NewDataList(l.Data()...)
			if l.GetName() != "" {
				newdl.SetName(l.GetName())
			}
			t.DataTable.AppendCols(newdl)
		}
	case Row:
		err := fromRowToDT(t, val)
		if err != nil {
			insyra.LogFatal("DT", "Push", "%v", err)
		}
	case []Row:
		for _, r := range val {
			err := fromRowToDT(t, r)
			if err != nil {
				insyra.LogFatal("DT", "Push", "%v", err)
			}
		}
	case Col:
		// 先創建新dt 當成row插入 再轉置
		// 轉置後抽出為dl 再插入
		temDT := PtrDT(insyra.NewDataTable())
		err := fromRowToDT(temDT, val)
		if err != nil {
			insyra.LogFatal("DT", "Push", "%v", err)
		}
		numRow, _ := temDT.Size()
		for i := range numRow {
			l := temDT.GetRow(i)
			t.DataTable.AppendCols(l)
		}
	case []Col:
		for _, r := range val {
			temDT := PtrDT(insyra.NewDataTable())
			err := fromRowToDT(temDT, r)
			if err != nil {
				insyra.LogFatal("DT", "Push", "%v", err)
			}
			numRow, _ := temDT.Size()
			for i := range numRow {
				l := temDT.GetRow(i)
				t.DataTable.AppendCols(l)
			}
		}
	default:
		insyra.LogFatal("DT", "Push", "got unexpected type %T", data)
	}
	return t
}

func fromRowToDT(t *dt, val map[any]any) error {
	strMap := make(map[string]any)
	if isIntKey(val) || isStrKey(val) {
		for k, v := range val {
			if isInt(k) {
				strMap[numberToColIndex(k.(int))] = v
			} else if isStr(k) {
				strMap[conv.ToString(k)] = v
			}
		}
		t.AppendRowsByColIndex(strMap)
	} else if isNameKey(val) {
		for k, v := range val {
			strMap[k.(name).value] = v
		}
		t.AppendRowsByColName(strMap)
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

func numberToColIndex(index int) string {
	name := ""
	for index >= 0 {
		name = fmt.Sprintf("%c%s", 'A'+(index%26), name)
		index = index/26 - 1
	}
	return name
}
