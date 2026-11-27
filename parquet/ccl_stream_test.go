package parquet

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// streamedRows ranges over StreamFilterWithCCL and returns the [A, B] of every
// row it gives, and how many batches it gave.
func streamedRows(t *testing.T, path, expr string, opts ...FilterWithCCLOptions) ([][]any, int) {
	t.Helper()
	var rows [][]any
	batches := 0
	for batch, err := range StreamFilterWithCCL(context.Background(), path, expr, opts...) {
		if err != nil {
			t.Fatalf("StreamFilterWithCCL(%q): %v", expr, err)
		}
		batches++
		rows = append(rows, filteredRows(t, batch)...)
	}
	return rows, batches
}

// The stream and FilterWithCCL both give what the loaded table's own CCL gives,
// across batch boundaries, for every route a filter can take (#415 part 3).
func TestStreamFilterWithCCLMatchesTheLoadedTable(t *testing.T) {
	path := writePlanFixture(t)
	for _, expr := range []string{"A > 100", "A > AVG(A)", "A > MEDIAN(A)", "A > -1", "COUNT(@) > 0", "LAG(A, 1)", "IF(# > 0, A.(# - 1), 0) > 1500"} {
		t.Run(expr, func(t *testing.T) {
			want := expectedFilter(t, path, expr)
			got, batches := streamedRows(t, path, expr)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("the stream gave %d rows, the loaded table keeps %d", len(got), len(want))
			}
			if len(want) > cclBatchSize && batches < 2 {
				t.Errorf("%d rows came in %d batch", len(got), batches)
			}
			table, err := FilterWithCCL(context.Background(), path, expr)
			if err != nil {
				t.Fatal(err)
			}
			if rows := filteredRows(t, table); !reflect.DeepEqual(rows, want) {
				t.Errorf("FilterWithCCL gave %d rows, the loaded table keeps %d", len(rows), len(want))
			}
		})
	}
}

func TestStreamFilterWithCCLGivesTheColumnsWhenNothingIsKept(t *testing.T) {
	path := writePlanFixture(t)
	n := 0
	for batch, err := range StreamFilterWithCCL(context.Background(), path, "A < 0") {
		if err != nil {
			t.Fatal(err)
		}
		n++
		if rows, cols := batch.Size(); rows != 0 || cols != 2 || !slices.Equal(batch.ColNames(), []string{"A", "B"}) {
			t.Errorf("got %d x %d with columns %q", rows, cols, batch.ColNames())
		}
	}
	if n != 1 {
		t.Errorf("%d batches for a filter that keeps nothing, want one empty table", n)
	}
}

func TestFilterWithCCLColumns(t *testing.T) {
	path := writePlanFixture(t)
	opt := FilterWithCCLOptions{Columns: []string{"B", "A"}}
	table, err := FilterWithCCL(context.Background(), path, "A > 2400", opt)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(table.ColNames(), []string{"B", "A"}) {
		t.Fatalf("columns %q, want B then A", table.ColNames())
	}
	if rows, _ := table.Size(); rows != 99 || table.GetColByNumber(1).Get(0) != any(int64(2401)) {
		t.Errorf("%d rows, first A %v", rows, table.GetColByNumber(1).Get(0))
	}
	// The filter reads a column the result leaves out.
	only, err := FilterWithCCL(context.Background(), path, "A > 2400", FilterWithCCLOptions{Columns: []string{"B"}})
	if err != nil || !slices.Equal(only.ColNames(), []string{"B"}) {
		t.Errorf("got %v, %v", only.ColNames(), err)
	}
	for _, bad := range [][]string{{"Z"}, {"A", "A"}} {
		before := cclDataPasses.Load()
		if _, err := FilterWithCCL(context.Background(), path, "A > 0", FilterWithCCLOptions{Columns: bad}); err == nil {
			t.Errorf("Columns %q gave no error", bad)
		}
		if cclDataPasses.Load() != before {
			t.Errorf("Columns %q was refused after reading the data", bad)
		}
	}
}

func TestStreamFilterWithCCLMaxRowsGivesTheRowsWithinTheLimit(t *testing.T) {
	path := writePlanFixture(t)
	for _, expr := range []string{"A > -1", "A > MEDIAN(A) - 2000", "LAG(A, 1)"} {
		got := 0
		var last error
		for batch, err := range StreamFilterWithCCL(context.Background(), path, expr, FilterWithCCLOptions{MaxRows: 1500}) {
			if err != nil {
				last = err
				continue
			}
			rows, _ := batch.Size()
			got += rows
		}
		if got != 1500 || !errors.Is(last, ErrCCLRowLimit) {
			t.Errorf("%s: %d rows then %v; want 1500 rows then ErrCCLRowLimit", expr, got, last)
		}
	}
}

// Leaving the loop stops the read: the reader goroutines end and no further
// pass starts.
func TestStreamFilterWithCCLStopsWhenTheLoopEnds(t *testing.T) {
	path := writePlanFixture(t)
	before := runtime.NumGoroutine()
	for range 20 {
		for batch, err := range StreamFilterWithCCL(context.Background(), path, "A > -1") {
			if err != nil {
				t.Fatal(err)
			}
			_ = batch
			break
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before+2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Errorf("%d goroutines before, %d after leaving 20 loops early", before, after)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range StreamFilterWithCCL(ctx, path, "A > -1") {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("a cancelled context gave %v", err)
		}
	}
	//lint:ignore SA1012 the nil context is what is being tested
	for _, err := range StreamFilterWithCCL(nil, path, "A > -1") { //nolint:staticcheck
		if !errors.Is(err, errNilContext) {
			t.Errorf("a nil context gave %v", err)
		}
	}
}

// A sink written batch by batch reads back as the filter's table.
func TestStreamFilterWithCCLIntoACSVSink(t *testing.T) {
	path := writePlanFixture(t)
	var buf bytes.Buffer
	header := true
	for batch, err := range StreamFilterWithCCL(context.Background(), path, "A > AVG(A)") {
		if err != nil {
			t.Fatal(err)
		}
		var part bytes.Buffer
		if err := batch.WriteCSV(&part); err != nil {
			t.Fatal(err)
		}
		b := part.Bytes()
		if !header {
			b = b[bytes.IndexByte(b, '\n')+1:]
		}
		header = false
		buf.Write(b)
	}
	back, err := insyra.ReadCSV(&buf)
	if err != nil {
		t.Fatal(err)
	}
	want := expectedFilter(t, path, "A > AVG(A)")
	if rows, _ := back.Size(); rows != len(want) || back.GetColByNumber(0).Get(0) != want[0][0] {
		t.Errorf("the CSV sink read back %d rows starting %v, want %d starting %v", rows, back.GetColByNumber(0).Get(0), len(want), want[0][0])
	}
}
