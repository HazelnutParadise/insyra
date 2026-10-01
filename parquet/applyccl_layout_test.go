package parquet

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/compress"
	"github.com/apache/arrow/go/v17/parquet/file"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

// These tests pin how ApplyCCL lays the file it writes back out: it keeps the
// codec each column had and the row group size of the original, one WriteOptions
// replaces both as Write uses it, and the new file goes through a temporary
// file of its own rather than through <path>.tmp.

// layoutOf reads a written file's layout back from its metadata: the rows in
// each row group, and the codec of every column of the first one.
func layoutOf(t *testing.T, path string) (rows []int64, codecs map[string]compress.Compression) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	r, err := file.NewParquetReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	md := r.MetaData()
	rows = make([]int64, 0, r.NumRowGroups())
	for i := 0; i < r.NumRowGroups(); i++ {
		rows = append(rows, md.RowGroup(i).NumRows())
	}
	codecs = make(map[string]compress.Compression)
	rg := md.RowGroup(0)
	for i := 0; i < rg.NumColumns(); i++ {
		chunk, err := rg.ColumnChunk(i)
		if err != nil {
			t.Fatal(err)
		}
		codecs[chunk.PathInSchema().String()] = chunk.Compression()
	}
	return rows, codecs
}

// assertCodecs fails naming the columns that differ.
func assertCodecs(t *testing.T, got map[string]compress.Compression, want map[string]compress.Compression) {
	t.Helper()
	for name, codec := range want {
		if g, ok := got[name]; !ok {
			t.Errorf("column %q is missing from the written file", name)
		} else if g != codec {
			t.Errorf("column %q used codec %d, want %d", name, g, codec)
		}
	}
}

// assertRows fails naming the row groups that differ.
func assertRows(t *testing.T, got, want []int64) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("row groups hold %v rows, want %v", got, want)
	}
}

// The measured defect: a 2,500-row Zstd file written as one row group came back
// uncompressed, in 1,000-row row groups.
func TestApplyCCLKeepsAZstdFileInOneRowGroup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zstd.parquet")
	if err := Write(mixedTable(2500), path, WriteOptions{Compression: CompressionZstd, RowGroupSize: 2500}); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	if err := ApplyCCL(context.Background(), path, "NEW('b') = ['id'] * 2"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	rows, codecs := layoutOf(t, path)
	assertRows(t, rows, []int64{2500})
	assertCodecs(t, codecs, map[string]compress.Compression{
		"id":    compress.Codecs.Zstd,
		"score": compress.Codecs.Zstd,
		"name":  compress.Codecs.Zstd,
		"maybe": compress.Codecs.Zstd,
		"b":     compress.Codecs.Zstd,
	})

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	col := dt.GetColByName("b")
	if col == nil {
		t.Fatalf("column \"b\" is missing from the result")
	}
	if got := fmt.Sprint(col.Get(10)); got != "20" {
		t.Errorf("column \"b\" value 10 = %s, want \"20\"", got)
	}
}

// The row group size comes from the original, not from Arrow's default of one
// row group per batch.
func TestApplyCCLKeepsTheRowGroupSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rg.parquet")
	if err := Write(mixedTable(2500), path, WriteOptions{Compression: CompressionSnappy, RowGroupSize: 700}); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	if err := ApplyCCL(context.Background(), path, "NEW('b') = 1"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	rows, codecs := layoutOf(t, path)
	assertRows(t, rows, []int64{700, 700, 700, 400})
	assertCodecs(t, codecs, map[string]compress.Compression{
		"id":    compress.Codecs.Snappy,
		"score": compress.Codecs.Snappy,
		"name":  compress.Codecs.Snappy,
		"maybe": compress.Codecs.Snappy,
		"b":     compress.Codecs.Snappy,
	})
}

// Each column keeps its own codec, and a column the script adds takes the
// original's first one, so a file that mixed codecs stays mixed.
func TestApplyCCLKeepsEachColumnsCodec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "per-column.parquet")
	writeTwoColumns(t, path, 30)

	if err := ApplyCCL(context.Background(), path, "NEW('c') = ['a'] + ['b']"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	_, codecs := layoutOf(t, path)
	assertCodecs(t, codecs, map[string]compress.Compression{
		"a": compress.Codecs.Zstd,
		"b": compress.Codecs.Snappy,
		"c": compress.Codecs.Zstd,
	})
}

