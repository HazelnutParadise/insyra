package insyra

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestStringIsTheUncolouredView(t *testing.T) {
	prev := Config.GetUseColoredOutput()
	Config.SetUseColoredOutput(true)
	defer Config.SetUseColoredOutput(prev)

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"), NewDataList("x", "y", "z").SetName("b"))
	s := dt.String()
	if strings.Contains(s, "\033[") {
		t.Errorf("String has colour codes:\n%q", s)
	}
	for _, want := range []string{"DataTable", "a", "b", "x", "z"} {
		if !strings.Contains(s, want) {
			t.Errorf("String lacks %q:\n%s", want, s)
		}
	}
	if fmt.Sprint(dt) != s || fmt.Sprintf("%v", dt) != s {
		t.Error("fmt does not use String")
	}
	var plain bytes.Buffer
	dt.renderRange(&plain, showStyle{width: 80})
	if plain.String() != s {
		t.Error("String is not the 80-column uncoloured view")
	}

	dl := NewDataList(1.5, "two", 3).SetName("v")
	ls := dl.String()
	if strings.Contains(ls, "\033[") || !strings.Contains(ls, "two") || !strings.Contains(ls, "DataList") || fmt.Sprint(dl) != ls {
		t.Errorf("DataList.String:\n%s", ls)
	}
}

func TestStringPreviewsALongTable(t *testing.T) {
	vals := make([]any, 1000)
	for i := range vals {
		vals[i] = i
	}
	s := NewDataTable(NewDataList(vals...)).String()
	if lines := strings.Count(s, "\n"); lines > 60 {
		t.Errorf("a 1000-row table gave %d lines", lines)
	}
	if !strings.Contains(s, "999") || strings.Contains(s, " 500 ") {
		t.Errorf("preview does not show the first and last rows only:\n%s", s)
	}
}

func TestCellsHoldingTablesAreSummarised(t *testing.T) {
	inner := NewDataTable(NewDataList(1, 2), NewDataList(3, 4))
	dt := NewDataTable(NewDataList(inner, NewDataList(7, 8, 9)))
	s := dt.String()
	if !strings.Contains(s, "DataTable(2x2)") || !strings.Contains(s, "DataList(3)") {
		t.Errorf("nested cells are not summarised:\n%s", s)
	}

	// A table holding itself must finish.
	self := NewDataTable(NewDataList(1, 2))
	self.UpdateElement(0, "A", self)
	if s := self.String(); !strings.Contains(s, "DataTable(2x1)") {
		t.Errorf("self-holding table:\n%s", s)
	}
	selfList := NewDataList(1)
	selfList.Append(selfList)
	if s := selfList.String(); !strings.Contains(s, "DataList(") {
		t.Errorf("self-holding list:\n%s", s)
	}
}
