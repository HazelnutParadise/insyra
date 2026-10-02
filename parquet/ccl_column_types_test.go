package parquet

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/decimal128"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/file"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

// parquet.Write only ever emits the types the builder infers from Go values, so
// a file holding a column of any other type cannot be produced by this package
// and every round trip of one is untested. These are the types other tools
// write — DuckDB's INTEGER, Spark's DateType, money as a decimal, a
// large_string, a list — read and written back by ApplyCCL, whose builder has
// none of them: a column no statement writes has to go back into the file as
// the file had it, whatever its type.

// manyTypesRows is how many rows writeManyTypesFile writes: more than the batch
// size ApplyCCL reads at a time, so a look-ahead holds rows back across batches
// and a written column's type is not settled from the first rows alone.
const manyTypesRows = 2500

// writeManyTypesFile writes a file of rows rows whose columns are of the types
// named here, and returns its path. The Arrow schema is stored in the file,
// which is how the column types survive the parquet round trip: a DATE column
// is a date32, a decimal is a decimal128 at its own scale, and a large_string
// stays one.
func writeManyTypesFile(t *testing.T, rows int) string {
	t.Helper()

	fields := []arrow.Field{
		{Name: "A", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "i32", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "f32", Type: arrow.PrimitiveTypes.Float32, Nullable: true},
		{Name: "d32", Type: arrow.FixedWidthTypes.Date32, Nullable: true},
		{Name: "dec", Type: &arrow.Decimal128Type{Precision: 10, Scale: 2}, Nullable: true},
		{Name: "ls", Type: arrow.BinaryTypes.LargeString, Nullable: true},
		{Name: "lst", Type: arrow.ListOf(arrow.PrimitiveTypes.Int64), Nullable: true},
		{Name: "u64", Type: arrow.PrimitiveTypes.Uint64, Nullable: true},
		{Name: "i16", Type: arrow.PrimitiveTypes.Int16, Nullable: true},
	}
	schema := arrow.NewSchema(fields, nil)

	mem := memory.DefaultAllocator
	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	a := b.Field(0).(*array.Float64Builder)
	i32 := b.Field(1).(*array.Int32Builder)
	f32 := b.Field(2).(*array.Float32Builder)
	d32 := b.Field(3).(*array.Date32Builder)
	dec := b.Field(4).(*array.Decimal128Builder)
	ls := b.Field(5).(*array.LargeStringBuilder)
	lst := b.Field(6).(*array.ListBuilder)
	listVals := lst.ValueBuilder().(*array.Int64Builder)
	u64 := b.Field(7).(*array.Uint64Builder)
	i16 := b.Field(8).(*array.Int16Builder)

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < rows; i++ {
		a.Append(float64(i + 1))
		// A missing value every seventh row, so the columns are not all full.
		if i%7 == 0 {
			i32.AppendNull()
		} else {
			i32.Append(int32(i % 1000))
		}
		f32.Append(float32(i) / 4)
		d32.Append(arrow.Date32FromTime(start.AddDate(0, 0, i)))
		dec.Append(decimal128.FromI64(int64(i) * 125))
		ls.Append(fmt.Sprint("s", i))
		if i%5 == 0 {
			lst.AppendNull()
		} else {
			lst.Append(true)
			listVals.AppendValues([]int64{int64(i), int64(i + 1)}, nil)
		}
		// A value above math.MaxInt64, which no int64 column can hold.
		u64.Append(math.MaxUint64 - uint64(i))
		i16.Append(int16(i % 300))
	}

	rec := b.NewRecord()
	defer rec.Release()

	path := filepath.Join(t.TempDir(), "many-types.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating the many-types file: %v", err)
	}
	w, err := pqarrow.NewFileWriter(schema, f, parquet.NewWriterProperties(),
		pqarrow.NewArrowWriterProperties(pqarrow.WithStoreSchema()))
	if err != nil {
		t.Fatalf("creating the writer for the many-types file: %v", err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatalf("writing the many-types file: %v", err)
	}
	// pqarrow's writer closes the underlying file.
	if err := w.Close(); err != nil {
		t.Fatalf("closing the many-types file: %v", err)
	}
	return path
}

// readArrowTable reads the file at path as Arrow, which is the only way to see
// the type a column came back with: Read gives the Go value every type is read
// as, and a list column is nil there however it was stored.
func readArrowTable(t *testing.T, path string) arrow.Table {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	r, err := file.NewParquetReader(f)
	if err != nil {
		t.Fatalf("reading the footer of %s: %v", path, err)
	}
	fr, err := pqarrow.NewFileReader(r, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		t.Fatalf("creating the Arrow reader for %s: %v", path, err)
	}
	tbl, err := fr.ReadTable(context.Background())
	if err != nil {
		t.Fatalf("reading %s as Arrow: %v", path, err)
	}
	// The Parquet reader owns f and closes it, so only it is closed here.
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("closing the Parquet reader of %s: %v", path, err)
		}
	})
	return tbl
}

