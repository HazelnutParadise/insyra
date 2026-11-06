package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

type scalarRow struct {
	id    *int64
	text  *string
	score *float64
	rank  *int32
	state *string
}

func scalarFixture(t *testing.T) []byte {
	t.Helper()
	dict := &arrow.DictionaryType{
		IndexType: arrow.PrimitiveTypes.Int8, ValueType: arrow.BinaryTypes.String, Ordered: true,
	}
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "text", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "score", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "rank", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "state", Type: dict, Nullable: true},
	}, nil)
	rows := []scalarRow{
		{ptr(int64(9007199254740993)), ptr("台灣"), ptr(1.25), ptr(int32(12)), ptr("ready")},
		{nil, ptr("🐈"), nil, nil, nil},
		{ptr(int64(0)), nil, ptr(-2.5), ptr(int32(0)), ptr("waiting")},
		{ptr(int64(4)), ptr("utf8"), ptr(float64(0)), ptr(int32(-7)), ptr("ready")},
		{ptr(int64(5)), ptr("last"), ptr(3.125), ptr(int32(9)), ptr("waiting")},
		{ptr(int64(6)), ptr("done"), ptr(-0.5), ptr(int32(3)), ptr("ready")},
	}

	var out bytes.Buffer
	writer, err := ipc.NewFileWriter(&out, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	for _, batch := range [][]scalarRow{rows[:4], rows[4:]} {
		for _, row := range batch {
			appendNullable(builder.Field(0).(*array.Int64Builder), row.id)
			appendNullable(builder.Field(1).(*array.StringBuilder), row.text)
			appendNullable(builder.Field(2).(*array.Float64Builder), row.score)
			appendNullable(builder.Field(3).(*array.Int32Builder), row.rank)
			if row.state == nil {
				builder.Field(4).(*array.BinaryDictionaryBuilder).AppendNull()
			} else if err := builder.Field(4).(*array.BinaryDictionaryBuilder).AppendString(*row.state); err != nil {
				t.Fatal(err)
			}
		}
		record := builder.NewRecordBatch()
		if err := writer.Write(record); err != nil {
			record.Release()
			t.Fatal(err)
		}
		record.Release()
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.Clone(out.Bytes())
}

func ptr[T any](v T) *T { return &v }

func appendNullable[T any](builder interface {
	Append(T)
	AppendNull()
}, value *T) {
	if value == nil {
		builder.AppendNull()
		return
	}
	builder.Append(*value)
}

func writeRecords(t *testing.T, schema *arrow.Schema, build func(*array.RecordBuilder)) []byte {
	t.Helper()
	var out bytes.Buffer
	writer, err := ipc.NewFileWriter(&out, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	build(builder)
	record := builder.NewRecordBatch()
	if err := writer.Write(record); err != nil {
		record.Release()
		t.Fatal(err)
	}
	record.Release()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.Clone(out.Bytes())
}

type closableReaderAt struct {
	*bytes.Reader
	closed bool
}

func (r *closableReaderAt) Close() error {
	r.closed = true
	return nil
}

func drain(t *testing.T, tables <-chan *insyra.DataTable, errs <-chan error) ([]*insyra.DataTable, error) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	var result []*insyra.DataTable
	for {
		select {
		case table, ok := <-tables:
			if !ok {
				select {
				case err := <-errs:
					return result, err
				case <-timer.C:
					t.Fatal("timed out waiting for terminal error")
				}
			}
			result = append(result, table)
		case <-timer.C:
			t.Fatal("timed out draining table channel")
		}
	}
}

func TestStreamFromPreservesScalarsBatchesAndOwnership(t *testing.T) {
	data := scalarFixture(t)
	source := &closableReaderAt{Reader: bytes.NewReader(data)}
	tables, errs := streamFrom(context.Background(), source, int64(len(data)), arrowipc.ReadOptions{}, 3)
	got, err := drain(t, tables, errs)
	if err != nil {
		t.Fatal(err)
	}
	if source.closed {
		t.Fatal("StreamFrom closed its caller-owned source")
	}
	if len(got) != 3 {
		t.Fatalf("got %d output tables, want 3 (native boundaries retained)", len(got))
	}
	for i, rows := range []int{3, 1, 2} {
		if got[i].GetColByNumber(0).Len() != rows {
			t.Fatalf("table %d has %d rows, want %d", i, got[i].GetColByNumber(0).Len(), rows)
		}
	}
	wantIDs := []any{int64(9007199254740993), nil, int64(0), int64(4), int64(5), int64(6)}
	wantText := []any{"台灣", "🐈", nil, "utf8", "last", "done"}
	wantScores := []any{float64(1.25), nil, float64(-2.5), float64(0), float64(3.125), float64(-0.5)}
	wantRanks := []any{int32(12), nil, int32(0), int32(-7), int32(9), int32(3)}
	wantStates := []any{"ready", nil, "waiting", "ready", "waiting", "ready"}
	for col, want := range [][]any{wantIDs, wantText, wantScores, wantRanks, wantStates} {
		var values []any
		for _, table := range got {
			values = append(values, table.GetColByNumber(col).Data()...)
		}
		if !reflect.DeepEqual(values, want) {
			t.Fatalf("column %d values=%#v want=%#v", col, values, want)
		}
	}
	if got[0] == got[1] {
		t.Fatal("separate output batches share a DataTable")
	}
	first := got[0].GetColByNumber(0).Data()
	got[0].UpdateColByNumber(0, insyra.NewDataList(int64(99), int64(98), int64(97)).SetName("id"))
	if !reflect.DeepEqual(got[1].GetColByNumber(0).Data(), []any{int64(4)}) {
		t.Fatal("mutating one output table changed another table")
	}
	copy(data, bytes.Repeat([]byte{0}, len(data)))
	if !reflect.DeepEqual(got[1].GetColByNumber(0).Data(), []any{int64(4)}) || first[0] != int64(9007199254740993) {
		t.Fatal("output cells depend on the source bytes or were not independently owned")
	}
}

func TestStreamSelectsColumnsInRequestOrder(t *testing.T) {
	data := scalarFixture(t)
	tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{
		Columns: namedColumns([]string{"state", "id"}),
	}, 2)
	got, err := drain(t, tables, errs)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	var ids, states []any
	for _, table := range got {
		names = []string{table.GetColNameByNumber(0), table.GetColNameByNumber(1)}
		ids = append(ids, table.GetColByNumber(1).Data()...)
		states = append(states, table.GetColByNumber(0).Data()...)
		if table.GetColByNumber(0).Len() > 2 {
			t.Fatalf("batch exceeded row cap: %d", table.GetColByNumber(0).Len())
		}
	}
	if !reflect.DeepEqual(names, []string{"state", "id"}) {
		t.Fatalf("column order=%v want [state id]", names)
	}
	if !reflect.DeepEqual(ids, []any{int64(9007199254740993), nil, int64(0), int64(4), int64(5), int64(6)}) {
		t.Fatalf("selected ids=%#v", ids)
	}
	if !reflect.DeepEqual(states, []any{"ready", nil, "waiting", "ready", "waiting", "ready"}) {
		t.Fatalf("selected states=%#v", states)
	}
}

