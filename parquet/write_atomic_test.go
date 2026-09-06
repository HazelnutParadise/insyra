package parquet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestWriteLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "w.parquet")
	dt := insyra.NewDataTable(insyra.NewDataList(1.0, 2.0).SetName("a"))
	if err := Write(dt, p); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "w.parquet" {
		t.Fatalf("unexpected files: %v", entries)
	}
}