// oneColumn returns the whole of the chunked column idx of tbl, so two tables
// that cut the same rows into different row groups can still be compared.
func oneColumn(t *testing.T, tbl arrow.Table, idx int) arrow.Array {
	t.Helper()

	col := tbl.Column(idx)
	arr, err := array.Concatenate(col.Data().Chunks(), memory.DefaultAllocator)
	if err != nil {
		t.Fatalf("joining column %q: %v", col.Name(), err)
	}
	return arr
}

// TestApplyCCLKeepsEveryUnwrittenColumn holds ApplyCCL on a file whose columns
// are of the types its own builder cannot write, whatever the script does to the
// rest of the file. Every column the script does not write comes back with the
// type the file gave it and every value of it, whether the script ran over one
// batch, held rows back across them, or computed a column over the whole file.
func TestApplyCCLKeepsEveryUnwrittenColumn(t *testing.T) {
	for _, script := range []string{
		// One batch at a time, a new column beside the file's own.
		"NEW('n') = 1",
		// A look-ahead longer than a batch, so rows are held back and handed on
		// in pieces the file's arrays have to be sliced with.
		"NEW('n') = LEAD(A, 1500)",
		// A statement computed over the whole of the column it reads, whose
		// answer is held in memory until the file is written.
		"NEW('n') = A - MEDIAN(A)",
		// A column of the file's own replaced, so every other column is
		// untouched beside one that is not.
		"['A'] = A * 2",
		// A written column of two kinds of value, which is what sends the file
		// through the second pass.
		"NEW('c') = IF(# < 1000, 1, 'x')",
	} {
		t.Run(script, func(t *testing.T) {
			path := writeManyTypesFile(t, manyTypesRows)
			before := readArrowTable(t, path)
			defer before.Release()

			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}

			after := readArrowTable(t, path)
			defer after.Release()

			if after.NumRows() != before.NumRows() {
				t.Fatalf("ApplyCCL(%q) left %d rows, the file had %d", script, after.NumRows(), before.NumRows())
			}

			// Column A is the one these scripts write, so every other column the
			// file had has to come back as it was.
			for i := 1; i < int(before.NumCols()); i++ {
				name := before.Schema().Field(i).Name
				idx := after.Schema().FieldIndices(name)
				if len(idx) == 0 {
					t.Errorf("ApplyCCL(%q) dropped column %q", script, name)
					continue
				}
				gotType := after.Schema().Field(idx[0]).Type
				wantType := before.Schema().Field(i).Type
				if !arrow.TypeEqual(gotType, wantType) {
					t.Errorf("ApplyCCL(%q) wrote column %q as %s, the file had %s", script, name, gotType, wantType)
					continue
				}

				want := oneColumn(t, before, i)
				defer want.Release()
				got := oneColumn(t, after, idx[0])
				defer got.Release()
				if !array.Equal(got, want) {
					t.Errorf("ApplyCCL(%q) changed column %q: %s is not the %s the file had",
						script, name, got, want)
				}
			}
		})
	}
}

// runArraysFixture is a run of six rows of one column whose array is the file's
// own, which is what every split, write and hand-on has to keep in step.
func runArraysFixture(t *testing.T) (cclRun, arrow.Array) {
	t.Helper()

	b := array.NewInt64Builder(memory.DefaultAllocator)
	defer b.Release()
	b.AppendValues([]int64{10, 20, 30, 40, 50, 60}, nil)
	arr := b.NewArray().(*array.Int64)
	t.Cleanup(arr.Release)

	cols := make([]any, arr.Len())
	for i := range cols {
		cols[i] = arr.Value(i)
	}
	return cclRun{names: []string{"A"}, cols: [][]any{cols}, arrays: []arrow.Array{arr}}, arr
}

// slicedArray returns the array a run of rows holds for the column at index, or
// nil with a message the caller failed on, so a run that lost it is reported
// rather than indexed into.
func slicedArray(t *testing.T, r cclRun, index int) arrow.Array {
	t.Helper()

	if index >= len(r.arrays) {
		t.Fatalf("the run holds %d arrays for %d names", len(r.arrays), len(r.names))
	}
	return r.arrays[index]
}

// int64Values reads every value of an int64 array, which is how a sliced array
// is compared with the rows the run's values hold.
func int64Values(t *testing.T, arr arrow.Array) []int64 {
	t.Helper()

	if arr == nil {
		return nil
	}
	col, ok := arr.(*array.Int64)
	if !ok {
		t.Fatalf("array is a %T, want *array.Int64", arr)
	}
	return col.Int64Values()
}

