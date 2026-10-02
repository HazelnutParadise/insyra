package parquet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow/go/v17/parquet/file"
)

// A Parquet file whose data page header is damaged reads as if it ended before
// that row group: Arrow records the page it cannot decode, then reports the end
// of the row group without asking for that error, and pqarrow turns the empty
// batch into io.EOF. Every reader here therefore compares what it read with the
// row count the footer holds for the row groups it was asked for, and says the
// file is damaged when the two differ.

// writeDamagedFile writes a 3,000-row file of one column with three row groups
// and overwrites the first eight bytes of row group damage's first data page,
// which is its page header, with 0xFF. Reading it stops before that row group.
func writeDamagedFile(t *testing.T, damage int) string {
	t.Helper()
	values := make([]any, 3000)
	for i := range values {
		values[i] = float64(i + 1)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(values...).SetName("A"))

	path := filepath.Join(t.TempDir(), "damaged.parquet")
	if err := Write(dt, path, WriteOptions{RowGroupSize: 1000}); err != nil {
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
	chunk, err := r.MetaData().RowGroup(damage).ColumnChunk(0)
	if err != nil {
		_ = r.Close()
		_ = f.Close()
		t.Fatal(err)
	}
	offset := chunk.DataPageOffset()
	if err := r.Close(); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	// Closing the reader closes the file it was given; the package's own
	// readers allow for that.
	if err := f.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if offset < 0 || offset+8 > int64(len(data)) {
		t.Fatalf("row group %d starts at %d, outside the %d-byte file", damage, offset, len(data))
	}
	for i := offset; i < offset+8; i++ {
		data[i] = 0xFF
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// checkDamaged asserts err says the file is damaged, that its metadata holds
// 3,000 rows and that it names row group where, and that no table came back
// with it.
func checkDamaged(t *testing.T, what string, err error, dt *insyra.DataTable, where string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s returned no error on a damaged file", what)
		return
	}
	msg := err.Error()
	t.Logf("%s: %v", what, err)
	for _, want := range []string{"is damaged", "3000", "row groups " + where} {
		if !strings.Contains(msg, want) {
			t.Errorf("%s returned %q, want it to mention %q", what, msg, want)
		}
	}
	if dt != nil {
		t.Errorf("%s returned a %d-row table with the error %v", what, dt.NumRows(), err)
	}
}

// TestReadRefusesADamagedRowGroup puts one damaged row group through every
// reader. Arrow's own answer is a short read with no error, so each of them has
// to say the file is damaged instead, and ApplyCCL has to leave the file alone.
func TestReadRefusesADamagedRowGroup(t *testing.T) {
	ctx := context.Background()
	for _, damage := range []int{2, 1} {
		t.Run(fmt.Sprintf("row group %d", damage), func(t *testing.T) {
			path := writeDamagedFile(t, damage)
			// Arrow reads a page it cannot decode as the end of its row
			// group and goes on with the next one, so the groups that did not
			// read in full are named explicitly.
			where := fmt.Sprintf("[%d]", damage)

			dt, err := Read(ctx, path, ReadOptions{})
			checkDamaged(t, "Read", err, dt, where)

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fromReader, err := ReadFrom(ctx, bytes.NewReader(data), int64(len(data)), ReadOptions{})
			checkDamaged(t, "ReadFrom", err, fromReader, where)

			list, err := ReadColumn(ctx, path, "A", ReadColumnOptions{})
			if err == nil {
				t.Errorf("ReadColumn returned %d values with no error on a damaged file", list.Len())
			} else {
				checkDamaged(t, "ReadColumn", err, nil, where)
			}

			var streamErr error
			for _, err := range Stream(ctx, path, ReadOptions{}, 500) {
				if err != nil {
					streamErr = err
				}
			}
			if streamErr == nil {
				t.Error("Stream ended with no error on a damaged file")
			} else if !strings.Contains(streamErr.Error(), "is damaged") {
				t.Errorf("Stream returned %v, want an error saying the file is damaged", streamErr)
			}

			filtered, err := FilterWithCCL(ctx, path, "A > 0")
			checkDamaged(t, "FilterWithCCL", err, filtered, where)

			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := ApplyCCL(ctx, path, "NEW('c') = A"); err == nil {
				t.Error("ApplyCCL returned no error on a damaged file")
			} else {
				checkDamaged(t, "ApplyCCL", err, nil, where)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, before) {
				t.Errorf("ApplyCCL changed the file it could not read: %d bytes became %d", len(before), len(after))
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".tmp") {
					t.Errorf("ApplyCCL left %s behind", e.Name())
				}
			}
		})
	}
}

// TestReadsTheIntactRowGroupsOfADamagedFile asks for the two row groups before
// the damaged one, so nothing is read past the damage and the read is whole.
func TestReadsTheIntactRowGroupsOfADamagedFile(t *testing.T) {
	ctx := context.Background()
	path := writeDamagedFile(t, 2)

	dt, err := Read(ctx, path, ReadOptions{RowGroups: []int{0, 1}})
	if err != nil {
		t.Fatalf("reading the intact row groups: %v", err)
	}
	if dt.NumRows() != 2000 {
		t.Fatalf("got %d rows, want 2000", dt.NumRows())
	}
	if first := dt.GetElement(0, "A"); first != float64(1) {
		t.Errorf("first row holds %v (%T), want 1", first, first)
	}
	if last := dt.GetElement(1999, "A"); last != float64(2000) {
		t.Errorf("last row holds %v (%T), want 2000", last, last)
	}
}

// TestReadsAroundADamagedMiddleRowGroup reads row groups 0 and 2 when row
// group 1 is damaged: they are intact, so no error is returned and the rows
// remain the ones from those groups in order.
func TestReadsAroundADamagedMiddleRowGroup(t *testing.T) {
	ctx := context.Background()
	path := writeDamagedFile(t, 1)

	dt, err := Read(ctx, path, ReadOptions{RowGroups: []int{0, 2}})
	if err != nil {
		t.Fatalf("reading intact row groups around a damaged one: %v", err)
	}
	if dt.NumRows() != 2000 {
		t.Fatalf("got %d rows, want 2000", dt.NumRows())
	}
	if first := dt.GetElement(0, "A"); first != float64(1) {
		t.Errorf("first row holds %v (%T), want 1", first, first)
	}
	if last := dt.GetElement(1999, "A"); last != float64(3000) {
		t.Errorf("last row holds %v (%T), want 3000", last, last)
	}
	if row1000 := dt.GetElement(999, "A"); row1000 != float64(1000) { // 0-based 999 is 1000th row
		t.Errorf("row 1000 holds %v (%T), want 1000", row1000, row1000)
	}
	if row1001 := dt.GetElement(1000, "A"); row1001 != float64(2001) { // 1-based 1001 is index 1000
		t.Errorf("row 1001 holds %v (%T), want 2001", row1001, row1001)
	}
}
