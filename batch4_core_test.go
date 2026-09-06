package insyra

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// SEC-7: a failed ToCSV / ToJSON leaves no file and no temp file behind; a
// successful one leaves only the target.
func TestToCSVAndToJSONAreAtomic(t *testing.T) {
	dir := t.TempDir()
	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))

	missing := filepath.Join(dir, "no", "such", "dir", "x.csv")
	if err := dt.ToCSV(missing, false, true, false); err == nil {
		t.Fatal("ToCSV into a missing directory returned nil")
	}
	if err := dt.ToJSON(filepath.Join(dir, "no", "x.json"), true); err == nil {
		t.Fatal("ToJSON into a missing directory returned nil")
	}

	csvPath := filepath.Join(dir, "ok.csv")
	if err := dt.ToCSV(csvPath, false, true, false); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(dir, "ok.json")
	if err := dt.ToJSON(jsonPath, true); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
	b, _ := os.ReadFile(csvPath)
	if !strings.HasPrefix(string(b), "a\n1\n") {
		t.Fatalf("unexpected CSV content %q", b)
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
