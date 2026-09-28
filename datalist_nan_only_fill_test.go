package insyra

import (
	"math"
	"reflect"
	"testing"
)

// The replacement FillNaNWithMean's deprecation note points to fills the same
// cells with the same mean, and leaves the other numbers as they were.
func TestReplaceNaNsWithObservedMeanMatchesFillNaNWithMean(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	a := NewDataList(1, nil, math.NaN(), 3, math.NaN())
	a.FillNaNWithMean()
	b := NewDataList(1, nil, math.NaN(), 3, math.NaN())
	b.ReplaceNaNsWith(b.Clone().ClearNilsAndNaNs().Mean())

	if err := a.Err(); err != nil {
		t.Fatalf("FillNaNWithMean: %v", err)
	}
	if err := b.Err(); err != nil {
		t.Fatalf("ReplaceNaNsWith: %v", err)
	}

	got, want := b.Data(), a.Data()
	if len(got) != len(want) {
		t.Fatalf("got %d cells, want %d", len(got), len(want))
	}
	for i := range want {
		if want[i] == nil || got[i] == nil {
			if want[i] != nil || got[i] != nil {
				t.Errorf("cell %d: got %v, FillNaNWithMean gave %v", i, got[i], want[i])
			}
			continue
		}
		gf, gok := ToFloat64Safe(got[i])
		wf, wok := ToFloat64Safe(want[i])
		if !gok || !wok || gf != wf {
			t.Errorf("cell %d: got %v, FillNaNWithMean gave %v", i, got[i], want[i])
		}
	}

	if !reflect.DeepEqual(got, []any{1, nil, 2.0, 3, 2.0}) {
		t.Errorf("got %#v, want the integers kept as int and NaN replaced by 2.0", got)
	}
}
