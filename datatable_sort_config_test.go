package insyra

import (
	"bytes"
	"log"
	"reflect"
	"strings"
	"testing"
)

// #233. A sort config selects its column with one of three fields, and Go
// cannot tell a config that names no column from one that says
// ColumnNumber: 0, because they are the same value. Both sort by the first
// column, as documented. These pin what SortBy does with each shape of config.

// Original order: id 2, 3, 1. Sorting by score descending puts id 1 first;
// sorting by id ascending puts id 1 first too, so the tests read the whole
// id column rather than one cell.
func sortConfigTable() *DataTable {
	return NewDataTable(
		NewDataList(2, 3, 1).SetName("id"),
		NewDataList(20, 10, 30).SetName("score"),
		NewDataList("b", "c", "a").SetName("name"),
	)
}

func ids(dt *DataTable) []any { return dt.GetColByNumber(0).Data() }

var unsorted = []any{2, 3, 1}

// captureSortLog pins the log level at Warning, so a warning reaches the log
// and a no-warning check cannot pass vacuously, and returns the log output.
func captureSortLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	level := Config.GetLogLevel()
	Config.SetLogLevel(LogLevelWarning)
	t.Cleanup(func() { Config.SetLogLevel(level) })
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func TestSortByFirstColumnOnlyWhenNothingIsNamed(t *testing.T) {
	buf := captureSortLog(t)

	for _, c := range []struct {
		label string
		cfg   DataTableSortConfig
		want  []any
	}{
		{"empty", DataTableSortConfig{}, []any{1, 2, 3}},
		{"only Descending", DataTableSortConfig{Descending: true}, []any{3, 2, 1}},
		{"ColumnNumber 0", DataTableSortConfig{ColumnNumber: 0}, []any{1, 2, 3}},
		// ColumnNumber is left at zero here too. It is the lowest precedence and
		// counts only when non-zero, so the name is used, not the first column.
		// Score ascending gives ids 3, 2, 1; the first column would give 1, 2, 3.
		{"ColumnName with ColumnNumber at zero", DataTableSortConfig{ColumnName: "score"}, []any{3, 2, 1}},
	} {
		buf.Reset()
		dt := sortConfigTable()
		dt.SortBy(c.cfg)
		if e := dt.Err(); e != nil {
			t.Errorf("%s: unexpected error: %v", c.label, e)
		}
		if got := ids(dt); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: sorted as %v, want %v", c.label, got, c.want)
		}
		if strings.Contains(buf.String(), "SortBy") {
			t.Errorf("%s: sorting by the first column logged a warning: %q", c.label, buf.String())
		}
	}
}

func TestSortByFirstColumnByIndex(t *testing.T) {
	dt := sortConfigTable()
	dt.SortBy(DataTableSortConfig{ColumnIndex: "A"})
	if e := dt.Err(); e != nil {
		t.Fatalf("unexpected error: %v", e)
	}
	if got := ids(dt); !reflect.DeepEqual(got, []any{1, 2, 3}) {
		t.Errorf("sorted as %v, want [1 2 3]", got)
	}
}

func TestSortByRefusesAMissingColumnNamingSortBy(t *testing.T) {
	for _, c := range []struct {
		label string
		cfg   DataTableSortConfig
	}{
		{"ColumnIndex that matches nothing", DataTableSortConfig{ColumnIndex: "Z"}},
		{"ColumnName that is not there", DataTableSortConfig{ColumnName: "scroe"}},
		{"ColumnNumber out of range", DataTableSortConfig{ColumnNumber: 99}},
		{"negative ColumnNumber", DataTableSortConfig{ColumnNumber: -1}},
	} {
		dt := sortConfigTable()
		dt.SortBy(c.cfg)
		e := dt.Err()
		if e == nil {
			t.Errorf("%s: no error", c.label)
			continue
		}
		// "GetCol" also covers GetColByName and GetColByNumber.
		msg := e.Error()
		if !strings.Contains(msg, "SortBy") || strings.Contains(msg, "GetCol") {
			t.Errorf("%s: the error should name SortBy and no internal lookup: %v", c.label, e)
		}
		if got := ids(dt); !reflect.DeepEqual(got, unsorted) {
			t.Errorf("%s: the table moved: %v", c.label, got)
		}
	}
}

func TestSortByIsAllOrNothing(t *testing.T) {
	dt := sortConfigTable()
	dt.SortBy(
		DataTableSortConfig{ColumnName: "score", Descending: true},
		DataTableSortConfig{ColumnName: "nope"},
	)
	if dt.Err() == nil {
		t.Error("no error for a bad second level")
	}
	if got := ids(dt); !reflect.DeepEqual(got, unsorted) {
		t.Errorf("the good level was applied although another was bad: %v", got)
	}
}

// Several selectors in one config are not an error: the precedence rule —
// index, then name, then number, the same one mkt's configs document — picks
// the column, and a warning says which fields were ignored. The warning is
// logged only, so Err() stays nil on a sort that ran.
func TestSortByWarnsAndFollowsPrecedence(t *testing.T) {
	buf := captureSortLog(t)

	for _, c := range []struct {
		label string
		cfg   DataTableSortConfig
		want  []any
	}{
		// B is score; descending puts id 1 (score 30) first. The name "id"
		// would have sorted by id instead.
		{"index and name", DataTableSortConfig{ColumnIndex: "B", ColumnName: "id", Descending: true}, []any{1, 2, 3}},
		// Name wins over a non-zero number. Score ascending gives ids 3, 2, 1;
		// column 2 is name, which would have given 1, 2, 3.
		{"name and number", DataTableSortConfig{ColumnName: "score", ColumnNumber: 2}, []any{3, 2, 1}},
	} {
		buf.Reset()
		dt := sortConfigTable()
		dt.SortBy(c.cfg)
		if e := dt.Err(); e != nil {
			t.Errorf("%s: several selectors should warn, not fail: %v", c.label, e)
		}
		if got := ids(dt); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: sorted as %v, want %v", c.label, got, c.want)
		}
		if !strings.Contains(buf.String(), "SortBy") {
			t.Errorf("%s: no warning was logged: %q", c.label, buf.String())
		}
	}
}
