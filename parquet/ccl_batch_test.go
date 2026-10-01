package parquet

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// These tests pin what cclBatchSize does not change: the parts of an
// expression that read beyond the current row are computed over the whole
// file before any batch is evaluated, so FilterWithCCL and ApplyCCL give the
// answer the loaded table gives whatever the batch size.

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

func TestFilterWithCCLComputesAnAggregateOverTheFile(t *testing.T) {
	path := writeCounting(t, 2500)

	// 2500 rows arrive as three batches of 1000, 1000 and 500, but AVG(A) is
	// computed over the file before any row is compared with it, so every row
	// is compared against 1250.5 and everything from 1251 on is kept — the same
	// answer the loaded table gives, rather than half of each batch.
	res, err := FilterWithCCL(context.Background(), path, "A > AVG(A)")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}

	rows, _ := res.Size()
	if rows != 1250 {
		t.Fatalf("rows = %d, want 1250 (everything above the file's average of 1250.5)", rows)
	}

	got := fmt.Sprint(res.GetColByNumber(0).Get(0))
	if got != "1251" {
		t.Fatalf("first kept value = %s, want \"1251\" (the first value above the file's average)", got)
	}
}

func TestFilterWithCCLRowIndexIsTheFileRow(t *testing.T) {
	path := writeCounting(t, 2500)

	// # is the row's position in the file, not in the batch it arrived in, so
	// "# == 0" keeps the file's first row and no other.
	res, err := FilterWithCCL(context.Background(), path, "# == 0")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}

	rows, _ := res.Size()
	if rows != 1 {
		t.Fatalf("rows = %d, want 1 (the file's first row)", rows)
	}

	want := []string{"1"}
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

func TestApplyCCLRowIndexIsTheFileRow(t *testing.T) {
	path := writeCounting(t, 2500)

	// # is the row's position in the file, so the written column counts up
	// across the batches: the value at file position 1000 is 1000, not 0 again
	// because a batch started there.
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

	for i, want := range map[int]string{999: "999", 1000: "1000", 1001: "1001"} {
		if got := fmt.Sprint(data[i]); got != want {
			t.Fatalf("column \"i\" value %d = %s, want %q", i, got, want)
		}
	}
}