// TestRunArraysFollowTheirRows holds the seam the writer reads a run's file
// arrays through: a run cut in two cuts its arrays with it, a run of no rows has
// none, a column a statement writes has none to write back, and a run's arrays
// are handed on beside its values.
func TestRunArraysFollowTheirRows(t *testing.T) {
	r, arr := runArraysFixture(t)

	t.Run("headRun takes the first rows of every array", func(t *testing.T) {
		got := headRun(r, 2)
		if got.rows() != 2 {
			t.Fatalf("headRun(r, 2) holds %d rows, want 2", got.rows())
		}
		if len(got.arrays) != len(got.names) {
			t.Fatalf("headRun(r, 2) holds %d arrays for %d names", len(got.arrays), len(got.names))
		}
		head := slicedArray(t, got, 0)
		if head == nil {
			t.Fatal("headRun(r, 2) dropped the array")
		}
		if want := []int64{10, 20}; !slices.Equal(int64Values(t, head), want) {
			t.Errorf("headRun(r, 2) array is %v, want %v", int64Values(t, head), want)
		}

		// Every row of the run is a run of its own, array and all.
		if all := headRun(r, r.rows()); all.arrays[0] != arr {
			t.Error("headRun over every row did not keep the run's own array")
		}
	})

	t.Run("tailRun takes the rows after them", func(t *testing.T) {
		got := tailRun(r, 2)
		if got.offset != r.offset+2 {
			t.Errorf("tailRun(r, 2) starts at row %d, want %d", got.offset, r.offset+2)
		}
		tail := slicedArray(t, got, 0)
		if tail == nil {
			t.Fatal("tailRun(r, 2) dropped the array")
		}
		if want := []int64{30, 40, 50, 60}; !slices.Equal(int64Values(t, tail), want) {
			t.Errorf("tailRun(r, 2) array is %v, want %v", int64Values(t, tail), want)
		}
		if tail.Len() != got.rows() {
			t.Errorf("tailRun(r, 2) array holds %d rows, the run holds %d", tail.Len(), got.rows())
		}
	})

	t.Run("a run of no rows has no arrays", func(t *testing.T) {
		got := tailRun(r, r.rows())
		if got.rows() != 0 {
			t.Fatalf("tailRun(r, 6) holds %d rows, want none", got.rows())
		}
		if got.arrays != nil {
			t.Errorf("tailRun(r, 6) holds %d arrays, want none", len(got.arrays))
		}
	})

	t.Run("runArrays has one entry per name", func(t *testing.T) {
		got := runArrays(cclRun{names: []string{"A", "B", "C"}, arrays: []arrow.Array{arr}})
		if len(got) != 3 {
			t.Fatalf("runArrays gave %d entries for 3 names", len(got))
		}
		if got[0] != arr {
			t.Error("runArrays did not hand back the run's own array")
		}
		if got[1] != nil || got[2] != nil {
			t.Errorf("runArrays gave %v for the names the run has no array for", got[1:])
		}
	})

	t.Run("a sequence's assignment drops its array", func(t *testing.T) {
		s := &sequenceStage{target: 0}
		got := s.write(r, []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0})
		if len(got.arrays) != 1 {
			t.Fatalf("the run came out with %d arrays, want 1", len(got.arrays))
		}
		if got.arrays[0] != nil {
			t.Error("the column a sequence assigned kept the array of the file's values")
		}
		if r.arrays[0] != arr {
			t.Error("writing into the run changed the array it was given")
		}
	})

	t.Run("a sequence's NEW adds no array", func(t *testing.T) {
		s := &sequenceStage{newName: "n", target: -1}
		got := s.write(r, []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0})
		if !slices.Equal(got.names, []string{"A", "n"}) {
			t.Fatalf("the run came out naming %v, want [A n]", got.names)
		}
		if len(got.arrays) != 2 {
			t.Fatalf("the run came out with %d arrays for 2 names", len(got.arrays))
		}
		if got.arrays[0] != arr {
			t.Error("the column a NEW leaves beside it lost the file's array")
		}
		if got.arrays[1] != nil {
			t.Error("a column a NEW created has an array of the file's")
		}
	})
}

// valued is an Arrow array that reads its cells as T, which is what every fixed
// width array of a type this file checks is.
type valued[T any] interface {
	arrow.Array
	Value(i int) T
}

// checkColumn holds every row of arr to want, which says what row i holds and
// false where it is a null. A is the array type the column is expected to be, so
// a column of another type fails here rather than at an index.
func checkColumn[T comparable, A valued[T]](t *testing.T, arr arrow.Array, want func(i int) (T, bool)) {
	t.Helper()

	got, ok := arr.(A)
	if !ok {
		t.Fatalf("the column is a %T, want %T", arr, *new(A))
	}
	if got.Len() != manyTypesRows {
		t.Fatalf("the column holds %d rows, want %d", got.Len(), manyTypesRows)
	}
	for i := range got.Len() {
		w, present := want(i)
		switch {
		case !present && !got.IsNull(i):
			t.Fatalf("row %d is %v, want a null", i, got.Value(i))
		case present && got.IsNull(i):
			t.Fatalf("row %d is a null, want %v", i, w)
		case present && got.Value(i) != w:
			t.Fatalf("row %d is %v, want %v", i, got.Value(i), w)
		}
	}
}

