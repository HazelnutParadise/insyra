package insyra

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
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

// wrappedTable embeds a table the way the isr wrappers do.
type wrappedTable struct{ *DataTable }

type wrappedList struct{ *DataList }

func TestCellsHoldingTablesAreSummarised(t *testing.T) {
	inner := NewDataTable(NewDataList(1, 2), NewDataList(3, 4))
	dt := NewDataTable(NewDataList(inner, NewDataList(7, 8, 9), wrappedTable{inner}, (*DataTable)(nil)))
	s := dt.String()
	if strings.Count(s, "<DataTable>") != 2 || !strings.Contains(s, "<DataList>") || !strings.Contains(s, "<nil>") {
		t.Errorf("nested cells are not summarised:\n%s", s)
	}

	// A table or list holding itself, directly or through a wrapper, must
	// finish.
	self := NewDataTable(NewDataList(1, 2))
	self.UpdateElement(0, "A", self)
	self.UpdateElement(1, "A", &wrappedTable{self})
	if s := self.String(); strings.Count(s, "<DataTable>") != 2 {
		t.Errorf("self-holding table:\n%s", s)
	}
	selfList := NewDataList(1)
	selfList.Append(selfList, wrappedList{selfList})
	if s := selfList.String(); strings.Count(s, "<DataList>") != 2 {
		t.Errorf("self-holding list:\n%s", s)
	}
}

func TestStringOfTablesHoldingEachOtherDoesNotDeadlock(t *testing.T) {
	tbl := NewDataTable(NewDataList(1))
	lst := NewDataList(tbl)
	tbl.UpdateElement(0, "A", lst)
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for i := 0; i < 200; i++ {
			wg.Add(2)
			go func() { defer wg.Done(); _ = tbl.String() }()
			go func() { defer wg.Done(); _ = lst.String() }()
		}
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("String on a table and a list holding each other deadlocked")
	}
}

func TestStringOfANilReceiver(t *testing.T) {
	if (*DataTable)(nil).String() != "<nil>" || (*DataList)(nil).String() != "<nil>" {
		t.Error("a nil receiver did not give <nil>")
	}
}
