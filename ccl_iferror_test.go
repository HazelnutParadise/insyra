package insyra

import (
	"errors"
	"testing"

	engineccl "github.com/HazelnutParadise/insyra/engine/ccl"
)

func TestIfErrorOnATable(t *testing.T) {
	dt := NewDataTable(NewDataList(6, 8).SetName("a"), NewDataList(2, 0).SetName("b"))
	dt.AddColUsingCCL("r", "IFERROR(A / B, -1)")
	if err := dt.Err(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("r").Data()
	if got[0] != 3.0 && got[0] != int64(3) {
		t.Errorf("row 0 = %#v", got[0])
	}
	if got[1] != int64(-1) && got[1] != -1.0 {
		t.Errorf("row 1 = %#v", got[1])
	}

	missing := NewDataTable(NewDataList(1).SetName("a"))
	missing.AddColUsingCCL("r", "IFERROR(['nope'] + 1, 0)")
	if missing.Err() == nil {
		t.Error("a column that does not exist should fail, not read as the fallback")
	}
	beyond := NewDataTable(NewDataList(1).SetName("a"))
	beyond.AddColUsingCCL("r", "IFERROR(E + 1, 0)")
	if beyond.Err() == nil {
		t.Error("a column past the last should fail, not read as the fallback")
	}
}

// An aggregate that fails under IFERROR is computed once, not once per row.
func TestIfErrorDoesNotRepeatAFailedAggregate(t *testing.T) {
	calls := 0
	engineccl.RegisterAggregateFunction("ZZFAILSONCE", func(args ...[]any) (any, error) {
		calls++
		return nil, errors.New("no")
	})
	col := NewDataList()
	for i := range 200 {
		col.Append(i)
	}
	dt := NewDataTable(col)
	dt.AddColUsingCCL("r", "IFERROR(A / ZZFAILSONCE(A), 0)")
	if err := dt.Err(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("the aggregate ran %d times", calls)
	}
}
