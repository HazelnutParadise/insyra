package parquet

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/arrow/memory"
	parquetlib "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// Parquet reads from an io.ReaderAt and writes to an io.Writer, and the path
// versions are the same code.

func sampleTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3, 4, 5).SetName("n"),
		insyra.NewDataList("a", "b", "c", "d", "e").SetName("s"),
	)
}

func writeDurationParquet(t *testing.T) string {
	t.Helper()
	mem := memory.DefaultAllocator
	durationSchema := arrow.NewSchema([]arrow.Field{{Name: "d", Type: arrow.FixedWidthTypes.Duration_ns, Nullable: true}}, nil)
	schemaBytes := flight.SerializeSchema(durationSchema, mem)
	// Keep the file column physically INT64 while preserving the Arrow Duration
	// annotation that v18 pqarrow cannot write directly.
	metadata := arrow.NewMetadata([]string{"ARROW:schema"}, []string{base64.StdEncoding.EncodeToString(schemaBytes)})
	physicalSchema := arrow.NewSchema([]arrow.Field{{Name: "d", Type: arrow.PrimitiveTypes.Int64, Nullable: true}}, &metadata)
	b := array.NewInt64Builder(mem)
	b.Append(1)
	b.Append(2)
	b.AppendNull()
	arr := b.NewArray()
	b.Release()
	defer arr.Release()
	record := array.NewRecordBatch(physicalSchema, []arrow.Array{arr}, 3)
	defer record.Release()
	path := filepath.Join(t.TempDir(), "duration.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w, err := pqarrow.NewFileWriter(physicalSchema, f, parquetlib.NewWriterProperties(), pqarrow.NewArrowWriterProperties(pqarrow.WithStoreSchema()))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(record); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertDurationValues(t *testing.T, got []any) {
	t.Helper()
	want := []any{int64(1), int64(2), nil}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("duration values=%#v want=%#v", got, want)
	}
}

func TestDurationSchemaReadsAsInt64AcrossAPIs(t *testing.T) {
	path := writeDurationParquet(t)
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertDurationValues(t, dt.GetColByNumber(0).Data())
	col, err := ReadColumn(context.Background(), path, "d", ReadColumnOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertDurationValues(t, col.Data())
	var streamed []any
	dc, ec := Stream(context.Background(), path, ReadOptions{}, 1)
	for batch := range dc {
		streamed = append(streamed, batch.GetColByNumber(0).Data()...)
	}
	if err := <-ec; err != nil {
		t.Fatal(err)
	}
	assertDurationValues(t, streamed)
	filtered, err := FilterWithCCL(context.Background(), path, "A > 0")
	if err != nil {
		t.Fatal(err)
	}
	if got := filtered.GetElementByNumberIndex(0, 0); got != int64(1) {
		t.Fatalf("filtered duration=%v (%T), want int64(1)", got, got)
	}
	if err := ApplyCCL(context.Background(), path, "NEW('copy') = A"); err != nil {
		t.Fatal(err)
	}
	updated, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertDurationValues(t, updated.GetColByNumber(0).Data())
}

func TestGetValDurationAsInt64(t *testing.T) {
	b := array.NewDurationBuilder(memory.DefaultAllocator, arrow.FixedWidthTypes.Duration_ns.(*arrow.DurationType))
	b.Append(1)
	b.Append(2)
	b.AppendNull()
	arr := b.NewArray()
	b.Release()
	defer arr.Release()
	got := []any{getVal(arr, 0), getVal(arr, 1), nil}
	want := []any{int64(1), int64(2), nil}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("duration values=%#v want=%#v", got, want)
	}
}

func TestReaderPropertiesKeepArrowV17PageLimits(t *testing.T) {
	props := readerProperties()
	if props.MaxCompressedPageSize != 1<<40 || props.MaxUncompressedPageSize != 1<<40 {
		t.Fatalf("page limits compressed=%d uncompressed=%d", props.MaxCompressedPageSize, props.MaxUncompressedPageSize)
	}
}

func TestWriteToMatchesWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.parquet")
	if err := Write(sampleTable(), path); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteTo(sampleTable(), &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), onDisk) {
		t.Fatalf("WriteTo wrote %d bytes that differ from the %d Write put on disk", buf.Len(), len(onDisk))
	}
}

func TestWriteUpdatesHardLinkInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "table.parquet")
	alias := filepath.Join(dir, "alias.parquet")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
	if err := Write(sampleTable(), path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(alias)
	if err != nil {
		t.Fatal(err)
	}
	contextPath := filepath.Join(dir, "context.parquet")
	if err := WriteContext(context.Background(), sampleTable(), contextPath); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("hard link has %d bytes, want newly written Parquet content (%d)", len(got), len(want))
	}
}

// closeRecorder is a writer that remembers whether anyone closed it.
type closeRecorder struct {
	bytes.Buffer
	closed bool
}

func (c *closeRecorder) Close() error { c.closed = true; return nil }

func TestWriteToLeavesTheWriterOpen(t *testing.T) {
	w := &closeRecorder{}
	if err := WriteTo(sampleTable(), w); err != nil {
		t.Fatal(err)
	}
	if w.closed {
		t.Fatal("WriteTo closed a writer it does not own")
	}
}

func TestReadFromMatchesRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.parquet")
	if err := Write(sampleTable(), path); err != nil {
		t.Fatal(err)
	}
	fromFile, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadFrom(context.Background(), bytes.NewReader(content), int64(len(content)), ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromReader.ColNames(), fromFile.ColNames()) || !reflect.DeepEqual(fromReader.To2DSlice(), fromFile.To2DSlice()) {
		t.Fatalf("ReadFrom %v %v, Read %v %v", fromReader.ColNames(), fromReader.To2DSlice(), fromFile.ColNames(), fromFile.To2DSlice())
	}
}

func TestStreamFromYieldsEveryRow(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTo(sampleTable(), &buf); err != nil {
		t.Fatal(err)
	}
	content := buf.Bytes()
	total := 0
	dtChan, errChan := StreamFrom(context.Background(), bytes.NewReader(content), int64(len(content)), ReadOptions{}, 2)
	for dt := range dtChan {
		rows, _ := dt.Size()
		total += rows
	}
	if err := <-errChan; err != nil {
		t.Fatalf("StreamFrom: %v", err)
	}
	if total != 5 {
		t.Fatalf("streamed %d rows, want 5", total)
	}
}
