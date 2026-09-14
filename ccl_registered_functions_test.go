package insyra

import (
	"sort"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/ccl"
)

// A registered aggregate or sequence function that panics ends the CCL call
// the way it did on v0.3.2: the method's own recover records the panic and the
// method returns nil. Turning the panic into an evaluation error inside the
// evaluator made the method return the receiver instead.
func TestAPanickingRegisteredCCLFunctionReturnsNil(t *testing.T) {
	quietLogs(t)
	ccl.RegisterAggregateFunction("ZZPANICAGG_REVIEW", func(args ...[]any) (any, error) { panic("boom") })
	ccl.RegisterSequenceFunction("ZZPANICSEQ_REVIEW", func(args ...[]any) ([]any, error) { panic("boom") })

	for _, expr := range []string{"ZZPANICAGG_REVIEW(A)", "ZZPANICSEQ_REVIEW(A)"} {
		dt := NewDataTable(NewDataList(1, 2, 3))
		if got := dt.AddColUsingCCL("R", expr); got != nil {
			t.Errorf("AddColUsingCCL(%q) returned the table, want nil", expr)
		}
		if dt.Err() == nil {
			t.Errorf("AddColUsingCCL(%q) recorded nothing", expr)
		}
		if dt.NumCols() != 1 {
			t.Errorf("AddColUsingCCL(%q) added a column", expr)
		}

		edit := NewDataTable(NewDataList(1, 2, 3))
		if got := edit.EditColByIndexUsingCCL("A", expr); got != nil {
			t.Errorf("EditColByIndexUsingCCL(%q) returned the table, want nil", expr)
		}
	}
}

// An aggregate receives its own copy of the column, as on v0.3.2. A registered
// aggregate that sorts its input in place must not change what the rest of the
// expression reads from the same column, nor the order a row reads it in.
func TestARegisteredAggregateCannotReorderTheColumnItReads(t *testing.T) {
	quietLogs(t)
	ccl.RegisterAggregateFunction("ZZSORTFIRST_REVIEW", func(args ...[]any) (any, error) {
		col := args[0]
		sort.Slice(col, func(i, j int) bool {
			a, _ := ToFloat64Safe(col[i])
			b, _ := ToFloat64Safe(col[j])
			return a < b
		})
		return col[0], nil
	})

	dt := NewDataTable(NewDataList(3, 1, 2))
	dt.AddColUsingCCL("R", "ZZSORTFIRST_REVIEW(A) + A.0")
	dt.AddColUsingCCL("S", "A.# + 0*ZZSORTFIRST_REVIEW(A)")
	if e := dt.Err(); e != nil {
		t.Fatalf("Err() = %v", e)
	}
	for row, wantS := range []float64{3, 1, 2} {
		if got, _ := ToFloat64Safe(dt.GetElement(row, "B")); got != 4 {
			t.Errorf("row %d: ZZSORTFIRST(A) + A.0 = %v, want 4 (1 + the unsorted A.0 = 3)", row, dt.GetElement(row, "B"))
		}
		if got, _ := ToFloat64Safe(dt.GetElement(row, "C")); got != wantS {
			t.Errorf("row %d: A.# + 0*ZZSORTFIRST(A) = %v, want %v", row, dt.GetElement(row, "C"), wantS)
		}
		if got, _ := ToFloat64Safe(dt.GetElement(row, "A")); got != wantS {
			t.Errorf("row %d: column A changed to %v", row, dt.GetElement(row, "A"))
		}
	}
}
