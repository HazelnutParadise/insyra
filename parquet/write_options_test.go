package parquet

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// mixedTable is a table with one column of every type the writer infers here,
// plus a column with nulls, so a row group split has to carry all of them.
func mixedTable(n int) *insyra.DataTable {
	ids := make([]any, n)
	scores := make([]any, n)
	names := make([]any, n)
	maybe := make([]any, n)
	for i := range n {
		ids[i] = int64(i)
		scores[i] = float64(i) * 1.5
		names[i] = fmt.Sprintf("n%d", i)
		if i%3 == 0 {
			maybe[i] = nil
		} else {
			maybe[i] = int64(i)
		}
	}
	return insyra.NewDataTable(
		insyra.NewDataList(ids...).SetName("id"),
		insyra.NewDataList(scores...).SetName("score"),
		insyra.NewDataList(names...).SetName("name"),
		insyra.NewDataList(maybe...).SetName("maybe"),
	)
}

// referenceBytes writes dt the way Write wrote before it took options, through
// Arrow's own WriteTable, so the comparison is against the old output rather
// than against WriteTo reading its own tail. chunkSize is WriteTable's chunk
// size, which is WriteOptions.RowGroupSize's counterpart.
func referenceBytes(t *testing.T, dt insyra.IDataTable, chunkSize int64) []byte {
	t.Helper()
	tbl, err := dataTableToArrowTable(context.Background(), dt)
	if err != nil {
		t.Fatal(err)
	}
	defer tbl.Release()
	var buf bytes.Buffer
	w, err := pqarrow.NewFileWriter(
		tbl.Schema(),
		&buf,
		parquet.NewWriterProperties(parquet.WithCreatedBy(fmt.Sprintf("go-insyra v%s", insyra.Version))),
		pqarrow.DefaultWriterProps(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteTable(tbl, chunkSize); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestWriteWithoutOptionsIsTheOldOutput(t *testing.T) {
	dt := mixedTable(30)
	ref := referenceBytes(t, dt, 1024*1024)

	var a bytes.Buffer
	if err := WriteTo(dt, &a); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, a.Bytes()) {
		t.Fatalf("WriteTo with no options wrote %d bytes, the old output is %d", a.Len(), len(ref))
	}

	var b bytes.Buffer
	if err := WriteTo(dt, &b, WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, b.Bytes()) {
		t.Fatalf("WriteTo with an empty WriteOptions wrote %d bytes, the old output is %d", b.Len(), len(ref))
	}
}

// TestWriteRowGroupsMatchWriteTable pins the row group loop against Arrow's own
// WriteTable at a size that splits the table, so a row group WriteTo lays out
// differently from WriteTable would show here.
func TestWriteRowGroupsMatchWriteTable(t *testing.T) {
	dt := mixedTable(25)
	ref := referenceBytes(t, dt, 7)

	var buf bytes.Buffer
	if err := WriteTo(dt, &buf, WriteOptions{RowGroupSize: 7}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, buf.Bytes()) {
		t.Fatalf("WriteTo with row groups of 7 wrote %d bytes, WriteTable's chunked output is %d", buf.Len(), len(ref))
	}
}

func TestWriteRowGroupSize(t *testing.T) {
	dt := mixedTable(25)
	path := filepath.Join(t.TempDir(), "rg.parquet")
	if err := Write(dt, path, WriteOptions{RowGroupSize: 10}); err != nil {
		t.Fatal(err)
	}

	info, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []int64
	for _, rg := range info.RowGroups {
		got = append(got, rg.NumRows)
	}
	if want := []int64{10, 10, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("row groups hold %v rows, want %v", got, want)
	}

	back, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertSameColumns(t, dt, back)
}

// assertSameColumns compares every named column of want with the column of the
// same name in got.
func assertSameColumns(t *testing.T, want, got *insyra.DataTable) {
	t.Helper()
	for _, name := range want.ColNames() {
		w := want.GetColByName(name).Data()
		g := got.GetColByName(name).Data()
		if !reflect.DeepEqual(w, g) {
			t.Fatalf("column %s came back as %v, want %v", name, g, w)
		}
	}
}

func TestWriteCompression(t *testing.T) {
	cases := []struct {
		compression Compression
		codec       compress.Compression
	}{
		{CompressionNone, compress.Codecs.Uncompressed},
		{CompressionSnappy, compress.Codecs.Snappy},
		{CompressionGzip, compress.Codecs.Gzip},
		{CompressionBrotli, compress.Codecs.Brotli},
		{CompressionZstd, compress.Codecs.Zstd},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("codec%d", int(c.compression)), func(t *testing.T) {
			dt := mixedTable(40)
			var buf bytes.Buffer
			if err := WriteTo(dt, &buf, WriteOptions{Compression: c.compression}); err != nil {
				t.Fatal(err)
			}
			content := buf.Bytes()

			r, err := file.NewParquetReader(bytes.NewReader(content))
			if err != nil {
				t.Fatal(err)
			}
			rg := r.MetaData().RowGroup(0)
			for i := 0; i < rg.NumColumns(); i++ {
				chunk, err := rg.ColumnChunk(i)
				if err != nil {
					t.Fatal(err)
				}
				if got := chunk.Compression(); got != c.codec {
					t.Fatalf("column chunk %d used codec %d, want %d", i, got, c.codec)
				}
			}
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}

			back, err := ReadFrom(context.Background(), bytes.NewReader(content), int64(len(content)), ReadOptions{})
			if err != nil {
				t.Fatal(err)
			}
			assertSameColumns(t, dt, back)
		})
	}
}

func TestWriteRefusesSettingsItCannotUse(t *testing.T) {
	cases := []struct {
		name string
		opts []WriteOptions
	}{
		{"negative row group size", []WriteOptions{{RowGroupSize: -1}}},
		{"unknown compression", []WriteOptions{{Compression: Compression(99)}}},
		{"two settings", []WriteOptions{{}, {}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dt := mixedTable(5)

			dir := t.TempDir()
			path := filepath.Join(dir, "bad.parquet")
			if err := Write(dt, path, c.opts...); err == nil {
				t.Fatal("Write accepted settings it cannot use")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("Write left %d file(s) behind: %v", len(entries), entries)
			}

			var buf bytes.Buffer
			if err := WriteTo(dt, &buf, c.opts...); err == nil {
				t.Fatal("WriteTo accepted settings it cannot use")
			}
			if buf.Len() != 0 {
				t.Fatalf("WriteTo wrote %d bytes before refusing", buf.Len())
			}
		})
	}
}
