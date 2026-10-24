package insyra

import (
	"bytes"
	"fmt"
	"regexp"
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

// An unnamed column of ints is three characters wide by its letter and its
// Go type, narrower than the word its DataType row prints; the row still
// prints the whole word.
func TestShowTypesFitsTheDataTypeWord(t *testing.T) {
	colored := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(colored)

	dt := NewDataTable(NewDataList(1, 2), NewDataList(nil, nil))
	var buf bytes.Buffer
	dt.ShowTypesTo(&buf)
	var row string
	for line := range strings.SplitSeq(buf.String(), "\n") {
		if strings.HasPrefix(line, "DataType") {
			row = line
			break
		}
	}
	if got := strings.Fields(row); len(got) != 3 || got[1] != "number" || got[2] != "empty" {
		t.Fatalf("DataType row = %q, want number and empty in full:\n%s", row, buf.String())
	}
}

// IN-9: ShowTypes sorted its column headers as plain strings, so a table with
// more than 26 columns printed A, AA, AB, B, ... while Show printed them in
// the real column order.
func TestShowTypesColumnOrderBeyondZ(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable()
	for i := 0; i < 28; i++ {
		dt.AppendCols(NewDataList(i))
	}
	coloured := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(coloured)

	var typesOut, showOut bytes.Buffer
	dt.ShowTypesRangeTo(&typesOut)
	dt.ShowRangeTo(&showOut)

	typeCols := headerColumns(typesOut.String())
	showCols := headerColumns(showOut.String())
	if len(typeCols) == 0 || len(showCols) == 0 {
		t.Fatalf("could not find the header row\ntypes:\n%s\nshow:\n%s", typesOut.String(), showOut.String())
	}
	if strings.Join(typeCols, ",") != strings.Join(showCols, ",") {
		t.Fatalf("ShowTypes column order %v differs from Show's %v", typeCols, showCols)
	}
}

// The same order holds when the columns have names. ShowTypes keys a named
// column as "AA(name)", and parsing that whole key as a column index failed, so
// it fell back to a string sort and printed A, AA, AB, B again.
func TestShowTypesColumnOrderBeyondZWithNamedColumns(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable()
	for i := 0; i < 28; i++ {
		dt.AppendCols(NewDataList(i).SetName(fmt.Sprintf("c%d", i)))
	}
	coloured := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(coloured)

	var typesOut, showOut bytes.Buffer
	dt.ShowTypesRangeTo(&typesOut)
	dt.ShowRangeTo(&showOut)

	typeCols := headerColumns(typesOut.String())
	showCols := headerColumns(showOut.String())
	if len(typeCols) == 0 || len(showCols) == 0 {
		t.Fatalf("could not find the header row\ntypes:\n%s\nshow:\n%s", typesOut.String(), showOut.String())
	}
	if strings.Join(typeCols, ",") != strings.Join(showCols, ",") {
		t.Fatalf("ShowTypes column order %v differs from Show's %v", typeCols, showCols)
	}
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// headerColumns returns the column labels from every "RowNames ..." header
// line, in the order they are printed (the display pages the columns).
func headerColumns(out string) []string {
	var cols []string
	for _, line := range strings.Split(ansiEscape.ReplaceAllString(out, ""), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "RowNames" {
			continue
		}
		cols = append(cols, fields[1:]...)
	}
	return cols
}
