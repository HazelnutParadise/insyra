package stats

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

type textInner struct {
	A float64
	B *float64
}

type textSample struct {
	textInner // embedded: A and B come first
	Big       float64
	Tiny      float64
	NaN       float64
	Count     int
	Method    string
	Ptr       *float64 // nil -> omitted
	Set       *int     // non-nil -> dereferenced
	Iface     any      // nil -> omitted
	Pair      [2]float64
	Short     []float64
	Long      []int
	Labels    []string
	Nested    struct {
		X int
		Y *int
	}
	Table  insyra.IDataTable
	List   insyra.IDataList
	hidden int
}

func TestFormatResult_Nil(t *testing.T) {
	got := formatResult("t", (*textSample)(nil))
	if got != "<nil>" {
		t.Errorf("formatResult with nil pointer = %q, want %q", got, "<nil>")
	}
	got = formatResult("t", nil)
	if got != "<nil>" {
		t.Errorf("formatResult with nil = %q, want %q", got, "<nil>")
	}
}

func TestFormatResult_Struct(t *testing.T) {
	bVal := 2.5
	setVal := 42
	s := &textSample{
		textInner: textInner{
			A: 1.0,
			B: &bVal,
		},
		Big:    1500000,
		Tiny:   1e-7,
		NaN:    math.NaN(),
		Count:  7,
		Method: "test",
		Set:    &setVal,
		Pair:   [2]float64{1.5, 2},
		Short:  []float64{1, 2, 3},
		Long:   make([]int, 100),
		Labels: []string{"a b", "c"},
		Nested: struct {
			X int
			Y *int
		}{X: 3, Y: nil},
		hidden: 9,
	}

	for i := range s.Long {
		s.Long[i] = i + 1
	}

	got := formatResult("Sample", s)

	// Check no ANSI codes
	if strings.Contains(got, "\x1b") {
		t.Errorf("output contains ANSI codes: %q", got)
	}

	// Check first line is title
	lines := strings.Split(got, "\n")
	if len(lines) == 0 || lines[0] != "Sample" {
		t.Errorf("first line = %q, want %q", lines[0], "Sample")
	}

	// Check field order and values by looking for expected patterns
	expectedPatterns := []string{
		"  A: 1",
		"  B: 2.5",
		"  Big: 1500000",
		"  Tiny: 1e-07",
		"  NaN: NaN",
		"  Count: 7",
		"  Method: test",
		"  Set: 42",
		"  Pair: [1.5 2]",
		"  Short: [1 2 3]",
		"  Long: [1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 ... 96 97 98 99 100] (100 values)",
		"  Labels: [\"a b\" \"c\"]",
		"  Nested: {X: 3}",
	}

	for _, pat := range expectedPatterns {
		if !slices.Contains(strings.Split(got, "\n"), pat) {
			t.Errorf("missing expected line %q in output:\n%s", pat, got)
		}
	}

	// Ensure hidden, Ptr (nil), Iface (nil) are absent
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "  hidden:") {
			t.Errorf("unexported field 'hidden' should not appear")
		}
	}
	if slices.Contains(strings.Split(got, "\n"), "  Ptr:") {
		t.Errorf("nil pointer field 'Ptr' should be omitted")
	}
	if slices.Contains(strings.Split(got, "\n"), "  Iface:") {
		t.Errorf("nil interface field 'Iface' should be omitted")
	}
}

func TestFormatResult_Table(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(
		insyra.NewDataList(1.0, 3.0).SetName("x"),
		insyra.NewDataList(2.5, 4.0).SetName("y"),
	)
	dt.SetRowNames([]string{"r1", "r2"})

	s := &textSample{
		Table: dt,
	}

	got := formatResult("WithTable", s)

	// Should have table header and grid
	if !slices.Contains(strings.Split(got, "\n"), "  Table:") {
		t.Errorf("missing 'Table:' line in output:\n%s", got)
	}

	// Check grid structure - should have header row with column names
	// and row names column. Tabwriter with padding 2 produces 2 spaces between columns.
	expectedGridLines := []string{
		"        x  y",
		"    r1  1  2.5",
		"    r2  3  4",
	}

	for _, line := range expectedGridLines {
		if !slices.Contains(strings.Split(got, "\n"), line) {
			t.Errorf("missing expected grid line %q in output:\n%s", line, got)
		}
	}
}

func TestFormatResult_Table_NoRowNames(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(
		insyra.NewDataList(1.0, 3.0).SetName("x"),
		insyra.NewDataList(2.5, 4.0).SetName("y"),
	)

	s := &textSample{
		Table: dt,
	}

	got := formatResult("NoRowNames", s)

	// Should have table header but no row-name column
	if !slices.Contains(strings.Split(got, "\n"), "  Table:") {
		t.Errorf("missing 'Table:' line in output:\n%s", got)
	}

	// Check grid without row names. Tabwriter with padding 2 produces 2 spaces between columns.
	expectedGridLines := []string{
		"    x  y",
		"    1  2.5",
		"    3  4",
	}

	for _, line := range expectedGridLines {
		if !slices.Contains(strings.Split(got, "\n"), line) {
			t.Errorf("missing expected grid line %q in output:\n%s", line, got)
		}
	}
}

