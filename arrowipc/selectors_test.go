package arrowipc_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

func TestColumnSelectorsDistinguishIndicesNamesAndPositions(t *testing.T) {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "B", Type: arrow.PrimitiveTypes.Int32},
		{Name: "price", Type: arrow.PrimitiveTypes.Int32},
	}, nil)
	data := writeRecords(t, schema, func(b *array.RecordBuilder) {
		b.Field(0).(*array.Int32Builder).Append(11)
		b.Field(1).(*array.Int32Builder).Append(22)
	})
	for _, tc := range []struct {
		selectors []any
		names     []string
		values    []any
	}{
		{[]any{"B"}, []string{"price"}, []any{int32(22)}},
		{[]any{insyra.Name("B")}, []string{"B"}, []any{int32(11)}},
		{[]any{1, -2}, []string{"price", "B"}, []any{int32(22), int32(11)}},
		{[]any{"b", insyra.Name("B")}, []string{"price", "B"}, []any{int32(22), int32(11)}},
	} {
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: tc.selectors}, 1)
		got, err := drain(t, tables, errs)
		if err != nil || len(got) != 1 {
			t.Fatalf("selectors %v: %d tables, %v", tc.selectors, len(got), err)
		}
		if !reflect.DeepEqual(got[0].ColNames(), tc.names) {
			t.Fatalf("selectors %v names %v, want %v", tc.selectors, got[0].ColNames(), tc.names)
		}
		for i, want := range tc.values {
			if value := got[0].GetColByNumber(i).Get(0); value != want {
				t.Fatalf("selectors %v cell %d = %v, want %v", tc.selectors, i, value, want)
			}
		}
	}
	for _, selectors := range [][]any{{"B", 1}, {"price"}, {nil}, {int64(1)}, {-3}, {"ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"}} {
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: selectors}, 1)
		got, err := drain(t, tables, errs)
		if err == nil || len(got) != 0 {
			t.Fatalf("invalid selectors %v: %d tables, %v", selectors, len(got), err)
		}
	}
}
