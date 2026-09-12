package insyra

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

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
