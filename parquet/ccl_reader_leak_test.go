package parquet

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// These tests pin that a FilterWithCCL or ApplyCCL call which gives up part way
// through takes its reader with it. Both start streamAsArrowRecord, whose
// producer goroutine hands each batch over an unbuffered channel and only checks
// ctx.Done() while it waits. An early return leaves the consumer gone, so
// without a context of its own the producer stays blocked on a batch nobody
// reads, holding the source file open for the life of the process. The caller's
// context does not help: a caller who passed context.Background() never ends
// it, which is exactly what the tests below do.

const (
	// leakTestRows is enough rows for the reader to have more than one
	// cclBatchSize batch to send, so a consumer that stops after the first one
	// leaves the producer waiting on a send.
	leakTestRows = 5000

	// leakTestCalls is how many times each call is made. One leaked reader is
	// easy to hide inside a noisy goroutine count; ten of them are not.
	leakTestCalls = 10
)

// leakFilterExpr and leakApplyScript both fail while the first batch is being
// evaluated, which is after the reader has started. Division by zero is
// reported by the CCL evaluator (internal/ccl applyOperator, "division by
// zero"), so both reach the row loop and then give up with the producer still
// holding a batch it wants to send.
const (
	leakFilterExpr  = "A / 0 > 1"
	leakApplyScript = "NEW('B') = A / 0"
)

// writeLeakTestFile writes a one-column file counting up from 1 to n as int64
// and returns its path.
func writeLeakTestFile(t *testing.T, n int) string {
	t.Helper()
	vals := make([]any, n)
	for i := range vals {
		vals[i] = int64(i + 1)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(vals...).SetName("A"))
	path := filepath.Join(t.TempDir(), "counting.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the counting table: %v", err)
	}
	return path
}

// waitForNoNewGoroutines polls the goroutine count every 10ms for up to 2s and
// reports whether it is back at before. The count is sampled once up front so
// the last poll is not the only one that gets to pass.
func waitForNoNewGoroutines(before int) (int, bool) {
	deadline := time.Now().Add(2 * time.Second)
	after := runtime.NumGoroutine()
	for {
		if after <= before {
			return after, true
		}
		if !time.Now().Before(deadline) {
			return after, false
		}
		time.Sleep(10 * time.Millisecond)
		after = runtime.NumGoroutine()
	}
}

func TestFilterWithCCLStopsItsReaderOnAnError(t *testing.T) {
	path := writeLeakTestFile(t, leakTestRows)

	before := runtime.NumGoroutine()
	for i := 0; i < leakTestCalls; i++ {
		if _, err := FilterWithCCL(context.Background(), path, leakFilterExpr); err == nil {
			t.Fatalf("call %d: FilterWithCCL(%q) returned no error", i+1, leakFilterExpr)
		}
	}
	after, settled := waitForNoNewGoroutines(before)
	if !settled {
		t.Fatalf("goroutines did not return to %d within 2s after %d failing FilterWithCCL calls: %d goroutines running, want at most %d",
			before, leakTestCalls, after, before)
	}
}

func TestApplyCCLStopsItsReaderOnAnError(t *testing.T) {
	path := writeLeakTestFile(t, leakTestRows)

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the file before the calls: %v", err)
	}

	before := runtime.NumGoroutine()
	for i := 0; i < leakTestCalls; i++ {
		if err := ApplyCCL(context.Background(), path, leakApplyScript); err == nil {
			t.Fatalf("call %d: ApplyCCL(%q) returned no error", i+1, leakApplyScript)
		}
	}

	// A failed run writes to a temporary file of its own and only renames it
	// into place once every batch and the footer are in, so the source has to
	// come back byte for byte.
	after1, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the file after the calls: %v", err)
	}
	if !bytes.Equal(original, after1) {
		t.Fatalf("the file changed: %d bytes before, %d bytes after %d failing ApplyCCL calls", len(original), len(after1), leakTestCalls)
	}

	after, settled := waitForNoNewGoroutines(before)
	if !settled {
		t.Fatalf("goroutines did not return to %d within 2s after %d failing ApplyCCL calls: %d goroutines running, want at most %d",
			before, leakTestCalls, after, before)
	}
}
