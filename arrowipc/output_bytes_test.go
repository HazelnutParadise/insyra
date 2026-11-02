package arrowipc_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

func TestOutputByteBudgetBoundsDictionaryExpansion(t *testing.T) {
	for _, valueType := range []arrow.DataType{arrow.BinaryTypes.String, arrow.BinaryTypes.Binary} {
		t.Run(valueType.Name(), func(t *testing.T) {
			typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: valueType}
			schema := arrow.NewSchema([]arrow.Field{{Name: "category", Type: typ}}, nil)
			data := writeRecords(t, schema, func(b *array.RecordBuilder) {
				db := b.Field(0).(*array.BinaryDictionaryBuilder)
				for i := 0; i < 32; i++ {
					if err := db.Append([]byte(strings.Repeat("x", 128))); err != nil {
						t.Fatal(err)
					}
				}
			})
			tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{MaxBatchBytes: 4096, MaxOutputBytes: 1024}, 32)
			got, err := drain(t, tables, errs)
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "MaxOutputBytes") {
				t.Fatalf("expanded dictionary: %d tables, %v", len(got), err)
			}
		})
	}
}

func TestOutputByteBudgetIncludesNullCellSlots(t *testing.T) {
	schema := arrow.NewSchema([]arrow.Field{{Name: "missing", Type: arrow.Null}}, nil)
	data := writeRecords(t, schema, func(b *array.RecordBuilder) {
		b.Field(0).AppendNulls(100)
	})
	tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{MaxOutputBytes: 512}, 100)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "MaxOutputBytes") {
		t.Fatalf("null cell slots: %d tables, %v", len(got), err)
	}
}

func TestOutputByteBudgetBoundaryAndNegativeLimit(t *testing.T) {
	schema := arrow.NewSchema([]arrow.Field{{Name: "text", Type: arrow.BinaryTypes.String}}, nil)
	data := writeRecords(t, schema, func(b *array.RecordBuilder) {
		b.Field(0).(*array.StringBuilder).Append(strings.Repeat("x", 128))
	})
	for _, limit := range []int64{-1, 143, 144} {
		tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{MaxOutputBytes: limit}, 1)
		got, err := drain(t, tables, errs)
		if limit == 144 {
			if len(got) != 1 || err != nil {
				t.Fatalf("exact budget: %d tables, %v", len(got), err)
			}
		} else if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "MaxOutputBytes") {
			t.Fatalf("output budget %d: %d tables, %v", limit, len(got), err)
		}
	}
}
