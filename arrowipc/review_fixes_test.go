package arrowipc_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// writeBatches writes one int64 column "n" as batches of the given row counts.
func writeBatches(t *testing.T, w io.Writer, rows ...int) {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{{Name: "n", Type: arrow.PrimitiveTypes.Int64}}, nil)
	writer, err := ipc.NewFileWriter(w, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	for _, count := range rows {
		ints := builder.Field(0).(*array.Int64Builder)
		ints.Reserve(count)
		for i := 0; i < count; i++ {
			ints.UnsafeAppend(int64(i))
		}
		record := builder.NewRecordBatch()
		if err := writer.Write(record); err != nil {
			t.Fatal(err)
		}
		record.Release()
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

// A file with more than 65,536 batches is read in full, as Inspect reports it.
func TestManyBatchesInspectAndStreamAgree(t *testing.T) {
	const batches = 70000
	rows := make([]int, batches)
	for i := range rows {
		rows[i] = 1
	}
	var buf bytes.Buffer
	writeBatches(t, &buf, rows...)
	data := buf.Bytes()
	info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil || info.NumBatches != batches {
		t.Fatalf("Inspect: %d batches, %v", info.NumBatches, err)
	}
	// Counted straight off the sequence: drain's per-table timeout is too
	// short for 70,000 tables on a slow runner.
	count := 0
	for _, err := range arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 10) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != batches {
		t.Fatalf("Stream: %d tables, want %d", count, batches)
	}
}

// A batch larger than 256 MiB is read when MaxBatchBytes allows it. It needs
// about 1 GiB of memory, so it runs only when asked for.
func TestBatchLargerThanTheDefaultLimitWhenAllowed(t *testing.T) {
	if os.Getenv("INSYRA_ARROWIPC_LARGE_TESTS") != "1" {
		t.Skip("set INSYRA_ARROWIPC_LARGE_TESTS=1 to read a 288 MB batch")
	}
	const rows = 36_000_000
	path := filepath.Join(t.TempDir(), "large.arrow")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writeBatches(t, f, rows)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	total := 0
	for table, err := range arrowipc.Stream(context.Background(), path, arrowipc.ReadOptions{MaxBatchBytes: 512 << 20, MaxOutputBytes: 1 << 30}, 4_000_000) {
		if err != nil {
			t.Fatal(err)
		}
		total += table.NumRows()
	}
	if total != rows {
		t.Fatalf("read %d rows, want %d", total, rows)
	}
	for _, err := range arrowipc.Stream(context.Background(), path, arrowipc.ReadOptions{}, 4_000_000) {
		if err == nil || !strings.Contains(err.Error(), "MaxBatchBytes") {
			t.Fatalf("default limit: %v, want a MaxBatchBytes error", err)
		}
	}
}

func durationFile(t *testing.T, unit arrow.TimeUnit, values ...int64) []byte {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{{Name: "d", Type: &arrow.DurationType{Unit: unit}}}, nil)
	var buf bytes.Buffer
	writer, err := ipc.NewFileWriter(&buf, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	for _, v := range values {
		builder.Field(0).(*array.DurationBuilder).Append(arrow.Duration(v))
	}
	record := builder.NewRecordBatch()
	defer record.Release()
	if err := writer.Write(record); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// 1500 milliseconds and 1500 nanoseconds are different durations.
func TestDurationKeepsItsUnit(t *testing.T) {
	for unit, want := range map[arrow.TimeUnit]time.Duration{
		arrow.Second:      1500 * time.Second,
		arrow.Millisecond: 1500 * time.Millisecond,
		arrow.Microsecond: 1500 * time.Microsecond,
		arrow.Nanosecond:  1500,
	} {
		data := durationFile(t, unit, 1500)
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 10)
		got, err := drain(t, tables, errs)
		if err != nil || len(got) != 1 {
			t.Fatalf("%s: %v", unit, err)
		}
		if cell := got[0].GetColByNumber(0).Data()[0]; cell != want {
			t.Fatalf("%s: cell = %#v, want %v", unit, cell, want)
		}
	}
	data := durationFile(t, arrow.Second, 1<<62)
	tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 10)
	if got, err := drain(t, tables, errs); err == nil || len(got) != 0 {
		t.Fatalf("a duration time.Duration cannot hold: %d tables, %v", len(got), err)
	}
}

// A size larger than the source is a truncated source, never a clean io.EOF.
func TestWrongSizeIsAnUnexpectedEOF(t *testing.T) {
	data := scalarFixture(t)
	_, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data))+100)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want io.ErrUnexpectedEOF", err)
	}
}

// Fields that share a name are renamed the way DataTable.AppendCols renames.
func TestRepeatedFieldNamesAreRenamed(t *testing.T) {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "x", Type: arrow.PrimitiveTypes.Int64},
		{Name: "x", Type: arrow.PrimitiveTypes.Int64},
		{Name: "x_1", Type: arrow.PrimitiveTypes.Int64},
	}, nil)
	var buf bytes.Buffer
	writer, err := ipc.NewFileWriter(&buf, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	for i := range 3 {
		builder.Field(i).(*array.Int64Builder).Append(int64(i))
	}
	record := builder.NewRecordBatch()
	defer record.Release()
	if err := writer.Write(record); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 10)
	got, err := drain(t, tables, errs)
	if err != nil || len(got) != 1 {
		t.Fatalf("%d tables, %v", len(got), err)
	}
	if names := got[0].ColNames(); !reflect.DeepEqual(names, []string{"x", "x_1", "x_1_1"}) {
		t.Fatalf("names = %v", names)
	}
}

// buffer_limits.go watches the reads arrow-go v18.8.0's FileReader makes. A
// different version may read in another order, so moving to one means
// checking that code against it and changing this version. go test's build
// information does not list dependencies, so go.mod is read instead.
func TestArrowGoVersionTheReadWatcherWasCheckedAgainst(t *testing.T) {
	mod, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(mod), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "github.com/apache/arrow-go/v18" {
			if fields[1] != "v18.8.0" {
				t.Fatalf("arrow-go is %s; check buffer_limits.go against it, then update this test", fields[1])
			}
			return
		}
	}
	t.Fatal("go.mod does not require github.com/apache/arrow-go/v18")
}
