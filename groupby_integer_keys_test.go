package insyra

import (
	"reflect"
	"testing"
)

// GroupBy, NUnique and the other grouped operations key an integer by its
// value whatever its Go width, uintptr included, the way Count and Counter
// match integers. A float stays apart from an integer of the same value, and
// floats of different widths merge by value.
func TestGroupByKeysIntegersByValue(t *testing.T) {
	keys := []any{
		1, int8(1), int16(1), int32(1), int64(1),
		uint(1), uint8(1), uint16(1), uint32(1), uint64(1), uintptr(1),
		1.0, float32(1),
		"1",
	}
	values := make([]any, len(keys))
	for i := range values {
		values[i] = i
	}
	dt := NewDataTable(NewDataList(keys...).SetName("k"), NewDataList(values...).SetName("v"))
	counts := dt.GroupBy(Name("k")).Count()
	if e := dt.Err(); e != nil {
		t.Fatalf("GroupBy recorded %v", e)
	}
	got := counts.GetColByNumber(counts.NumCols() - 1).Data()
	if want := []any{11, 2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("group sizes = %v over keys %v, want %v (integers, floats, text)",
			got, counts.GetColByNumber(0).Data(), want)
	}
}

func TestNUniqueCountsIntegersByValue(t *testing.T) {
	dt := NewDataTable(
		NewDataList("g", "g", "g", "g", "g").SetName("group"),
		NewDataList(1, int64(1), uintptr(1), 1.0, "1").SetName("v"),
	)
	out := dt.GroupBy(Name("group")).Aggregate(AggregateConfig{SourceCol: Name("v"), Op: OpNUnique})
	if got := out.GetColByNumber(1).Data(); !reflect.DeepEqual(got, []any{3}) {
		t.Fatalf("NUnique = %v, want [3]: one integer, one float, one string", got)
	}
}
