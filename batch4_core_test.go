package insyra

import (
	"bytes"
	"errors"
	"io"
	"strings"
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

// closeFailer accepts every write and fails to close, the way a file on a full
// or network filesystem reports a write it could not complete.
type closeFailer struct{ bytes.Buffer }

func (*closeFailer) Close() error { return errors.New("close: no space left on device") }

// ToCSV and ToCSVWithOptions return the error from closing the file when the
// write itself succeeded; it used to be discarded, reporting a lost write as
// success.
func TestToCSVReportsCloseFailure(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3))
	w := &closeFailer{}
	err := dt.writeCSVAndClose(w, CSVWriteOptions{})
	if err == nil || !strings.Contains(err.Error(), "no space left") {
		t.Fatalf("writeCSVAndClose with a failing Close returned %v", err)
	}
	if w.Len() == 0 {
		t.Fatal("nothing was written before the close")
	}
	// A write error still wins over the close error.
	if err := dt.writeCSVAndClose(struct {
		io.Writer
		io.Closer
	}{&failingWriter{}, &closeFailer{}}, CSVWriteOptions{}); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("a write error should be returned before a close error, got %v", err)
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
