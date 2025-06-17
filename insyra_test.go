package insyra_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func TestProcessData(t *testing.T) {
	dl := insyra.NewDataList(1, 2, 3)
	s, len := insyra.ProcessData(dl)
	if len != 3 {
		t.Errorf("ProcessData() did not return the correct length")
	}
	if s == nil {
		t.Errorf("ProcessData() did not return the correct slice")
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
