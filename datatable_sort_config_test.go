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
// Col: 0, because they are the same value. The owner ruled that both
// sort by the first column, as documented. These pin what SortBy does with
// each shape of config.

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

func TestSortByFirstColumnOnlyWhenNothingIsNamed(t *testing.T) {
	// The documented default is not worth a warning. TestMain turns logging
	// down to Fatal, which would make the no-warning check pass vacuously.
	restoreConfig(t)
	Config.SetLogLevel(LogLevelWarning)
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	for _, c := range []struct {
		label string
		cfg   DataTableSortConfig
		want  []any
	}{
		{"empty", DataTableSortConfig{}, []any{1, 2, 3}},
		{"only Descending", DataTableSortConfig{Descending: true}, []any{3, 2, 1}},
		{"position 0", DataTableSortConfig{Col: 0}, []any{1, 2, 3}},
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
	dt.SortBy(DataTableSortConfig{Col: "A"})
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
		{"an index that matches nothing", DataTableSortConfig{Col: "Z"}},
		{"a name that is not there", DataTableSortConfig{Col: Name("scroe")}},
		{"a number out of range", DataTableSortConfig{Col: 99}},
		{"a number further back than the table", DataTableSortConfig{Col: -99}},
	} {
		dt := sortConfigTable()
		dt.SortBy(c.cfg)
		e := dt.Err()
		if e == nil {
			t.Errorf("%s: no error", c.label)
			continue
		}
		msg := e.Error()
		if !strings.Contains(msg, "SortBy") || strings.Contains(msg, "GetColByName") || strings.Contains(msg, "GetColByNumber") {
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
		DataTableSortConfig{Col: Name("score"), Descending: true},
		DataTableSortConfig{Col: Name("nope")},
	)
	if dt.Err() == nil {
		t.Error("no error for a bad second level")
	}
	if got := ids(dt); !reflect.DeepEqual(got, unsorted) {
		t.Errorf("the good level was applied although another was bad: %v", got)
	}
}

// One field takes every form the library's selector takes, and the spellings
// of the same column sort the same way.
func TestSortByTakesEverySelectorForm(t *testing.T) {
	for _, c := range []struct {
		label string
		cfg   DataTableSortConfig
		want  []any
	}{
		// B is score; ascending gives ids 3, 2, 1.
		{"excel index", DataTableSortConfig{Col: "B"}, []any{3, 2, 1}},
		{"name", DataTableSortConfig{Col: Name("score")}, []any{3, 2, 1}},
		{"position", DataTableSortConfig{Col: 1}, []any{3, 2, 1}},
		{"position from the end", DataTableSortConfig{Col: -2}, []any{3, 2, 1}},
	} {
		dt := sortConfigTable()
		dt.SortBy(c.cfg)
		if e := dt.Err(); e != nil {
			t.Errorf("%s: unexpected error: %v", c.label, e)
			continue
		}
		if got := ids(dt); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: sorted as %v, want %v", c.label, got, c.want)
		}
	}
}
