package insyra

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	gsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// A cell Go cannot compare used to crash the library. convertSQLValue keeps a
// binary column as []byte on purpose, so reading a BLOB column and counting it
// took the process down, and searching for a value plainly in the list
// reported "not found".

type anyField struct{ X any }

func blobTable(t *testing.T) *DataTable {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(gsqlite.New(gsqlite.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE files (name TEXT, payload BLOB)`)
	db.Exec(`INSERT INTO files VALUES ('a', X'00FF41'), ('b', X'00FF41'), ('c', X'00FF42')`)

	dt, err := ReadSQL(db, "files")
	if err != nil {
		t.Fatal(err)
	}
	return dt
}

func TestBlobColumnCanBeCountedAndFound(t *testing.T) {
	col := blobTable(t).GetColByNumber(1)
	want := []byte{0x00, 0xff, 0x41}

	counter := col.Counter() // used to panic: hash of unhashable type []uint8
	if got := counter[ToMapKey(want)]; got != 2 {
		t.Errorf("Counter reports %d for a value held by two rows, want 2", got)
	}
	if got := col.Count(want); got != 2 {
		t.Errorf("Count reports %d for the same value, want 2 (Count and Counter must agree)", got)
	}
	if got := len(col.FindAll(want)); got != 2 {
		t.Errorf("FindAll found %d, want 2", got)
	}
}

// reflect.TypeOf(v).Comparable() says true for both of these and == still
// panics, so the obvious guard is not the right one.
func TestTypesThatLookComparableAndAreNot(t *testing.T) {
	for _, v := range []any{
		[2]any{[]int{1}, 2},
		anyField{[]int{1}},
	} {
		dl := NewDataList()
		dl.Append(v)
		dl.Append(v)
		counter := dl.Counter()
		if got := counter[ToMapKey(v)]; got != 2 {
			t.Errorf("%T: Counter reports %d, want 2", v, got)
		}
		if got := dl.Count(v); got != 2 {
			t.Errorf("%T: Count reports %d, want 2", v, got)
		}
	}
}

// The encoding is the identity, so it has to descend: %v alone makes the
// integer 1 and the string "1" look alike inside a []any.
func TestNestedValuesThatPrintAlikeAreNotTheSame(t *testing.T) {
	a, b := []any{1}, []any{"1"}
	if encodeCell(a) == encodeCell(b) {
		t.Errorf("[]any{1} and []any{\"1\"} share an identity: %q", encodeCell(a))
	}

	dl := NewDataList()
	dl.Append(a)
	dl.Append(b)
	if got := len(dl.Counter()); got != 2 {
		t.Errorf("Counter merged two distinct values into %d group(s)", got)
	}
}

func TestMapIdentityDoesNotDependOnIterationOrder(t *testing.T) {
	m := map[string]int{"z": 1, "a": 2, "m": 3, "q": 4, "b": 5}
	first := encodeCell(m)
	for i := 0; i < 50; i++ {
		if encodeCell(m) != first {
			t.Fatalf("the identity of a map changed between calls")
		}
	}
}

func TestSelfReferentialValueDoesNotExhaustTheStack(t *testing.T) {
	cyclic := []any{nil}
	cyclic[0] = cyclic
	// A fatal stack overflow cannot be recovered, so reaching the next line at
	// all is the assertion.
	_ = encodeCell(cyclic)
}

// A value that refers to itself more than once used to double its work at every
// level down to the depth limit, so counting it never finished. A reference
// back to a container still being encoded is written as a marker instead.
func TestSelfReferentialValueIsEncodedPromptly(t *testing.T) {
	s := []any{nil, nil}
	s[0] = s
	s[1] = s
	m := map[string]any{}
	m["a"] = m
	m["b"] = m

	done := make(chan map[any]int, 1)
	go func() { done <- NewDataList(Cell(s), Cell(s), Cell(m)).Counter() }()
	select {
	case counter := <-done:
		if got := counter[ToMapKey(s)]; got != 2 {
			t.Errorf("Counter[s] = %d, want 2", got)
		}
		if got := counter[ToMapKey(m)]; got != 1 {
			t.Errorf("Counter[m] = %d, want 1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Counter on a value that refers to itself twice did not finish")
	}

	// Distinct cyclic values are told apart, the same content is the same
	// value, and the answer does not change between calls.
	a := []any{nil, 1}
	a[0] = a
	b := []any{nil, 2}
	b[0] = b
	c := []any{nil, 1}
	c[0] = c
	if encodeCell(a) == encodeCell(b) {
		t.Error("cyclic values with different content encode alike")
	}
	if encodeCell(a) != encodeCell(c) {
		t.Error("cyclic values with the same content encode differently")
	}
	if first, again := encodeCell(a), encodeCell(a); first != again {
		t.Error("the encoding of a cyclic value changed between calls")
	}
	if got := NewDataList(Cell(a), Cell(b), Cell(c)).Count(a); got != 2 {
		t.Errorf("Count(a) = %d, want 2", got)
	}
}

// A value that shares a sub-value without containing itself, built as
// x = []any{x, x} over and over, has 2^depth paths through it, and encoding
// every path took 675 ms at depth 22 and never finished at depth 40. Its
// identity must still come from its content, whether or not the content is
// shared.
func TestSharedSubValueIsEncodedPromptly(t *testing.T) {
	shared := func(depth, leaf int) any {
		var x any = leaf
		for range depth {
			x = []any{x, x}
		}
		return x
	}
	var unshared func(depth int) any
	unshared = func(depth int) any {
		if depth == 0 {
			return 1
		}
		return []any{unshared(depth - 1), unshared(depth - 1)}
	}

	// Each level of []any is two levels of encoding, a slice and the interface
	// holding it, so depth 30 is the deepest whose leaves the encoder still
	// reads and depth 40 is cut short by maxCellEncodeDepth.
	for _, depth := range []int{30, 40} {
		x, y, other := shared(depth, 1), shared(depth, 1), shared(depth, 2)
		type result struct {
			count   int
			elapsed time.Duration
		}
		done := make(chan result, 1)
		go func() {
			start := time.Now()
			n := NewDataList(Cell(x), Cell(y), Cell(other)).Count(x)
			done <- result{n, time.Since(start)}
		}()
		select {
		case r := <-done:
			want := 2 // x and y, not other
			if depth > 30 {
				want = 3 // beyond the limit the leaves are not read
			}
			if r.count != want {
				t.Errorf("depth %d: Count(x) = %d, want %d", depth, r.count, want)
			}
			if r.elapsed > 100*time.Millisecond {
				t.Errorf("depth %d: Count took %v, want under 100ms", depth, r.elapsed)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("depth %d: Count on a value sharing its sub-values did not finish", depth)
		}
	}

	// Sharing does not change identity: the same content built without
	// sharing is the same value, and one different leaf is a different value.
	tree := unshared(12)
	if encodeCell(tree) != encodeCell(shared(12, 1)) {
		t.Error("the same content encodes differently with and without sharing")
	}
	if first, again := encodeCell(tree), encodeCell(tree); first != again {
		t.Error("the encoding of a large nested value changed between calls")
	}
	node := tree.([]any)
	for range 11 {
		node = node[1].([]any)
	}
	node[1] = 2
	if encodeCell(tree) == encodeCell(shared(12, 1)) {
		t.Error("values differing in one deep leaf encode alike")
	}
}

func TestComparableValuesAreStillKeyedByThemselves(t *testing.T) {
	dl := NewDataList(1, 1, "a", 2.5, true)
	counter := dl.Counter()
	if counter[1] != 2 || counter["a"] != 1 || counter[2.5] != 1 || counter[true] != 1 {
		t.Errorf("ordinary values no longer key by themselves: %v", counter)
	}
}

func TestPrintingACounterStaysReadable(t *testing.T) {
	dl := NewDataList()
	dl.Append(make([]byte, 4096))
	dl.Append([]byte{0x00, 0xff, 0x41})
	out := fmt.Sprintf("%v", dl.Counter())
	if len(out) > 200 {
		t.Errorf("a 4096-byte cell made the printed counter %d characters long", len(out))
	}
	if !strings.Contains(out, "00ff41") {
		t.Errorf("the short value is not readable in the output: %s", out)
	}
}

// A value short enough to show whole must not lose its last character or two:
// that reads as a broken rendering rather than as a truncation, and nothing
// tries to close the bracket afterwards because a string cell's own content
// can contain one.
func TestASmallCompositeIsNotCutShort(t *testing.T) {
	for _, v := range []any{
		map[string]int{"a": 1, "b": 2},
		[][]int{{1}, {2, 3}},
		[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		[]string{"a", "b"},
	} {
		got := fmt.Sprintf("%v", ToMapKey(v))
		if strings.Contains(got, "…") {
			t.Errorf("%T rendered truncated: %s", v, got)
		}
		if opens, closes := strings.Count(got, "["), strings.Count(got, "]"); opens != closes {
			t.Errorf("%T rendered with unbalanced brackets: %s", v, got)
		}
	}
	// The binary values a column actually holds show whole: a UUID and an MD5
	// are 16 bytes, a SHA-1 20, a SHA-256 32. This is what the display limit
	// is set from, so it is pinned here rather than left to the constant.
	for name, size := range map[string]int{"UUID": 16, "MD5": 16, "SHA-1": 20, "SHA-256": 32} {
		got := fmt.Sprintf("%v", ToMapKey(make([]byte, size)))
		if strings.Contains(got, "…") {
			t.Errorf("a %s (%d bytes) rendered truncated: %s", name, size, got)
		}
	}

	// Longer than that is cut, and says so.
	got := fmt.Sprintf("%v", ToMapKey(make([]byte, 64)))
	if !strings.Contains(got, "…") {
		t.Errorf("a 64-byte value was not truncated: %s", got)
	}
}

// Scalar group keys must not move: GroupBy, Pivot and Merge depend on them.
func TestScalarGroupKeysAreUnchanged(t *testing.T) {
	for _, c := range []struct {
		in   []any
		want string
	}{
		{[]any{"a"}, "s:a"},
		{[]any{int64(3)}, "i:3"},
		{[]any{2.5}, "f:2.5"},
		{[]any{true}, "b:1"},
		{[]any{nil}, "n:"},
		{[]any{"a", 1}, "s:a\x1ei:1"},
	} {
		if got := encodeGroupKey(c.in); got != c.want {
			t.Errorf("encodeGroupKey(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Every DataTable method that compares by value goes through the same
// valueMatcher seam as DataList, but that is a reason to believe
// they agree, not evidence. These pin the DataTable side: a cell Go cannot
// compare is found, counted and dropped by content, and a different value is
// not.
func TestDataTableValueMethodsMatchUncomparableCells(t *testing.T) {
	found := []byte{0x00, 0xff, 0x41}
	// A separate slice with the same content: == would be false for these even
	// if Go allowed it, so matching has to be by content.
	same := []byte{0x00, 0xff, 0x41}
	absent := []byte{0x09}

	table := func() *DataTable {
		a := NewDataList()
		a.Append(found, same, "x")
		b := NewDataList()
		b.Append("p", "q", []byte{0x01})
		return NewDataTable(a.SetName("a"), b.SetName("b"))
	}

	t.Run("Count", func(t *testing.T) {
		if got := table().Count(found); got != 2 {
			t.Errorf("Count = %d, want 2", got)
		}
		if got := table().Count(absent); got != 0 {
			t.Errorf("Count of a value that is not there = %d, want 0", got)
		}
	})

	t.Run("Counter", func(t *testing.T) {
		if got := table().Counter()[ToMapKey(found)]; got != 2 {
			t.Errorf("Counter = %d, want 2", got)
		}
	})

	t.Run("FindRowsIfContains", func(t *testing.T) {
		if got := table().FindRowsIfContains(found); len(got) != 2 || got[0] != 0 || got[1] != 1 {
			t.Errorf("got rows %v, want [0 1]", got)
		}
		if got := table().FindRowsIfContains(absent); len(got) != 0 {
			t.Errorf("a value that is not there matched rows %v", got)
		}
	})

	t.Run("FindRowsIfContainsAll", func(t *testing.T) {
		if got := table().FindRowsIfContainsAll(found, "p"); len(got) != 1 || got[0] != 0 {
			t.Errorf("got rows %v, want [0]", got)
		}
		if got := table().FindRowsIfContainsAll(found, absent); len(got) != 0 {
			t.Errorf("a row matched although one value is not there: %v", got)
		}
	})

	t.Run("FindColsIfContains", func(t *testing.T) {
		if got := table().FindColsIfContains(found); len(got) != 1 || got[0] != "A" {
			t.Errorf("got cols %v, want [A]", got)
		}
		if got := table().FindColsIfContains(absent); len(got) != 0 {
			t.Errorf("a value that is not there matched cols %v", got)
		}
	})

	t.Run("FindColsIfContainsAll", func(t *testing.T) {
		if got := table().FindColsIfContainsAll(found, "x"); len(got) != 1 || got[0] != "A" {
			t.Errorf("got cols %v, want [A]", got)
		}
		if got := table().FindColsIfContainsAll(found, "p"); len(got) != 0 {
			t.Errorf("a column matched although the two values are in different columns: %v", got)
		}
	})

	t.Run("DropColsContain", func(t *testing.T) {
		dt := table()
		dt.DropColsContain(found)
		if _, cols := dt.Size(); cols != 1 {
			t.Errorf("%d columns left, want 1", cols)
		}
		dt = table()
		dt.DropColsContain(absent)
		if _, cols := dt.Size(); cols != 2 {
			t.Errorf("a value that is not there dropped a column: %d left, want 2", cols)
		}
	})

	t.Run("DropRowsContain", func(t *testing.T) {
		dt := table()
		dt.DropRowsContain(found)
		if rows, _ := dt.Size(); rows != 1 {
			t.Errorf("%d rows left, want 1", rows)
		}
		dt = table()
		dt.DropRowsContain(absent)
		if rows, _ := dt.Size(); rows != 3 {
			t.Errorf("a value that is not there dropped a row: %d left, want 3", rows)
		}
	})

	t.Run("UpdateElement then find", func(t *testing.T) {
		dt := table()
		dt.UpdateElement(2, "A", absent)
		if got := dt.FindRowsIfContains(absent); len(got) != 1 || got[0] != 2 {
			t.Errorf("a value written into a cell was not found again: %v", got)
		}
	})
}

// IsEqualTo and IsTheSameAs compare cell by cell, type included. A cell Go
// cannot compare used to count as unequal even against an identical copy; it
// is now compared by its type and content, like every lookup. A scalar NaN
// cell stays unequal to another NaN, as it always has been here.
func TestIsEqualToComparesUncomparableCellsByContent(t *testing.T) {
	build := func(vals ...any) *DataList {
		dl := NewDataList()
		dl.Append(vals...)
		return dl
	}

	a := build([]byte{0x00, 0xff}, anyField{[]int{1}})
	if !a.IsEqualTo(build([]byte{0x00, 0xff}, anyField{[]int{1}})) {
		t.Error("two lists holding the same uncomparable values are not equal")
	}
	if a.IsEqualTo(build([]byte{0x00, 0xfe}, anyField{[]int{1}})) {
		t.Error("lists holding different bytes are equal")
	}
	if a.IsEqualTo(build([]int{0, 255}, anyField{[]int{1}})) {
		t.Error("a []byte and a []int with the same numbers are equal; the type is part of the identity")
	}

	if NewDataList(math.NaN()).IsEqualTo(NewDataList(math.NaN())) {
		t.Error("a scalar NaN cell is now equal to another NaN; that stays unequal on this line")
	}

	// Inside a value Go cannot compare, a NaN is part of the encoded content,
	// so two such containers holding NaN are the same value. Lookups agree.
	x := build([]any{math.NaN()})
	if !x.IsEqualTo(build([]any{math.NaN()})) {
		t.Error("two []any{NaN} cells are not equal")
	}
	if got := x.Count([]any{math.NaN()}); got != 1 {
		t.Errorf("Count([]any{NaN}) = %d, want 1", got)
	}
}

// A value that knows how to write itself should print that way in a counter,
// not as its internal fields. decimal.Decimal — what finance.ScheduleTable
// puts in cells and what a Parquet Decimal128 column reads as — used to print
// as big.Int's sign and words.

// zzStringerSlice is uncomparable, because it holds a slice, and knows its own
// text. That is the shape of a decimal: a big.Int inside, a String outside.
type zzStringerSlice struct{ v []int }

func (s zzStringerSlice) String() string { return fmt.Sprintf("%d values", len(s.v)) }

func TestAStandInPrintsTheWayTheValueWritesItself(t *testing.T) {
	got := fmt.Sprintf("%v", ToMapKey(zzStringerSlice{[]int{1, 2}}))
	if !strings.Contains(got, "2 values") {
		t.Errorf("the stand-in did not print the value's own text: %s", got)
	}
	if strings.Contains(got, "i:1") {
		t.Errorf("the stand-in printed the encoding instead: %s", got)
	}
}

func TestAValueWithNoTextPrintsItsEncoding(t *testing.T) {
	for _, v := range []any{[]byte{0x00, 0xff}, []int{1, 2}, map[string]int{"a": 1}} {
		got := fmt.Sprintf("%v", ToMapKey(v))
		if !strings.ContainsAny(got, ":[{") && !strings.Contains(got, "00ff") {
			t.Errorf("%T no longer shows its encoding: %s", v, got)
		}
	}
}

// The text is for reading. Identity still comes from the encoding, so two
// values whose text matches but whose contents differ stay apart.
func TestTextDoesNotDecideIdentity(t *testing.T) {
	a := zzStringerSlice{[]int{1, 2}}
	b := zzStringerSlice{[]int{3, 4}} // same String(), different content
	if a.String() != b.String() {
		t.Fatal("the fixture no longer has two values with the same text")
	}
	dl := NewDataList(Cell(a), Cell(b))
	if got := len(dl.Counter()); got != 2 {
		t.Errorf("two values with the same text were merged into %d group(s)", got)
	}
}

// A String that runs long is cut like any other content.
func TestALongTextIsTruncated(t *testing.T) {
	got := fmt.Sprintf("%v", ToMapKey(zzLongStringer{}))
	if !strings.Contains(got, "…") {
		t.Errorf("a long text was not truncated: %s", got)
	}
}

type zzLongStringer struct{ v []int }

// The slice field is what makes it uncomparable, and it is read so the
// field is not dead.
func (l zzLongStringer) String() string { return strings.Repeat("x", 200-len(l.v)) }
