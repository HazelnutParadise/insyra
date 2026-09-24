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

// Deprecated: Use UseDT instead.
// PtrDT converts a DataTable or DT to a *DT.
// You should no longer use this function, use UseDT instead.
func PtrDT[T *insyra.DataTable | dt](t T) *dt {
	return UseDT(t)
}

// From converts a DataList, DL, Row, Col, []DL, []Row, []Col, CSV, map[string]any, or map[int]any to a DataTable.
// nolint:govet
func (d dt) From(item any) *dt {
	t := dt{}
	switch val := item.(type) {
	case [][]any, [][]int, [][]float64, [][]string, [][]bool, [][]uint, [][]int8, [][]int16, [][]int32, [][]int64, [][]uint8, [][]uint16, [][]uint32, [][]uint64, [][]float32, [][]complex64, [][]complex128, [][]uintptr:
		converted, err := insyra.Slice2DToDataTable(val)
		if err != nil {
			insyra.LogWarning("isr", "DT.From", "%v", err)
		}
		t.DataTable = converted
	case *insyra.DataList:
		t.DataTable = insyra.NewDataTable(val)
	case *dl:
		t.DataTable = insyra.NewDataTable(val.DataList)
	case []*insyra.DataList:
		t.DataTable = insyra.NewDataTable()
		for _, l := range val {
			t.AppendCols(l)
		}
	case []dl:
		t.DataTable = insyra.NewDataTable()
		for _, l := range val {
			t.AppendCols(l.DataList)
		}
	case DLs:
		t.DataTable = insyra.NewDataTable()
		for _, l := range val {
			newdl := insyra.NewDataList(l.Data()...)
			if l.GetName() != "" {
				newdl.SetName(l.GetName())
			}
			t.AppendCols(newdl)
		}
	case Row:
		t.DataTable = insyra.NewDataTable()
		err := fromRowToDT(&t, val)
		if err != nil {
			recordDT(&t, "DT.From", "%v", err)
		}
	case Rows:
		t.DataTable = insyra.NewDataTable()
		for _, r := range val {
			err := fromRowToDT(&t, r)
			if err != nil {
				recordDT(&t, "DT.From", "%v", err)
			}
		}
	case Col:
		t.DataTable = insyra.NewDataTable()
		err := fromRowToDT(&t, val)
		if err != nil {
			recordDT(&t, "DT.From", "%v", err)
		}
		t.Transpose()
	case Cols:
		t.DataTable = insyra.NewDataTable()
		for _, r := range val {
			err := fromRowToDT(&t, r)
			if err != nil {
				recordDT(&t, "DT.From", "%v", err)
			}
		}
		t.Transpose()
	case Excel:
		t.DataTable = insyra.NewDataTable()
		var err error
		if val.FilePath == "" {
			insyra.LogWarning("isr", "DT.From", "Excel FilePath cannot be empty")
		}
		t.DataTable, err = insyra.ReadExcelSheet(val.FilePath, val.SheetName, val.InputOpts.FirstCol2RowNames, val.InputOpts.FirstRow2ColNames)
		if err != nil {
			recordDT(&t, "DT.From", "%v", err)
		}
	case CSV:
		t.DataTable = insyra.NewDataTable()
		var err error
		opts := insyra.CSVReadOptions{
			FirstColToRowNames: val.InputOpts.FirstCol2RowNames,
			FirstRowToColNames: val.InputOpts.FirstRow2ColNames,
			Encoding:           val.InputOpts.Encoding,
			RawStrings:         val.InputOpts.RawStrings,
			AllowRaggedRows:    val.InputOpts.AllowRaggedRows,
			TrimLeadingSpace:   val.InputOpts.TrimLeadingSpace,
		}
		if val.FilePath != "" {
			t.DataTable, err = insyra.ReadCSV_FileWithOptions(val.FilePath, opts)
		} else {
			t.DataTable, err = insyra.ReadCSV_StringWithOptions(val.String, opts)
		}
		if err != nil {
			recordDT(&t, "DT.From", "%v", err)
		}
	case JSON:
		t.DataTable = insyra.NewDataTable()
		var err error
		if val.FilePath != "" {
			t.DataTable, err = insyra.ReadJSON_File(val.FilePath)
		} else {
			t.DataTable, err = insyra.ReadJSON(val.Bytes)
		}
		if err != nil {
			recordDT(&t, "DT.From", "%v", err)
		}
	case map[string]any:
		t.DataTable = insyra.NewDataTable().AppendRowsByColIndex(val)
	case map[int]any:
		// AppendRowsByColIndex wants an Excel-style index, so key 0 has to
		// become "A". conv.ToString made it "0", which the method rejected —
		// every key, so this documented input always produced an empty table.
		// numberToColIndex is the same helper the Row path uses.
		strV := make(map[string]any)
		for k, v := range val {
			strV[numberToColIndex(k)] = v
		}
		t.DataTable = insyra.NewDataTable().AppendRowsByColIndex(strV)
	case nil:
		// do nothing, return an empty DataTable
	default:
		insyra.LogWarning("isr", "DT.From", "got unexpected type %T", item)
	}
	return &t
}

