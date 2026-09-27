package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

// roundTripCell encodes v, writes the encoded value as JSON, reads it back
// through a decoder that keeps numbers as json.Number, and decodes the cell
// again — the path a stored variable takes between two sessions.
func roundTripCell(t *testing.T, v any) any {
	t.Helper()
	tag, value, err := encodeCell(v)
	if err != nil {
		t.Fatalf("encodeCell(%T) returned %v", v, err)
	}
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal of the encoded %T: %v", v, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var read any
	if err := dec.Decode(&read); err != nil {
		t.Fatalf("decoding %s: %v", b, err)
	}
	got, err := decodeCell(tag, read)
	if err != nil {
		t.Fatalf("decodeCell(%q, %#v) returned %v", tag, read, err)
	}
	return got
}

// encodeRejects asserts that encodeCell refuses v and that the message names
// the type it refused, so the caller learns which value to look at.
func encodeRejects(t *testing.T, v any, wantSubstring string) {
	t.Helper()
	tag, value, err := encodeCell(v)
	if err == nil {
		t.Fatalf("encodeCell(%T) accepted the value and returned tag %q, value %#v", v, tag, value)
	}
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Errorf("encodeCell(%T) error = %q, want it to mention %q", v, err.Error(), wantSubstring)
	}
}

func TestCellCodec_TagIsTheGoTypeName(t *testing.T) {
	// The tag is what a stored file carries, so it must be the name the Go
	// type prints, not a spelling chosen by the codec.
	for _, v := range []any{
		int8(1), uint16(1), float32(1), float64(1), true, "s",
		[]byte{1}, []any{json.Number("1")}, map[string]any{}, json.Number("1"),
		time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), decimal.Decimal{},
	} {
		want := fmt.Sprintf("%T", v)
		tag, _, err := encodeCell(v)
		if err != nil {
			t.Errorf("encodeCell(%T) returned %v", v, err)
			continue
		}
		if tag != want {
			t.Errorf("encodeCell(%T) tag = %q, want %q", v, tag, want)
		}
	}
}

func TestCellCodec_RoundTripsSignedIntegers(t *testing.T) {
	// 2^53+1 is not representable as a float64, so it also proves the value
	// travels as a written integer rather than through a float.
	for _, want := range []any{
		int(7),
		int8(-128),
		int16(32767),
		int32(-5),
		int64(math.MaxInt64),
		int64(math.MinInt64),
		int64(9007199254740993),
	} {
		got := roundTripCell(t, want)
		if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", want) {
			t.Errorf("%T round trip returned %T", want, got)
			continue
		}
		if got != want {
			t.Errorf("%T round trip = %v, want %v", want, got, want)
		}
	}
}

func TestCellCodec_RoundTripsUnsignedIntegers(t *testing.T) {
	for _, want := range []any{
		uint(3),
		uint8(255),
		uint16(1),
		uint32(4294967295),
		uint64(math.MaxUint64),
	} {
		got := roundTripCell(t, want)
		if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", want) {
			t.Errorf("%T round trip returned %T", want, got)
			continue
		}
		if got != want {
			t.Errorf("%T round trip = %v, want %v", want, got, want)
		}
	}
}

func TestCellCodec_RoundTripsFloat64ByBits(t *testing.T) {
	// Bits, not ==: 3.0 must not come back as an integer, and a negative zero
	// must keep its sign.
	for _, want := range []float64{
		3.0, 0.1, math.SmallestNonzeroFloat64, math.MaxFloat64, math.Copysign(0, -1),
	} {
		got, ok := roundTripCell(t, want).(float64)
		if !ok {
			t.Errorf("float64 %v round trip returned %T, want float64", want, roundTripCell(t, want))
			continue
		}
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Errorf("float64 %v round trip = %v (%016x), want %016x", want, got, math.Float64bits(got), math.Float64bits(want))
		}
	}
}

func TestCellCodec_RoundTripsFloat32ByBits(t *testing.T) {
	for _, want := range []float32{0.1, -1.5, math.SmallestNonzeroFloat32, math.MaxFloat32} {
		got, ok := roundTripCell(t, want).(float32)
		if !ok {
			t.Errorf("float32 %v round trip returned %T, want float32", want, roundTripCell(t, want))
			continue
		}
		if math.Float32bits(got) != math.Float32bits(want) {
			t.Errorf("float32 %v round trip = %v (%08x), want %08x", want, got, math.Float32bits(got), math.Float32bits(want))
		}
	}
}

