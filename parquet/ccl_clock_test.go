package parquet

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

func TestParquetCCLClockIsOneMoment(t *testing.T) {
	col := insyra.NewDataList().SetName("A")
	for i := range 5000 {
		col.Append(i)
	}
	path := filepath.Join(t.TempDir(), "clock.parquet")
	if err := Write(insyra.NewDataTable(col), path); err != nil {
		t.Fatal(err)
	}
	if err := ApplyCCL(context.Background(), path, "NEW('t') = NOW()"); err != nil {
		t.Fatal(err)
	}
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	cells := dt.GetColByName("t").Data()
	first, ok := cells[0].(time.Time)
	if !ok {
		t.Fatalf("first cell is %#v", cells[0])
	}
	for i, v := range cells {
		if v != any(first) {
			t.Fatalf("row %d is %v, row 0 is %v", i, v, first)
		}
	}

	kept, err := FilterWithCCL(context.Background(), path, "YEAR(TODAY()) >= 2000")
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := kept.Size(); rows != 5000 {
		t.Errorf("FilterWithCCL kept %d rows", rows)
	}
}
