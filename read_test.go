package insyra_test

import (
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func TestSlice2DToDataTable(t *testing.T) {
	// Test with [][]any
	data := [][]any{
		{1, "Alice", 3.5},
		{2, "Bob", 4.0},
		{3, "Charlie", 2.8},
	}

	dt, err := insyra.Slice2DToDataTable(data)
	if err != nil {
		t.Errorf("Slice2DToDataTable() returned error: %v", err)
		return
	}
	if dt == nil {
		t.Errorf("Slice2DToDataTable() returned nil DataTable")
		return
	}
	dt.Show()
	if len(dt.ColNames()) != 3 {
		t.Errorf("Slice2DToDataTable() did not create the correct number of columns")
		return
	}
	if dt.GetElement(1, "A") != 2 {
		t.Errorf("Slice2DToDataTable() did not set the correct data, expected 2, got %v", dt.GetElement(1, "A"))
		return
	}
	if dt.GetElement(0, "B") != "Alice" {
		t.Errorf("Slice2DToDataTable() did not set the correct data, expected 'Alice', got '%s'", dt.GetElement(0, "B"))
		return
	}
	if dt.GetElement(2, "C") != 2.8 {
		t.Errorf("Slice2DToDataTable() did not set the correct data, expected 2.8, got %v", dt.GetElement(2, "C"))
		return
	}
}

// Test with [][]int64
func TestSlice2DToDataTable_Int64(t *testing.T) {
	data := [][]int64{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	dt, err := insyra.Slice2DToDataTable(data)
	if err != nil {
		t.Errorf("Slice2DToDataTable() with [][]int64 returned error: %v", err)
		return
	}
	if dt == nil {
		t.Errorf("Slice2DToDataTable() with [][]int64 returned nil DataTable")
		return
	}
	dt.Show()
	if len(dt.ColNames()) != 3 {
		t.Errorf("Slice2DToDataTable() with [][]int64 did not create the correct number of columns")
		return
	}
	if dt.GetElement(1, "A") != int64(4) {
		t.Errorf("Slice2DToDataTable() with [][]int64 expected int64(4), got %v", dt.GetElement(1, "A"))
		return
	}
}

// Test with [][]float64
func TestSlice2DToDataTable_Float64(t *testing.T) {
	data := [][]float64{
		{1.1, 2.2, 3.3},
		{4.4, 5.5, 6.6},
		{7.7, 8.8, 9.9},
	}

	dt, err := insyra.Slice2DToDataTable(data)
	if err != nil {
		t.Errorf("Slice2DToDataTable() with [][]float64 returned error: %v", err)
		return
	}
	if dt == nil {
		t.Errorf("Slice2DToDataTable() with [][]float64 returned nil DataTable")
		return
	}
	dt.Show()
	if len(dt.ColNames()) != 3 {
		t.Errorf("Slice2DToDataTable() with [][]float64 did not create the correct number of columns")
		return
	}
	if dt.GetElement(0, "A") != 1.1 {
		t.Errorf("Slice2DToDataTable() with [][]float64 expected 1.1, got %v", dt.GetElement(0, "A"))
		return
	}
}

// Test with [][]string
func TestSlice2DToDataTable_String(t *testing.T) {
	data := [][]string{
		{"Alice", "Bob", "Charlie"},
		{"Denver", "New York", "San Francisco"},
		{"Engineer", "Manager", "Developer"},
	}

	dt, err := insyra.Slice2DToDataTable(data)
	if err != nil {
		t.Errorf("Slice2DToDataTable() with [][]string returned error: %v", err)
		return
	}
	if dt == nil {
		t.Errorf("Slice2DToDataTable() with [][]string returned nil DataTable")
		return
	}
	dt.Show()
	if len(dt.ColNames()) != 3 {
		t.Errorf("Slice2DToDataTable() with [][]string did not create the correct number of columns")
		return
	}
	if dt.GetElement(0, "A") != "Alice" {
		t.Errorf("Slice2DToDataTable() with [][]string expected 'Alice', got %v", dt.GetElement(0, "A"))
		return
	}
}

// Test with inconsistent row lengths
func TestSlice2DToDataTable_InconsistentLengths(t *testing.T) {
	data := [][]any{
		{1, "Alice", 3.5},
		{2, "Bob"},                   // 少一列
		{3, "Charlie", 2.8, "Extra"}, // 多一列
	}

	dt, err := insyra.Slice2DToDataTable(data)
	if err != nil {
		t.Errorf("Slice2DToDataTable() with inconsistent lengths returned error: %v", err)
		return
	}
	if dt == nil {
		t.Errorf("Slice2DToDataTable() with inconsistent lengths returned nil DataTable")
		return
	}
	dt.Show()
	// 應該以第一行的列數為準
	if len(dt.ColNames()) != 3 {
		t.Errorf("Slice2DToDataTable() with inconsistent lengths expected 3 columns, got %d", len(dt.ColNames()))
		return
	}
	// 第二行的第三列應該是 nil
	if dt.GetElement(1, "C") != nil {
		t.Errorf("Slice2DToDataTable() with inconsistent lengths expected nil for missing cell, got %v", dt.GetElement(1, "C"))
		return
	}
}

// Test error cases
func TestSlice2DToDataTable_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		wantErr bool
	}{
		{
			name:    "nil input",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "empty slice",
			data:    [][]any{},
			wantErr: true,
		},
		{
			name:    "not a 2D slice",
			data:    []int{1, 2, 3},
			wantErr: true,
		},
		{
			name:    "first row empty",
			data:    [][]any{{}},
			wantErr: true,
		},
		{
			name:    "row is not a slice",
			data:    []any{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := insyra.Slice2DToDataTable(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slice2DToDataTable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && dt == nil {
				t.Errorf("Slice2DToDataTable() returned nil DataTable when no error expected")
			}
			if tt.wantErr && dt != nil {
				t.Errorf("Slice2DToDataTable() returned DataTable when error was expected: %v", err)
			}
		})
	}
}

func TestReadCSV_String(t *testing.T) {
	csvData := "name,age,city\nJohn,30,NYC\nJane,25,LA"
	dtt := isr.DT.From(isr.CSV{
		String: csvData,
		InputOpts: isr.CSV_inOpts{
			FirstRow2ColNames: true,  // 第一行作為列名
			FirstCol2RowNames: false, // 第一列作為行名
		},
	})
	if dtt == nil {
		t.Errorf("ReadCSV_String() returned nil")
		return
	}
	if len(dtt.ColNames()) != 3 {
		t.Errorf("ReadCSV_String() did not parse the correct number of columns")
		return
	}
	if dtt.GetColByName("name").Data()[0] != "John" {
		t.Errorf("ReadCSV_String() did not parse the correct data, expected 'John', got '%s'", dtt.GetColByName("name").Data()[0])
		return
	}
	// Column-level type inference: an all-integer column parses as int64 (not
	// float64) so large-integer columns keep full precision. See inferCSVColumnTypes.
	if dtt.GetColByName("age").Data()[1] != int64(25) {
		t.Errorf("ReadCSV_String() did not parse the correct data, expected int64(25), got %v (%T)", dtt.GetColByName("age").Data()[1], dtt.GetColByName("age").Data()[1])
		return
	}
	if dtt.GetColByName("city").Data()[0] != "NYC" {
		t.Errorf("ReadCSV_String() did not parse the correct data, expected 'NYC', got '%s'", dtt.GetColByName("city").Data()[0])
		return
	}

}

func TestReadJSON(t *testing.T) {
	jsonData := `[
		{"name": "John", "age": 30, "city": "NYC"},
		{"name": "Jane", "age": 25, "city": "LA"}
	]`
	dtt := isr.DT.From(isr.JSON{
		Bytes: []byte(jsonData),
	})
	if dtt == nil {
		t.Errorf("ReadJSON() returned nil")
		return
	}
	if len(dtt.ColNames()) != 3 {
		t.Errorf("ReadJSON() did not parse the correct number of columns")
		return
	}
	if dtt.GetColByName("name").Data()[0] != "John" {
		t.Errorf("ReadJSON() did not parse the correct data, expected 'John', got '%s'", dtt.GetColByName("name").Data()[0])
		return
	}
	// Integer JSON literals (30, 25) are typed as int64 so large integers keep
	// full precision; decimals stay float64. See coerceJSONNumber.
	if dtt.GetColByName("age").Data()[1] != int64(25) {
		t.Errorf("ReadJSON() did not parse the correct data, expected int64(25), got %v (%T)", dtt.GetColByName("age").Data()[1], dtt.GetColByName("age").Data()[1])
		return
	}
	if dtt.GetColByName("city").Data()[0] != "NYC" {
		t.Errorf("ReadJSON() did not parse the correct data, expected 'NYC', got '%s'", dtt.GetColByName("city").Data()[0])
		return
	}
}

// Issue #188: RawStrings mode loads every cell as its original string —
// leading zeros in stock IDs survive, thousand-separated amounts stay verbatim,
// and empty cells stay "" instead of becoming NaN.
func TestReadCSV_StringWithOptions_RawStrings(t *testing.T) {
	csvData := "股票代號,集保庫存,成交均價\n2330,1000,600.855\n0050,\"2,000\",100.14\n00878,1500,\n"
	dt, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{
		FirstRowToColNames: true,
		RawStrings:         true,
	})
	if err != nil {
		t.Fatalf("ReadCSV_StringWithOptions: %v", err)
	}

	ids := dt.GetColByName("股票代號").Data()
	wantIDs := []any{"2330", "0050", "00878"}
	for i, want := range wantIDs {
		if ids[i] != want {
			t.Errorf("stock ID row %d: expected %q, got %v (%T)", i, want, ids[i], ids[i])
		}
	}

	inventory := dt.GetColByName("集保庫存").Data()
	if inventory[1] != "2,000" {
		t.Errorf("inventory row 1: expected \"2,000\", got %v (%T)", inventory[1], inventory[1])
	}

	prices := dt.GetColByName("成交均價").Data()
	if prices[0] != "600.855" {
		t.Errorf("price row 0: expected \"600.855\", got %v (%T)", prices[0], prices[0])
	}
	if prices[2] != "" {
		t.Errorf("price row 2 (empty cell): expected \"\", got %v (%T)", prices[2], prices[2])
	}
}

