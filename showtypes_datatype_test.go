package insyra

import (
	"bytes"
	"strings"
	"testing"
)

// ShowTypes prints each cell's Go type; the DataType row on top sums each
// column up in one word, so a reader sees both the detail and the verdict.
func TestShowTypesLeadsWithTheDataType(t *testing.T) {
	colored := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(colored)

	dt := NewDataTable(
		NewDataList(10, 2.5, nil).SetName("price"),
		NewDataList("a", nil, "b").SetName("city"),
		NewDataList(1, "a", nil).SetName("notes"),
	)
	var buf bytes.Buffer
	dt.ShowTypesTo(&buf)
	lines := strings.Split(buf.String(), "\n")
	var row string
	for i, line := range lines {
		if strings.HasPrefix(line, "DataType") {
			row = line
			if i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], "---") {
				t.Errorf("the DataType row is not set apart from the cell rows:\n%s", buf.String())
			}
			break
		}
		if strings.HasPrefix(line, "0:") {
			t.Fatalf("a cell row came before the DataType row:\n%s", buf.String())
		}
	}
	if got := strings.Fields(row); len(got) != 4 || got[1] != "number" || got[2] != "string" || got[3] != "mixed" {
		t.Fatalf("DataType row = %q, want number, string and mixed:\n%s", row, buf.String())
	}

	buf.Reset()
	NewDataList(1, "a").ShowTypesTo(&buf)
	if !strings.Contains(buf.String(), "DataType: mixed") {
		t.Fatalf("a DataList's type info does not state its data type:\n%s", buf.String())
	}
}
