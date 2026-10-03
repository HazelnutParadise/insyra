package parquet

import (
	"bytes"
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// ApplyCCL writes the file back through its own builder, so a column it writes
// has to come out with the type and the missing values Write would give it. The
// tests below hold the written file to the one Write would produce for the same
// script on the same data: the script run on the table Read gives, then written
// with Write and read back. A column's type settled from the first rows the file
// arrives in, a value that cannot be converted and a column type the builder
// does not build are all ways that used to end in a panic or a wrong answer.

// expectedWrittenFile returns what ApplyCCL has to leave behind for the file at
// path: the script run on the table Read gives, then written with Write and read
// back, which is the file Write would put on disk for the table the script
// leaves.
func expectedWrittenFile(t *testing.T, path, script string) *insyra.DataTable {
	t.Helper()
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read(%s): %v", path, err)
	}
	dt.ExecuteCCL(script)
	if dt.Err() != nil {
		t.Fatalf("ExecuteCCL(%q) on the loaded table: %v", script, dt.Err())
	}

	out := filepath.Join(t.TempDir(), "expected.parquet")
	if err := Write(dt, out); err != nil {
		t.Fatalf("Write of the table ExecuteCCL(%q) leaves: %v", script, err)
	}
	want, err := Read(context.Background(), out, ReadOptions{})
	if err != nil {
		t.Fatalf("Read of the written expectation: %v", err)
	}
	if want.Err() != nil {
		t.Fatalf("the expectation carries an error: %v", want.Err())
	}
	return want
}

// sameWrittenCell compares one cell of two tables: the same Go type, and for a
// float64 the same bits, so a value a rounding step away is a difference and
// two NaNs are the same value. Anything of a different type is a difference too,
// which is the point: an int64 written where Write would write a float64 is a
// different file, even when the numbers agree.
func sameWrittenCell(got, want any) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	if reflect.TypeOf(got) != reflect.TypeOf(want) {
		return false
	}
	switch w := want.(type) {
	case float64:
		g := got.(float64)
		if math.IsNaN(g) && math.IsNaN(w) {
			return true
		}
		return math.Float64bits(g) == math.Float64bits(w)
	case []byte:
		return bytes.Equal(got.([]byte), w)
	case time.Time:
		return got.(time.Time).Equal(w)
	default:
		return reflect.DeepEqual(got, want)
	}
}

// sameWrittenTable reports whether got holds what want does: the same columns in
// the same order, and every cell the same value of the same type.
func sameWrittenTable(t *testing.T, got, want *insyra.DataTable, script string) {
	t.Helper()
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}
	gotNames, wantNames := got.ColNames(), want.ColNames()
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("ApplyCCL(%q) wrote the columns %v, Write writes %v", script, gotNames, wantNames)
	}
	for _, name := range wantNames {
		gotData := got.GetColByName(name).Data()
		wantData := want.GetColByName(name).Data()
		if len(gotData) != len(wantData) {
			t.Fatalf("ApplyCCL(%q) column %q holds %d values, Write writes %d",
				script, name, len(gotData), len(wantData))
		}
		for i, w := range wantData {
			if !sameWrittenCell(gotData[i], w) {
				t.Fatalf("ApplyCCL(%q) column %q row %d = %v (%T), Write writes %v (%T)",
					script, name, i, gotData[i], gotData[i], w, w)
			}
		}
	}
}

// writtenFileScripts are the scripts whose written column changes type past the
// first rows the file arrives in: a value of another kind beside the first
// batch's, a number where the first batch held whole numbers, a boolean where
// the first batch held numbers, a replacement of a column of the file, and a
// window wider than the file's first row group.
var writtenFileScripts = []string{
	"NEW('c') = IF(A > 1200, 'big', A)",
	"NEW('c') = IF(# < 1000, B, B / 2)",
	"NEW('c') = IF(# < 1000, B, B > 3)",
	"['B'] = B / 2",
	"['B'] = B > 3",
	"NEW('r') = ROLLING_MEAN(A, 1500)",
}

