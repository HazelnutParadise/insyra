package arrowipc_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func streamFixture(t *testing.T) []byte {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "text", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "score", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "count", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "status", Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: arrow.BinaryTypes.String, Ordered: true}, Nullable: true},
		{Name: "refs", Type: arrow.ListOf(arrow.PrimitiveTypes.Int64), Nullable: true},
	}, nil)
	var out bytes.Buffer
	w, err := ipc.NewFileWriter(&out, ipc.WithSchema(schema), ipc.WithZstd())
	if err != nil {
		t.Fatal(err)
	}
	b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer b.Release()
	for batch := 0; batch < 3; batch++ {
		for row := 0; row < 3; row++ {
			if row == 1 {
				for _, builder := range b.Fields() {
					builder.AppendNull()
				}
				continue
			}
			id := int64(0)
			if row == 0 {
				id = 9007199254740993 + int64(batch)
			}
			b.Field(0).(*array.Int64Builder).Append(id)
			b.Field(1).(*array.StringBuilder).Append("台灣 🐈")
			b.Field(2).(*array.Float64Builder).Append(1.25)
			b.Field(3).(*array.Int32Builder).Append(7)
			if err := b.Field(4).(*array.BinaryDictionaryBuilder).AppendString("Traced"); err != nil {
				t.Fatal(err)
			}
			list := b.Field(5).(*array.ListBuilder)
			list.Append(true)
			list.ValueBuilder().(*array.Int64Builder).Append(id)
		}
		rec := b.NewRecordBatch()
		err = w.Write(rec)
		rec.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

var selectedScalarColumns = []string{"status", "count", "score", "text", "id"}

func collectTables(t *testing.T, tables <-chan *insyra.DataTable, errs <-chan error) ([]*insyra.DataTable, error) {
	t.Helper()
	var out []*insyra.DataTable
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case table, ok := <-tables:
			if !ok {
				return out, <-errs
			}
			out = append(out, table)
		case <-timeout.C:
			t.Fatal("stream did not close")
			return nil, nil
		}
	}
}

func TestSelectedBatchesOwnExactCells(t *testing.T) {
	data := streamFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tables, errs := streamFrom(ctx, bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns(selectedScalarColumns)}, 2)
	got, err := collectTables(t, tables, errs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("got %d tables, want 6", len(got))
	}
	clear(data)
	var ids []any
	for i, dt := range got {
		rows, cols := dt.Size()
		wantRows := 2
		if i%2 == 1 {
			wantRows = 1
		}
		if rows != wantRows || cols != 5 {
			t.Fatalf("table %d shape %dx%d", i, rows, cols)
		}
		for col, name := range selectedScalarColumns {
			if dt.GetColByNumber(col).GetName() != name {
				t.Fatalf("column %d not %s", col, name)
			}
		}
		ids = append(ids, dt.GetColByNumber(4).Data()...)
		if i%2 == 0 {
			want := []any{"Traced", int32(7), float64(1.25), "台灣 🐈", int64(9007199254740993 + i/2)}
			for j, v := range want {
				if !reflect.DeepEqual(dt.GetElementByNumberIndex(0, j), v) {
					t.Fatalf("cell %d/%d = %#v, want %#v", i, j, dt.GetElementByNumberIndex(0, j), v)
				}
			}
			for j := 0; j < 5; j++ {
				if dt.GetElementByNumberIndex(1, j) != nil {
					t.Fatal("null slot changed")
				}
			}
		}
	}
	want := []any{int64(9007199254740993), nil, int64(0), int64(9007199254740994), nil, int64(0), int64(9007199254740995), nil, int64(0)}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("IDs %#v", ids)
	}
}

func TestUnsupportedSelectionAndInvalidOptions(t *testing.T) {
	data := streamFixture(t)
	for _, tc := range []struct {
		columns []string
		size    int
		match   string
	}{
		{nil, 2, "refs"}, {[]string{"refs"}, 2, "list"}, {[]string{"absent"}, 2, "absent"},
		{[]string{"id", "id"}, 2, "id"}, {[]string{"id"}, 0, "batchSize"}, {[]string{"id"}, -1, "batchSize"},
	} {
		t.Run(tc.match, func(t *testing.T) {
			tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns(tc.columns)}, tc.size)
			got, err := collectTables(t, tables, errs)
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), tc.match) {
				t.Fatalf("tables=%d err=%v, want %s", len(got), err, tc.match)
			}
		})
	}
}

func TestStreamCancellation(t *testing.T) {
	data := streamFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tables, errs := streamFrom(ctx, bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 1)
	got, err := collectTables(t, tables, errs)
	if len(got) != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancel: %d %v", len(got), err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	tables, errs = streamFrom(ctx, bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 1)
	select {
	case _, ok := <-tables:
		if !ok {
			t.Fatalf("no first table: %v", <-errs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no first table")
	}
	cancel()
	_, err = collectTables(t, tables, errs)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked-send cancel: %v", err)
	}
}

func TestPathAndCallerOwnedSource(t *testing.T) {
	data := streamFixture(t)
	path := filepath.Join(t.TempDir(), "sample.feather")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := arrowipc.Inspect(path)
	if err != nil || info.NumBatches != 3 {
		t.Fatalf("inspect: %#v %v", info, err)
	}
	tables, errs := streamPath(context.Background(), path, arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 4)
	got, err := collectTables(t, tables, errs)
	if err != nil || len(got) != 3 {
		t.Fatalf("path: %d %v", len(got), err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := arrowipc.InspectFrom(context.Background(), f, int64(len(data))); err != nil {
		t.Fatal(err)
	}
	tables, errs = streamFrom(context.Background(), f, int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 4)
	if _, err := collectTables(t, tables, errs); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stat(); err != nil {
		t.Fatal("reader closed caller source:", err)
	}
}

func TestInvalidSourcesAndContexts(t *testing.T) {
	valid := streamFixture(t)
	for name, data := range map[string][]byte{"notArrow": []byte("this is definitely not an Arrow IPC file"), "truncated": valid[:len(valid)-8], "empty": nil} {
		t.Run(name, func(t *testing.T) {
			if _, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data))); err == nil {
				t.Fatal("inspection succeeded")
			}
			tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 2)
			if got, err := collectTables(t, tables, errs); err == nil || len(got) != 0 {
				t.Fatalf("tables=%d err=%v", len(got), err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := arrowipc.InspectFrom(ctx, bytes.NewReader(valid), int64(len(valid))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := arrowipc.InspectFrom(nil, //nolint:staticcheck // Test the documented nil-context rejection.
		bytes.NewReader(valid), int64(len(valid))); err == nil {
		t.Fatal("nil context accepted")
	}
	tables, errs := streamFrom(nil, //nolint:staticcheck // Test the documented nil-context rejection.
		bytes.NewReader(valid), int64(len(valid)), arrowipc.ReadOptions{}, 1)
	if got, err := collectTables(t, tables, errs); err == nil || len(got) != 0 {
		t.Fatalf("nil context: %d %v", len(got), err)
	}
}
