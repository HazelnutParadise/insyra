package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"runtime"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	flatbuffers "github.com/google/flatbuffers/go"
)

func schemaOnlyFixture(t *testing.T, schema *arrow.Schema) []byte {
	t.Helper()
	var b bytes.Buffer
	w, err := ipc.NewFileWriter(&b, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func footerTable(data []byte) flatbuffers.Table {
	start := len(data) - 10 - int(binary.LittleEndian.Uint32(data[len(data)-10:]))
	footer := data[start : len(data)-10]
	return flatbuffers.Table{Bytes: footer, Pos: flatbuffers.GetUOffsetT(footer)}
}
func indirectTable(parent flatbuffers.Table, slot flatbuffers.VOffsetT) flatbuffers.Table {
	return flatbuffers.Table{Bytes: parent.Bytes, Pos: parent.Indirect(parent.Pos + flatbuffers.UOffsetT(parent.Offset(slot)))}
}
func TestMalformedSchemaCountDoesNotAmplifyAllocation(t *testing.T) {
	data := schemaOnlyFixture(t, arrow.NewSchema([]arrow.Field{{Name: "x", Type: arrow.PrimitiveTypes.Int64}}, nil))
	schema := indirectTable(footerTable(data), 6)
	start := schema.Vector(flatbuffers.UOffsetT(schema.Offset(6)))
	binary.LittleEndian.PutUint32(schema.Bytes[start-4:start], 65536)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	runtime.ReadMemStats(&after)
	if err == nil {
		t.Fatal("corrupt schema count accepted")
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 512<<10 {
		t.Fatalf("%d-byte input allocated %d bytes before rejecting bad field count", len(data), allocated)
	}
}
func TestSchemaNestingLimit(t *testing.T) {
	typ := arrow.PrimitiveTypes.Int64
	for i := 0; i < 65; i++ {
		typ = arrow.ListOf(typ)
	}
	data := schemaOnlyFixture(t, arrow.NewSchema([]arrow.Field{{Name: "deep", Type: typ}}, nil))
	_, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatalf("deep schema: %v", err)
	}
}
func TestUnknownExtensionIsNotSilentlyReadAsStorage(t *testing.T) {
	md := arrow.NewMetadata([]string{"ARROW:extension:name", "ARROW:extension:metadata"}, []string{"insyra.test.unknown", "{}"})
	schema := arrow.NewSchema([]arrow.Field{{Name: "special", Type: arrow.PrimitiveTypes.Int64, Metadata: md}}, nil)
	data := writeRecords(t, schema, func(b *array.RecordBuilder) { b.Field(0).(*array.Int64Builder).Append(42) })
	info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := info.Schema.Field(0).Metadata.GetValue("ARROW:extension:name"); !ok {
		t.Fatal("extension metadata lost during inspection")
	}
	tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 1)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "special") || !strings.Contains(err.Error(), "extension") {
		t.Fatalf("unknown extension: %d tables, %v", len(got), err)
	}
}