// TestApplyCCLKeepsANarrowAssignedType holds a column of a type narrower than
// the 64-bit ones to the rule every assigned column keeps: it stays the type the
// file gave it while every value written into it is one that type holds, and
// takes the type Write gives its values the moment one is not. Each script runs
// on a file of its own.
func TestApplyCCLKeepsANarrowAssignedType(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		script string
		column string
		want   arrow.DataType
		check  func(t *testing.T, arr arrow.Array)
	}{
		{
			script: "['i32'] = ['i32'] * 2",
			column: "i32",
			want:   arrow.PrimitiveTypes.Int32,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[int32, *array.Int32](t, arr, func(i int) (int32, bool) {
					// A missing value is a 0 in arithmetic, so it is written as one.
					if i%7 == 0 {
						return 0, true
					}
					return int32(i%1000) * 2, true
				})
			},
		},
		{
			script: "['i32'] = ['i32'] / 3",
			column: "i32",
			want:   arrow.PrimitiveTypes.Float64,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[float64, *array.Float64](t, arr, func(i int) (float64, bool) {
					if i%7 == 0 {
						return 0, true
					}
					return float64(i%1000) / 3, true
				})
			},
		},
		{
			script: "['f32'] = ['f32'] * 2",
			column: "f32",
			want:   arrow.PrimitiveTypes.Float32,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[float32, *array.Float32](t, arr, func(i int) (float32, bool) {
					return float32(i) / 4 * 2, true
				})
			},
		},
		{
			script: "['f32'] = ['f32'] + 0.1",
			column: "f32",
			want:   arrow.PrimitiveTypes.Float64,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[float64, *array.Float64](t, arr, func(i int) (float64, bool) {
					return float64(float32(i)/4) + 0.1, true
				})
			},
		},
		{
			script: "['d32'] = LAG(['d32'], 1)",
			column: "d32",
			want:   arrow.FixedWidthTypes.Date32,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[arrow.Date32, *array.Date32](t, arr, func(i int) (arrow.Date32, bool) {
					// No row above the first, so it has nothing to take.
					return arrow.Date32FromTime(start.AddDate(0, 0, i-1)), i > 0
				})
			},
		},
		{
			// The operands are int32s: CCL does no arithmetic on an int16 cell, so the
			// int16 column is assigned numbers computed from another column. 999 *
			// 1000 is past int16's range, so the column is not an int16 any more.
			script: "['i16'] = ['i32'] * 1000",
			column: "i16",
			want:   arrow.PrimitiveTypes.Float64,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[float64, *array.Float64](t, arr, func(i int) (float64, bool) {
					if i%7 == 0 {
						return 0, true
					}
					return float64(i%1000) * 1000, true
				})
			},
		},
		{
			script: "['i16'] = ['i32'] - 1",
			column: "i16",
			want:   arrow.PrimitiveTypes.Int16,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[int16, *array.Int16](t, arr, func(i int) (int16, bool) {
					if i%7 == 0 {
						return -1, true
					}
					return int16(i%1000) - 1, true
				})
			},
		},
		{
			// Moved down a row, an int16 cell goes through CCL as the cell it is.
			script: "['i16'] = LAG(['i16'], 1)",
			column: "i16",
			want:   arrow.PrimitiveTypes.Int16,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[int16, *array.Int16](t, arr, func(i int) (int16, bool) {
					return int16((i - 1) % 300), i > 0
				})
			},
		},
		{
			// Every value of the first batch fits an int16 and a later one does not,
			// which is a type settled wrongly the first time and settled again from
			// the whole file.
			script: "['i16'] = IF(# < 1500, 1, 40000)",
			column: "i16",
			want:   arrow.PrimitiveTypes.Float64,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[float64, *array.Float64](t, arr, func(i int) (float64, bool) {
					if i < 1500 {
						return 1, true
					}
					return 40000, true
				})
			},
		},
		{
			// 0.5 is a float32 and 0.1 is not, which comes only in a later batch.
			script: "['f32'] = IF(# < 1500, 0.5, 0.1)",
			column: "f32",
			want:   arrow.PrimitiveTypes.Float64,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[float64, *array.Float64](t, arr, func(i int) (float64, bool) {
					if i < 1500 {
						return 0.5, true
					}
					return 0.1, true
				})
			},
		},
		{
			script: "['u64'] = ['u64']",
			column: "u64",
			want:   arrow.PrimitiveTypes.Uint64,
			check: func(t *testing.T, arr arrow.Array) {
				checkColumn[uint64, *array.Uint64](t, arr, func(i int) (uint64, bool) {
					return math.MaxUint64 - uint64(i), true
				})
			},
		},
	} {
		t.Run(tt.script, func(t *testing.T) {
			path := writeManyTypesFile(t, manyTypesRows)
			if err := ApplyCCL(context.Background(), path, tt.script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", tt.script, err)
			}

			tbl := readArrowTable(t, path)
			defer tbl.Release()
			idx := tbl.Schema().FieldIndices(tt.column)
			if len(idx) == 0 {
				t.Fatalf("ApplyCCL(%q) dropped column %q", tt.script, tt.column)
			}
			arr := oneColumn(t, tbl, idx[0])
			defer arr.Release()

			if got := arr.DataType(); !arrow.TypeEqual(got, tt.want) {
				t.Fatalf("ApplyCCL(%q) wrote column %q as %s, want %s", tt.script, tt.column, got, tt.want)
			}
			tt.check(t, arr)
		})
	}
}

