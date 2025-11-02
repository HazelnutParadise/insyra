package insyra_test

import (
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
	if dtt.GetColByName("age").Data()[1] != 25.0 {
		t.Errorf("ReadCSV_String() did not parse the correct data, expected 25, got %v", dtt.GetColByName("age").Data()[1])
		return
	}
	if dtt.GetColByName("city").Data()[0] != "NYC" {
		t.Errorf("ReadCSV_String() did not parse the correct data, expected 'NYC', got '%s'", dtt.GetColByName("city").Data()[0])
		return
	}

}

func TestReadJSON_Bytes(t *testing.T) {
	jsonData := `[
		{"name": "John", "age": 30, "city": "NYC"},
		{"name": "Jane", "age": 25, "city": "LA"}
	]`
	dtt := isr.DT.From(isr.JSON{
		Bytes: []byte(jsonData),
	})
	if dtt == nil {
		t.Errorf("ReadJSON_Bytes() returned nil")
		return
	}
	if len(dtt.ColNames()) != 3 {
		t.Errorf("ReadJSON_Bytes() did not parse the correct number of columns")
		return
	}
	if dtt.GetColByName("name").Data()[0] != "John" {
		t.Errorf("ReadJSON_Bytes() did not parse the correct data, expected 'John', got '%s'", dtt.GetColByName("name").Data()[0])
		return
	}
	if dtt.GetColByName("age").Data()[1] != 25.0 {
		t.Errorf("ReadJSON_Bytes() did not parse the correct data, expected 25, got %v", dtt.GetColByName("age").Data()[1])
		return
	}
	if dtt.GetColByName("city").Data()[0] != "NYC" {
		t.Errorf("ReadJSON_Bytes() did not parse the correct data, expected 'NYC', got '%s'", dtt.GetColByName("city").Data()[0])
		return
	}
}
