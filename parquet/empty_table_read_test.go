package parquet

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// A file with columns and no rows came back from Read as one row of nils: an
// empty column was handed to NewDataList as nil, which makes a list holding
// one nil value (#415).
func TestReadingAFileWithNoRowsGivesNoRows(t *testing.T) {
	empty := insyra.NewDataTable(insyra.NewDataList().SetName("p"), insyra.NewDataList().SetName("q"))
	path := filepath.Join(t.TempDir(), "empty.parquet")
	if err := Write(empty, path); err != nil {
		t.Fatal(err)
	}
	back, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if rows, cols := back.Size(); rows != 0 || cols != 2 || !slices.Equal(back.ColNames(), []string{"p", "q"}) {
		t.Errorf("Read gave %d x %d with columns %q, want 0 x 2 named p and q", rows, cols, back.ColNames())
	}
	tables, errs := Stream(context.Background(), path, ReadOptions{}, 10)
	for dt := range tables {
		if rows, _ := dt.Size(); rows != 0 {
			t.Errorf("Stream gave a batch of %d rows from an empty file", rows)
		}
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
}