// TestHoldsEveryValueInNarrowTypes holds each type narrower than the 64-bit ones
// to exactly the values it holds: a whole number inside its range from any Go
// integer or from a float, a float32 only for a float that survives the trip
// through it, and a date only for midnight UTC.
func TestHoldsEveryValueInNarrowTypes(t *testing.T) {
	midnight := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	plusEight := time.FixedZone("UTC+8", 8*3600)
	// 2024-03-05 08:00 at UTC+8 is midnight UTC: the same instant as midnight.
	midnightElsewhere := time.Date(2024, 3, 5, 8, 0, 0, 0, plusEight)
	const float32Limit = 1 << 24

	var (
		int8T    = arrow.PrimitiveTypes.Int8
		int16T   = arrow.PrimitiveTypes.Int16
		int32T   = arrow.PrimitiveTypes.Int32
		uint8T   = arrow.PrimitiveTypes.Uint8
		uint16T  = arrow.PrimitiveTypes.Uint16
		uint32T  = arrow.PrimitiveTypes.Uint32
		uint64T  = arrow.PrimitiveTypes.Uint64
		float32T = arrow.PrimitiveTypes.Float32
		date32T  = arrow.FixedWidthTypes.Date32
		date64T  = arrow.FixedWidthTypes.Date64
	)

	for _, tt := range []struct {
		dtype arrow.DataType
		value any
		want  bool
	}{
		{int8T, 127, true},
		{int8T, 128, false},
		{int8T, -128, true},
		{int8T, -129, false},
		{int8T, int64(127), true},
		{int8T, uint64(128), false},
		{int8T, uint8(127), true},
		{int8T, 5.0, true},
		{int8T, float32(-100), true},
		{int8T, 1.5, false},
		{int8T, 128.0, false},
		{int8T, math.NaN(), false},
		{int8T, math.Inf(1), false},
		{int8T, "7", false},
		{int8T, true, false},
		{int8T, midnight, false},
		{int8T, []byte{1}, false},

		{int16T, 32767, true},
		{int16T, 32768, false},
		{int16T, -32768, true},
		{int16T, -32769, false},
		{int16T, 299 * 1000.0, false},
		{int16T, 299.0, true},

		{int32T, math.MaxInt32, true},
		{int32T, math.MaxInt32 + 1, false},
		{int32T, math.MinInt32, true},
		{int32T, math.MinInt32 - 1, false},
		{int32T, uint32(math.MaxUint32), false},
		{int32T, float64(math.MaxInt32), true},
		{int32T, float64(math.MaxInt32) + 1, false},
		{int32T, 0.5, false},

		{uint8T, 255, true},
		{uint8T, 256, false},
		{uint8T, 0, true},
		{uint8T, -1, false},
		{uint8T, -1.0, false},
		{uint8T, math.Copysign(0, -1), true},
		{uint8T, 255.0, true},
		{uint8T, 255.5, false},

		{uint16T, 65535, true},
		{uint16T, 65536, false},
		{uint16T, -1, false},

		{uint32T, uint32(math.MaxUint32), true},
		{uint32T, int64(math.MaxUint32) + 1, false},
		{uint32T, -1, false},
		{uint32T, float64(math.MaxUint32), true},

		{uint64T, uint64(math.MaxUint64), true},
		{uint64T, uint(math.MaxUint64), true},
		{uint64T, int64(math.MaxInt64), true},
		{uint64T, int64(-1), false},
		{uint64T, 0, true},
		{uint64T, float64(1 << 63), true},
		{uint64T, float64(1 << 53), true},
		{uint64T, math.Ldexp(1, 64), false},
		{uint64T, -1.0, false},
		{uint64T, 0.5, false},

		{float32T, float32(0.1), true},
		{float32T, 0.1, false},
		{float32T, 0.5, true},
		{float32T, float64(float32(0.1)), true},
		{float32T, math.NaN(), true},
		{float32T, math.Inf(1), true},
		{float32T, math.Inf(-1), true},
		{float32T, math.MaxFloat32, true},
		{float32T, 1e300, false},
		{float32T, 1e-50, false},
		{float32T, float32Limit, true},
		{float32T, float32Limit + 1, false},
		{float32T, -float32Limit, true},
		{float32T, -float32Limit - 1, false},
		{float32T, int64(float32Limit), true},
		{float32T, int64(-float32Limit) - 1, false},
		{float32T, uint64(float32Limit), true},
		{float32T, uint64(float32Limit) + 1, false},
		{float32T, int64(math.MinInt64), false},
		{float32T, "0.5", false},
		{float32T, true, false},

		{date32T, midnight, true},
		{date32T, midnightElsewhere, true},
		{date32T, midnight.Add(time.Second), false},
		{date32T, midnight.Add(time.Nanosecond), false},
		{date32T, midnight.Add(12 * time.Hour), false},
		{date32T, time.Date(2024, 3, 5, 0, 0, 0, 0, plusEight), false},
		{date32T, time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), true},
		{date32T, time.Date(6_000_000, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{date32T, 19000, false},
		{date32T, "2024-03-05", false},

		{date64T, midnight, true},
		{date64T, midnightElsewhere, true},
		{date64T, midnight.Add(time.Second), false},
		{date64T, midnight.Add(time.Millisecond), false},
		{date64T, midnight.Add(time.Nanosecond), false},
		{date64T, midnight.Add(12 * time.Hour), false},
		{date64T, time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), true},
		{date64T, 19000, false},
	} {
		name := fmt.Sprintf("%s holds %T %v", tt.dtype, tt.value, tt.value)
		t.Run(name, func(t *testing.T) {
			var kinds columnKinds
			kinds.add([]any{tt.value})
			if got := holdsEveryValue(&kinds, tt.dtype); got != tt.want {
				t.Errorf("holdsEveryValue(%s) of %v (%T) = %v, want %v", tt.dtype, tt.value, tt.value, got, tt.want)
			}
		})
	}

	all := []arrow.DataType{int8T, int16T, int32T, uint8T, uint16T, uint32T, uint64T, float32T, date32T, date64T}

	t.Run("a column of nothing but missing values holds every type", func(t *testing.T) {
		var kinds columnKinds
		kinds.add([]any{nil, nil})
		for _, dtype := range all {
			if !holdsEveryValue(&kinds, dtype) {
				t.Errorf("holdsEveryValue(%s) refused a column with no value in it", dtype)
			}
		}
	})

	t.Run("a missing value is no reason to refuse a type", func(t *testing.T) {
		var kinds columnKinds
		kinds.add([]any{nil, 1, nil, 127})
		if !holdsEveryValue(&kinds, int8T) {
			t.Error("holdsEveryValue(int8) refused 1 and 127 beside missing values")
		}
		var dates columnKinds
		dates.add([]any{nil, midnight, nil})
		if !holdsEveryValue(&dates, date32T) {
			t.Error("holdsEveryValue(date32) refused a midnight beside missing values")
		}
	})

	t.Run("a value in a later batch counts", func(t *testing.T) {
		var kinds columnKinds
		kinds.add([]any{1, 2, 3})
		kinds.add([]any{nil})
		if !holdsEveryValue(&kinds, int8T) {
			t.Fatal("holdsEveryValue(int8) refused a column of 1, 2 and 3")
		}
		kinds.add([]any{128})
		if holdsEveryValue(&kinds, int8T) {
			t.Error("holdsEveryValue(int8) held a column with a 128 in a later batch")
		}
		// A value one type cannot hold says nothing about another.
		if !holdsEveryValue(&kinds, int16T) {
			t.Error("holdsEveryValue(int16) refused a column the int8 refusal does not touch")
		}
	})

	t.Run("a mix of kinds is held by no narrow type", func(t *testing.T) {
		for _, mix := range [][]any{
			{1, "x"},
			{1, true},
			{1, midnight},
			{1, []byte{1}},
			{midnight, 1},
			{midnight, "x"},
		} {
			var kinds columnKinds
			kinds.add(mix)
			for _, dtype := range all {
				if holdsEveryValue(&kinds, dtype) {
					t.Errorf("holdsEveryValue(%s) held a column of %v", dtype, mix)
				}
			}
		}
	})

	t.Run("a number is not a date and a date is not a number", func(t *testing.T) {
		var dates columnKinds
		dates.add([]any{midnight})
		for _, dtype := range []arrow.DataType{int8T, int16T, int32T, uint8T, uint16T, uint32T, uint64T, float32T} {
			if holdsEveryValue(&dates, dtype) {
				t.Errorf("holdsEveryValue(%s) held a column of dates", dtype)
			}
		}
		var numbers columnKinds
		numbers.add([]any{1, 2})
		for _, dtype := range []arrow.DataType{date32T, date64T} {
			if holdsEveryValue(&numbers, dtype) {
				t.Errorf("holdsEveryValue(%s) held a column of numbers", dtype)
			}
		}
	})
}