// writeTwoColumns writes 30 rows of two int64 columns, a Zstd and a Snappy, which
// WriteOptions cannot express because it sets one codec for the whole file.
func writeTwoColumns(t *testing.T, path string, n int) {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "a", Type: arrow.PrimitiveTypes.Int64},
		{Name: "b", Type: arrow.PrimitiveTypes.Int64},
	}, nil)
	b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer b.Release()
	for i := range n {
		b.Field(0).(*array.Int64Builder).Append(int64(i))
		b.Field(1).(*array.Int64Builder).Append(int64(i * 10))
	}
	rec := b.NewRecord()
	defer rec.Release()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w, err := pqarrow.NewFileWriter(
		rec.Schema(),
		f,
		parquet.NewWriterProperties(
			parquet.WithCompressionFor("a", compress.Codecs.Zstd),
			parquet.WithCompressionFor("b", compress.Codecs.Snappy),
		),
		pqarrow.DefaultWriterProps(),
	)
	if err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	// Close closes the sink too, so f is not closed again here.
	if err := w.Write(rec); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// One WriteOptions replaces both, as Write uses it.
func TestApplyCCLTakesWriteOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opts.parquet")
	if err := Write(mixedTable(2500), path, WriteOptions{Compression: CompressionZstd}); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	err := ApplyCCL(context.Background(), path, "NEW('b') = 1", WriteOptions{Compression: CompressionGzip, RowGroupSize: 1000})
	if err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	rows, codecs := layoutOf(t, path)
	assertRows(t, rows, []int64{1000, 1000, 500})
	assertCodecs(t, codecs, map[string]compress.Compression{
		"id":    compress.Codecs.Gzip,
		"score": compress.Codecs.Gzip,
		"name":  compress.Codecs.Gzip,
		"maybe": compress.Codecs.Gzip,
		"b":     compress.Codecs.Gzip,
	})
}

// Settings Write refuses are refused here too, before the file is read, so
// nothing on disk moves.
func TestApplyCCLRefusesSettingsItCannotUse(t *testing.T) {
	cases := []struct {
		name string
		opts []WriteOptions
	}{
		{"negative row group size", []WriteOptions{{RowGroupSize: -1}}},
		{"two settings", []WriteOptions{{}, {}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "bad.parquet")
			if err := Write(mixedTable(50), path); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			if err := ApplyCCL(context.Background(), path, "NEW('b') = 1", c.opts...); err == nil {
				t.Fatal("ApplyCCL accepted settings it cannot use")
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("the original file is gone: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Error("the original file changed despite the refusal")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Errorf("leftover files in the directory: %v", entries)
			}
		})
	}
}

// The old implementation wrote through <path>.tmp, so a file of that name in
// the same directory was truncated and then removed. The temporary file has a
// name of its own, so a file of that name is nobody's business.
func TestApplyCCLLeavesAFileNamedLikeItsTempAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mine.parquet")
	if err := Write(mixedTable(50), path); err != nil {
		t.Fatal(err)
	}
	sibling := path + ".tmp"
	if err := os.WriteFile(sibling, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ApplyCCL(context.Background(), path, "NEW('b') = 1"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	got, err := os.ReadFile(sibling)
	if err != nil {
		t.Fatalf("the file at %s is gone: %v", sibling, err)
	}
	if string(got) != "mine" {
		t.Errorf("%s holds %q, want %q", sibling, got, "mine")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("directory holds %d files, want 2: %v", len(entries), entries)
	}
}

// An input with no rows leaves the original alone rather than replacing it with
// an empty file.
func TestApplyCCLLeavesAnEmptyFileAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.parquet")
	if err := Write(mixedTable(0), path); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := ApplyCCL(context.Background(), path, "NEW('b') = 1"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("an input with no rows was replaced: %d bytes became %d", len(before), len(after))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("leftover files in the directory: %v", entries)
	}
}