func TestCellCodec_RoundTripsNonFiniteFloats(t *testing.T) {
	for _, want := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		got, ok := roundTripCell(t, want).(float64)
		if !ok {
			t.Errorf("float64 %v round trip returned %T, want float64", want, roundTripCell(t, want))
			continue
		}
		if !sameSpecialFloat64(got, want) {
			t.Errorf("float64 %v round trip = %v, want %v", want, got, want)
		}
	}
	for _, want := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		got, ok := roundTripCell(t, want).(float32)
		if !ok {
			t.Errorf("float32 %v round trip returned %T, want float32", want, roundTripCell(t, want))
			continue
		}
		if !sameSpecialFloat64(float64(got), float64(want)) {
			t.Errorf("float32 %v round trip = %v, want %v", want, got, want)
		}
	}
}

// sameSpecialFloat64 compares NaN by being NaN and infinities by sign, which
// == cannot do.
func sameSpecialFloat64(got, want float64) bool {
	switch {
	case math.IsNaN(want):
		return math.IsNaN(got)
	case math.IsInf(want, 1):
		return math.IsInf(got, 1)
	case math.IsInf(want, -1):
		return math.IsInf(got, -1)
	default:
		return got == want
	}
}

func TestCellCodec_RoundTripsBoolStringAndNumber(t *testing.T) {
	for _, want := range []any{true, false, "", "文字", json.Number("12.50")} {
		got := roundTripCell(t, want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%#v round trip = %#v (%T), want the same value and type", want, got, got)
		}
	}
}

func TestCellCodec_RoundTripsByteSlices(t *testing.T) {
	for _, want := range [][]byte{{0, 255, 10}, {}} {
		got, ok := roundTripCell(t, want).([]byte)
		if !ok {
			t.Errorf("[]byte %v round trip returned %T", want, roundTripCell(t, want))
			continue
		}
		// An empty slice must not come back nil: it is a different value to
		// anything comparing against it, and it prints as <nil>.
		if !bytes.Equal(got, want) || (len(got) == 0 && got == nil) {
			t.Errorf("[]byte %v round trip = %#v", want, got)
		}
	}
}

func TestCellCodec_RoundTripsTime(t *testing.T) {
	t.Run("UTC", func(t *testing.T) {
		want := time.Date(2024, 1, 2, 3, 4, 5, 6, time.UTC)
		got, ok := roundTripCell(t, want).(time.Time)
		if !ok {
			t.Fatalf("time round trip returned %T", roundTripCell(t, want))
		}
		if !got.Equal(want) {
			t.Errorf("time round trip = %v, want %v", got, want)
		}
		if got.Location() != time.UTC {
			t.Errorf("time round trip location = %v, want UTC", got.Location())
		}
	})

	t.Run("Local", func(t *testing.T) {
		want := time.Date(2024, 1, 2, 3, 4, 5, 0, time.Local)
		got, ok := roundTripCell(t, want).(time.Time)
		if !ok {
			t.Fatalf("time round trip returned %T", roundTripCell(t, want))
		}
		if !got.Equal(want) {
			t.Errorf("time round trip = %v, want %v", got, want)
		}
		_, wantOffset := want.Zone()
		if _, gotOffset := got.Zone(); gotOffset != wantOffset {
			t.Errorf("time round trip offset = %d, want %d", gotOffset, wantOffset)
		}
		if wantOffset != 0 {
			// time.Parse reads a stored offset back as time.Local when the
			// local zone has that offset. A zero offset is written as Z and
			// comes back in UTC, which is the same instant in a different
			// location, so the identity only holds off UTC.
			if got.Location() != time.Local {
				t.Errorf("time round trip location = %v, want time.Local", got.Location())
			}
		}
	})

	t.Run("FixedZone", func(t *testing.T) {
		want := time.Date(2024, 1, 2, 3, 4, 5, 0, time.FixedZone("", 5*3600+1800))
		got, ok := roundTripCell(t, want).(time.Time)
		if !ok {
			t.Fatalf("time round trip returned %T", roundTripCell(t, want))
		}
		if !got.Equal(want) {
			t.Errorf("time round trip = %v, want %v", got, want)
		}
		_, wantOffset := want.Zone()
		if _, gotOffset := got.Zone(); gotOffset != wantOffset {
			t.Errorf("time round trip offset = %d, want %d", gotOffset, wantOffset)
		}
	})

	t.Run("year out of range", func(t *testing.T) {
		encodeRejects(t, time.Date(10000, 1, 2, 3, 4, 5, 0, time.UTC), "10000")
	})
}