// Options-based CSV loading must reproduce legacy loading exactly, including
// column type inference.
func TestReadCSV_WithOptions_ZeroValueMatchesLegacy(t *testing.T) {
	csvData := "id,val,note\n0050,600.855,a\n2330,100.14,b\n"
	legacy, err := insyra.ReadCSV_String(csvData, false, true)
	if err != nil {
		t.Fatalf("ReadCSV_String: %v", err)
	}
	withOpts, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{FirstRowToColNames: true})
	if err != nil {
		t.Fatalf("ReadCSV_StringWithOptions: %v", err)
	}

	for _, col := range []string{"id", "val", "note"} {
		l, w := legacy.GetColByName(col).Data(), withOpts.GetColByName(col).Data()
		if len(l) != len(w) {
			t.Fatalf("column %s: length mismatch %d vs %d", col, len(l), len(w))
		}
		for i := range l {
			if l[i] != w[i] {
				t.Errorf("column %s row %d: legacy %v (%T) vs options %v (%T)", col, i, l[i], l[i], w[i], w[i])
			}
		}
	}
	// Inference still applies with zero-value options: the id column is all-int.
	if withOpts.GetColByName("id").Data()[0] != int64(50) {
		t.Errorf("expected inferred int64(50), got %v (%T)", withOpts.GetColByName("id").Data()[0], withOpts.GetColByName("id").Data()[0])
	}
}

