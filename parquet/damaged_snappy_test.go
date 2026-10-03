package parquet

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/parquet/file"
)

// writeDamagedSnappyFile writes a 3,000-row Snappy file of one column in three
// row groups and overwrites eight bytes inside the compressed body of the last
// row group's first page, past its header, so the page header still decodes
// and the decompressor is what meets the damage.
func writeDamagedSnappyFile(t *testing.T) string {
	t.Helper()
	values := make([]any, 3000)
	for i := range values {
		values[i] = float64(i + 1)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(values...).SetName("A"))

	path := filepath.Join(t.TempDir(), "damaged-snappy.parquet")
	if err := Write(dt, path, WriteOptions{Compression: CompressionSnappy, RowGroupSize: 1000}); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := file.NewParquetReader(f)
	if err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	chunk, err := r.MetaData().RowGroup(2).ColumnChunk(0)
	if err != nil {
		_ = r.Close()
		t.Fatal(err)
	}
	offset := chunk.DataPageOffset()
	if chunk.HasDictionaryPage() {
		offset = chunk.DictionaryPageOffset()
	}
	// Closing the reader closes the file it was given.
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A page header of one float64 column takes well under 20 bytes, and the
	// page body is longer than 28.
	const skip = 20
	if offset < 0 || offset+skip+8 > int64(len(data)) {
		t.Fatalf("row group 2 starts at %d, too close to the end of the %d-byte file", offset, len(data))
	}
	for i := offset + skip; i < offset+skip+8; i++ {
		data[i] = 0xFF
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestReadRefusesADamagedSnappyPage reads a file whose Snappy page body is
// damaged. Arrow v17 decompressed pages in a goroutine of its own and panicked
// there with "snappy: corrupt input", which ended the test binary; every reader
// has to return an error instead, and ApplyCCL has to leave the file alone.
func TestReadRefusesADamagedSnappyPage(t *testing.T) {
	ctx := context.Background()
	path := writeDamagedSnappyFile(t)

	if dt, err := Read(ctx, path, ReadOptions{}); err == nil {
		t.Errorf("Read returned %d rows with no error on a damaged Snappy page", dt.NumRows())
	} else {
		t.Logf("Read: %v", err)
	}

	var streamErr error
	for _, err := range Stream(ctx, path, ReadOptions{}, 500) {
		if err != nil {
			streamErr = err
		}
	}
	if streamErr == nil {
		t.Error("Stream ended with no error on a damaged Snappy page")
	}

	if _, err := FilterWithCCL(ctx, path, "A > 0"); err == nil {
		t.Error("FilterWithCCL returned no error on a damaged Snappy page")
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyCCL(ctx, path, "NEW('c') = A"); err == nil {
		t.Error("ApplyCCL returned no error on a damaged Snappy page")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Errorf("ApplyCCL changed the file it could not read: %d bytes became %d", len(before), len(after))
	}

	// The intact row groups still read in full.
	dt, err := Read(ctx, path, ReadOptions{RowGroups: []int{0, 1}})
	if err != nil {
		t.Fatalf("reading the intact row groups: %v", err)
	}
	if dt.NumRows() != 2000 {
		t.Errorf("the intact row groups read %d rows, want 2000", dt.NumRows())
	}
}