// TestBuildArrowArrayBuildsNarrowTypes holds each narrow type's builder to the
// values holdsEveryValue lets it hold: whatever Go type they arrive as, they come
// out as the value itself in the type's own width, and a missing value is a null.
func TestBuildArrowArrayBuildsNarrowTypes(t *testing.T) {
	midnight := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		dtype  arrow.DataType
		values []any
		want   []string
	}{
		{arrow.PrimitiveTypes.Int8, []any{1, nil, int64(-128), 127.0, uint8(7), float32(-3)}, []string{"1", "null", "-128", "127", "7", "-3"}},
		{arrow.PrimitiveTypes.Int16, []any{-32768, 32767.0, nil}, []string{"-32768", "32767", "null"}},
		{arrow.PrimitiveTypes.Int32, []any{math.MinInt32, float64(math.MaxInt32), nil}, []string{"-2147483648", "2147483647", "null"}},
		{arrow.PrimitiveTypes.Uint8, []any{0, 255.0, math.Copysign(0, -1), nil}, []string{"0", "255", "0", "null"}},
		{arrow.PrimitiveTypes.Uint16, []any{65535, uint16(1), nil}, []string{"65535", "1", "null"}},
		{arrow.PrimitiveTypes.Uint32, []any{uint32(math.MaxUint32), 7.0, nil}, []string{"4294967295", "7", "null"}},
		{arrow.PrimitiveTypes.Uint64, []any{uint64(math.MaxUint64), int64(5), float64(1 << 63), nil}, []string{"18446744073709551615", "5", "9223372036854775808", "null"}},
		{arrow.PrimitiveTypes.Float32, []any{float32(0.1), 0.5, nil, int64(1 << 24), -3, math.Inf(1)}, []string{"0.1", "0.5", "null", "1.6777216e+07", "-3", "+Inf"}},
		{arrow.FixedWidthTypes.Date32, []any{midnight, nil, midnight.AddDate(0, 0, -1)}, []string{"2024-03-05", "null", "2024-03-04"}},
		{arrow.FixedWidthTypes.Date64, []any{midnight, nil, midnight.AddDate(0, 0, 1)}, []string{"2024-03-05", "null", "2024-03-06"}},
	} {
		t.Run(tt.dtype.String(), func(t *testing.T) {
			arr, err := buildArrowArray(memory.DefaultAllocator, "c", tt.values, tt.dtype)
			if err != nil {
				t.Fatalf("buildArrowArray(%v) as %s: %v", tt.values, tt.dtype, err)
			}
			defer arr.Release()
			if !arrow.TypeEqual(arr.DataType(), tt.dtype) {
				t.Fatalf("buildArrowArray built a %s array, want %s", arr.DataType(), tt.dtype)
			}
			got := make([]string, arr.Len())
			for i := range got {
				if arr.IsNull(i) {
					got[i] = "null"
				} else {
					got[i] = arr.ValueStr(i)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildArrowArray(%v) as %s holds %v, want %v", tt.values, tt.dtype, got, tt.want)
			}
		})
	}
}

