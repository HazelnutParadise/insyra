package parquet

import (
	"bytes"
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// arrow-go v18 refuses a page above 64 MiB compressed unless the reader's
// limits are raised. One 70 MiB value of random bytes makes a page that size.
func TestAPageAboveArrowsDefaultLimitReads(t *testing.T) {
	if testing.Short() {
		t.Skip("writes a 70 MiB value")
	}
	blob := make([]byte, 70<<20)
	if _, err := rand.Read(blob); err != nil {
		t.Fatal(err)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(insyra.Cell(blob)).SetName("blob"))
	path := filepath.Join(t.TempDir(), "big.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatal(err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read refused the large page: %v", err)
	}
	b, ok := got.GetColByNumber(0).Get(0).([]byte)
	if !ok || !bytes.Equal(b, blob) {
		t.Fatalf("the 70 MiB value did not read back intact (%T)", got.GetColByNumber(0).Get(0))
	}
}
