package insyra

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// maxCellEncodeDepth bounds how far encodeCell descends.
//
// A value nested too deeply would otherwise exhaust the stack, and running out
// of stack in Go is a fatal error that recover cannot catch: the whole program
// would end. The limit keeps encoding from overflowing the stack. Beyond it a
// value is identified by its address, which is what == means for a reference
// anyway.
// A value that contains itself does not reach the limit: a reference back to a
// slice or map still being encoded is written as a back-reference (cellRef).
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

// UncomparableKey stands in for a cell value that Go cannot use as a map key —
// a slice, a map, or anything containing one. It is what ToMapKey returns
// for such a value, and what appears as the key in a Counter result.
//
// Two of them are equal exactly when the values they stand for count and match
// as the same value. The content they carry is deliberately not exported: it
// is the encoder's format, and it will change. Build one with ToMapKey rather
// than by hand.
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

// comparableCell reports whether == on v is safe.
//
// reflect.TypeOf(v).Comparable() is not this test: an array of interfaces and
// a struct with an interface field both report true, and both panic on ==
// when what they hold is a slice. reflect.Value.Comparable inspects the value.
func comparableCell(v any) bool {
	if v == nil {
		return true
	}
	return reflect.ValueOf(v).Comparable()
}

// splitCellEncoding returns v's type name and the encoding of its content.
func splitCellEncoding(v any) (typ, content string) {
	typ = fmt.Sprintf("%T", v)
	var b strings.Builder
	writeCellValue(&b, reflect.ValueOf(v), 0, nil, nil)
	return typ, b.String()
}

// encodeCell returns the canonical identity of a cell value: two values encode
// alike exactly when they should count and match as the same value.
//
// Scalars use the same encoding as encodeGroupKey. Composite values descend,
// which %v does not: it writes []any{1} and []any{"1"} identically. Grouping
// (encodeGroupKey, uniqueKey) still uses its own %T:%v fallback on this line,
// because moving it here would change GroupBy, Pivot and Merge keys.
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

// cellRef names a slice or map being encoded: the same type, backing pointer
// and length is the same value.
type cellRef struct {
	typ    reflect.Type
	ptr    uintptr
	length int
}

// cellMemoKey names a slice or map written at a depth. The depth is part of it
// because maxCellEncodeDepth can cut a value short at one depth and not at
// another.
type cellMemoKey struct {
	ref   cellRef
	depth int
}

// noCellRef is what writeCellValue returns when what it wrote refers back to
// no slice or map enclosing it.
const noCellRef = math.MaxInt

// maxInlineCellBytes is the longest encoding a nested slice or map that holds
// another slice or map is written as. A longer one is written as the SHA-256
// digest of that encoding instead.
//
// A value that shares a sub-value, x = []any{x, x} repeated, has an encoding
// that doubles at every level: a terabyte by depth 40. Memoizing a shared
// sub-value saves the work of walking it again but not the length of writing
// it again, and the digest bounds that length. It is a function of the content
// alone, so equal content still encodes alike whether it is shared or copied.
// The outermost value is never digested, so a printed key still shows its
// content, and neither is a container that holds no other, whose encoding
// cannot double.
const maxInlineCellBytes = 1024

// writeContainer writes the non-empty slice or map rv and returns the lowest
// path index a back-reference in it points to.
//
// A container already on path is written as its distance up the path. Without
// that, a value that refers to itself twice, s[0] = s; s[1] = s, doubles the
// work at every level down to maxCellEncodeDepth and never finishes; with it,
// the encoding stays structural, so two cyclic values of the same shape and
// content encode alike and different content still encodes differently.
//
// A container this call has already written at the same depth is written from
// memo. One that refers back to a container enclosing it is not memoized,
// because the distance would be wrong anywhere else.
func writeContainer(b *strings.Builder, rv reflect.Value, depth int, path []cellRef, memo map[cellMemoKey]string) int {
	ref := cellRef{typ: rv.Type(), ptr: rv.Pointer(), length: rv.Len()}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == ref {
			b.WriteString("r:" + strconv.Itoa(len(path)-i))
			return i
		}
	}
	writeElems := writeSliceElems
	if rv.Kind() == reflect.Map {
		writeElems = writeMapEntries
	}
	own := len(path)
	path = append(path, ref)
	if depth == 0 || !holdsContainer(rv) {
		// The outermost value is met again only through a cycle, which the
		// path catches, and a container holding no other shares nothing and
		// refers to nothing: neither is worth memoizing.
		writeElems(b, rv, depth, path, memo)
		return noCellRef
	}
	key := cellMemoKey{ref: ref, depth: depth}
	if memo == nil {
		// Made by the first container that needs one and shared by everything
		// inside it, so a value holding no nested container allocates none.
		memo = map[cellMemoKey]string{}
	} else if text, ok := memo[key]; ok {
		b.WriteString(text)
		return noCellRef
	}
	var inner strings.Builder
	lowest := writeElems(&inner, rv, depth, path, memo)
	text := inner.String()
	if len(text) > maxInlineCellBytes {
		sum := sha256.Sum256([]byte(text))
		text = "h:" + hex.EncodeToString(sum[:])
	}
	b.WriteString(text)
	if lowest < own {
		return lowest
	}
	memo[key] = text
	return noCellRef
}