// TestApplyCCLWritesWhatWriteWouldWrite holds every column the script writes to
// the type Write gives a table holding the same values, which is the rule the
// column types are settled by: from the values themselves, not from the first
// rows the file arrives in.
func TestApplyCCLWritesWhatWriteWouldWrite(t *testing.T) {
	for _, script := range writtenFileScripts {
		t.Run(script, func(t *testing.T) {
			want := expectedWrittenFile(t, writeWholeFileFixture(t), script)

			path := writeWholeFileFixture(t)
			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}
			got, err := Read(context.Background(), path, ReadOptions{})
			if err != nil {
				t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
			}
			sameWrittenTable(t, got, want, script)
		})
	}
}

// writeSmallRowGroupFixture writes the whole-file fixture's data again with row
// groups smaller than the batch ApplyCCL reads, so a column's type cannot be
// settled from the first row group and the writer's own zero takes the place of
// the missing values that lead the column.
func writeSmallRowGroupFixture(t *testing.T) string {
	t.Helper()
	src := writeWholeFileFixture(t)
	dt, err := Read(context.Background(), src, ReadOptions{})
	if err != nil {
		t.Fatalf("Read(%s): %v", src, err)
	}
	path := filepath.Join(t.TempDir(), "small-row-groups.parquet")
	if err := Write(dt, path, WriteOptions{RowGroupSize: 1000}); err != nil {
		t.Fatalf("writing the small-row-group fixture: %v", err)
	}
	return path
}

// TestApplyCCLRetypesWithSmallRowGroups holds the same window in a file whose
// row groups are shorter than the window: the first run of the pipeline answers
// nothing for it, so a type settled from that run alone is the type of nothing
// and every number after it would be written as text.
func TestApplyCCLRetypesWithSmallRowGroups(t *testing.T) {
	const script = "NEW('r') = ROLLING_MEAN(A, 1500)"

	want := expectedWrittenFile(t, writeSmallRowGroupFixture(t), script)

	path := writeSmallRowGroupFixture(t)
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	sameWrittenTable(t, got, want, script)

	data := got.GetColByName("r").Data()
	values := 0
	for i, v := range data {
		if v == nil {
			continue
		}
		values++
		if _, ok := v.(float64); !ok {
			t.Fatalf("ApplyCCL(%q) column r row %d = %v (%T), want a float64",
				script, i, v, v)
		}
	}
	if values == 0 {
		t.Fatalf("ApplyCCL(%q) wrote no value into column r at all", script)
	}
}

// writeRequiredColumnFixture writes a file of ten rows whose only column A is
// declared not nullable, which a file written by Write never is, and returns its
// path.
func writeRequiredColumnFixture(t *testing.T) string {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "A", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	}, nil)

	builder := array.NewFloat64Builder(memory.DefaultAllocator)
	defer builder.Release()
	for i := 1; i <= 10; i++ {
		builder.Append(float64(i))
	}
	arr := builder.NewArray()
	defer arr.Release()
	rec := array.NewRecordBatch(schema, []arrow.Array{arr}, 10)
	defer rec.Release()

	path := filepath.Join(t.TempDir(), "required.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating the required-column fixture: %v", err)
	}
	w, err := pqarrow.NewFileWriter(rec.Schema(), f, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("creating the writer for the required-column fixture: %v", err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatalf("writing the required-column fixture: %v", err)
	}
	// Close writes the footer and closes the file the writer was given.
	if err := w.Close(); err != nil {
		t.Fatalf("closing the required-column fixture: %v", err)
	}
	return path
}

// TestApplyCCLKeepsAMissingValueInARequiredColumn holds a written column against
// a field the file declared not nullable: LAG has no value for the first row,
// and a field that cannot hold a missing value drops it, so the file comes back
// holding the writer's zero there.
func TestApplyCCLKeepsAMissingValueInARequiredColumn(t *testing.T) {
	const script = "['A'] = LAG(A, 1)"

	path := writeRequiredColumnFixture(t)
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}

	data := got.GetColByName("A").Data()
	if len(data) != 10 {
		t.Fatalf("ApplyCCL(%q) wrote column A with %d values, want 10", script, len(data))
	}
	if data[0] != nil {
		t.Errorf("ApplyCCL(%q) column A row 0 = %v (%T), want a missing value",
			script, data[0], data[0])
	}
	if got, want := data[1], 1.0; !sameWrittenCell(got, want) {
		t.Errorf("ApplyCCL(%q) column A row 1 = %v (%T), want %v (%T)",
			script, got, got, want, want)
	}
}

