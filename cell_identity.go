package insyra

import (
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// maxCellEncodeDepth bounds how far encodeCell descends.
//
// A value that contains itself would otherwise recurse forever, and running
// out of stack in Go is a fatal error that recover cannot catch — the library
// would terminate, which it promises never to do. Beyond the limit a value is
// identified by its address, which is what == means for a reference anyway.
const maxCellEncodeDepth = 64

// uncomparableDisplayBytes is how much of a value's content String renders
// before cutting it off. A counter holding one large blob has to stay
// readable when the whole map is printed.
//
// 64 is the width at which the binary values a column actually holds render
// whole: a UUID and an MD5 are 32 hex characters, a SHA-1 is 40 and a SHA-256
// is 64. A SHA-512 and a real blob are longer than any limit worth setting,
// so they are cut and the ellipsis says so.
//
// It also clears every small composite, which has to show whole: losing its
// last character or two reads as a broken rendering rather than as a
// truncation, and a two-entry map encodes to 17 characters, a ten-element
// []int to 42. Nothing tries to close an unterminated bracket, because a
// string cell's own content can contain one and the closer would be a guess.
const uncomparableDisplayBytes = 64

// UncomparableKey stands in for a cell value that cannot serve as a map key:
// a slice, a map or anything containing one, which Go refuses to compare, and
// a NaN or anything containing one, which compares unequal to itself and so
// would make an entry nobody could look up. It is what ToMapKey returns for
// such a value, and what appears as the key in a Counter result.
//
// Two of them are equal exactly when the values they stand for count, match
// and group as the same value. The content they carry is deliberately not
// exported: it is the encoder's format, shared with grouping, and it will
// change. Build one with ToMapKey rather than by hand.
type UncomparableKey struct {
	// Type is the Go type of the value, as %T would write it.
	Type string

	content string

	// text is what the value's own String returned, when it has one. It is
	// for reading only: identity comes from content, because a String may be
	// lossy and two distinct values whose text matched would otherwise merge
	// into one count.
	//
	// It takes part in == like any field, which is safe because it is a
	// function of the same value content came from: equal values produce
	// equal text, so no new key appears, and two different values that
	// happened to share an encoding are told apart rather than merged. The
	// one thing that would break it is a String that returns different text
	// for the same value, which would count that value twice. Nothing can
	// detect that, so it is written down rather than guarded.
	text string
}

// String renders the type with a shortened form of the content, so printing a
// whole Counter stays readable even when a cell holds a large value.
func (k UncomparableKey) String() string {
	// What the value writes for itself, when it wrote anything: a decimal
	// reads as -340.0221114815 rather than as big.Int's sign and words.
	if k.text != "" {
		return k.Type + "(" + truncateForDisplay(k.text) + ")"
	}
	// A byte sequence drops its marker: hex describes itself, and a blob is
	// the common case here. Composite values keep theirs, because two groups
	// that differ only by the type of a nested value have to look different.
	c := strings.TrimPrefix(k.content, "x:")
	return k.Type + "(" + truncateForDisplay(c) + ")"
}

func truncateForDisplay(s string) string {
	if len(s) > uncomparableDisplayBytes {
		return s[:uncomparableDisplayBytes] + "…"
	}
	return s
}

// ToMapKey converts v into something usable as a map key: v itself when Go
// can compare it, and an UncomparableKey when it cannot.
//
// It exists because a cell can hold a slice — a []byte read from a SQL BLOB
// column, say — and indexing any map with one panics. That is the caller's
// own map operation, which no library can guard, so build the key through
// ToMapKey for a tally, a set, an index or a dedup over cell values, and to read
// a Counter result.
//
// Integer widths are not merged, because a map keyed by Go values keeps
// int(1) and int64(1) apart and this changes nothing about that. A CSV load
// stores integers as int64, so ToMapKey(1) finds nothing in a map built from
// loaded data. To ask how often one value appears, prefer Count(v), which
// matches integers by value.
func ToMapKey(v any) any {
	if comparableCell(v) {
		return v
	}
	typ, content := splitCellEncoding(v)
	key := UncomparableKey{Type: typ, content: content}
	if s, ok := v.(fmt.Stringer); ok {
		key.text = s.String()
	}
	return key
}

// comparableCell reports whether v can serve as a map key — which asks not
// only whether == is legal but whether it is useful.
//
// Two things make it false. A value Go refuses to compare panics on ==, and
// reflect.TypeOf(v).Comparable() is not the test for that: an array of
// interfaces and a struct with an interface field both report true and both
// panic when what they hold is a slice, so it has to be
// reflect.Value.Comparable, which inspects the value.
//
// A value that does not equal itself is the other. A NaN is comparable by the
// type system and NaN != NaN, so using one as a map key makes an entry nobody
// can look up: three NaNs in a column became three counts of one. Self-
// equality catches the nested shapes too — an array or a struct holding a NaN
// compares element by element and so is unequal to itself as well.
func comparableCell(v any) bool {
	if v == nil {
		return true
	}
	if !reflect.ValueOf(v).Comparable() {
		return false
	}
	// Safe now: == cannot panic on a value reflect says is comparable.
	return v == v
}

// splitCellEncoding returns v's type name and the encoding of its content.
func splitCellEncoding(v any) (typ, content string) {
	typ = fmt.Sprintf("%T", v)
	var b strings.Builder
	writeCellValue(&b, reflect.ValueOf(v), 0)
	return typ, b.String()
}

// encodeCell returns the canonical identity of a cell value: two values encode
// alike exactly when they should count, match and group as the same value.
//
// Scalars keep the encoding grouping has always used, so ordinary group keys
// are unchanged byte for byte. Composite values descend, which the previous
// %v fallback did not: it wrote []any{1} and []any{"1"} identically, so an
// integer and a string inside a slice were treated as one value.
func encodeCell(v any) string {
	if v == nil {
		return "n:"
	}
	switch tv := v.(type) {
	case string:
		return "s:" + tv
	case bool:
		if tv {
			return "b:1"
		}
		return "b:0"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("i:%v", v)
	case float32, float64:
		return encodeCellFloat(toFloatForKey(v))
	}
	typ, content := splitCellEncoding(v)
	return "o:" + typ + ":" + content
}

func toFloatForKey(v any) float64 {
	switch f := v.(type) {
	case float32:
		return float64(f)
	case float64:
		return f
	}
	return math.NaN()
}

func encodeCellFloat(f float64) string {
	if math.IsNaN(f) {
		return "f:NaN"
	}
	return fmt.Sprintf("f:%v", f)
}

// writeCellValue encodes rv through reflection, so it can descend into struct
// fields that are not exported. It must not call Interface() for that reason.
func writeCellValue(b *strings.Builder, rv reflect.Value, depth int) {
	if !rv.IsValid() {
		b.WriteString("n:")
		return
	}
	if depth > maxCellEncodeDepth {
		writeCellAddress(b, rv)
		return
	}

	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			b.WriteString("n:")
			return
		}
		if rv.Kind() == reflect.Pointer {
			// A pointer is identified the way == identifies it.
			writeCellAddress(b, rv)
			return
		}
		writeCellValue(b, rv.Elem(), depth+1)

	case reflect.String:
		b.WriteString("s:")
		b.WriteString(rv.String())

	case reflect.Bool:
		if rv.Bool() {
			b.WriteString("b:1")
		} else {
			b.WriteString("b:0")
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString("i:")
		b.WriteString(strconv.FormatInt(rv.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		b.WriteString("i:")
		b.WriteString(strconv.FormatUint(rv.Uint(), 10))

	case reflect.Float32, reflect.Float64:
		b.WriteString(encodeCellFloat(rv.Float()))

	case reflect.Complex64, reflect.Complex128:
		fmt.Fprintf(b, "c:%v", rv.Complex())

	case reflect.Slice, reflect.Array:
		// A byte sequence is written as hex: it is the compact form, it is what
		// a reader of a printed counter can make sense of, and it keeps the key
		// of a large blob to twice its length rather than five times.
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			b.WriteString("x:")
			b.WriteString(hex.EncodeToString(cellBytes(rv)))
			return
		}
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			b.WriteString("n:")
			return
		}
		b.WriteByte('[')
		for i := 0; i < rv.Len(); i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCellValue(b, rv.Index(i), depth+1)
		}
		b.WriteByte(']')

	case reflect.Map:
		if rv.IsNil() {
			b.WriteString("n:")
			return
		}
		// Sorted by encoded key, so the identity does not depend on the order
		// Go happens to iterate the map in.
		entries := make([]string, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			var kb, vb strings.Builder
			writeCellValue(&kb, iter.Key(), depth+1)
			writeCellValue(&vb, iter.Value(), depth+1)
			entries = append(entries, kb.String()+"="+vb.String())
		}
		sort.Strings(entries)
		b.WriteByte('{')
		b.WriteString(strings.Join(entries, ","))
		b.WriteByte('}')

	case reflect.Struct:
		b.WriteByte('{')
		for i := 0; i < rv.NumField(); i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCellValue(b, rv.Field(i), depth+1)
		}
		b.WriteByte('}')

	default:
		// Channels and funcs are identified by address, as == identifies them.
		writeCellAddress(b, rv)
	}
}

// cellBytes reads a byte slice or byte array without requiring the value to be
// exported or addressable.
func cellBytes(rv reflect.Value) []byte {
	if rv.Kind() == reflect.Slice {
		out := make([]byte, rv.Len())
		reflect.Copy(reflect.ValueOf(out), rv)
		return out
	}
	out := make([]byte, rv.Len())
	for i := range out {
		out[i] = byte(rv.Index(i).Uint())
	}
	return out
}

func writeCellAddress(b *strings.Builder, rv reflect.Value) {
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		fmt.Fprintf(b, "p:%s:%#x", rv.Type(), rv.Pointer())
	default:
		fmt.Fprintf(b, "p:%s", rv.Type())
	}
}
