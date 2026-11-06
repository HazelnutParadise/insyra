package arrowipc_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func TestDictionaryNullValueAndNullIndexRemainNull(t *testing.T) {
	typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: arrow.BinaryTypes.String, Ordered: true}
	schema := arrow.NewSchema([]arrow.Field{{Name: "category", Type: typ, Nullable: true}}, nil)
	indices := array.NewInt8Builder(memory.DefaultAllocator)
	defer indices.Release()
	indices.AppendValues([]int8{0, 1, 0}, []bool{true, true, false})
	idx := indices.NewArray()
	defer idx.Release()
	values := array.NewStringBuilder(memory.DefaultAllocator)
	defer values.Release()
	values.AppendNull()
	values.Append("ready")
	dict := values.NewArray()
	defer dict.Release()
	col := array.NewDictionaryArray(typ, idx, dict)
	defer col.Release()
	rec := array.NewRecordBatch(schema, []arrow.Array{col}, 3)
	defer rec.Release()
	var buf bytes.Buffer
	w, err := ipc.NewFileWriter(&buf, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	tables, errs := streamFrom(context.Background(), bytes.NewReader(buf.Bytes()), int64(buf.Len()), arrowipc.ReadOptions{}, 2)
	got, err := drain(t, tables, errs)
	if err != nil {
		t.Fatal(err)
	}
	var cells []any
	for _, dt := range got {
		cells = append(cells, dt.GetColByNumber(0).Data()...)
	}
	if !reflect.DeepEqual(cells, []any{nil, "ready", nil}) {
		t.Fatalf("dictionary cells %#v", cells)
	}
}