func TestReadCSV_StringWithOptions_LiteralZeroValueMatchesLegacyDefaults(t *testing.T) {
	csvData := "1,2\n3,4\n"
	legacy, err := insyra.ReadCSV_String(csvData, false, false)
	if err != nil {
		t.Fatalf("ReadCSV_String: %v", err)
	}
	withOpts, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{})
	if err != nil {
		t.Fatalf("ReadCSV_StringWithOptions: %v", err)
	}
	if got, want := withOpts.ColNames(), legacy.ColNames(); len(got) != len(want) {
		t.Fatalf("column count mismatch: options=%v legacy=%v", got, want)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("column %d name: options=%q legacy=%q", i, got[i], want[i])
			}
		}
	}
	for colIndex := 0; colIndex < withOpts.NumCols(); colIndex++ {
		got, want := withOpts.GetColByNumber(colIndex).Data(), legacy.GetColByNumber(colIndex).Data()
		if len(got) != len(want) {
			t.Fatalf("column %d length mismatch: options=%v legacy=%v", colIndex, got, want)
		}
		for rowIndex := range got {
			if got[rowIndex] != want[rowIndex] {
				t.Errorf("cell (%d,%d): options=%v legacy=%v", rowIndex, colIndex, got[rowIndex], want[rowIndex])
			}
		}
	}
}

