package insyra

import (
	"bytes"
	"strings"
	"testing"
)

func showHeadTailTable() *DataTable {
	return NewDataTable(
		NewDataList(10.0, nil, 30.0).SetName("price"),
		NewDataList("Taipei", nil, "Taichung").SetName("city"),
		NewDataList(nil, nil, nil).SetName("blank"),
	)
}

func TestShowHeadAndTail(t *testing.T) {
	colored := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(colored)

	dl := NewDataList("first", "middle", "last")
	var head, tail, want bytes.Buffer
	dl.ShowHeadTo(&head, 2)
	dl.ShowRangeTo(&want, 2)
	if head.String() != want.String() {
		t.Errorf("ShowHead(2) differs from ShowRange(2):\n%s\n%s", head.String(), want.String())
	}
	want.Reset()
	dl.ShowTailTo(&tail, 1)
	dl.ShowRangeTo(&want, -1)
	if tail.String() != want.String() {
		t.Errorf("ShowTail(1) differs from ShowRange(-1):\n%s\n%s", tail.String(), want.String())
	}

	dt := showHeadTailTable()
	var th, tw bytes.Buffer
	dt.ShowHeadTo(&th, 1)
	dt.ShowRangeTo(&tw, 1)
	if th.String() != tw.String() {
		t.Errorf("DataTable.ShowHead(1) differs from ShowRange(1)")
	}
	th.Reset()
	tw.Reset()
	dt.ShowTailTo(&th, 1)
	dt.ShowRangeTo(&tw, -1)
	if th.String() != tw.String() {
		t.Errorf("DataTable.ShowTail(1) differs from ShowRange(-1)")
	}

	var bad bytes.Buffer
	dl.ShowHeadTo(&bad, 0)
	dt.ShowTailTo(&bad, -2)
	if strings.Count(bad.String(), "ERROR") != 2 {
		t.Errorf("a count that is not positive was accepted:\n%s", bad.String())
	}
}
