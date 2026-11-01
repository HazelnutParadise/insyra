package insyra

import "testing"

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
}
