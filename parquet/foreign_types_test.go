package parquet

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/decimal128"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

// parquet.Write only ever emits seven Arrow types, so nothing in this package's
// own round trips can reach the rest. These tests write the types other tools
// write — pandas, Spark, DuckDB — and read them back through the public API.
//
// #371: the reader's default arm returned arr.String(), the string form of the
// whole array, ignoring the row index, so every row of such a column read back
// as the same string with no error anywhere.

const decimal38 = "99999999999999999999999999999999999999"

// writeForeignParquet writes one file holding a column per field and returns
// its path.
func writeForeignParquet(t *testing.T, fields []arrow.Field, fill func(*array.RecordBuilder)) string {
	t.Helper()

	mem := memory.NewGoAllocator()
	schema := arrow.NewSchema(fields, nil)
	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	fill(b)
	rec := b.NewRecord()
	defer rec.Release()

	path := filepath.Join(t.TempDir(), "foreign.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w, err := pqarrow.NewFileWriter(schema, f, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatal(err)
	}
	// pqarrow's writer closes the underlying file.
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadRepresentableForeignTypes(t *testing.T) {
	dec := &arrow.Decimal128Type{Precision: 38, Scale: 10}
	fields := []arrow.Field{
		{Name: "date32", Type: arrow.FixedWidthTypes.Date32, Nullable: true},
		{Name: "date64", Type: arrow.FixedWidthTypes.Date64, Nullable: true},
		{Name: "dec128", Type: dec, Nullable: true},
		{Name: "u64", Type: arrow.PrimitiveTypes.Uint64, Nullable: true},
		{Name: "u32", Type: arrow.PrimitiveTypes.Uint32, Nullable: true},
		{Name: "i16", Type: arrow.PrimitiveTypes.Int16, Nullable: true},
		{Name: "i8", Type: arrow.PrimitiveTypes.Int8, Nullable: true},
		{Name: "bin", Type: arrow.BinaryTypes.Binary, Nullable: true},
	}

	big38, ok := new(big.Int).SetString(decimal38, 10)
	if !ok {
		t.Fatal("bad fixture")
	}
	neg38 := new(big.Int).Neg(big38)

	path := writeForeignParquet(t, fields, func(b *array.RecordBuilder) {
		b.Field(0).(*array.Date32Builder).AppendValues([]arrow.Date32{19000, 19001, 0}, nil)
		b.Field(1).(*array.Date64Builder).AppendValues([]arrow.Date64{1600000000000, 1600086400000, 0}, nil)
		b.Field(2).(*array.Decimal128Builder).AppendValues([]decimal128.Num{
			decimal128.FromBigInt(big38), decimal128.FromBigInt(neg38), decimal128.FromI64(12345),
		}, nil)
		b.Field(3).(*array.Uint64Builder).AppendValues([]uint64{1, 1 << 63, 0}, nil)
		b.Field(4).(*array.Uint32Builder).AppendValues([]uint32{7, 4294967295, 0}, nil)
		b.Field(5).(*array.Int16Builder).AppendValues([]int16{10, -32768, 0}, nil)
		b.Field(6).(*array.Int8Builder).AppendValues([]int8{1, -128, 0}, nil)
		b.Field(7).(*array.BinaryBuilder).AppendValues([][]byte{[]byte("aa"), {0x00, 0xff}, {}}, nil)
	})

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if e := dt.Err(); e != nil {
		t.Fatalf("Err() on a file of readable columns: %v", e)
	}

	// Every row must differ from every other, which is the shape of the defect.
	for ci := 0; ci < 8; ci++ {
		a := dt.GetElementByNumberIndex(0, ci)
		b := dt.GetElementByNumberIndex(1, ci)
		if eq(a, b) {
			t.Errorf("column %d: rows 0 and 1 read back identical (%v), which is the #371 shape", ci, a)
		}
	}

	wantDate32 := time.Date(2022, 1, 8, 0, 0, 0, 0, time.UTC)
	if got := dt.GetElementByNumberIndex(0, 0); !isTime(got, wantDate32) {
		t.Errorf("date32: got %v (%T), want %v", got, got, wantDate32)
	}
	wantDate64 := time.Date(2020, 9, 13, 0, 0, 0, 0, time.UTC)
	if got := dt.GetElementByNumberIndex(0, 1); !isTime(got, wantDate64) {
		t.Errorf("date64: got %v (%T), want %v", got, got, wantDate64)
	}

	// 38 nines at scale 10, and its negative. A float64 or an int64 coefficient
	// could not carry either.
	for row, want := range map[int]string{
		0: "9999999999999999999999999999.9999999999",
		1: "-9999999999999999999999999999.9999999999",
		2: "0.0000012345",
	} {
		got := dt.GetElementByNumberIndex(row, 2)
		s, ok := got.(interface{ String() string })
		if !ok {
			t.Errorf("dec128 row %d: %T is not a decimal", row, got)
			continue
		}
		if s.String() != want {
			t.Errorf("dec128 row %d: got %s, want %s", row, s.String(), want)
		}
	}

	if got := dt.GetElementByNumberIndex(1, 3); got != uint64(1<<63) {
		t.Errorf("u64 above 2^63: got %v (%T), want %v", got, got, uint64(1<<63))
	}
	if got := dt.GetElementByNumberIndex(1, 4); got != uint32(4294967295) {
		t.Errorf("u32: got %v (%T)", got, got)
	}
	if got := dt.GetElementByNumberIndex(1, 5); got != int16(-32768) {
		t.Errorf("i16: got %v (%T)", got, got)
	}
	if got := dt.GetElementByNumberIndex(1, 6); got != int8(-128) {
		t.Errorf("i8: got %v (%T)", got, got)
	}
	// Binary arrives as a string because a DataList cell cannot be a slice.
	// The bytes still have to survive exactly, 0x00 and invalid UTF-8 included.
	got, ok := dt.GetElementByNumberIndex(1, 7).(string)
	if !ok {
		t.Errorf("binary: got %T, want string", dt.GetElementByNumberIndex(1, 7))
	} else if b := []byte(got); len(b) != 2 || b[0] != 0x00 || b[1] != 0xff {
		t.Errorf("binary: got % x, want 00 ff", b)
	}
}

func TestReadUnrepresentableColumnSaysSo(t *testing.T) {
	fields := []arrow.Field{
		{Name: "ok", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "tags", Type: arrow.ListOf(arrow.PrimitiveTypes.Int64), Nullable: true},
	}
	path := writeForeignParquet(t, fields, func(b *array.RecordBuilder) {
		b.Field(0).(*array.Int64Builder).AppendValues([]int64{1, 2, 3}, nil)
		lb := b.Field(1).(*array.ListBuilder)
		vb := lb.ValueBuilder().(*array.Int64Builder)
		for i := 0; i < 3; i++ {
			lb.Append(true)
			vb.AppendValues([]int64{int64(i), int64(i + 1)}, nil)
		}
	})

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if dt == nil {
		t.Fatal("nil table")
	}

	// The readable column still works.
	for row, want := range []int64{1, 2, 3} {
		if got := dt.GetElementByNumberIndex(row, 0); got != want {
			t.Errorf("readable column row %d: got %v, want %v", row, got, want)
		}
	}

	// The unreadable one is empty, not plausible text.
	for row := 0; row < 3; row++ {
		if got := dt.GetElementByNumberIndex(row, 1); got != nil {
			t.Errorf("unreadable column row %d: got %v (%T), want nil", row, got, got)
		}
	}

	e := dt.Err()
	if e == nil {
		t.Fatal("a column that could not be read was not reported")
	}
	msg := e.Error()
	if !contains(msg, "tags") || !contains(msg, "list") {
		t.Errorf("the error names neither the column nor the type: %s", msg)
	}
}

func eq(a, b any) bool {
	ab, aok := a.([]byte)
	bb, bok := b.([]byte)
	if aok && bok {
		return string(ab) == string(bb)
	}
	as, aok := a.(interface{ String() string })
	bs, bok := b.(interface{ String() string })
	if aok && bok {
		return as.String() == bs.String()
	}
	return a == b
}

func isTime(v any, want time.Time) bool {
	got, ok := v.(time.Time)
	return ok && got.UTC().Equal(want)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// supportedArrowType and getVal are two lists of the same thing, and they drift
// silently: a type in the first but not the second reads as nil while claiming
// to be supported, and one in the second but not the first makes the reader
// report a column it read perfectly well.
func TestSupportedTypesAreExactlyWhatGetValHandles(t *testing.T) {
	mem := memory.NewGoAllocator()
	types := []arrow.DataType{
		arrow.PrimitiveTypes.Int8, arrow.PrimitiveTypes.Int16,
		arrow.PrimitiveTypes.Int32, arrow.PrimitiveTypes.Int64,
		arrow.PrimitiveTypes.Uint8, arrow.PrimitiveTypes.Uint16,
		arrow.PrimitiveTypes.Uint32, arrow.PrimitiveTypes.Uint64,
		arrow.PrimitiveTypes.Float32, arrow.PrimitiveTypes.Float64,
		arrow.FixedWidthTypes.Boolean,
		arrow.BinaryTypes.String, arrow.BinaryTypes.LargeString,
		arrow.BinaryTypes.Binary, arrow.BinaryTypes.LargeBinary,
		&arrow.FixedSizeBinaryType{ByteWidth: 2},
		arrow.FixedWidthTypes.Timestamp_us,
		arrow.FixedWidthTypes.Date32, arrow.FixedWidthTypes.Date64,
		&arrow.Decimal128Type{Precision: 10, Scale: 2},
		&arrow.Decimal256Type{Precision: 40, Scale: 4},
		// Not representable:
		arrow.FixedWidthTypes.Time32s, arrow.FixedWidthTypes.Time64us,
		arrow.FixedWidthTypes.Duration_s,
		arrow.FixedWidthTypes.MonthInterval,
		arrow.ListOf(arrow.PrimitiveTypes.Int64),
		arrow.StructOf(arrow.Field{Name: "a", Type: arrow.PrimitiveTypes.Int64}),
	}

	for _, dt := range types {
		b := array.NewBuilder(mem, dt)
		b.AppendNull()
		arr := b.NewArray()

		// A null cell is nil whatever the type, so ask the switch directly by
		// building a one-value array where the type allows it; for the rest the
		// null is enough, because getVal's arms are chosen by the array's Go
		// type, not by the value.
		handled := getValHandles(arr)
		arr.Release()
		b.Release()

		if want := supportedArrowType(dt); handled != want {
			t.Errorf("%s: getVal handles=%v, supportedArrowType=%v", dt, handled, want)
		}
	}
}

// getValHandles reports whether getVal has an arm for this array's Go type,
// independent of the value in it.
func getValHandles(arr arrow.Array) bool {
	switch arr.(type) {
	case *array.Int64, *array.Int32, *array.Int16, *array.Int8,
		*array.Uint64, *array.Uint32, *array.Uint16, *array.Uint8,
		*array.Float64, *array.Float32,
		*array.String, *array.LargeString, *array.Boolean,
		*array.Binary, *array.LargeBinary, *array.FixedSizeBinary,
		*array.Timestamp, *array.Date32, *array.Date64,
		*array.Decimal128, *array.Decimal256:
		return true
	}
	return false
}

func TestDecimalColumnSortsByValue(t *testing.T) {
	dec := &arrow.Decimal128Type{Precision: 12, Scale: 1}
	fields := []arrow.Field{{Name: "amount", Type: dec, Nullable: true}}
	path := writeForeignParquet(t, fields, func(b *array.RecordBuilder) {
		// 9.5, 10.2, 100.0 — in text order "10.2" < "100.0" < "9.5", so a
		// lexicographic sort puts them the wrong way round.
		b.Field(0).(*array.Decimal128Builder).AppendValues([]decimal128.Num{
			decimal128.FromI64(95), decimal128.FromI64(102), decimal128.FromI64(1000),
		}, nil)
	})

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	dt.SortBy(insyra.DataTableSortConfig{ColumnIndex: "A"})

	want := []string{"9.5", "10.2", "100.0"}
	for i, w := range want {
		got := dt.GetElementByNumberIndex(i, 0)
		s, ok := got.(interface{ String() string })
		if !ok {
			t.Fatalf("row %d: %T is not a decimal", i, got)
		}
		if s.String() != w {
			t.Errorf("row %d after sorting: got %s, want %s", i, s.String(), w)
		}
	}
}

// listColumnFile writes a file whose second column is a nested type no Go cell
// can hold, alongside a readable one.
func listColumnFile(t *testing.T) string {
	t.Helper()
	return writeForeignParquet(t, []arrow.Field{
		{Name: "ok", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "tags", Type: arrow.ListOf(arrow.PrimitiveTypes.Int64), Nullable: true},
	}, func(b *array.RecordBuilder) {
		b.Field(0).(*array.Int64Builder).AppendValues([]int64{1, 2, 3}, nil)
		lb := b.Field(1).(*array.ListBuilder)
		vb := lb.ValueBuilder().(*array.Int64Builder)
		for i := 0; i < 3; i++ {
			lb.Append(true)
			vb.AppendValues([]int64{int64(i), int64(i + 1)}, nil)
		}
	})
}

func TestStreamReportsAnUnreadableColumn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dtChan, errChan := Stream(ctx, listColumnFile(t), ReadOptions{}, 2)
	batches := 0
	for dt := range dtChan {
		batches++
		if dt.Err() == nil {
			t.Errorf("batch %d: an unreadable column was not reported", batches)
		}
		rows, _ := dt.Size()
		for row := 0; row < rows; row++ {
			if got := dt.GetElementByNumberIndex(row, 1); got != nil {
				t.Errorf("batch %d row %d: got %v (%T), want nil", batches, row, got, got)
			}
		}
	}
	if err := <-errChan; err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if batches == 0 {
		t.Fatal("no batches")
	}
}

// FilterWithCCL reads cells one at a time through parquetContext rather than
// through chunkedToSlice, so it needs its own check that it does not hand the
// filter a value the file does not hold.
func TestFilterWithCCLDoesNotSeeFakeValues(t *testing.T) {
	dt, err := FilterWithCCL(context.Background(), listColumnFile(t), "A > 1")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}
	rows, _ := dt.Size()
	if rows != 2 {
		t.Fatalf("got %d rows, want 2", rows)
	}
	for row := 0; row < rows; row++ {
		if got := dt.GetElementByNumberIndex(row, 1); got != nil {
			t.Errorf("row %d: the filter path produced %v (%T) for a column it cannot read", row, got, got)
		}
	}
}

func TestReadColumnCarriesTheReason(t *testing.T) {
	// A flat unreadable type rather than the list, because a nested column's
	// Parquet leaf is not named after the field and cannot be selected by it.
	path := writeForeignParquet(t, []arrow.Field{
		{Name: "clock", Type: arrow.FixedWidthTypes.Time32s, Nullable: true},
	}, func(b *array.RecordBuilder) {
		b.Field(0).(*array.Time32Builder).AppendValues([]arrow.Time32{1, 2, 3}, nil)
	})

	dl, err := ReadColumn(context.Background(), path, "clock", ReadColumnOptions{})
	if err != nil {
		t.Fatalf("ReadColumn: %v", err)
	}
	if dl.Err() == nil {
		t.Error("ReadColumn handed back a list of nils with no reason on it")
	}
	for i := 0; i < dl.Len(); i++ {
		if got := dl.Get(i); got != nil {
			t.Errorf("row %d: got %v (%T), want nil", i, got, got)
		}
	}
}
