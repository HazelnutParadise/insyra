package insyra

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// showLayoutPair builds two tables that differ only in one row that the
// display does not show: a wide text cell and a long row name in one, short
// ones in the other. Column A is numeric so the stat line is printed too.
func showLayoutPair(rows, hidden int) (wide, narrow *DataTable) {
	build := func(text, rowName string) *DataTable {
		nums := make([]any, rows)
		texts := make([]any, rows)
		for i := range rows {
			nums[i] = i
			texts[i] = fmt.Sprintf("t%d", i)
		}
		texts[hidden] = text
		dt := NewDataTable(NewDataList(nums...).SetName("n"), NewDataList(texts...).SetName("s"))
		dt.SetRowNameByIndex(hidden, rowName)
		return dt
	}
	return build(strings.Repeat("w", 60), "a-very-long-row-name"), build("x", "r")
}

// ShowRange sizes its columns from the rows it prints. It used to format
// every cell of the table to measure widths, so ShowRange(5) on a million
// rows did a million rows of work and a wide cell far below the range
// widened the columns it printed.
func TestShowRangeMeasuresOnlyShownRows(t *testing.T) {
	wide, narrow := showLayoutPair(10, 7)
	var a, b bytes.Buffer
	wide.ShowRangeTo(&a, 5)
	narrow.ShowRangeTo(&b, 5)
	if a.String() != b.String() {
		t.Fatalf("a hidden row changed ShowRange(5):\n%s\nversus\n%s", a.String(), b.String())
	}
	if !strings.Contains(a.String(), "t4") || strings.Contains(a.String(), "t5") {
		t.Fatalf("ShowRange(5) printed the wrong rows:\n%s", a.String())
	}
}

// The default view of a long table prints the first 20 and last 5 rows. A
// row between them is measured no more than it is printed.
func TestShowMeasuresOnlyTheRowsItPrints(t *testing.T) {
	wide, narrow := showLayoutPair(30, 22)
	var a, b bytes.Buffer
	wide.ShowTo(&a)
	narrow.ShowTo(&b)
	if a.String() != b.String() {
		t.Fatalf("a hidden row changed Show:\n%s\nversus\n%s", a.String(), b.String())
	}
	if !strings.Contains(a.String(), "t19") || !strings.Contains(a.String(), "t25") || strings.Contains(a.String(), "t22") {
		t.Fatalf("Show printed the wrong rows:\n%s", a.String())
	}
}

// aStringTypeWithAVeryLongName is text to DataType, so the column's DataType
// row stays the same, while its type label is far wider than "string".
type aStringTypeWithAVeryLongName string

func TestShowTypesRangeMeasuresOnlyShownRows(t *testing.T) {
	build := func(hiddenValue any) *DataTable {
		return NewDataTable(NewDataList("a", "b", "c", hiddenValue).SetName("s"))
	}
	var a, b bytes.Buffer
	build(aStringTypeWithAVeryLongName("d")).ShowTypesRangeTo(&a, 3)
	build("d").ShowTypesRangeTo(&b, 3)
	if a.String() != b.String() {
		t.Fatalf("a hidden row changed ShowTypesRange(3):\n%s\nversus\n%s", a.String(), b.String())
	}
}

// Showable is exported so a caller can name the type Show takes.
func TestShowableIsExported(t *testing.T) {
	things := []Showable{NewDataTable(NewDataList(1)), NewDataList(1)}
	for _, s := range things {
		Show("x", s, 1)
	}
}

func millionRowTable() *DataTable {
	const n = 1_000_000
	a := make([]any, n)
	b := make([]any, n)
	c := make([]any, n)
	for i := range n {
		a[i] = i
		b[i] = float64(i) / 3
		c[i] = "row"
	}
	return NewDataTable(NewDataList(a...).SetName("a"), NewDataList(b...).SetName("b"), NewDataList(c...).SetName("c"))
}

// ShowRange(5) on a 1,000,000 x 3 table: #236 measured 475 ms per call.
func BenchmarkShowRangeFiveOfAMillionRows(b *testing.B) {
	dt := millionRowTable()
	b.ResetTimer()
	for b.Loop() {
		dt.ShowRangeTo(io.Discard, 5)
	}
}

func BenchmarkShowOfAMillionRows(b *testing.B) {
	dt := millionRowTable()
	b.ResetTimer()
	for b.Loop() {
		dt.ShowTo(io.Discard)
	}
}

func BenchmarkShowTypesRangeFiveOfAMillionRows(b *testing.B) {
	dt := millionRowTable()
	b.ResetTimer()
	for b.Loop() {
		dt.ShowTypesRangeTo(io.Discard, 5)
	}
}
