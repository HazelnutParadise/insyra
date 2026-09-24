package parquet

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

func writeRows(t *testing.T, n int, rowGroupSize int64) string {
	t.Helper()
	vals := make([]any, n)
	for i := range vals {
		vals[i] = float64(i)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(vals...).SetName("a"))
	tbl, err := dataTableToArrowTable(dt)
	if err != nil {
		t.Fatal(err)
	}
	defer tbl.Release()
	p := filepath.Join(t.TempDir(), "x.parquet")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	w, err := pqarrow.NewFileWriter(tbl.Schema(), f, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteTable(tbl, rowGroupSize); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadColumnMaxValuesRefusesBeforeReading(t *testing.T) {
	p := writeRows(t, 1000, 1000)
	dl, err := ReadColumn(context.Background(), p, "a", ReadColumnOptions{MaxValues: 10})
	if err == nil || !strings.Contains(err.Error(), "1000") || !strings.Contains(err.Error(), "10") {
		t.Fatalf("expected error naming 1000 and 10, got %v", err)
	}
	if dl != nil {
		t.Fatal("expected nil DataList")
	}
}

func TestReadColumnMaxValuesWithinLimit(t *testing.T) {
	p := writeRows(t, 1000, 1000)
	dl, err := ReadColumn(context.Background(), p, "a", ReadColumnOptions{MaxValues: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if dl.Len() != 1000 {
		t.Fatalf("expected 1000 values, got %d", dl.Len())
	}
}

func TestReadColumnMaxValuesCountsSelectedRowGroups(t *testing.T) {
	p := writeRows(t, 1000, 500)
	info, err := Inspect(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.NumRowGroups != 2 {
		t.Fatalf("fixture should have 2 row groups, got %d", info.NumRowGroups)
	}
	dl, err := ReadColumn(context.Background(), p, "a", ReadColumnOptions{RowGroups: []int{0}, MaxValues: 500})
	if err != nil {
		t.Fatal(err)
	}
	if dl.Len() != 500 {
		t.Fatalf("expected 500 values, got %d", dl.Len())
	}
}
