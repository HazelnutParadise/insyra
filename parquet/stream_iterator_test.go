package parquet

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// Stream is ranged over directly. Leaving the loop stops the reader, so a
// caller who breaks early without cancelling anything leaves nothing running.

// rowsFile writes n rows so a stream with a small batch size yields several
// batches.
func rowsFile(t *testing.T, n int) string {
	t.Helper()
	values := make([]any, n)
	for i := range values {
		values[i] = i
	}
	path := filepath.Join(t.TempDir(), "rows.parquet")
	if err := Write(insyra.NewDataTable(insyra.NewDataList(values...).SetName("v")), path); err != nil {
		t.Fatalf("writing the file: %v", err)
	}
	return path
}

func TestStreamYieldsEveryRow(t *testing.T) {
	total := 0
	for dt, err := range Stream(context.Background(), rowsFile(t, 25), ReadOptions{}, 10) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		rows, _ := dt.Size()
		total += rows
	}
	if total != 25 {
		t.Fatalf("read %d rows, want 25", total)
	}
}

func TestStreamStopsWhenTheLoopBreaks(t *testing.T) {
	path := rowsFile(t, 50)
	// Settle whatever the test runtime started before measuring.
	runtime.GC()
	before := runtime.NumGoroutine()

	for _, err := range Stream(context.Background(), path, ReadOptions{}, 5) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		break
	}

	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutines still running after the loop broke, %d before it started", runtime.NumGoroutine(), before)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStreamReportsAMissingFileOnce(t *testing.T) {
	yields := 0
	for dt, err := range Stream(context.Background(), filepath.Join(t.TempDir(), "nope.parquet"), ReadOptions{}, 10) {
		yields++
		if dt != nil {
			t.Errorf("a failure should come with a nil table, got %v", dt)
		}
		if err == nil {
			t.Error("a missing file should yield an error")
		}
	}
	if yields != 1 {
		t.Fatalf("yielded %d times, want exactly one failure", yields)
	}
}
