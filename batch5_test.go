package insyra

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"
)

// IN-3, seen from the public API: a wrong comparator corrupts sorting.
func TestSortAndRankExactForLargeIntegers(t *testing.T) {
	const twoP53 = int64(1) << 53
	dl := NewDataList(twoP53+1, twoP53, twoP53+2)
	dl.Sort()
	want := []any{twoP53, twoP53 + 1, twoP53 + 2}
	got := dl.Data()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Sort = %v, want %v", got, want)
		}
	}

	ranks := NewDataList(twoP53+1, twoP53, twoP53+2).Rank().Data()
	for i, w := range []float64{2, 1, 3} {
		if ranks[i] != w {
			t.Fatalf("Rank = %v, want [2 1 3]", ranks)
		}
	}
}

// IN-2: Close() must not silently swallow an operation that was already
// waiting for the lock — the caller has no way to learn it never ran.
func TestCloseDoesNotDropWaitingOperations(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList(1)
	started := make(chan struct{})
	appended := make(chan struct{})

	go func() {
		dl.AtomicDo(func(inner *DataList) {
			close(started)
			// Give the other goroutine time to queue up on the mutex.
			time.Sleep(100 * time.Millisecond)
			inner.Close()
		})
	}()

	<-started
	go func() {
		dl.Append(2)
		close(appended)
	}()

	select {
	case <-appended:
	case <-time.After(3 * time.Second):
		t.Fatal("Append never returned")
	}
	if n := dl.Len(); n != 2 {
		t.Fatalf("Len() = %d, want 2: Close() dropped the queued Append", n)
	}
}

// IN-9: ShowTypes sorted its column headers as plain strings, so a table with
// more than 26 columns printed A, AA, AB, B, ... while Show printed them in
// the real column order.
func TestShowTypesColumnOrderBeyondZ(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable()
	for i := 0; i < 28; i++ {
		dt.AppendCols(NewDataList(i))
	}
	coloured := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(coloured)

	var typesOut, showOut bytes.Buffer
	dt.ShowTypesRangeTo(&typesOut)
	dt.ShowRangeTo(&showOut)

	typeCols := headerColumns(typesOut.String())
	showCols := headerColumns(showOut.String())
	if len(typeCols) == 0 || len(showCols) == 0 {
		t.Fatalf("could not find the header row\ntypes:\n%s\nshow:\n%s", typesOut.String(), showOut.String())
	}
	if strings.Join(typeCols, ",") != strings.Join(showCols, ",") {
		t.Fatalf("ShowTypes column order %v differs from Show's %v", typeCols, showCols)
	}
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// headerColumns returns the column labels from every "RowNames ..." header
// line, in the order they are printed (the display pages the columns).
func headerColumns(out string) []string {
	var cols []string
	for _, line := range strings.Split(ansiEscape.ReplaceAllString(out, ""), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "RowNames" {
			continue
		}
		cols = append(cols, fields[1:]...)
	}
	return cols
}

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
