package insyra

import (
	"errors"
	"testing"
	"time"
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
	if err := dt.writeCSV(&failingWriter{}, false, false, false); err == nil {
		t.Fatal("writeCSV to a failing writer returned nil")
	}
}

// IN-1: AtomicDoAll called from inside AtomicDo on one of its instances must
// not deadlock against another goroutine doing the mirror image.
func TestAtomicDoAllNestedInAtomicDoDoesNotDeadlock(t *testing.T) {
	a := NewDataList(1)
	b := NewDataList(2)
	done := make(chan struct{}, 2)
	go func() {
		a.AtomicDo(func(*DataList) {
			time.Sleep(20 * time.Millisecond)
			AtomicDoAll(func() {}, a, b)
		})
		done <- struct{}{}
	}()
	go func() {
		b.AtomicDo(func(*DataList) {
			time.Sleep(20 * time.Millisecond)
			AtomicDoAll(func() {}, a, b)
		})
		done <- struct{}{}
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("deadlock: nested AtomicDoAll never returned")
		}
	}
}
