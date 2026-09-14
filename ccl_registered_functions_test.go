package insyra

import (
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
