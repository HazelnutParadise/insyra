package insyra

import (
	"bytes"
	"log"
	"sort"
	"strings"
	"sync/atomic"
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

// An aggregate's argument is a whole column only when it is a sequence
// function call. A row read such as @.0 is one value, even on a table whose
// column count happens to equal its row count: ZZLEN(@.0) gave 3 on a 3x3
// table and 1 on a 2x3 one, where v0.3.2 gave 1 for both.
func TestAnAggregateArgumentIsAColumnOnlyWhenItIsASequenceCall(t *testing.T) {
	quietLogs(t)
	ccl.RegisterAggregateFunction("ZZLEN_REVIEW", func(args ...[]any) (any, error) {
		return float64(len(args[0])), nil
	})

	for cols, newCol := range map[int]string{3: "D", 2: "C"} {
		lists := make([]*DataList, cols)
		for c := range lists {
			lists[c] = NewDataList(1, 2, 3)
		}
		dt := NewDataTable(lists...)
		dt.AddColUsingCCL("R", "ZZLEN_REVIEW(@.0)")
		if e := dt.Err(); e != nil {
			t.Fatalf("%d columns: Err() = %v", cols, e)
		}
		for row := 0; row < 3; row++ {
			if got, _ := ToFloat64Safe(dt.GetElement(row, newCol)); got != 1 {
				t.Errorf("%d columns, row %d: ZZLEN(@.0) = %v, want 1", cols, row, dt.GetElement(row, newCol))
			}
		}
	}

	// A sequence function nested in an aggregate still hands over the column.
	dt := NewDataTable(NewDataList(1, 2, 3))
	dt.AddColUsingCCL("R", "ZZLEN_REVIEW(LAG(A, 1))")
	for row := 0; row < 3; row++ {
		if got, _ := ToFloat64Safe(dt.GetElement(row, "B")); got != 3 {
			t.Errorf("row %d: ZZLEN(LAG(A, 1)) = %v, want 3", row, dt.GetElement(row, "B"))
		}
	}
}

// On a table with no rows the row loop never runs, so v0.3.2 never called an
// aggregate inside a row-dependent expression there, and folding must not call
// it once up front. (A bare aggregate is row-independent and is evaluated once,
// as it always was.) Nor may an expression reach Go's standard logger, which
// bypasses insyra's own, by trying to restore row 0 of an empty table.
func TestAZeroRowTableCallsNoAggregateAndLogsNothing(t *testing.T) {
	quietLogs(t)
	var calls atomic.Int64
	ccl.RegisterAggregateFunction("ZZCOUNTCALLS_REVIEW", func(args ...[]any) (any, error) {
		calls.Add(1)
		return 0.0, nil
	})

	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	dt := NewDataTable()
	dt.AppendCols(NewDataList().SetName("A"))
	dt.AddColUsingCCL("R", "A + ZZCOUNTCALLS_REVIEW(A)")
	dt.AddColUsingCCL("S", "A + SUM(A * 2)")

	if n := calls.Load(); n != 0 {
		t.Errorf("an aggregate in a row-dependent expression was called %d times on a table with no rows, want 0", n)
	}
	if strings.Contains(buf.String(), "restore row index") {
		t.Errorf("the standard logger received: %q", buf.String())
	}
}
