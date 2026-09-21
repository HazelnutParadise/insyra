package insyra

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// Every column parameter in the library takes the same selector: a string is
// an Excel-style index, Name(...) is a column name, an int is a position.
// These tests pin the three forms, what a wrong one says, and that a column
// named like an index cannot quietly steal it.

func selectorTable() *DataTable {
	price := NewDataList(1, 2, 3)
	price.SetName("price")
	qty := NewDataList(10, 20, 30)
	qty.SetName("qty")
	dt := NewDataTable()
	dt.AppendCols(price, qty)
	return dt
}

func firstCell(t *testing.T, dl *DataList) any {
	t.Helper()
	if dl == nil {
		t.Fatal("column is nil")
	}
	data := dl.Data()
	if len(data) == 0 {
		t.Fatal("column is empty")
	}
	return data[0]
}

func TestGetColTakesTheThreeSelectorForms(t *testing.T) {
	for _, c := range []struct {
		label string
		sel   any
		want  any
	}{
		{"excel index", "A", 1},
		{"lowercase excel index", "a", 1},
		{"name", Name("price"), 1},
		{"position", 0, 1},
		{"second by index", "B", 10},
		{"second by name", Name("qty"), 10},
		{"second by position", 1, 10},
		{"position from the end", -1, 10},
	} {
		dt := selectorTable()
		got := dt.GetCol(c.sel)
		if err := dt.Err(); err != nil {
			t.Errorf("%s: %v", c.label, err)
			continue
		}
		if value := firstCell(t, got); value != c.want {
			t.Errorf("%s: first value = %v, want %v", c.label, value, c.want)
		}
	}
}

func TestGetColRefusesANameWrittenAsABareString(t *testing.T) {
	dt := selectorTable()
	dt.GetCol("price")
	err := dt.Err()
	if err == nil {
		t.Fatal("a name written as a bare string must not resolve")
	}
	if !strings.Contains(err.Error(), `Name("price")`) {
		t.Errorf("the failure does not say how to write the name: %v", err)
	}
}

func TestGetColUsesTheIndexWhenAColumnIsNamedLikeOne(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelWarning)
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	first := NewDataList(1, 2, 3)
	first.SetName("B")
	second := NewDataList(10, 20, 30)
	second.SetName("other")
	dt := NewDataTable()
	dt.AppendCols(first, second)

	got := dt.GetCol("B")
	if err := dt.Err(); err != nil {
		t.Fatalf("an in-range index must still resolve: %v", err)
	}
	if value := firstCell(t, got); value != 10 {
		t.Errorf("first value = %v, want 10: the index wins, not the name", value)
	}
	if !strings.Contains(buf.String(), "B") {
		t.Errorf("a column named like the index given must be reported: %q", buf.String())
	}
}

func TestColumnSelectorOfAnotherTypeIsRefused(t *testing.T) {
	dt := selectorTable()
	dt.GetCol(3.5)
	err := dt.Err()
	if err == nil {
		t.Fatal("a float is not a column selector")
	}
	if !strings.Contains(err.Error(), "float64") {
		t.Errorf("the failure does not name the type it was given: %v", err)
	}
}

func TestExplicitColumnMethodsAgreeWithTheSelector(t *testing.T) {
	dt := selectorTable()
	byIndex := firstCell(t, dt.GetColByIndex("B"))
	byName := firstCell(t, dt.GetColByName("qty"))
	byNumber := firstCell(t, dt.GetColByNumber(1))
	generic := firstCell(t, dt.GetCol(Name("qty")))
	if err := dt.Err(); err != nil {
		t.Fatalf("unexpected failure: %v", err)
	}
	if byIndex != byName || byName != byNumber || byNumber != generic {
		t.Errorf("the four spellings disagree: %v %v %v %v", byIndex, byName, byNumber, generic)
	}
}

func TestGetColByIndexDoesNotTryTheName(t *testing.T) {
	dt := selectorTable()
	dt.GetColByIndex("price")
	err := dt.Err()
	if err == nil {
		t.Fatal("GetColByIndex must read its argument as an index only")
	}
	if !strings.Contains(err.Error(), "PRICE") {
		t.Errorf("the failure does not name the index it decoded: %v", err)
	}
}