func TestCellCodec_RoundTripsDecimal(t *testing.T) {
	want, err := decimal.ParseExact("12.30")
	if err != nil {
		t.Fatalf("ParseExact: %v", err)
	}
	got, ok := roundTripCell(t, want).(decimal.Decimal)
	if !ok {
		t.Fatalf("decimal round trip returned %T", roundTripCell(t, want))
	}
	// String alone is not enough: ParseExact keeps trailing zeros and only
	// reports the same scale, so a 12.3 that lost its scale would still print
	// the same digits.
	if got.String() != "12.30" || got.Scale() != 2 {
		t.Errorf("decimal round trip = %s (scale %d), want 12.30 (scale 2)", got.String(), got.Scale())
	}
}

func TestCellCodec_RejectsDecimalWithNegativeScale(t *testing.T) {
	// A negative scale has no plain-decimal spelling, so String cannot render
	// it and the cell must be refused before anything tries.
	encodeRejects(t, decimal.NewFromScaledInt(big.NewInt(5), -1), "-1")
}

func TestCellCodec_RoundTripsNestedJSON(t *testing.T) {
	// The shape insyra.ReadJSON leaves a nested JSON document in.
	for _, want := range []any{
		map[string]any{
			"a": json.Number("1"),
			"b": []any{"x", true, nil, map[string]any{"c": json.Number("2.5")}},
		},
		[]any{json.Number("3"), "y"},
	} {
		got := roundTripCell(t, want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%#v round trip = %#v, want the same value", want, got)
		}
	}
}

func TestCellCodec_RejectsLeafItCannotRestore(t *testing.T) {
	encodeRejects(t, map[string]any{"a": 1.5}, "float64")
	encodeRejects(t, map[string]any{"a": []any{1}}, "int")
}

func TestCellCodec_RejectsUnsupportedTypes(t *testing.T) {
	// A named type like time.Duration is now supported. Structs and
	// maps with non-string keys remain unsupported.
	for _, v := range []any{struct{ X int }{1}, map[int]any{}} {
		encodeRejects(t, v, fmt.Sprintf("%T", v))
	}
}

func TestCellCodec_NilCellHasNoTag(t *testing.T) {
	tag, value, err := encodeCell(nil)
	if err != nil {
		t.Fatalf("encodeCell(nil) returned %v", err)
	}
	if tag != "" || value != nil {
		t.Errorf("encodeCell(nil) = (%q, %#v), want (\"\", nil)", tag, value)
	}
	// A stored null is a nil cell whatever the file claims it was.
	got, err := decodeCell("int64", nil)
	if err != nil || got != nil {
		t.Errorf("decodeCell(\"int64\", nil) = (%#v, %v), want (nil, nil)", got, err)
	}
}

func TestCellCodec_DecodeRejectsUnreadableCells(t *testing.T) {
	cases := []struct {
		name  string
		tag   string
		value any
	}{
		{"int tag with a string", "int64", "12"},
		{"int8 out of range", "int8", json.Number("300")},
		{"unknown float name", "float64", "nan"},
		{"unknown tag", "bogus", "x"},
		{"no tag", "", "x"},
		{"string tag with a number", "string", json.Number("1")},
		{"bytes that are not base64", "[]uint8", "not base64!"},
		{"time that is not RFC 3339", "time.Time", "yesterday"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, err := decodeCell(c.tag, c.value); err == nil {
				t.Errorf("decodeCell(%q, %#v) = %#v, want an error", c.tag, c.value, got)
			}
		})
	}
}

// A value that does not match its tag must name both, so the reader can tell
// which part of the stored cell is wrong.
func TestCellCodec_DecodeErrorNamesTheTagAndTheValue(t *testing.T) {
	_, err := decodeCell("int64", "12")
	if err == nil {
		t.Fatal("decodeCell(\"int64\", \"12\") returned no error")
	}
	if message := err.Error(); !strings.Contains(message, "int64") || !strings.Contains(message, "string") {
		t.Errorf("decodeCell(\"int64\", \"12\") error = %q, want it to name the tag and the value's type", message)
	}
}

