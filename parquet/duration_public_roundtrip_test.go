package parquet

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// Write and Read are the public pair a caller uses, so the Duration column is
// proved through them and not through the helper that lowers it.
func TestWriteThenReadKeepsADurationColumn(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList(90*time.Second, 36*time.Hour, 0*time.Second).SetName("elapsed"))
	path := filepath.Join(t.TempDir(), "elapsed.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatal(err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []any{90 * time.Second, 36 * time.Hour, 0 * time.Second}
	col := got.GetColByNumber(0)
	for i, w := range want {
		if v := col.Get(i); v != w {
			t.Fatalf("row %d = %#v (%T), want %#v", i, v, v, w)
		}
	}
}