// TestBuildArrowArrayRefusesWhatANarrowTypeCannotHold holds each narrow type's
// builder to an error, naming the column and the type, for a value it cannot hold
// as it is, where a conversion would have written another number: a value that
// wraps around, a fraction cut to a whole number, a float rounded to a float32, a
// text, or a time of day in a date.
func TestBuildArrowArrayRefusesWhatANarrowTypeCannotHold(t *testing.T) {
	midnight := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		dtype arrow.DataType
		value any
	}{
		{arrow.PrimitiveTypes.Int8, 128},
		{arrow.PrimitiveTypes.Int8, -129},
		{arrow.PrimitiveTypes.Int8, 1.5},
		{arrow.PrimitiveTypes.Int8, math.NaN()},
		{arrow.PrimitiveTypes.Int8, "7"},
		{arrow.PrimitiveTypes.Int8, true},
		{arrow.PrimitiveTypes.Int16, 32768},
		{arrow.PrimitiveTypes.Int32, math.MaxInt32 + 1},
		{arrow.PrimitiveTypes.Int32, 0.5},
		{arrow.PrimitiveTypes.Uint8, -1},
		{arrow.PrimitiveTypes.Uint8, 256},
		{arrow.PrimitiveTypes.Uint16, 65536},
		{arrow.PrimitiveTypes.Uint32, int64(math.MaxUint32) + 1},
		{arrow.PrimitiveTypes.Uint64, -1},
		{arrow.PrimitiveTypes.Uint64, math.Ldexp(1, 64)},
		{arrow.PrimitiveTypes.Float32, 0.1},
		{arrow.PrimitiveTypes.Float32, 1 << 25},
		{arrow.PrimitiveTypes.Float32, "0.5"},
		{arrow.FixedWidthTypes.Date32, midnight.Add(time.Hour)},
		{arrow.FixedWidthTypes.Date32, 19000},
		{arrow.FixedWidthTypes.Date64, midnight.Add(time.Millisecond)},
		{arrow.FixedWidthTypes.Date64, "2024-03-05"},
	} {
		name := fmt.Sprintf("%s refuses %T %v", tt.dtype, tt.value, tt.value)
		t.Run(name, func(t *testing.T) {
			arr, err := buildArrowArray(memory.DefaultAllocator, "c", []any{tt.value}, tt.dtype)
			if err == nil {
				arr.Release()
				t.Fatalf("buildArrowArray wrote %v (%T) as %s without an error", tt.value, tt.value, tt.dtype)
			}
			if !strings.Contains(err.Error(), `"c"`) || !strings.Contains(err.Error(), tt.dtype.String()) {
				t.Errorf("error %q names neither the column nor %s", err, tt.dtype)
			}
		})
	}
}

// narrowTypeRows is how many rows writeNarrowTypesFile writes.
const narrowTypeRows = 40

// narrowTypeColumn is one column of writeNarrowTypesFile: the type it has, and
// the value its row i holds as the string an Arrow array of that type gives it.
type narrowTypeColumn struct {
	name  string
	dtype arrow.DataType
	want  func(i int) string
}