func TestFormatResult_Table_Empty(t *testing.T) {
	dt := insyra.NewDataTable()

	s := &textSample{
		Table: dt,
	}

	got := formatResult("EmptyTable", s)

	if !slices.Contains(strings.Split(got, "\n"), "  Table: (empty table)") {
		t.Errorf("empty table should show '(empty table)':\n%s", got)
	}
}

func TestFormatResult_Table_ManyRows(t *testing.T) {
	dt := insyra.NewDataTable()
	col := insyra.NewDataList()
	for i := 1; i <= 100; i++ {
		col.Append(float64(i))
	}
	dt.AppendCols(col.SetName("val"))

	s := &textSample{
		Table: dt,
	}

	got := formatResult("ManyRows", s)

	// Should have first 20 rows, "...", last 5 rows, and "(100 rows)"
	lines := strings.Split(got, "\n")
	dataRowCount := 0
	for _, line := range lines {
		// Data rows start with 4 spaces followed by a digit (the row value)
		if len(line) >= 5 && line[:4] == "    " && line[4] >= '0' && line[4] <= '9' {
			dataRowCount++
		}
	}
	if dataRowCount != 25 {
		t.Errorf("expected 25 data rows (20 head + 5 tail), got %d", dataRowCount)
	}
	if !slices.Contains(strings.Split(got, "\n"), "    ...") {
		t.Errorf("missing '...' line for truncated table")
	}
	if !slices.Contains(strings.Split(got, "\n"), "    (100 rows)") {
		t.Errorf("missing '(100 rows)' line")
	}
}

func TestFormatResult_List(t *testing.T) {
	dl := insyra.NewDataList("u", 2)

	s := &textSample{
		List: dl,
	}

	got := formatResult("WithList", s)

	if !slices.Contains(strings.Split(got, "\n"), `  List: ["u" 2]`) {
		t.Errorf("list output mismatch:\n%s", got)
	}
}

func TestFormatResult_ShortListWhole(t *testing.T) {
	// 60 elements should print whole with no count
	dl := insyra.NewDataList()
	for i := 1; i <= 60; i++ {
		dl.Append(float64(i))
	}

	s := &textSample{
		List: dl,
	}

	got := formatResult("ShortList", s)

	if !slices.Contains(strings.Split(got, "\n"), "  List: [1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60]") {
		t.Errorf("60-element list should print whole:\n%s", got)
	}
	if slices.Contains(strings.Split(got, "\n"), " (60 values)") {
		t.Errorf("60-element list should not show count")
	}
}

func TestFormatResult_NestedPointer(t *testing.T) {
	yVal := 5
	s := &textSample{
		Nested: struct {
			X int
			Y *int
		}{X: 3, Y: &yVal},
	}

	got := formatResult("NestedPtr", s)

	if !slices.Contains(strings.Split(got, "\n"), "  Nested: {X: 3, Y: 5}") {
		t.Errorf("nested struct with non-nil pointer:\n%s", got)
	}
}

func TestFormatResult_NilPtrInSlice(t *testing.T) {
	one := 1.0
	type withPtrSlice struct {
		Ptrs []*float64
	}
	s := &withPtrSlice{
		Ptrs: []*float64{nil, &one},
	}
	got := formatResult("Test", s)
	want := "  Ptrs: [<nil> 1]"
	if !slices.Contains(strings.Split(got, "\n"), want) {
		t.Errorf("missing expected line %q in output:\n%s", want, got)
	}
}

func TestFormatResult_WholeOutput(t *testing.T) {
	bVal := 2.5
	setVal := 42
	dt := insyra.NewDataTable()
	dt.AppendCols(
		insyra.NewDataList(1.0, 3.0).SetName("x"),
		insyra.NewDataList(2.5, 4.0).SetName("y"),
	)
	dt.SetRowNames([]string{"r1", "r2"})
	s := &textSample{
		textInner: textInner{
			A: 1.0,
			B: &bVal,
		},
		Big:    1500000,
		Tiny:   1e-7,
		NaN:    math.NaN(),
		Count:  7,
		Method: "test",
		Set:    &setVal,
		Pair:   [2]float64{1.5, 2},
		Short:  []float64{1, 2, 3},
		Long:   make([]int, 100),
		Labels: []string{"a b", "c"},
		Nested: struct {
			X int
			Y *int
		}{X: 3, Y: nil},
		Table:  dt,
		List:   insyra.NewDataList("u", 2),
		hidden: 9,
	}
	for i := range s.Long {
		s.Long[i] = i + 1
	}

	got := formatResult("Sample", s)

	want := `Sample
  A: 1
  B: 2.5
  Big: 1500000
  Tiny: 1e-07
  NaN: NaN
  Count: 7
  Method: test
  Set: 42
  Pair: [1.5 2]
  Short: [1 2 3]
  Long: [1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 ... 96 97 98 99 100] (100 values)
  Labels: ["a b" "c"]
  Nested: {X: 3}
  Table:
        x  y
    r1  1  2.5
    r2  3  4
  List: ["u" 2]`

	if got != want {
		t.Errorf("formatResult output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
