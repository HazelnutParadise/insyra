package insyra

import (
	"fmt"
	"math"
	"slices"
	"testing"
)

// A CSV load stores integers as int64; Go code writes them as int. Every value
// lookup compared the two with ==, and inside an any, int64(2) == int(2) is
// false, so Count(2) on a CSV column returned 0 and Replace(2, 0) changed
// nothing. These tests hold int64 data and search with Go int literals, the
// shape of that failure.

func csvInts() *DataList { return NewDataList(int64(1), int64(2), int64(3), int64(2)) }

func csvTable() *DataTable {
	return NewDataTable(
		NewDataList("a", "b", "a", "b").SetName("g"),
		NewDataList(int64(1), int64(2), int64(3), int64(2)).SetName("v"),
	)
}

func TestDataListValueLookupsMatchIntegersAcrossTypes(t *testing.T) {
	if n := csvInts().Count(2); n != 2 {
		t.Errorf("Count(2) = %d, want 2", n)
	}
	if i := csvInts().FindFirst(2); i != 1 {
		t.Errorf("FindFirst(2) = %v, want 1", i)
	}
	if i := csvInts().FindLast(2); i != 3 {
		t.Errorf("FindLast(2) = %v, want 3", i)
	}
	if got := csvInts().FindAll(2); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("FindAll(2) = %v, want [1 3]", got)
	}
	for _, c := range []struct {
		name string
		got  *DataList
		want string
	}{
		{"ReplaceFirst(2, 9)", csvInts().ReplaceFirst(2, 9), "[1 9 3 2]"},
		{"ReplaceLast(2, 9)", csvInts().ReplaceLast(2, 9), "[1 2 3 9]"},
		{"ReplaceAll(2, 9)", csvInts().ReplaceAll(2, 9), "[1 9 3 9]"},
		{"DropAll(2)", csvInts().DropAll(2), "[1 3]"},
	} {
		if s := fmt.Sprint(c.got.Data()); s != c.want {
			t.Errorf("%s left %s, want %s", c.name, s, c.want)
		}
	}
}

func TestIntegerMatchingRules(t *testing.T) {
	for _, c := range []struct {
		name  string
		data  []any
		value any
		want  int
	}{
		{"uint8 cell, int literal", []any{uint8(2)}, 2, 1},
		{"int32 cell, int64 value", []any{int32(-5)}, int64(-5), 1},
		{"uint64 above int64's range", []any{uint64(math.MaxUint64)}, uint64(math.MaxUint64), 1},
		{"a negative value never equals an unsigned one", []any{int64(-1)}, uint64(math.MaxUint64), 0},
		{"a float cell is not an integer", []any{2.0}, 2, 0},
		{"an integer cell is not a float", []any{int64(2)}, 2.0, 0},
		{"a string is not a number", []any{"2"}, 2, 0},
		{"NaN still matches NaN", []any{math.NaN()}, math.NaN(), 1},
	} {
		if got := NewDataList(c.data...).Count(c.value); got != c.want {
			t.Errorf("%s: Count = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestDataTableValueLookupsMatchIntegersAcrossTypes(t *testing.T) {
	if n := csvTable().Count(2); n != 2 {
		t.Errorf("Count(2) = %d, want 2", n)
	}
	if got := csvTable().FindRowsIfContains(2); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("FindRowsIfContains(2) = %v, want [1 3]", got)
	}
	if got := csvTable().FindRowsIfContainsAll("b", 2); !slices.Equal(got, []int{1, 3}) {
		t.Errorf(`FindRowsIfContainsAll("b", 2) = %v, want [1 3]`, got)
	}
	if got := csvTable().FindColsIfContains(2); !slices.Equal(got, []string{"B"}) {
		t.Errorf("FindColsIfContains(2) = %v, want [B]", got)
	}
	if got := csvTable().FindColsIfContainsAll(1, 3); !slices.Equal(got, []string{"B"}) {
		t.Errorf("FindColsIfContainsAll(1, 3) = %v, want [B]", got)
	}
	if n := csvTable().Replace(2, 0).Count(int64(2)); n != 0 {
		t.Errorf("Replace(2, 0) left %d cells holding 2", n)
	}
	if v := csvTable().ReplaceInRow(1, 2, 0).GetElement(1, "B"); fmt.Sprint(v) != "0" {
		t.Errorf("ReplaceInRow(1, 2, 0) left %v in row 1", v)
	}
	if n := csvTable().ReplaceInCol("B", 2, 0).Count(int64(2)); n != 0 {
		t.Errorf(`ReplaceInCol("B", 2, 0) left %d cells holding 2`, n)
	}
	if n := csvTable().DropRowsContain(2).NumRows(); n != 2 {
		t.Errorf("DropRowsContain(2) kept %d rows, want 2", n)
	}
	if n := csvTable().DropColsContain(2).NumCols(); n != 1 {
		t.Errorf("DropColsContain(2) kept %d columns, want 1", n)
	}
}

func TestEncoderCategoriesMatchIntegersAcrossTypes(t *testing.T) {
	src := NewDataTable(NewDataList(int64(1), int64(2), int64(3), int64(2)).SetName("v"))
	out, _, err := src.OrdinalEncode(OrdinalEncodeOptions{Column: "v", Order: []any{1, 2, 3}, Unknown: UnknownError})
	if err != nil {
		t.Fatalf("OrdinalEncode with Order written as int literals: %v", err)
	}
	if got := fmt.Sprint(out.GetColByNumber(0).Data()); got != "[0 1 2 1]" {
		t.Errorf("OrdinalEncode gave %s, want [0 1 2 1]", got)
	}

	fitted := NewDataTable(NewDataList(1, 2, 3).SetName("v"))
	_, enc, err := fitted.LabelEncode(LabelEncodeOptions{Column: "v", Unknown: UnknownError})
	if err != nil {
		t.Fatalf("LabelEncode: %v", err)
	}
	moved, err := enc.Transform(NewDataTable(NewDataList(int64(3), int64(1)).SetName("v")))
	if err != nil {
		t.Fatalf("an encoder fitted on int data rejected int64 data: %v", err)
	}
	if got := fmt.Sprint(moved.GetColByNumber(0).Data()); got != "[2 0]" {
		t.Errorf("Transform gave %s, want [2 0]", got)
	}

	// int(1) and int64(1) in one column are one category, not two that
	// render as the same column name.
	mixed := NewDataTable(NewDataList(1, int64(1), 2).SetName("v"))
	_, oh, err := mixed.OneHotEncode(OneHotOptions{Columns: []string{"v"}})
	if err != nil {
		t.Fatalf("OneHotEncode on a column mixing int and int64: %v", err)
	}
	if cats := oh.Categories()["v"]; len(cats) != 2 {
		t.Errorf("categories %v, want two (1 and 2)", cats)
	}
}

// IsEqualTo asks whether two lists hold identical data, not whether a value
// occurs, so it keeps telling int(1) from int64(1).
func TestIsEqualToStaysTypeStrict(t *testing.T) {
	if NewDataList(1).IsEqualTo(NewDataList(int64(1))) {
		t.Error("IsEqualTo treated int(1) and int64(1) as identical data")
	}
}