func TestReadCSV_FileWithOptions_RawStrings(t *testing.T) {
	path := t.TempDir() + "/stocks.csv"
	if err := os.WriteFile(path, []byte("id,qty\n0050,1000\n00878,\n"), 0o644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	dt, err := insyra.ReadCSV_FileWithOptions(path, insyra.CSVReadOptions{
		FirstRowToColNames: true,
		RawStrings:         true,
	})
	if err != nil {
		t.Fatalf("ReadCSV_FileWithOptions: %v", err)
	}
	if got := dt.GetColByName("id").Data()[0]; got != "0050" {
		t.Errorf("expected \"0050\", got %v (%T)", got, got)
	}
	if got := dt.GetColByName("qty").Data()[1]; got != "" {
		t.Errorf("empty cell: expected \"\", got %v (%T)", got, got)
	}
}

// isr passes CSV_inOpts.RawStrings through to the options-based reader.
func TestReadCSV_ISR_RawStrings(t *testing.T) {
	dtt := isr.DT.From(isr.CSV{
		String: "id,price\n0050,100.14\n",
		InputOpts: isr.CSV_inOpts{
			FirstRow2ColNames: true,
			RawStrings:        true,
		},
	})
	if got := dtt.GetColByName("id").Data()[0]; got != "0050" {
		t.Errorf("expected \"0050\", got %v (%T)", got, got)
	}
	if got := dtt.GetColByName("price").Data()[0]; got != "100.14" {
		t.Errorf("expected \"100.14\", got %v (%T)", got, got)
	}
}

func TestReadCSV_StringWithOptions_RaggedRows(t *testing.T) {
	cases := []struct {
		name     string
		csv      string
		rownames bool
		check    func(t *testing.T, dt *insyra.DataTable)
	}{
		{
			name:     "trailer note is padded",
			csv:      "id,value,note\n1,2,ok\n以上資料僅供參考\n",
			rownames: true,
			check: func(t *testing.T, dt *insyra.DataTable) {
				if got, ok := dt.GetRowNameByIndex(1); !ok || got != "以上資料僅供參考" {
					t.Errorf("expected trailer note as row name, got (%q, %v)", got, ok)
				}
				for _, index := range []int{0, 1} {
					if got := dt.GetColByNumber(index).Data()[1]; got != "" {
						t.Errorf("expected padded empty cell in column %d, got %v", index, got)
					}
				}
			},
		},
		{
			name: "trailing comma keeps empty extra column",
			csv:  "id,value\n1,2,\n3,4\n",
			check: func(t *testing.T, dt *insyra.DataTable) {
				if got := dt.NumCols(); got != 3 {
					t.Fatalf("expected 3 columns, got %d", got)
				}
				if got := dt.GetColByNumber(2).Data(); got[0] != "" || got[1] != "" {
					t.Errorf("expected empty extra column, got %v", got)
				}
			},
		},
		{
			name: "non-empty extra cells are retained",
			csv:  "id,value\n1,2\n3,4,extra,more\n",
			check: func(t *testing.T, dt *insyra.DataTable) {
				if got := dt.NumCols(); got != 4 {
					t.Fatalf("expected 4 columns, got %d", got)
				}
				if got := dt.GetColByNumber(2).Data(); got[0] != "" || got[1] != "extra" {
					t.Errorf("unexpected first extra column: %v", got)
				}
				if got := dt.GetColByNumber(3).Data(); got[0] != "" || got[1] != "more" {
					t.Errorf("unexpected second extra column: %v", got)
				}
			},
		},
		{
			name:     "row names and header names stay aligned",
			csv:      "label,value\nr1,10\ntrailer\nr2,20,extra\n",
			rownames: true,
			check: func(t *testing.T, dt *insyra.DataTable) {
				if got, ok := dt.GetRowNameByIndex(1); !ok || got != "trailer" {
					t.Errorf("expected trailer row name, got (%q, %v)", got, ok)
				}
				if got := dt.GetColByName("value").Data()[1]; got != "" {
					t.Errorf("expected empty value beside trailer row, got %v", got)
				}
				if got := dt.GetColByNumber(1).Data()[2]; got != "extra" {
					t.Errorf("expected extra cell in auto-named column, got %v", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dt, err := insyra.ReadCSV_StringWithOptions(tc.csv, insyra.CSVReadOptions{
				FirstColToRowNames: tc.rownames,
				FirstRowToColNames: true,
				RawStrings:         true,
				AllowRaggedRows:    true,
			})
			if err != nil {
				t.Fatalf("ReadCSV_StringWithOptions: %v", err)
			}
			tc.check(t, dt)
			for colIndex := 0; colIndex < dt.NumCols(); colIndex++ {
				for rowIndex, value := range dt.GetColByNumber(colIndex).Data() {
					if _, ok := value.(string); !ok {
						t.Errorf("cell (%d,%d) has type %T, want string", rowIndex, colIndex, value)
					}
				}
			}
		})
	}
}

func TestReadCSV_StringWithOptions_TrimLeadingSpace(t *testing.T) {
	csvData := "id,name,amount\n2330, \"1,000\",600.86\n"
	if _, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{}); err == nil {
		t.Fatal("expected zero-value options to reject a space before a quote")
	}
	dt, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{
		FirstRowToColNames: true,
		TrimLeadingSpace:   true,
	})
	if err != nil {
		t.Fatalf("TrimLeadingSpace should parse quoted field: %v", err)
	}
	if got := dt.GetColByName("name").Data()[0]; got != "1,000" {
		t.Errorf("expected quoted value 1,000, got %v", got)
	}
}

func TestReadCSV_StringWithOptions_ZeroValueRejectsRaggedRows(t *testing.T) {
	for _, csvData := range []string{
		"id,value\n1\n",
		"id,value\n1,2,\n",
		"id,value\n1,2,extra\n",
	} {
		if _, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{}); err == nil {
			t.Errorf("expected zero-value options to reject %q", csvData)
		}
	}
}

