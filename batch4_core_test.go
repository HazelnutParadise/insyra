package insyra

import (
	"errors"
	"sync"
	"testing"
)

type failingWriter struct{ n int }

func (w *failingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return 0, errors.New("disk full")
}

// SEC-2: a write failure surfaces as an error even when it only happens at
// the final flush.
func TestToCSVReportsWriteFailure(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3))
	if err := dt.writeCSV(&failingWriter{}, CSVWriteOptions{}); err == nil {
		t.Fatal("writeCSV to a failing writer returned nil")
	}
}

// AtomicDoAll called from inside AtomicDo on one of its instances skips the
// instance this goroutine already holds and still locks the others. When it
// ran the callback inline instead, AppendCols inside dt.AtomicDo read col's
// data unlocked and raced this goroutine's col.Append (reported under -race).
func TestAtomicDoAllNestedInAtomicDoLocksTheOthers(t *testing.T) {
	dt := NewDataTable()
	col := NewDataList(1, 2, 3)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			col.Append(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			dt.AtomicDo(func(tbl *DataTable) { tbl.AppendCols(col) })
		}
	}()
	wg.Wait()
	if got := dt.getMaxColLength(); got < 3 {
		t.Fatalf("appended columns are shorter than the source list: %d", got)
	}
}
