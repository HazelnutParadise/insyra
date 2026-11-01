package parquet

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestParquetCCLIfError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iferror.parquet")
	table := insyra.NewDataTable(insyra.NewDataList(6.0, 8.0).SetName("a"), insyra.NewDataList(2.0, 0.0).SetName("b"))
	if err := Write(table, path); err != nil {
		t.Fatal(err)
	}
	kept, err := FilterWithCCL(context.Background(), path, "IFERROR(A / B, -1) > 0")
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := kept.Size(); rows != 1 {
		t.Errorf("FilterWithCCL kept %d rows, want 1", rows)
	}
	if err := ApplyCCL(context.Background(), path, "NEW('r') = IFERROR(A / B, -1)"); err != nil {
		t.Fatal(err)
	}
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("r").Data()
	if got[0] != 3.0 || got[1] != -1.0 {
		t.Errorf("r = %#v", got)
	}
}