func TestReadCSV_FileAndStringWithOptions_RaggedRowsMatch(t *testing.T) {
	// The `""` and whitespace-only lines parse to a single empty field; both
	// must survive the file path (which once re-serialized records and lost
	// them as blank lines) exactly like the string path.
	csvData := "id,name\n1, \"Alice\"\n\"\"\ntrailer\n   \n2,Bob,extra\n"
	opts := insyra.CSVReadOptions{
		FirstRowToColNames: true,
		RawStrings:         true,
		AllowRaggedRows:    true,
		TrimLeadingSpace:   true,
	}
	fromString, err := insyra.ReadCSV_StringWithOptions(csvData, opts)
	if err != nil {
		t.Fatalf("ReadCSV_StringWithOptions: %v", err)
	}
	path := t.TempDir() + "/ragged.csv"
	if err := os.WriteFile(path, []byte(csvData), 0o644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	fromFile, err := insyra.ReadCSV_FileWithOptions(path, opts)
	if err != nil {
		t.Fatalf("ReadCSV_FileWithOptions: %v", err)
	}

	if got := fromString.NumRows(); got != 5 {
		t.Fatalf("expected 5 data rows from string, got %d", got)
	}
	if got := fromFile.NumRows(); got != 5 {
		t.Fatalf("expected 5 data rows from file, got %d", got)
	}
	if got, want := fromFile.ColNames(), fromString.ColNames(); len(got) != len(want) {
		t.Fatalf("column count mismatch: file=%v string=%v", got, want)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("column %d name: file=%q string=%q", i, got[i], want[i])
			}
		}
	}
	for colIndex := 0; colIndex < fromString.NumCols(); colIndex++ {
		got, want := fromFile.GetColByNumber(colIndex).Data(), fromString.GetColByNumber(colIndex).Data()
		if len(got) != len(want) {
			t.Fatalf("column %d length mismatch: file=%v string=%v", colIndex, got, want)
		}
		for rowIndex := range got {
			if got[rowIndex] != want[rowIndex] {
				t.Errorf("cell (%d,%d): file=%v string=%v", rowIndex, colIndex, got[rowIndex], want[rowIndex])
			}
		}
	}
}

