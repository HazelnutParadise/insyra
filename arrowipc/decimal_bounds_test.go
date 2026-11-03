package arrowipc_test

import (
	"bytes"
	"context"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/decimal256"
)

func TestUnsupportedDecimalParametersFailBeforeCells(t *testing.T) {
	for _, typ := range []arrow.DataType{
		&arrow.Decimal128Type{Precision: 38, Scale: math.MaxInt32},
		&arrow.Decimal128Type{Precision: 0, Scale: 0},
		&arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: &arrow.Decimal128Type{Precision: 38, Scale: math.MaxInt32}},
		&arrow.Decimal128Type{Precision: 38, Scale: math.MinInt32},
		&arrow.Decimal128Type{Precision: 38, Scale: -1},
		&arrow.Decimal128Type{Precision: 38, Scale: 39},
		&arrow.Decimal128Type{Precision: 39, Scale: 1},
		&arrow.Decimal256Type{Precision: 76, Scale: math.MaxInt32},
		&arrow.Decimal256Type{Precision: 76, Scale: -1},
		&arrow.Decimal256Type{Precision: 77, Scale: 1},
	} {
		t.Run(typ.String(), func(t *testing.T) {
			schema := arrow.NewSchema([]arrow.Field{{Name: "amount", Type: typ}, {Name: "id", Type: arrow.PrimitiveTypes.Int64}}, nil)
			data := writeRecords(t, schema, func(b *array.RecordBuilder) {
				switch a := b.Field(0).(type) {
				case *array.Decimal128Builder:
					a.Append(decimal128.FromI64(1))
				case *array.Decimal256Builder:
					a.Append(decimal256.FromI64(1))
				case *array.Decimal128DictionaryBuilder:
					if err := a.Append(decimal128.FromI64(1)); err != nil {
						t.Fatal(err)
					}
				}
				b.Field(1).(*array.Int64Builder).Append(7)
			})
			info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
			if err != nil || !arrow.TypeEqual(info.Schema.Field(0).Type, typ) {
				t.Fatalf("inspection lost decimal type: %v", err)
			}
			tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 1)
			got, err := drain(t, tables, errs)
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "amount") || !strings.Contains(err.Error(), "decimal") {
				t.Fatalf("%s: %d tables, error %v", typ, len(got), err)
			}
			tables, errs = streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 1)
			got, err = drain(t, tables, errs)
			if err != nil || len(got) != 1 || got[0].GetColByNumber(0).Get(0) != int64(7) {
				t.Fatalf("excluding unsupported decimal: %d tables, %v", len(got), err)
			}

		})
	}
}
