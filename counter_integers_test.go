package insyra

import (
	"math"
	"testing"
)

// Counter keys every integer by its value, as int when int can hold it, so
// the literal a Go caller writes finds it whatever width the cell was stored
// at. Owner's ruling of 2026-09-27 on the AGENTS.md follow-up of 2026-09-11.
func TestCounterMergesIntegerWidthsUnderInt(t *testing.T) {
	dl := NewDataList()
	for _, v := range []any{int64(5), 5, uint8(5), int32(5), "5", 5.0, int64(3)} {
		dl.Append(v)
	}
	counter := dl.Counter()

	if got := counter[5]; got != 4 {
		t.Errorf("counter[5] = %d, want 4 (int64, int, uint8 and int32 fives)", got)
	}
	if got := dl.Count(5); counter[5] != got {
		t.Errorf("counter[5] = %d disagrees with Count(5) = %d", counter[5], got)
	}
	if counter["5"] != 1 || counter[5.0] != 1 || counter[3] != 1 {
		t.Errorf("text and float must stay apart from the integer: %v", counter)
	}
	if len(counter) != 4 {
		t.Errorf("got %d keys, want 4: %v", len(counter), counter)
	}
	for k := range counter {
		switch k.(type) {
		case int, string, float64:
		default:
			t.Errorf("key %v is a %T; an integer inside int's range must be keyed as int", k, k)
		}
	}
}

// A CSV load stores integers as int64, which is where the old keying hurt:
// counter[5] found nothing on a column holding two fives.
func TestCounterOnLoadedDataFindsTheIntLiteral(t *testing.T) {
	dt, err := ReadCSVString("qty\n5\n5\n3\n")
	if err != nil {
		t.Fatal(err)
	}
	col := dt.GetColByNumber(0)
	if _, ok := col.Get(0).(int64); !ok {
		t.Fatalf("fixture: expected the CSV reader to store int64, got %T", col.Get(0))
	}
	counter := col.Counter()
	if got := counter[5]; got != 2 {
		t.Errorf("counter[5] = %d on a loaded column with two fives, want 2", got)
	}
	if got := counter[ToMapKey(int64(5))]; got != 2 {
		t.Errorf("counter[ToMapKey(int64(5))] = %d, want 2", got)
	}
}

func TestTableCounterMergesIntegerWidthsAcrossColumns(t *testing.T) {
	dt := NewDataTable(NewDataList(int64(7)), NewDataList(7))
	if got := dt.Counter()[7]; got != 2 {
		t.Errorf("dt.Counter()[7] = %d, want 2 (int64 in one column, int in the other)", got)
	}
}

// Outside int's range there is no int to key by, so the value keeps a type
// that holds it: uint64 for an unsigned one. Two unsigned widths holding the
// same value still merge, because Count says they are the same value.
func TestCounterKeysIntegersBeyondIntByValue(t *testing.T) {
	big := uint64(math.MaxUint64)
	dl := NewDataList()
	dl.Append(big)
	dl.Append(uint(math.MaxUint))
	dl.Append(-1)
	counter := dl.Counter()

	if math.MaxUint == math.MaxUint64 {
		if got := counter[big]; got != 2 {
			t.Errorf("counter[uint64 max] = %d, want 2 (uint64 and uint holding the same value)", got)
		}
	}
	if counter[-1] != 1 {
		t.Errorf("a negative int must not merge with an unsigned value: %v", counter)
	}
	if _, ok := ToMapKey(big).(uint64); !ok {
		t.Errorf("ToMapKey(uint64 max) is %T, want uint64", ToMapKey(big))
	}
	if k := ToMapKey(uint8(9)); k != any(9) {
		t.Errorf("ToMapKey(uint8(9)) = %v (%T), want int 9", k, k)
	}
}