// Extra columns are numbered by their ordinal in the file, so the auto name
// does not shift when FirstColToRowNames consumes the first field.
func TestReadCSV_StringWithOptions_RaggedExtraColNameStableAcrossRowNames(t *testing.T) {
	csvData := "label,value\nr1,10\nr2,20,extra\n"
	for _, rownames := range []bool{false, true} {
		dt, err := insyra.ReadCSV_StringWithOptions(csvData, insyra.CSVReadOptions{
			FirstColToRowNames: rownames,
			FirstRowToColNames: true,
			RawStrings:         true,
			AllowRaggedRows:    true,
		})
		if err != nil {
			t.Fatalf("rownames=%v: %v", rownames, err)
		}
		col := dt.GetColByName("extra_3")
		if col == nil {
			t.Fatalf("rownames=%v: column extra_3 not found (cols=%v)", rownames, dt.ColNames())
		}
		if got := col.Data()[1]; got != "extra" {
			t.Errorf("rownames=%v: expected extra cell in extra_3, got %v", rownames, got)
		}
	}
}

// Padded cells count as empty for type inference: an otherwise-integer column
// becomes float64 with NaN once a short row pads it. Use RawStrings to keep
// cells verbatim instead.
func TestReadCSV_StringWithOptions_RaggedPaddingAffectsInference(t *testing.T) {
	dt, err := insyra.ReadCSV_StringWithOptions("id,qty\n1,10\n2\n", insyra.CSVReadOptions{
		FirstRowToColNames: true,
		AllowRaggedRows:    true,
	})
	if err != nil {
		t.Fatalf("ReadCSV_StringWithOptions: %v", err)
	}
	qty := dt.GetColByName("qty").Data()
	if v, ok := qty[0].(float64); !ok || v != 10 {
		t.Errorf("expected float64 10, got %v (%T)", qty[0], qty[0])
	}
	if v, ok := qty[1].(float64); !ok || !math.IsNaN(v) {
		t.Errorf("expected NaN for padded cell, got %v (%T)", qty[1], qty[1])
	}
}