// roundTripColumn encodes a column, writes the encoded column as JSON, reads it
// back through a decoder that keeps numbers as json.Number, and decodes the
// column again — the path a stored column takes between two sessions. It hands
// back the JSON as well, so a test can check the shape the file has and not
// only the cells it restores.
func roundTripColumn(t *testing.T, name string, cells []any) ([]byte, string, []any) {
	t.Helper()
	column, err := encodeColumn(name, cells)
	if err != nil {
		t.Fatalf("encodeColumn(%q, %#v) returned %v", name, cells, err)
	}
	b, err := json.Marshal(column)
	if err != nil {
		t.Fatalf("json.Marshal of the encoded column: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var read any
	if err := dec.Decode(&read); err != nil {
		t.Fatalf("decoding %s: %v", b, err)
	}
	gotName, gotCells, err := decodeColumn(read)
	if err != nil {
		t.Fatalf("decodeColumn(%#v) returned %v", read, err)
	}
	return b, gotName, gotCells
}

func TestColumnCodec_HomogeneousColumnStoresTheTypeOnce(t *testing.T) {
	// One type for the whole column, not one per cell, and a nil cell changes
	// nothing: it has no type of its own to disagree with.
	want := []any{1.5, nil, math.NaN(), 3.0}
	raw, name, got := roundTripColumn(t, "zeta", want)
	if !strings.Contains(string(raw), `"type":"float64"`) {
		t.Errorf("stored column = %s, want it to carry \"type\":\"float64\"", raw)
	}
	if strings.Contains(string(raw), `"types"`) {
		t.Errorf("stored column = %s, want no \"types\" for a column whose cells all share a type", raw)
	}
	if name != "zeta" {
		t.Errorf("decoded name = %q, want %q", name, "zeta")
	}
	if len(got) != len(want) {
		t.Fatalf("decoded %d cells, want %d", len(got), len(want))
	}
	// The type has to come back with the value: a 3.0 stored as a 3 is an
	// integer to anything that reads it.
	for _, i := range []int{0, 3} {
		f, isFloat := got[i].(float64)
		if !isFloat || f != want[i] {
			t.Errorf("decoded cells[%d] = %#v (%T), want the float64 %v", i, got[i], got[i], want[i])
		}
	}
	if got[1] != nil {
		t.Errorf("decoded cells[1] = %#v, want nil", got[1])
	}
	if f, isFloat := got[2].(float64); !isFloat || !math.IsNaN(f) {
		t.Errorf("decoded cells[2] = %#v (%T), want a float64 NaN", got[2], got[2])
	}
}

func TestColumnCodec_MixedColumnStoresATagPerCell(t *testing.T) {
	// Cells that disagree on type cannot share a column-wide tag, and the
	// nil cell's tag is the empty string rather than a guess.
	raw, _, got := roundTripColumn(t, "", []any{int64(1), "a", nil, true})
	if !strings.Contains(string(raw), `"types":["int64","string","","bool"]`) {
		t.Errorf("stored column = %s, want one tag per cell", raw)
	}
	if strings.Contains(string(raw), `"type":`) {
		t.Errorf("stored column = %s, want no column-wide \"type\" for a mixed column", raw)
	}
	want := []any{int64(1), "a", nil, true}
	if len(got) != len(want) {
		t.Fatalf("decoded %d cells, want %d", len(got), len(want))
	}
	for i := range want {
		if fmt.Sprintf("%T", got[i]) != fmt.Sprintf("%T", want[i]) || got[i] != want[i] {
			t.Errorf("decoded cells[%d] = %#v (%T), want %#v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}

func TestColumnCodec_AllNilColumnStoresNoType(t *testing.T) {
	// A column with nothing in it has no type to claim, and saying one would
	// have it change if a cell were ever added.
	raw, _, got := roundTripColumn(t, "nothing", []any{nil, nil})
	if strings.Contains(string(raw), `"type"`) {
		t.Errorf("stored column = %s, want neither \"type\" nor \"types\"", raw)
	}
	if len(got) != 2 {
		t.Fatalf("decoded %d cells, want 2", len(got))
	}
	for i, cell := range got {
		if cell != nil {
			t.Errorf("decoded cells[%d] = %#v, want nil", i, cell)
		}
	}
}

func TestColumnCodec_EmptyColumnStoresAnEmptyArray(t *testing.T) {
	// nil and [] write as the same "null", so an empty column that wrote nil
	// would come back with nothing to tell it from a column that never existed.
	raw, _, got := roundTripColumn(t, "none", []any{})
	if !strings.Contains(string(raw), `"values":[]`) {
		t.Errorf("stored column = %s, want \"values\":[]", raw)
	}
	if len(got) != 0 {
		t.Errorf("decoded %d cells, want 0", len(got))
	}
	if got == nil {
		t.Error("decoded cells = nil, want an empty non-nil slice")
	}
}

func TestColumnCodec_EmptyNameIsNotStored(t *testing.T) {
	raw, name, got := roundTripColumn(t, "", []any{"x"})
	if strings.Contains(string(raw), `"name"`) {
		t.Errorf("stored column = %s, want no \"name\" for an unnamed column", raw)
	}
	if name != "" {
		t.Errorf("decoded name = %q, want %q", name, "")
	}
	if len(got) != 1 || got[0] != "x" {
		t.Errorf("decoded cells = %#v, want [\"x\"]", got)
	}
}

// A cell the environment cannot store must be reported by its row, because
// that is the row the user has to look at.
func TestColumnCodec_EncodeErrorNamesTheRow(t *testing.T) {
	_, err := encodeColumn("mixed", []any{1, struct{ X int }{1}, "a"})
	if err == nil {
		t.Fatal("encodeColumn accepted a cell it cannot store")
	}
	if message := err.Error(); !strings.Contains(message, "row 1") || !strings.Contains(message, "struct { X int }") {
		t.Errorf("encodeColumn error = %q, want it to name row 1 and the type it refused", message)
	}
}

func TestColumnCodec_DecodeRejectsUnreadableColumns(t *testing.T) {
	cases := []struct {
		name string
		raw  any
	}{
		{"not an object", []any{}},
		{"name that is not a string", map[string]any{"name": json.Number("1"), "values": []any{}}},
		{"no values", map[string]any{"name": "zeta"}},
		{"values that are not an array", map[string]any{"values": "none"}},
		{"fewer types than values", map[string]any{"values": []any{"a", "b"}, "types": []any{"string"}}},
		{"more types than values", map[string]any{"values": []any{"a"}, "types": []any{"string", "string"}}},
		{"types that are not an array", map[string]any{"values": []any{"a"}, "types": "string"}},
		{"a type that is not a string", map[string]any{"values": []any{"a"}, "types": []any{json.Number("1")}}},
		{"a column type that is not a string", map[string]any{"type": json.Number("1"), "values": []any{"a"}}},
		{"a value with no type at all", map[string]any{"values": []any{"x"}}},
		{"a type no cell can be restored from", map[string]any{"type": "bogus", "values": []any{"x"}}},
		{"a value that does not match its type", map[string]any{"type": "int64", "values": []any{"x"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if name, cells, err := decodeColumn(c.raw); err == nil {
				t.Errorf("decodeColumn(%#v) = (%q, %#v), want an error", c.raw, name, cells)
			}
		})
	}
}

// A column-wide type that only some of the cells can be restored from is a
// mistake in the file, and the row it went wrong on is part of the message.
func TestColumnCodec_DecodeErrorNamesTheRow(t *testing.T) {
	// Row 0 restores, so the message can only come from row 1: a 1.5 stored
	// under an int64 tag is a number, but not one that tag can hold.
	raw := map[string]any{
		"values": []any{"a", json.Number("1.5")},
		"types":  []any{"string", "int64"},
	}
	_, _, err := decodeColumn(raw)
	if err == nil {
		t.Fatalf("decodeColumn(%#v) returned no error", raw)
	}
	if message := err.Error(); !strings.Contains(message, "row 1") {
		t.Errorf("decodeColumn error = %q, want it to name row 1", message)
	}
}

func TestCellCodec_RoundTripsDuration(t *testing.T) {
	for _, want := range []time.Duration{
		90 * time.Minute,
		-5,
		time.Duration(math.MinInt64),
	} {
		t.Run(fmt.Sprintf("%v", want), func(t *testing.T) {
			got := roundTripCell(t, want)
			if fmt.Sprintf("%T", got) != "time.Duration" {
				t.Errorf("duration round trip returned %T, want time.Duration", got)
				return
			}
			if got != want {
				t.Errorf("duration round trip = %v, want %v", got, want)
			}
		})
	}
}

func TestCellCodec_RoundTripsInvalidUTF8String(t *testing.T) {
	// A string that is not valid UTF-8 must survive round-trip byte-for-byte.
	want := "a\xffb"
	got := roundTripCell(t, want)
	if fmt.Sprintf("%T", got) != "string" {
		t.Errorf("invalid UTF-8 string round trip returned %T, want string", got)
	}
	if got != want {
		t.Errorf("invalid UTF-8 string round trip = %q, want %q", got, want)
	}
	// A valid UTF-8 string is stored as a JSON string, not an object.
	valid := "正常"
	tag, value, err := encodeCell(valid)
	if err != nil {
		t.Fatalf("encodeCell(%q) returned %v", valid, err)
	}
	if tag != "string" {
		t.Errorf("valid string tag = %q, want string", tag)
	}
	if _, isString := value.(string); !isString {
		t.Errorf("valid string encoded value is %T, want string", value)
	}
}

func TestCellCodec_ColumnWithMixedUTF8Strings(t *testing.T) {
	// A column containing both valid and invalid UTF-8 strings must
	// round-trip both values and keep Type as "string".
	cells := []any{"ok", "a\xffb"}
	b, name, got := roundTripColumn(t, "mixed", cells)
	if name != "mixed" {
		t.Errorf("decoded column name = %q, want %q", name, "mixed")
	}
	if len(got) != 2 {
		t.Fatalf("decoded %d cells, want 2", len(got))
	}
	for i, want := range cells {
		if got[i] != want {
			t.Errorf("decoded cells[%d] = %q, want %q", i, got[i], want)
		}
	}
	// The column should have a single "string" type.
	if !strings.Contains(string(b), `"type":"string"`) {
		t.Errorf("stored column = %s, want it to carry \"type\":\"string\"", b)
	}
}

func TestCellCodec_RejectsMapWithInvalidUTF8KeyOrValue(t *testing.T) {
	// Map value that is not valid UTF-8.
	encodeRejects(t, map[string]any{"k": "a\xffb"}, "not valid UTF-8")
	// Map key that is not valid UTF-8.
	encodeRejects(t, map[string]any{"a\xffb": "v"}, "not valid UTF-8")
}

func TestCellCodec_RoundTripsTimeWithSecondOffset(t *testing.T) {
	// A time zone whose offset has seconds (e.g., pre-1883 America/New_York
	// at -04:56:02) must be stored as UTC to preserve the instant, because
	// RFC 3339 only writes offsets to the minute.
	want := time.Date(1880, 1, 1, 12, 0, 0, 0, time.FixedZone("LMT", -(4*3600+56*60+2)))
	got, ok := roundTripCell(t, want).(time.Time)
	if !ok {
		t.Fatalf("time round trip returned %T", roundTripCell(t, want))
	}
	if !got.Equal(want) {
		t.Errorf("time round trip = %v, want %v", got, want)
	}
	if got.Location() != time.UTC {
		t.Errorf("time round trip location = %v, want UTC", got.Location())
	}
}

func TestCellCodec_RejectsSelfReferentialSlice(t *testing.T) {
	// A slice that contains itself must be rejected with a depth error,
	// not cause a stack overflow.
	s := []any{nil}
	s[0] = s
	encodeRejects(t, s, "nested more than 64 levels")
}

func TestCellCodec_NestingDepthBoundary(t *testing.T) {
	// 64 levels of nesting is allowed.
	var deep64 any = "x"
	for i := 0; i < 64; i++ {
		deep64 = []any{deep64}
	}
	_, _, err := encodeCell(deep64)
	if err != nil {
		t.Errorf("64-level nested value encodeCell returned %v", err)
	}
	// Verify it can round-trip.
	got := roundTripCell(t, deep64)
	if !reflect.DeepEqual(got, deep64) {
		t.Errorf("64-level nested value round trip = %#v, want %#v", got, deep64)
	}

	// 65 levels is rejected.
	deep65 := []any{deep64}
	encodeRejects(t, deep65, "nested more than 64 levels")
}

func TestCellCodec_DecodeRejectsInvalidDurationAndBase64(t *testing.T) {
	// duration tag with a non-json.Number value.
	if got, err := decodeCell("time.Duration", "5"); err == nil {
		t.Errorf("decodeCell(\"time.Duration\", \"5\") = %#v, want an error", got)
	}
	// string tag with a malformed base64 map.
	if got, err := decodeCell("string", map[string]any{"base64": "!!"}); err == nil {
		t.Errorf("decodeCell(\"string\", map[base64:!!]) = %#v, want an error", got)
	}
}
