package parquet

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// These tests pin how cclBatchSize is used: each batch is evaluated on its
// own, so an expression that reads more than the current row sees only that
// batch. The size itself is not a tuning knob — changing it would change the
// rows FilterWithCCL keeps and the values ApplyCCL writes.

// writeCounting writes a one-column file counting up from 1 to n and returns
// its path.
func writeCounting(t *testing.T, n int) string {
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

func TestFilterWithCCLEvaluatesAnAggregatePerBatch(t *testing.T) {
	path := writeCounting(t, 2500)

	// 2500 rows arrive as three batches of 1000, 1000 and 500. Each batch is
	// compared against its own average — 500.5, 1500.5 and 2250.5 — so half of
	// every batch is kept, not half of the table. The same filter evaluated over
	// the whole table would compare against 1250.5 and keep values from 1251 on.
	res, err := FilterWithCCL(context.Background(), path, "A > AVG(A)")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}

	rows, _ := res.Size()
	if rows != 1250 {
		t.Fatalf("rows = %d, want 1250 (half of each batch)", rows)
	}

	got := fmt.Sprint(res.GetColByNumber(0).Get(0))
	if got != "501" {
		t.Fatalf("first kept value = %s, want \"501\" (first value of the first batch above its average)", got)
	}
}

func TestFilterWithCCLRowIndexRestartsEachBatch(t *testing.T) {
	path := writeCounting(t, 2500)

	// The row index restarts at zero in every batch, so "# == 0" keeps the
	// first row of each of the three batches.
	res, err := FilterWithCCL(context.Background(), path, "# == 0")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}

	rows, _ := res.Size()
	if rows != 3 {
		t.Fatalf("rows = %d, want 3 (one per batch)", rows)
	}

	want := []string{"1", "1001", "2001"}
	got := res.GetColByNumber(0).Data()
	if len(got) != len(want) {
		t.Fatalf("kept %d values, want %d", len(got), len(want))
	}
	for i, w := range want {
		if s := fmt.Sprint(got[i]); s != w {
			t.Fatalf("kept value %d = %s, want %q", i, s, w)
		}
	}
}

func TestApplyCCLRowIndexRestartsEachBatch(t *testing.T) {
	path := writeCounting(t, 2500)

	// The written column restarts at zero in every batch, so the value at file
	// position 1000 is 0 again rather than 1000.
	if err := ApplyCCL(context.Background(), path, "NEW('i') = #"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	res, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	col := res.GetColByName("i")
	if col == nil {
		t.Fatalf("column \"i\" is missing from the result")
	}
	data := col.Data()
	if len(data) != 2500 {
		t.Fatalf("column \"i\" holds %d values, want 2500", len(data))
	}

	for i, want := range map[int]string{999: "999", 1000: "0", 1001: "1"} {
		if got := fmt.Sprint(data[i]); got != want {
			t.Fatalf("column \"i\" value %d = %s, want %q", i, got, want)
		}
	}
}