// writeTimeAndBytesFixture writes a file holding the two column types Write
// gives a table that only a cell's own type can decide: a timestamp and a byte
// slice, one cell each.
func writeTimeAndBytesFixture(t *testing.T) string {
	t.Helper()
	stamp := time.Date(2026, 3, 14, 15, 9, 26, 535897932, time.UTC)
	dt := insyra.NewDataTable(
		insyra.NewDataList(stamp, stamp.Add(time.Hour)).SetName("when"),
		insyra.NewDataList(
			insyra.Cell([]byte{0x00, 0x7f, 0xff}),
			insyra.Cell([]byte{0xde, 0xad}),
		).SetName("blob"),
	)
	path := filepath.Join(t.TempDir(), "time-and-bytes.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the time-and-bytes fixture: %v", err)
	}
	return path
}

// TestApplyCCLRoundTripsTimeAndBytes holds the two column types that only the
// builder can carry through a file ApplyCCL writes back: a timestamp and a byte
// slice have no other answer, so a column of either kind used to be written as
// text or refused.
func TestApplyCCLRoundTripsTimeAndBytes(t *testing.T) {
	const script = "NEW('n') = 1"

	want := expectedWrittenFile(t, writeTimeAndBytesFixture(t), script)

	path := writeTimeAndBytesFixture(t)
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	sameWrittenTable(t, got, want, script)

	// Said of the two columns themselves, so a change in either is named.
	when := got.GetColByName("when").Data()
	if len(when) != 2 {
		t.Fatalf("ApplyCCL(%q) wrote column when with %d values, want 2", script, len(when))
	}
	if _, ok := when[0].(time.Time); !ok {
		t.Errorf("ApplyCCL(%q) column when row 0 = %v (%T), want a time.Time", script, when[0], when[0])
	}
	blob := got.GetColByName("blob").Data()
	if len(blob) != 2 {
		t.Fatalf("ApplyCCL(%q) wrote column blob with %d values, want 2", script, len(blob))
	}
	if _, ok := blob[0].([]byte); !ok {
		t.Errorf("ApplyCCL(%q) column blob row 0 = %v (%T), want a []byte", script, blob[0], blob[0])
	}
}

// writeIntColumnFixture writes a file whose only column n holds 1 to 5 as an
// int64, the column Write gives a table of whole numbers, and returns its path.
func writeIntColumnFixture(t *testing.T) string {
	t.Helper()
	dt := insyra.NewDataTable(
		insyra.NewDataList(int64(1), int64(2), int64(3), int64(4), int64(5)).SetName("n"),
	)
	path := filepath.Join(t.TempDir(), "int-column.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the int-column fixture: %v", err)
	}
	return path
}

// TestApplyCCLKeepsAnAssignedColumnsTypeWhenItFits holds a column a statement
// assigns to at the type the file gave it, as long as every value written into it
// is one that type holds without loss.
//
// CCL's number literals are float64, so every arithmetic answer is a float64 and
// a column of whole numbers used to be written as one: ['n'] = A * 2 on a column
// of int64 came back holding 2.0 instead of 2, with the file's schema changed for
// nothing. A value the column's own type cannot hold still takes the type Write
// gives its values, which is what a fraction, a boolean or a missing value in an
// int64 column has to do.
func TestApplyCCLKeepsAnAssignedColumnsTypeWhenItFits(t *testing.T) {
	for _, tt := range []struct {
		script string
		want   []any
	}{
		// The whole numbers a literal-multiplied column answers: the file's int64
		// holds every one of them.
		{script: "['n'] = A * 2", want: []any{int64(2), int64(4), int64(6), int64(8), int64(10)}},
		// A fraction the int64 cannot hold: the column widens, as Write does.
		{script: "['n'] = A / 2", want: []any{0.5, 1.0, 1.5, 2.0, 2.5}},
		// A value of another kind altogether.
		{script: "['n'] = A > 2", want: []any{false, false, true, true, true}},
		// A missing value is no reason to change a type: every value the column
		// holds is an int64 the file's column already holds.
		{script: "['n'] = IF(A > 2, A, NULL)", want: []any{nil, nil, int64(3), int64(4), int64(5)}},
	} {
		t.Run(tt.script, func(t *testing.T) {
			path := writeIntColumnFixture(t)
			if err := ApplyCCL(context.Background(), path, tt.script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", tt.script, err)
			}

			got, err := Read(context.Background(), path, ReadOptions{})
			if err != nil {
				t.Fatalf("Read after ApplyCCL(%q): %v", tt.script, err)
			}
			if got.Err() != nil {
				t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
			}

			data := got.GetColByName("n").Data()
			if len(data) != len(tt.want) {
				t.Fatalf("ApplyCCL(%q) wrote column n with %d values, want %d",
					tt.script, len(data), len(tt.want))
			}
			for i, w := range tt.want {
				if !sameWrittenCell(data[i], w) {
					t.Errorf("ApplyCCL(%q) column n row %d = %v (%T), want %v (%T)",
						tt.script, i, data[i], data[i], w, w)
				}
			}
		})
	}
}