func TestStreamPathAndInvalidColumnSelections(t *testing.T) {
	data := scalarFixture(t)
	path := filepath.Join(t.TempDir(), "fixture.arrow")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	tables, errs := streamPath(context.Background(), path, arrowipc.ReadOptions{}, 10)
	got, err := drain(t, tables, errs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].GetColByNumber(0).Len() != 4 || got[1].GetColByNumber(0).Len() != 2 {
		t.Fatalf("native record boundaries not retained: got %d tables", len(got))
	}

	for _, columns := range [][]string{{"missing"}, {"id", "id"}} {
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns(columns)}, 10)
		output, err := drain(t, tables, errs)
		if err == nil || len(output) != 0 {
			t.Fatalf("columns %v: got %d tables and error %v", columns, len(output), err)
		}
	}
}

func TestStreamRejectsAmbiguousSelectedName(t *testing.T) {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "same", Type: arrow.PrimitiveTypes.Int32},
		{Name: "same", Type: arrow.PrimitiveTypes.Int32},
	}, nil)
	data := writeRecords(t, schema, func(builder *array.RecordBuilder) {
		builder.Field(0).(*array.Int32Builder).Append(1)
		builder.Field(1).(*array.Int32Builder).Append(2)
	})
	tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"same"})}, 1)
	got, err := drain(t, tables, errs)
	if err == nil || len(got) != 0 || !strings.Contains(strings.ToLower(err.Error()), "same") {
		t.Fatalf("ambiguous selection: tables=%d err=%v", len(got), err)
	}
}

