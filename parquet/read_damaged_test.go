package parquet

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// damagedFiles writes the two whole files a damaged one is made of: a 1,000-row
// file whose leading bytes are laid over a 3,000-row file, so the data pages
// come from the first and the footer from the second. The test that walks k
// needs the shorter file's length, so it calls this rather than
// damagedParquet.
func damagedFiles(t *testing.T) (short, long []byte) {
	t.Helper()
	var a, b bytes.Buffer
	if err := WriteTo(mixedTable(1000), &a); err != nil {
		t.Fatal(err)
	}
	if err := WriteTo(mixedTable(3000), &b); err != nil {
		t.Fatal(err)
	}
	return a.Bytes(), b.Bytes()
}

// damagedParquet returns a copy of the 3,000-row file with the first k bytes of
// the 1,000-row file laid over it. k past the shorter file's length is clamped,
// so a caller walking k up to that length cannot slice out of range.
func damagedParquet(t *testing.T, k int) []byte {
	t.Helper()
	short, long := damagedFiles(t)
	if k > len(short) {
		k = len(short)
	}
	data := append([]byte(nil), long...)
	copy(data, short[:k])
	return data
}

// noPanic runs f and turns a panic into a failure naming what panicked, because
// a panic crossing these calls is the defect these tests pin.
func noPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", what, r)
		}
	}()
	f()
}

// TestReadFromNeverPanicsOnADamagedFile walks k over the shorter file and reads
// what comes out. Arrow's reader dereferences a nil pointer on some of these
// offsets; every one of them has to come back as a table or an error, and at
// least one as an error naming the file.
func TestReadFromNeverPanicsOnADamagedFile(t *testing.T) {
	short, _ := damagedFiles(t)
	ctx := context.Background()

	cases := 0
	errs := 0
	named := 0
	for k := 100; k <= len(short); k += 53 {
		data := damagedParquet(t, k)
		var err error
		noPanic(t, fmt.Sprintf("ReadFrom at k=%d", k), func() {
			_, err = ReadFrom(ctx, bytes.NewReader(data), int64(len(data)), ReadOptions{})
		})
		cases++
		if err == nil {
			continue
		}
		errs++
		if strings.Contains(err.Error(), "not a readable Parquet file") {
			named++
			continue
		}
		t.Logf("k=%d returned another error: %v", k, err)
	}
	t.Logf("%d of %d offsets returned an error, %d of them naming the file", errs, cases, named)

	if errs == 0 {
		t.Fatalf("no offset made ReadFrom return an error, out of %d read", cases)
	}
	if named == 0 {
		t.Error("no error said the file is not a readable Parquet file")
	}
}

// TestEveryReaderRefusesADamagedFile takes one offset that made Arrow's reader
// panic and puts it through every reader, which had no recover of its own.
func TestEveryReaderRefusesADamagedFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "damaged.parquet")
	if err := os.WriteFile(path, damagedParquet(t, 8103), 0o600); err != nil {
		t.Fatal(err)
	}

	var readErr error
	noPanic(t, "Read", func() {
		_, readErr = Read(ctx, path, ReadOptions{})
	})
	if readErr == nil {
		t.Error("Read returned no error on a damaged file")
	}

	noPanic(t, "Inspect", func() {
		info, err := Inspect(path)
		t.Logf("Inspect returned %d rows and error %v", info.NumRows, err)
	})

	var streamErr error
	noPanic(t, "Stream", func() {
		for _, err := range Stream(ctx, path, ReadOptions{}, 100) {
			if err != nil {
				streamErr = err
				return
			}
		}
	})
	if streamErr == nil {
		t.Error("Stream reported no error on a damaged file")
	}

	var filterErr error
	noPanic(t, "FilterWithCCL", func() {
		_, filterErr = FilterWithCCL(ctx, path, "A > 0")
	})
	if filterErr == nil {
		t.Error("FilterWithCCL returned no error on a damaged file")
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var applyErr error
	noPanic(t, "ApplyCCL", func() {
		applyErr = ApplyCCL(ctx, path, "NEW('z') = 1")
	})
	if applyErr == nil {
		t.Error("ApplyCCL returned no error on a damaged file")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Errorf("ApplyCCL changed the file it failed to read: %d bytes became %d", len(before), len(after))
	}
}

// panickingAt is a source whose every read panics, the way a range reader
// backed by a corrupt block or a driver that faults can.
type panickingAt struct{ reason string }

func (p panickingAt) ReadAt([]byte, int64) (int, error) { panic(p.reason) }

// TestAReaderPanicBecomesAnError covers the readers that cannot let a panic
// reach the caller on their own: StreamFrom reads in a goroutine, where a panic
// would take the process down instead of the caller. The overlapping file
// reaches neither, because Arrow's record reader reports an error there first,
// so the two paths are pinned here instead.
func TestAReaderPanicBecomesAnError(t *testing.T) {
	ctx := context.Background()

	var readErr error
	noPanic(t, "ReadFrom", func() {
		_, readErr = ReadFrom(ctx, panickingAt{"the source faulted"}, 4096, ReadOptions{})
	})
	if readErr == nil || !strings.Contains(readErr.Error(), "not a readable Parquet file") {
		t.Errorf("ReadFrom returned %v, want an error saying the file is not readable", readErr)
	}

	var streamErr error
	noPanic(t, "StreamFrom", func() {
		for _, err := range StreamFrom(ctx, panickingAt{"the source faulted"}, 4096, ReadOptions{}, 10) {
			if err != nil {
				streamErr = err
			}
		}
	})
	if streamErr == nil || !strings.Contains(streamErr.Error(), "not a readable Parquet file") {
		t.Errorf("StreamFrom returned %v, want an error saying the file is not readable", streamErr)
	}
}