// TestApplyCCLKeepsATimestampColumnsUnit holds a written timestamp column to the
// field the file gave it. A column written as the type its values answer is the
// type Write infers, which is timestamp[ns] whatever unit and zone the file's own
// column was written at, so a file written elsewhere came back holding instants
// its schema no longer described.
func TestApplyCCLKeepsATimestampColumnsUnit(t *testing.T) {
	const script = "['t'] = LAG(['t'], 1)"

	start := time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC)
	stamps := make([]any, 4)
	for i := range stamps {
		stamps[i] = start.Add(time.Duration(i) * time.Hour)
	}
	dt := insyra.NewDataTable(
		insyra.NewDataList(stamps...).SetName("t"),
		insyra.NewDataList(int64(1), int64(2), int64(3), int64(4)).SetName("n"),
	)
	path := filepath.Join(t.TempDir(), "timestamps.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the timestamp fixture: %v", err)
	}

	before, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect before ApplyCCL: %v", err)
	}
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	after, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect after ApplyCCL: %v", err)
	}

	// Every column's name and type, the written one included: only nullability
	// may change, and nullability is not what Read reports here.
	if len(before.Columns) != len(after.Columns) {
		t.Fatalf("ApplyCCL(%q) wrote %d columns, the file had %d",
			script, len(after.Columns), len(before.Columns))
	}
	for i, want := range before.Columns {
		got := after.Columns[i]
		if got.Name != want.Name {
			t.Errorf("ApplyCCL(%q) wrote column %d as %q, the file had %q", script, i, got.Name, want.Name)
			continue
		}
		if got.PhysicalType != want.PhysicalType || got.LogicalType != want.LogicalType {
			t.Errorf("ApplyCCL(%q) wrote column %q as %s %s, the file had %s %s",
				script, got.Name, got.PhysicalType, got.LogicalType, want.PhysicalType, want.LogicalType)
		}
	}

	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}

	// LAG moves the column one row down: the first row has no earlier row to
	// take, and every other row holds the instant one row above it held.
	want := []any{nil, start, start.Add(time.Hour), start.Add(2 * time.Hour)}
	data := got.GetColByName("t").Data()
	if len(data) != len(want) {
		t.Fatalf("ApplyCCL(%q) wrote column t with %d values, want %d", script, len(data), len(want))
	}
	for i, w := range want {
		if !sameWrittenCell(data[i], w) {
			t.Errorf("ApplyCCL(%q) column t row %d = %v (%T), want %v (%T)",
				script, i, data[i], data[i], w, w)
		}
	}
}

// writeDateColumnFixture writes a file whose only column d has a type the
// builder cannot build an array of, which a file written by Write never has,
// and returns its path.
func writeDateColumnFixture(t *testing.T) string {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "d", Type: arrow.FixedWidthTypes.Date32, Nullable: true},
	}, nil)

	builder := array.NewDate32Builder(memory.DefaultAllocator)
	defer builder.Release()
	for i := 0; i < 5; i++ {
		builder.Append(arrow.Date32FromTime(time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)))
	}
	arr := builder.NewArray()
	defer arr.Release()
	rec := array.NewRecordBatch(schema, []arrow.Array{arr}, 5)
	defer rec.Release()

	path := filepath.Join(t.TempDir(), "dates.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating the date-column fixture: %v", err)
	}
	w, err := pqarrow.NewFileWriter(rec.Schema(), f, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("creating the writer for the date-column fixture: %v", err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatalf("writing the date-column fixture: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing the date-column fixture: %v", err)
	}
	return path
}