func TestInspectListAndStreamExplicitlyRefusesListTypes(t *testing.T) {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "value", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "items", Type: arrow.ListOf(arrow.PrimitiveTypes.Int64), Nullable: true},
		{Name: "nested", Type: arrow.ListOf(arrow.ListOf(arrow.PrimitiveTypes.Int64)), Nullable: true},
	}, nil)
	data := writeRecords(t, schema, func(builder *array.RecordBuilder) {
		builder.Field(0).(*array.Int32Builder).Append(7)
		list := builder.Field(1).(*array.ListBuilder)
		list.Append(true)
		list.ValueBuilder().(*array.Int64Builder).Append(9007199254740993)
		outer := builder.Field(2).(*array.ListBuilder)
		outer.Append(true)
		inner := outer.ValueBuilder().(*array.ListBuilder)
		inner.Append(true)
		inner.ValueBuilder().(*array.Int64Builder).Append(8)
	})
	info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if info.Columns[1].Type != arrow.ListOf(arrow.PrimitiveTypes.Int64).String() ||
		info.Columns[2].Type != arrow.ListOf(arrow.ListOf(arrow.PrimitiveTypes.Int64)).String() {
		t.Fatalf("inspection did not preserve list schema: %#v", info.Columns)
	}
	for _, column := range []string{"items", "nested"} {
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{column})}, 1)
		got, err := drain(t, tables, errs)
		if err == nil || len(got) != 0 || !strings.Contains(err.Error(), column) {
			t.Fatalf("selected %s: tables=%d err=%v", column, len(got), err)
		}
	}
	tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"value"})}, 1)
	got, err := drain(t, tables, errs)
	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0].GetColByNumber(0).Data(), []any{int32(7)}) {
		t.Fatalf("supported selection: tables=%d err=%v", len(got), err)
	}
}

func TestStreamRejectsNilAndCancelledContexts(t *testing.T) {
	data := scalarFixture(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, ctx := range map[string]context.Context{"nil": nil, "cancelled": cancelled} {
		t.Run(name, func(t *testing.T) {
			tables, errs := streamFrom(ctx, bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 2)
			got, err := drain(t, tables, errs)
			if err == nil || len(got) != 0 {
				t.Fatalf("got %d tables and error %v", len(got), err)
			}
			if name == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("error %v does not wrap context.Canceled", err)
			}
		})
	}
}

type bodyReadSignal struct {
	data      []byte
	footerAt  int64
	bodyRead  chan struct{}
	signalOne sync.Once
}

func (r *bodyReadSignal) ReadAt(p []byte, off int64) (int, error) {
	n, err := bytes.NewReader(r.data).ReadAt(p, off)
	if off < r.footerAt && off+int64(len(p)) > 8 {
		r.signalOne.Do(func() { close(r.bodyRead) })
	}
	return n, err
}

func TestStreamCancellationWhileSendIsBlocked(t *testing.T) {
	data := scalarFixture(t)
	footerLen := int64(binary.LittleEndian.Uint32(data[len(data)-10:]))
	source := &bodyReadSignal{
		data: data, footerAt: int64(len(data)) - 10 - footerLen, bodyRead: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	tables, errs := streamFrom(ctx, source, int64(len(data)), arrowipc.ReadOptions{}, 10)
	select {
	case <-source.bodyRead:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("stream did not read a record body")
	}
	cancel()
	// Keep the output channel without a receiver until the producer exits.
	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("blocked send returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled producer stayed blocked")
	}
	for range tables {
		t.Fatal("cancelled producer delivered a table without a receiver")
	}
}

func TestStreamReportsTruncatedAndNonArrowSources(t *testing.T) {
	valid := scalarFixture(t)
	for name, data := range map[string][]byte{
		"truncated": valid[:len(valid)-1],
		"non-arrow": []byte("this is not an Arrow IPC file"),
	} {
		t.Run(name, func(t *testing.T) {
			tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 1)
			got, err := drain(t, tables, errs)
			if err == nil || len(got) != 0 {
				t.Fatalf("got %d tables and error %v", len(got), err)
			}
		})
	}
}

func TestStreamFromDoesNotCloseSourceOnReadError(t *testing.T) {
	source := &closableReaderAt{Reader: bytes.NewReader([]byte("not an Arrow IPC file"))}
	tables, errs := streamFrom(context.Background(), source, int64(source.Size()), arrowipc.ReadOptions{}, 1)
	_, err := drain(t, tables, errs)
	if err == nil {
		t.Fatal("expected invalid-source error")
	}
	if source.closed {
		t.Fatal("StreamFrom closed its caller-owned source after an error")
	}
}

func TestStreamChannelsCloseOnInvalidInput(t *testing.T) {
	tables, errs := streamFrom(context.Background(), readerAtPanic{}, 64, arrowipc.ReadOptions{}, 1)
	_, err := drain(t, tables, errs)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "reader") {
		t.Fatalf("panic source error=%v", err)
	}
}

type readerAtPanic struct{}

func (readerAtPanic) ReadAt([]byte, int64) (int, error) { panic(fmt.Errorf("source panic")) }

var _ io.ReaderAt = readerAtPanic{}