// holdsContainer reports whether an element of the slice or map rv may be, or
// may hold, a slice or map. It looks through interfaces and counts any array
// or struct.
func holdsContainer(rv reflect.Value) bool {
	switch rv.Type().Elem().Kind() {
	case reflect.Interface:
	case reflect.Slice, reflect.Map, reflect.Array, reflect.Struct:
		return true
	default:
		return false
	}
	isContainer := func(v reflect.Value) bool {
		for v.Kind() == reflect.Interface && !v.IsNil() {
			v = v.Elem()
		}
		switch v.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array, reflect.Struct:
			return true
		}
		return false
	}
	if rv.Kind() == reflect.Map {
		iter := rv.MapRange()
		for iter.Next() {
			if isContainer(iter.Value()) {
				return true
			}
		}
		return false
	}
	for i := 0; i < rv.Len(); i++ {
		if isContainer(rv.Index(i)) {
			return true
		}
	}
	return false
}

// writeSliceElems writes the elements of the slice or array rv.
func writeSliceElems(b *strings.Builder, rv reflect.Value, depth int, path []cellRef, memo map[cellMemoKey]string) int {
	lowest := noCellRef
	b.WriteByte('[')
	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		lowest = min(lowest, writeCellValue(b, rv.Index(i), depth+1, path, memo))
	}
	b.WriteByte(']')
	return lowest
}

// writeMapEntries writes the entries of the map rv, sorted by their encoding,
// so the identity does not depend on the order Go iterates the map in.
func writeMapEntries(b *strings.Builder, rv reflect.Value, depth int, path []cellRef, memo map[cellMemoKey]string) int {
	type entry struct {
		key string
		val reflect.Value
	}
	// A key holds no slice or map, so its encoding does not depend on what was
	// written before it. Writing the values in key order keeps what memo holds,
	// and so the result, from depending on the iteration order as well.
	entries := make([]entry, 0, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		var kb strings.Builder
		writeCellValue(&kb, iter.Key(), depth+1, path, memo)
		entries = append(entries, entry{key: kb.String(), val: iter.Value()})
	}
	slices.SortFunc(entries, func(x, y entry) int { return strings.Compare(x.key, y.key) })
	lowest := noCellRef
	texts := make([]string, len(entries))
	for i, e := range entries {
		var eb strings.Builder
		eb.WriteString(e.key)
		eb.WriteByte('=')
		lowest = min(lowest, writeCellValue(&eb, e.val, depth+1, path, memo))
		texts[i] = eb.String()
	}
	// Sorted again by the whole entry, so two keys that encode alike still give
	// one order.
	sort.Strings(texts)
	b.WriteByte('{')
	for i, text := range texts {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(text)
	}
	b.WriteByte('}')
	return lowest
}

// writeCellValue encodes rv through reflection, so it can descend into struct
// fields that are not exported. It must not call Interface() for that reason.
// path holds the slices and maps enclosing rv, and memo the slices and maps
// this call has already written. It returns the lowest path index a
// back-reference in what it wrote points to, or noCellRef.
func writeCellValue(b *strings.Builder, rv reflect.Value, depth int, path []cellRef, memo map[cellMemoKey]string) int {
	if !rv.IsValid() {
		b.WriteString("n:")
		return noCellRef
	}
	if depth > maxCellEncodeDepth {
		writeCellAddress(b, rv)
		return noCellRef
	}

	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			b.WriteString("n:")
			return noCellRef
		}
		if rv.Kind() == reflect.Pointer {
			// A pointer is identified the way == identifies it.
			writeCellAddress(b, rv)
			return noCellRef
		}
		return writeCellValue(b, rv.Elem(), depth+1, path, memo)

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
			return noCellRef
		}
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			b.WriteString("n:")
			return noCellRef
		}
		if rv.Kind() == reflect.Slice && rv.Len() > 0 {
			return writeContainer(b, rv, depth, path, memo)
		}
		return writeSliceElems(b, rv, depth, path, memo)

	case reflect.Map:
		if rv.IsNil() {
			b.WriteString("n:")
			return noCellRef
		}
		if rv.Len() > 0 {
			return writeContainer(b, rv, depth, path, memo)
		}
		b.WriteString("{}")

	case reflect.Struct:
		lowest := noCellRef
		b.WriteByte('{')
		for i := 0; i < rv.NumField(); i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			lowest = min(lowest, writeCellValue(b, rv.Field(i), depth+1, path, memo))
		}
		b.WriteByte('}')
		return lowest

	default:
		// Channels and funcs are identified by address, as == identifies them.
		writeCellAddress(b, rv)
	}
	return noCellRef
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