// TestApplyCCLRefusesAValueItCannotWriteWithoutPanicking holds the two ways a
// value cannot be written, both of which used to end the process: a column type
// the builder does not build, and a value that does not convert to the type the
// column has. The second is the one a column whose first rows decided its type
// reaches, so ApplyCCL itself is held to refusing it and leaving the file alone.
func TestApplyCCLRefusesAValueItCannotWriteWithoutPanicking(t *testing.T) {
	t.Run("a column type the builder does not build", func(t *testing.T) {
		arr, err := buildArrowArray(memory.DefaultAllocator, "d", []any{time.Now()}, arrow.ListOf(arrow.PrimitiveTypes.Int64))
		if err == nil {
			got := arr.DataType()
			arr.Release()
			t.Fatalf("buildArrowArray built a %s array for a list column", got)
		}
		if !strings.Contains(err.Error(), `"d"`) || !strings.Contains(err.Error(), "list") {
			t.Errorf("error %q names neither the column nor its type", err)
		}
	})

	t.Run("a value that does not convert", func(t *testing.T) {
		for _, tt := range []struct {
			dtype arrow.DataType
			value any
			want  string
		}{
			{dtype: arrow.PrimitiveTypes.Float64, value: "big", want: "float64"},
			{dtype: arrow.PrimitiveTypes.Int64, value: true, want: "int64"},
			{dtype: arrow.FixedWidthTypes.Boolean, value: []byte{1}, want: "bool"},
		} {
			arr, err := buildArrowArray(memory.DefaultAllocator, "c", []any{tt.value}, tt.dtype)
			if err == nil {
				arr.Release()
				t.Errorf("buildArrowArray wrote %v as %s without an error", tt.value, tt.dtype)
				continue
			}
			if !strings.Contains(err.Error(), `"c"`) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q names neither the column nor %s", err, tt.want)
			}
		}
	})

	t.Run("ApplyCCL writes back a column it does not write", func(t *testing.T) {
		// A column no statement writes is written back from the file's own data,
		// so a type the builder does not build is no reason to refuse the file.
		const script = "NEW('n') = 1"

		path := writeDateColumnFixture(t)
		want, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("reading the fixture: %v", err)
		}
		if err := ApplyCCL(context.Background(), path, script); err != nil {
			t.Fatalf("ApplyCCL(%q): %v", script, err)
		}
		got, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("reading the file back: %v", err)
		}
		wantD, gotD := want.GetColByName("d").Data(), got.GetColByName("d").Data()
		if len(gotD) != len(wantD) {
			t.Fatalf("ApplyCCL(%q) column \"d\" holds %d values, want %d", script, len(gotD), len(wantD))
		}
		for i, w := range wantD {
			wt, wok := w.(time.Time)
			gt, gok := gotD[i].(time.Time)
			if wok != gok || (wok && !gt.Equal(wt)) || (!wok && gotD[i] != w) {
				t.Fatalf("ApplyCCL(%q) column \"d\" row %d = %v (%T), want %v (%T)", script, i, gotD[i], gotD[i], w, w)
			}
		}
	})
}

// writeFloatAndBigIntFixture writes a file of two rows whose column F is a float64
// and whose column I is an int64 holding first, which is a value a float64 can
// hold only when it is within 2^53 of zero.
func writeFloatAndBigIntFixture(t *testing.T, first int64) string {
	t.Helper()
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.5, 2.5).SetName("F"),
		insyra.NewDataList(first, int64(3)).SetName("I"),
	)
	path := filepath.Join(t.TempDir(), "float-and-int.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the float-and-int fixture: %v", err)
	}
	return path
}