// narrowTypeColumns are the columns of writeNarrowTypesFile, one for each type
// narrower than the 64-bit ones that a column read from a file can have. Every
// value is one its type holds and, but for the 64-bit unsigned column, one a
// smaller type could not hold. There is no date64 column: Parquet has one DATE
// type, which counts days, and Arrow reads it as a date32 even from a file that
// stores a date64 schema, so no file read has a date64 column and its builder is
// held by TestBuildArrowArrayBuildsNarrowTypes alone.
var narrowTypeColumns = []narrowTypeColumn{
	{"i8", arrow.PrimitiveTypes.Int8, func(i int) string { return fmt.Sprint(i - 20) }},
	{"i16", arrow.PrimitiveTypes.Int16, func(i int) string { return fmt.Sprint(i*100 - 2000) }},
	{"i32", arrow.PrimitiveTypes.Int32, func(i int) string { return fmt.Sprint(i*1_000_000 - 20_000_000) }},
	{"u8", arrow.PrimitiveTypes.Uint8, func(i int) string { return fmt.Sprint(i * 5) }},
	{"u16", arrow.PrimitiveTypes.Uint16, func(i int) string { return fmt.Sprint(i * 1000) }},
	{"u32", arrow.PrimitiveTypes.Uint32, func(i int) string { return fmt.Sprint(uint32(i) * 100_000_000) }},
	{"u64", arrow.PrimitiveTypes.Uint64, func(i int) string { return fmt.Sprint(math.MaxUint64 - uint64(i)) }},
	{"f32", arrow.PrimitiveTypes.Float32, func(i int) string { return fmt.Sprint(float32(i) / 8) }},
	{"d32", arrow.FixedWidthTypes.Date32, func(i int) string { return narrowTypeDay(i) }},
}

// narrowTypeDay is the day writeNarrowTypesFile gives row i: the days start on
// 2024-01-01.
func narrowTypeDay(i int) string {
	return time.Date(2024, 1, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// writeNarrowTypesFile writes narrowTypeRows rows of narrowTypeColumns, and
// returns the path of the file.
func writeNarrowTypesFile(t *testing.T) string {
	t.Helper()

	fields := make([]arrow.Field, len(narrowTypeColumns))
	for i, c := range narrowTypeColumns {
		fields[i] = arrow.Field{Name: c.name, Type: c.dtype, Nullable: true}
	}
	schema := arrow.NewSchema(fields, nil)

	b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer b.Release()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range narrowTypeRows {
		b.Field(0).(*array.Int8Builder).Append(int8(i - 20))
		b.Field(1).(*array.Int16Builder).Append(int16(i*100 - 2000))
		b.Field(2).(*array.Int32Builder).Append(int32(i*1_000_000 - 20_000_000))
		b.Field(3).(*array.Uint8Builder).Append(uint8(i * 5))
		b.Field(4).(*array.Uint16Builder).Append(uint16(i * 1000))
		b.Field(5).(*array.Uint32Builder).Append(uint32(i) * 100_000_000)
		b.Field(6).(*array.Uint64Builder).Append(math.MaxUint64 - uint64(i))
		b.Field(7).(*array.Float32Builder).Append(float32(i) / 8)
		b.Field(8).(*array.Date32Builder).Append(arrow.Date32FromTime(start.AddDate(0, 0, i)))
	}
	rec := b.NewRecord()
	defer rec.Release()

	path := filepath.Join(t.TempDir(), "narrow-types.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating the narrow-types file: %v", err)
	}
	w, err := pqarrow.NewFileWriter(schema, f, parquet.NewWriterProperties(),
		pqarrow.NewArrowWriterProperties(pqarrow.WithStoreSchema()))
	if err != nil {
		t.Fatalf("creating the writer for the narrow-types file: %v", err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatalf("writing the narrow-types file: %v", err)
	}
	// pqarrow's writer closes the underlying file.
	if err := w.Close(); err != nil {
		t.Fatalf("closing the narrow-types file: %v", err)
	}
	return path
}

// TestApplyCCLBuildsEveryNarrowType assigns each narrow column the value of the
// row above it, which changes no value and so asks for the type the file gave the
// column: it comes back as that type, row 0 a null and every other row the row
// above it, as it was written and read by Arrow rather than by Go.
func TestApplyCCLBuildsEveryNarrowType(t *testing.T) {
	for _, c := range narrowTypeColumns {
		script := fmt.Sprintf("['%s'] = LAG(['%s'], 1)", c.name, c.name)

		t.Run(script, func(t *testing.T) {
			path := writeNarrowTypesFile(t)
			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}

			tbl := readArrowTable(t, path)
			defer tbl.Release()
			idx := tbl.Schema().FieldIndices(c.name)
			if len(idx) == 0 {
				t.Fatalf("ApplyCCL(%q) dropped column %q", script, c.name)
			}
			arr := oneColumn(t, tbl, idx[0])
			defer arr.Release()

			if got := arr.DataType(); !arrow.TypeEqual(got, c.dtype) {
				t.Fatalf("ApplyCCL(%q) wrote column %q as %s, want %s", script, c.name, got, c.dtype)
			}
			if arr.Len() != narrowTypeRows {
				t.Fatalf("ApplyCCL(%q) left %d rows, want %d", script, arr.Len(), narrowTypeRows)
			}
			if !arr.IsNull(0) {
				t.Errorf("row 0 is %s, want a null", arr.ValueStr(0))
			}
			for i := 1; i < arr.Len(); i++ {
				if arr.IsNull(i) {
					t.Fatalf("row %d is a null, want %s", i, c.want(i-1))
				}
				if got, want := arr.ValueStr(i), c.want(i-1); got != want {
					t.Fatalf("row %d is %s, want %s", i, got, want)
				}
			}
		})
	}
}
