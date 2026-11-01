package insyra

import (
	"bytes"
	"strings"
	"testing"
)

// IN-8: ShowRange(2, -1) is documented as "to the end" but excludes the last
// item. The documented way to reach the end is a nil end; this pins both.
func TestShowRangeNegativeEndMatchesDocs(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList(10, 20, 30, 40, 50)

	var excl bytes.Buffer
	dl.ShowRangeTo(&excl, 2, -1)
	if strings.Contains(excl.String(), "50") {
		t.Fatalf("ShowRange(2, -1) should exclude the last item (Python slice semantics):\n%s", excl.String())
	}
	if !strings.Contains(excl.String(), "40") {
		t.Fatalf("ShowRange(2, -1) should include index 3:\n%s", excl.String())
	}

	var toEnd bytes.Buffer
	dl.ShowRangeTo(&toEnd, 2, nil)
	if !strings.Contains(toEnd.String(), "50") {
		t.Fatalf("ShowRange(2, nil) should run to the end:\n%s", toEnd.String())
	}
}

// ShowRange takes nothing, a count, or a start and an end. A third value or a
// value that is not an int used to be ignored and the whole table shown, so
// the caller never learned that the range they asked for was not the one
// displayed.
func TestShowRangeRefusesWhatItCannotRead(t *testing.T) {
	colored := Config.GetUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(colored)

	dt := fillTable()
	dl := NewDataList(1, 2, 3)
	bad := [][]any{{1, 2, 3}, {"a"}, {0, "b"}}
	for _, args := range bad {
		for name, show := range map[string]func(*bytes.Buffer){
			"DataTable.ShowRangeTo":      func(b *bytes.Buffer) { dt.ShowRangeTo(b, args...) },
			"DataTable.ShowTypesRangeTo": func(b *bytes.Buffer) { dt.ShowTypesRangeTo(b, args...) },
			"DataList.ShowRangeTo":       func(b *bytes.Buffer) { dl.ShowRangeTo(b, args...) },
			"DataList.ShowTypesRangeTo":  func(b *bytes.Buffer) { dl.ShowTypesRangeTo(b, args...) },
		} {
			var buf bytes.Buffer
			show(&buf)
			if !strings.Contains(buf.String(), "ERROR") {
				t.Errorf("%s(%v) showed:\n%s", name, args, buf.String())
			}
		}
	}
}

func TestShowHeadAndTail(t *testing.T) {
	colored := Config.GetUseColoredOutput()
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

	dt := fillTable()
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