// TestApplyCCLKeepsALargeIntegerExact holds a float64 column that is assigned
// whole numbers to what a float64 holds exactly: every integer up to 2^53 and no
// further. Written back as a float64 an int64 above that limit comes back as a
// neighbour of itself, which is a number nobody wrote, so the column takes the
// type Write gives its values instead, as it does for a fraction in an int64
// column.
func TestApplyCCLKeepsALargeIntegerExact(t *testing.T) {
	const script = "['F'] = ['I']"

	for _, tt := range []struct {
		name  string
		first int64
		// want is what the first row of F holds afterwards.
		want any
	}{
		{name: "one past 2^53 is not a float64", first: 1<<53 + 1, want: int64(1<<53 + 1)},
		{name: "a negative one past 2^53 is not a float64", first: -(1<<53 + 1), want: int64(-(1<<53 + 1))},
		{name: "2^53 itself is a float64", first: 1 << 53, want: float64(1 << 53)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFloatAndBigIntFixture(t, tt.first)
			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}
			got, err := Read(context.Background(), path, ReadOptions{})
			if err != nil {
				t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
			}
			if got.Err() != nil {
				t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
			}

			data := got.GetColByName("F").Data()
			if len(data) != 2 {
				t.Fatalf("ApplyCCL(%q) wrote column F with %d values, want 2", script, len(data))
			}
			if !sameWrittenCell(data[0], tt.want) {
				t.Errorf("ApplyCCL(%q) column F row 0 = %v (%T), want %v (%T)",
					script, data[0], data[0], tt.want, tt.want)
			}
			// The other row is the 3 either type holds, written as the column's type.
			if f, ok := data[1].(float64); ok && f != 3 {
				t.Errorf("ApplyCCL(%q) column F row 1 = %v, want 3", script, f)
			} else if i, ok := data[1].(int64); ok && i != 3 {
				t.Errorf("ApplyCCL(%q) column F row 1 = %v, want 3", script, i)
			}
		})
	}
}

// TestHoldsEveryValueInAFloat64Column holds the float64 type to the numbers a
// float64 holds exactly: any float, and an integer of any Go type whose size is
// at most 2^53.
func TestHoldsEveryValueInAFloat64Column(t *testing.T) {
	const limit = 1 << 53
	for _, tt := range []struct {
		name  string
		value any
		want  bool
	}{
		{name: "a float64", value: 0.1, want: true},
		{name: "a float64 above 2^53", value: 1e300, want: true},
		{name: "a float32", value: float32(0.1), want: true},
		{name: "an int64 at 2^53", value: int64(limit), want: true},
		{name: "an int64 at -2^53", value: int64(-limit), want: true},
		{name: "an int64 past 2^53", value: int64(limit + 1), want: false},
		{name: "an int64 past -2^53", value: int64(-limit - 1), want: false},
		{name: "an int past 2^53", value: int(limit + 1), want: false},
		{name: "an int32 at its largest", value: int32(math.MaxInt32), want: true},
		{name: "a uint32 at its largest", value: uint32(math.MaxUint32), want: true},
		{name: "a uint64 at 2^53", value: uint64(limit), want: true},
		{name: "a uint64 past 2^53", value: uint64(limit + 1), want: false},
		{name: "a uint64 at its largest", value: uint64(math.MaxUint64), want: false},
		{name: "a uint past 2^53", value: uint(limit + 1), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var kinds columnKinds
			kinds.add([]any{tt.value})
			if got := holdsEveryValue(&kinds, arrow.PrimitiveTypes.Float64); got != tt.want {
				t.Errorf("holdsEveryValue(float64) of %v (%T) = %v, want %v", tt.value, tt.value, got, tt.want)
			}
		})
	}

	// One value past the limit is enough, wherever it stands among the others,
	// and every batch added counts.
	var kinds columnKinds
	kinds.add([]any{1.5, int64(3), nil})
	kinds.add([]any{int64(limit + 1)})
	if holdsEveryValue(&kinds, arrow.PrimitiveTypes.Float64) {
		t.Errorf("holdsEveryValue(float64) held a column that had a value past 2^53 in a later batch")
	}
}

// writeMissingTailFixture writes 3,000 rows of one float64 column A in row groups
// of 1,000, whose last 1,000 rows are missing values, and returns its path.
func writeMissingTailFixture(t *testing.T) string {
	t.Helper()
	a := make([]any, 3000)
	for i := range a {
		if i >= 2000 {
			continue
		}
		a[i] = float64(i + 1)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(a...).SetName("A"))
	path := filepath.Join(t.TempDir(), "missing-tail.parquet")
	if err := Write(dt, path, WriteOptions{RowGroupSize: 1000}); err != nil {
		t.Fatalf("writing the missing-tail fixture: %v", err)
	}
	return path
}

