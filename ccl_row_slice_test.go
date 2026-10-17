package insyra

import (
	"reflect"
	"testing"
)

func TestCCLRowShapedValues(t *testing.T) {
	dt := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
	dt.AddColUsingCCL("r", "A:B")
	if err := dt.PopErr(); err != nil {
		t.Fatalf("A:B: %v", err)
	}
	got := dt.GetColByName("r").Data()
	want := [][]any{{10, 1}, {20, 0}, {30, 3}}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Fatalf("A:B = %v, want %v", got, want)
		}
	}

	dt.AddColUsingCCL("prev", "LAG(@, 1)")
	if err := dt.PopErr(); err != nil {
		t.Fatalf("LAG(@, 1): %v", err)
	}
	prev := dt.GetColByName("prev").Data()
	if prev[0] != nil {
		t.Errorf("LAG(@, 1): first row = %v, want nil", prev[0])
	}
	if len(prev) != 3 {
		t.Fatalf("LAG(@, 1) has %d rows, want 3", len(prev))
	}
}

func TestCCLRangeConsumersUnchanged(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4, 5, 6).SetName("a"),
		NewDataList(10, 20, 30, 40, 50, 60).SetName("b"),
		NewDataList(100, 200, 300, 400, 500, 600).SetName("c"),
	)
	for _, tc := range []struct {
		expr string
		want float64
	}{
		{"SUM(A:C)", 2331},
		{"SUM(A.(0:1))", 3},
	} {
		table := dt.Clone()
		table.AddColUsingCCL("r", tc.expr)
		if err := table.PopErr(); err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		for _, v := range table.GetColByName("r").Data() {
			if v != tc.want {
				t.Errorf("%s = %v, want %v in every row", tc.expr, v, tc.want)
				break
			}
		}
	}
}
