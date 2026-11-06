package arrowipc_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/arrowipc"
	insyraparquet "github.com/HazelnutParadise/insyra/parquet"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	arrowparquet "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

func TestParquetScalarParity(t *testing.T) {
	cases := []struct {
		typ    arrow.DataType
		values string
	}{
		{arrow.PrimitiveTypes.Int8, `[12,null,0]`},
		{arrow.PrimitiveTypes.Int16, `[-1234,null,0]`},
		{arrow.PrimitiveTypes.Int32, `[2147483647,null,0]`},
		{arrow.PrimitiveTypes.Int64, `[9007199254740993,null,0]`},
		{arrow.PrimitiveTypes.Uint8, `[255,null,0]`},
		{arrow.PrimitiveTypes.Uint16, `[65535,null,0]`},
		{arrow.PrimitiveTypes.Uint32, `[4294967295,null,0]`},
		{arrow.PrimitiveTypes.Uint64, `[18446744073709551615,null,0]`},
		{arrow.PrimitiveTypes.Float32, `[1.25,null,0]`},
		{arrow.PrimitiveTypes.Float64, `[-2.5,null,0]`},
		{arrow.FixedWidthTypes.Boolean, `[true,null,false]`},
		{arrow.BinaryTypes.String, `["台灣 🐈",null,""]`},
		{arrow.BinaryTypes.LargeString, `["UTF-8",null,""]`},
		{arrow.BinaryTypes.Binary, `["AP8=",null,""]`},
		{arrow.BinaryTypes.LargeBinary, `["AP8=",null,""]`},
		{&arrow.FixedSizeBinaryType{ByteWidth: 2}, `["AP8=",null,"AAA="]`},
		{&arrow.TimestampType{Unit: arrow.Microsecond}, `["2026-01-01 12:34:56.123456",null,"1970-01-01 00:00:00"]`},
		{arrow.FixedWidthTypes.Date32, `["2026-01-01",null,"1970-01-01"]`},
		{arrow.FixedWidthTypes.Date64, `["2026-01-01",null,"1970-01-01"]`},
		{&arrow.DurationType{Unit: arrow.Microsecond}, `[123456,null,0]`},
		{&arrow.Decimal128Type{Precision: 38, Scale: 4}, `["123456789012345678901234567890.1234",null,"0"]`},
		{&arrow.Decimal256Type{Precision: 76, Scale: 4}, `["12345678901234567890123456789012345678901234567890.1234",null,"0"]`},
	}
	for _, tc := range cases {
		t.Run(tc.typ.String(), func(t *testing.T) {
			schema := arrow.NewSchema([]arrow.Field{{Name: "value", Type: tc.typ, Nullable: true}}, nil)
			b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
			defer b.Release()
			if err := b.Field(0).UnmarshalJSON([]byte(tc.values)); err != nil {
				t.Fatal(err)
			}
			rec := b.NewRecordBatch()
			defer rec.Release()
			var ipcbuf bytes.Buffer
			iw, err := ipc.NewFileWriter(&ipcbuf, ipc.WithSchema(schema))
			if err != nil {
				t.Fatal(err)
			}
			if err = iw.Write(rec); err != nil {
				t.Fatal(err)
			}
			if err = iw.Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "reference.parquet")
			if tc.typ.ID() == arrow.DURATION {
				// Arrow's Parquet writer refuses Duration. Insyra's Parquet
				// writer stores a time.Duration column with its unit, and its
				// reader returns time.Duration again: that is the reference.
				durations := insyra.NewDataList(123456*time.Microsecond, nil, time.Duration(0)).SetName("value")
				if err := insyraparquet.Write(insyra.NewDataTable(durations), path); err != nil {
					t.Fatal(err)
				}
			} else {
				writeParquetReference(t, path, schema, rec)
			}
			reference, err := insyraparquet.Read(context.Background(), path, insyraparquet.ReadOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if e := reference.Err(); e != nil {
				t.Fatal(e)
			}
			tables, errs := streamFrom(context.Background(), bytes.NewReader(ipcbuf.Bytes()), int64(ipcbuf.Len()), arrowipc.ReadOptions{}, 1)
			var got []any
			for table := range tables {
				got = append(got, table.GetColByNumber(0).Data()...)
			}
			if err := <-errs; err != nil {
				t.Fatal(err)
			}
			want := reference.GetColByNumber(0).Data()
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("IPC %s != Parquet %s", fmt.Sprintf("%#v", got), fmt.Sprintf("%#v", want))
			}
			if got[1] != nil {
				t.Fatalf("null became %T(%v)", got[1], got[1])
			}
		})
	}
}

func writeParquetReference(t *testing.T, path string, schema *arrow.Schema, rec arrow.RecordBatch) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	pw, err := pqarrow.NewFileWriter(schema, f, arrowparquet.NewWriterProperties(), pqarrow.NewArrowWriterProperties(pqarrow.WithStoreSchema()))
	if err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err = pw.Write(rec); err != nil {
		_ = pw.Close()
		t.Fatal(err)
	}
	if err = pw.Close(); err != nil {
		t.Fatal(err)
	}
}