// TestApplyCCLDoesNotRewriteForMissingValues holds the type of a written column to
// the values it has seen so far, not to the ones in the batch at hand. A batch
// whose column is missing everywhere holds no kind of value, which says nothing
// against the type already settled, so the file is not read and written a second
// time for it.
func TestApplyCCLDoesNotRewriteForMissingValues(t *testing.T) {
	apply := func(t *testing.T, script string) int64 {
		t.Helper()
		want := expectedWrittenFile(t, writeMissingTailFixture(t), script)

		path := writeMissingTailFixture(t)
		writtenColumnTypesCalls.Store(0)
		if err := ApplyCCL(context.Background(), path, script); err != nil {
			t.Fatalf("ApplyCCL(%q): %v", script, err)
		}
		rewrites := writtenColumnTypesCalls.Load()

		got, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
		}
		sameWrittenTable(t, got, want, script)
		return rewrites
	}

	// The column has numbers from its first row and none in its last row group,
	// the second type a batch of nothing but missing values could be taken for.
	for _, script := range []string{
		"NEW('c') = A",
		"NEW('c') = LEAD(A, 1500)",
	} {
		t.Run(script, func(t *testing.T) {
			if rewrites := apply(t, script); rewrites != 0 {
				t.Errorf("ApplyCCL(%q) settled the types from the whole file %d times, want 0", script, rewrites)
			}
		})
	}

	// The first row group holds no value of c at all, so its type can only be
	// guessed, and the numbers after it may make the guess wrong once.
	t.Run("NEW('c') = LAG(A, 1500)", func(t *testing.T) {
		if rewrites := apply(t, "NEW('c') = LAG(A, 1500)"); rewrites > 1 {
			t.Errorf("ApplyCCL settled the types from the whole file %d times, want at most 1", rewrites)
		}
	})
}

// TestApplyCCLKeepsATimeZone holds a timestamp column to the zone the file gave
// it. The writer used to leave the file's Arrow schema out, and a file keeps a
// column's zone only there: parquet itself knows only whether the instants are
// in UTC, so every zoned column came back as UTC, whether or not the script wrote
// to it.
func TestApplyCCLKeepsATimeZone(t *testing.T) {
	zoned := &arrow.TimestampType{Unit: arrow.Millisecond, TimeZone: "Asia/Taipei"}

	writeFixture := func(t *testing.T) string {
		t.Helper()
		schema := arrow.NewSchema([]arrow.Field{
			{Name: "t", Type: zoned, Nullable: true},
			{Name: "n", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		}, nil)

		start := time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC)
		tb := array.NewTimestampBuilder(memory.DefaultAllocator, zoned)
		defer tb.Release()
		nb := array.NewInt64Builder(memory.DefaultAllocator)
		defer nb.Release()
		for i := 0; i < 4; i++ {
			ts, err := arrow.TimestampFromTime(start.Add(time.Duration(i)*time.Hour), arrow.Millisecond)
			if err != nil {
				t.Fatalf("building the timestamp fixture: %v", err)
			}
			tb.Append(ts)
			nb.Append(int64(i + 1))
		}
		tArr, nArr := tb.NewArray(), nb.NewArray()
		defer tArr.Release()
		defer nArr.Release()
		rec := array.NewRecordBatch(schema, []arrow.Array{tArr, nArr}, 4)
		defer rec.Release()

		path := filepath.Join(t.TempDir(), "zoned.parquet")
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("creating the zoned fixture: %v", err)
		}
		w, err := pqarrow.NewFileWriter(rec.Schema(), f, parquet.NewWriterProperties(),
			pqarrow.NewArrowWriterProperties(pqarrow.WithStoreSchema()))
		if err != nil {
			t.Fatalf("creating the writer for the zoned fixture: %v", err)
		}
		if err := w.Write(rec); err != nil {
			t.Fatalf("writing the zoned fixture: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("closing the zoned fixture: %v", err)
		}
		return path
	}

	// schemaOf reads the Arrow schema a file declares, which is where the zone is.
	schemaOf := func(t *testing.T, path string) *arrow.Schema {
		t.Helper()
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("opening %s: %v", path, err)
		}
		defer func() { _ = f.Close() }()
		r, err := file.NewParquetReader(f)
		if err != nil {
			t.Fatalf("reading the parquet file: %v", err)
		}
		defer func() { _ = r.Close() }()
		fr, err := pqarrow.NewFileReader(r, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
		if err != nil {
			t.Fatalf("reading the Arrow schema: %v", err)
		}
		schema, err := fr.Schema()
		if err != nil {
			t.Fatalf("reading the Arrow schema: %v", err)
		}
		return schema
	}

	if got := schemaOf(t, writeFixture(t)).Field(0).Type; !arrow.TypeEqual(got, zoned) {
		t.Fatalf("the fixture's column t is %s, want %s", got, zoned)
	}

	for _, script := range []string{
		"NEW('c') = 1",          // the column is not written
		"['t'] = LAG(['t'], 1)", // the column is written, with values it already held
	} {
		t.Run(script, func(t *testing.T) {
			path := writeFixture(t)
			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}
			schema := schemaOf(t, path)
			idx := schema.FieldIndices("t")
			if len(idx) != 1 {
				t.Fatalf("ApplyCCL(%q) left %d columns named t, want 1", script, len(idx))
			}
			if got := schema.Field(idx[0]).Type; !arrow.TypeEqual(got, zoned) {
				t.Errorf("ApplyCCL(%q) wrote column t as %s, want %s", script, got, zoned)
			}
		})
	}
}