// Of is an alias for From.
// It converts a DataList, DL, Row, Col, []DL, []Row, []Col, CSV, map[string]any, or map[int]any to a DataTable.
func (d dt) Of(item any) *dt {
	return d.From(item)
}

// Col returns a DL that contains the column at the specified index.
func (t *dt) Col(col any) *dl {
	var l dl
	switch v := col.(type) {
	case int:
		l.DataList = t.GetColByNumber(v)
	case string:
		l.DataList = t.GetCol(v)
	case name:
		colDt := t.FilterColsByColNameEqualTo(v.value)
		l.DataList = colDt.GetColByNumber(0)
	default:
		insyra.LogWarning("isr", "DT.Col", "got unexpected selector type %T; use an int, an Excel index string, or isr.Name", col)
	}
	return &l
}

// Row returns a DL that contains the row at the specified index.
func (t *dt) Row(row any) *dl {
	var l dl
	switch v := row.(type) {
	case int:
		l.DataList = t.GetRow(v)
	case name:
		rowDt := t.FilterRowsByRowNameEqualTo(v.value)
		l.DataList = rowDt.GetRow(0)
	default:
		insyra.LogWarning("isr", "DT.Row", "got unexpected selector type %T; use an int or isr.Name", row)
	}
	return &l
}

// At returns the element at the specified row and column.
func (t *dt) At(row any, col any) any {
	switch v := col.(type) {
	case int:
		switch r := row.(type) {
		case int:
			return t.GetElementByNumberIndex(r, v)
		case name:
			rowDt := t.FilterRowsByRowNameEqualTo(r.value)
			return rowDt.GetElementByNumberIndex(0, v)
		default:
			insyra.LogWarning("DT", "At", "got unexpected type %T. Returning nil.", row)
		}
	case string:
		switch r := row.(type) {
		case int:
			return t.GetElement(r, v)
		case name:
			rowDt := t.FilterRowsByRowNameEqualTo(r.value)
			return rowDt.GetElement(0, v)
		default:
			insyra.LogWarning("DT", "At", "got unexpected type %T. Returning nil.", row)
		}
	case name:
		switch r := row.(type) {
		case int:
			colDt := t.FilterColsByColNameEqualTo(v.value)
			return colDt.GetElementByNumberIndex(r, 0)
		case name:
			rowDt := t.FilterRowsByRowNameEqualTo(r.value)
			colDt := rowDt.FilterColsByColNameEqualTo(v.value)
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
		t.AppendCols(val)
	case *dl:
		t.AppendCols(val.DataList)
	case []*insyra.DataList:
		t.AtomicDo(func(dt *insyra.DataTable) {
			for _, l := range val {
				dt.AppendCols(l)
			}
		})
	case []dl:
		for _, l := range val {
			t.AppendCols(l.DataList)
		}
	case DLs:
		t.AtomicDo(func(dt *insyra.DataTable) {
			for _, l := range val {
				newdl := insyra.NewDataList(l.Data()...)
				if l.GetName() != "" {
					newdl.SetName(l.GetName())
				}
				dt.AppendCols(newdl)
			}
		})
	case Row:
		err := fromRowToDT(t, val)
		if err != nil {
			recordDT(t, "DT.Push", "%v", err)
		}
	case []Row:
		t.AtomicDo(func(dt *insyra.DataTable) {
			for _, r := range val {
				// A row that cannot be added is skipped and the rest still go in.
				if err := fromRowToDT(UseDT(dt), r); err != nil {
					dt.SetErr("isr", "DT.Push", "%v", err)
				}
			}
		})
	case Col:
		// 先創建新dt 當成row插入 再轉置
		// 轉置後抽出為dl 再插入
		temDT := UseDT(insyra.NewDataTable())
		err := fromRowToDT(temDT, val)
		if err != nil {
			recordDT(t, "DT.Push", "%v", err)
		}
		numRow, _ := temDT.Size()
		// Snapshot rows via temDT's own lock (top level) before entering t's actor,
		// so we don't nest a cross-instance AtomicDo (which would not lock temDT).
		rows := make([]*insyra.DataList, 0, numRow)
		for i := range numRow {
			rows = append(rows, temDT.GetRow(i))
		}
		t.AtomicDo(func(dt *insyra.DataTable) {
			for _, l := range rows {
				dt.AppendCols(l)
			}
		})
	case []Col:
		for _, r := range val {
			temDT := UseDT(insyra.NewDataTable())
			err := fromRowToDT(temDT, r)
			if err != nil {
				recordDT(t, "DT.Push", "%v", err)
			}
			numRow, _ := temDT.Size()
			rows := make([]*insyra.DataList, 0, numRow)
			for i := range numRow {
				rows = append(rows, temDT.GetRow(i))
			}
			t.AtomicDo(func(dt *insyra.DataTable) {
				for _, l := range rows {
					dt.AppendCols(l)
				}
			})
		}
	default:
		recordDT(t, "DT.Push", "got unexpected type %T", data)
	}
	return t
}

// CCL executes CCL statements on the DataTable and returns the updated DataTable.
func (t *dt) CCL(cclStatements string) *dt {
	return UseDT(t.ExecuteCCL(cclStatements))
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
