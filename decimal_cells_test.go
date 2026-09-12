package insyra

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/TimLai666/go-decimal/decimal"
)

// Two places in the library put a fixed-point decimal in a cell —
// finance.ScheduleTable and a Parquet Decimal128 column — and the numeric
// path used to answer NaN on both, silently.

func dec(s string) decimal.Decimal {
	return decimal.MustParse(decimal.Context{Scale: 10}, s)
}

func TestADecimalColumnCanBeAveraged(t *testing.T) {
	dl := NewDataList(Cell(dec("1.50")), Cell(dec("2.50")), Cell(dec("3.00")))
	if got := dl.Mean(); got != 7.0/3.0 {
		t.Errorf("Mean = %v, want %v", got, 7.0/3.0)
	}
	if got := dl.Sum(); got != 7.0 {
		t.Errorf("Sum = %v, want 7", got)
	}
}

func TestIsNumericAgreesWithReadingADecimal(t *testing.T) {
	d := dec("2.50")
	f, ok := utils.ToFloat64Safe(d)
	if !ok {
		t.Fatalf("ToFloat64Safe refused a decimal")
	}
	if f != 2.5 {
		t.Errorf("ToFloat64Safe = %v, want 2.5", f)
	}
	if !IsNumeric(d) {
		t.Error("IsNumeric says a decimal is not a number, disagreeing with the read path")
	}
}

func TestADecimalSortsAmongNumbers(t *testing.T) {
	dl := NewDataList(Cell(dec("2.50")), 1.0, Cell(dec("10.00")), 3.0)
	dl.Sort()
	var got []string
	for i := 0; i < dl.Len(); i++ {
		got = append(got, fmt.Sprintf("%v", dl.Get(i)))
	}
	want := []string{"1", "2.5000000000", "3", "10.0000000000"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sorted as %v, want interleaved by value %v", got, want)
			break
		}
	}
}

// Two decimals still compare exactly: a float64 carries about 16 significant
// digits and these differ in the 20th.
func TestTwoDecimalsCompareExactly(t *testing.T) {
	a := dec("10000000000000000000.0000000001")
	b := dec("10000000000000000000.0000000002")
	if algorithms.CompareAny(a, b) >= 0 {
		t.Error("two decimals differing beyond float64 precision compared equal or reversed")
	}
}

// The shape is two methods plus text that parses as a number, so a type that
// merely has the methods is not swept in.
type zzNotADecimal struct{}

func (zzNotADecimal) String() string { return "not a number" }
func (zzNotADecimal) Scale() int32   { return 2 }

func TestALookalikeIsNotANumber(t *testing.T) {
	if _, ok := utils.ToFloat64Safe(zzNotADecimal{}); ok {
		t.Error("a type whose text is not a number was read as one")
	}
	if IsNumeric(zzNotADecimal{}) {
		t.Error("IsNumeric accepted a lookalike")
	}
}

// Being a number does not make it a map key: big.Int holds a slice.
func TestADecimalIsStillNotAMapKey(t *testing.T) {
	if _, wrapped := ToMapKey(dec("2.50")).(UncomparableKey); !wrapped {
		t.Error("a decimal was used as a map key directly")
	}
}