// writeNoRowsFixture writes a file with the one float64 column A and no rows at
// all, which is a file with a schema and nothing in it, and returns its path.
func writeNoRowsFixture(t *testing.T) string {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "A", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	}, nil)

	builder := array.NewFloat64Builder(memory.DefaultAllocator)
	defer builder.Release()
	arr := builder.NewArray()
	defer arr.Release()
	rec := array.NewRecordBatch(schema, []arrow.Array{arr}, 0)
	defer rec.Release()

	path := filepath.Join(t.TempDir(), "no-rows.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating the no-rows fixture: %v", err)
	}
	w, err := pqarrow.NewFileWriter(rec.Schema(), f, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("creating the writer for the no-rows fixture: %v", err)
	}
	if err := w.Write(rec); err != nil {
		t.Fatalf("writing the no-rows fixture: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing the no-rows fixture: %v", err)
	}
	return path
}

// TestCCLComputesAggregatesOnAFileWithNoRows holds a file with no rows to what the
// loaded table answers: an expression whose sequence function takes an aggregate
// for its period has a value for it over no rows, so it is an answer and not an
// error, and an input with no rows leaves the file as it was.
func TestCCLComputesAggregatesOnAFileWithNoRows(t *testing.T) {
	const expr = "LAG(A, COUNT(A))"

	t.Run("ApplyCCL", func(t *testing.T) {
		script := "NEW('c') = " + expr
		path := writeNoRowsFixture(t)
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the fixture: %v", err)
		}
		if err := ApplyCCL(context.Background(), path, script); err != nil {
			t.Fatalf("ApplyCCL(%q): %v", script, err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the fixture back: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("ApplyCCL(%q) changed a file with no rows (%d bytes before, %d after)",
				script, len(before), len(after))
		}
	})

	t.Run("FilterWithCCL", func(t *testing.T) {
		res, err := FilterWithCCL(context.Background(), writeNoRowsFixture(t), expr)
		if err != nil {
			t.Fatalf("FilterWithCCL(%q): %v", expr, err)
		}
		if res.Err() != nil {
			t.Fatalf("FilterWithCCL's table carries an error: %v", res.Err())
		}
		if got := res.ColNames(); !reflect.DeepEqual(got, []string{"A"}) {
			t.Errorf("FilterWithCCL(%q) returned the columns %v, want [A]", expr, got)
		}
		if rows, _ := res.Size(); rows != 0 {
			t.Errorf("FilterWithCCL(%q) kept %d rows of a file with none", expr, rows)
		}
	})
}
