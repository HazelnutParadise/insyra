package insyra

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// IN-18 of #337: IsNumeric lives here and answers through reflection, so a
// named type over a numeric kind is a number to it. ToFloat64Safe is a plain
// type switch in internal/utils and said the same value was not convertible.
// One of them had to be wrong about every user-defined numeric type.

type celsius float64
type counter int

func TestIsNumericAndToFloat64SafeAgree(t *testing.T) {
	values := []any{
		celsius(36.6), counter(3),
		int(1), int64(2), uint8(3), float32(4), float64(5),
		"6", nil, true, []int{1}, struct{}{},
	}
	for _, v := range values {
		numeric := IsNumeric(v)
		_, convertible := utils.ToFloat64Safe(v)
		if numeric != convertible {
			t.Errorf("%#v: IsNumeric says %v, ToFloat64Safe says %v", v, numeric, convertible)
		}
	}

	// And the conversion is right, not merely possible.
	if got, ok := utils.ToFloat64Safe(celsius(36.6)); !ok || math.Abs(got-36.6) > 1e-12 {
		t.Errorf("ToFloat64Safe(celsius(36.6)) = (%v, %v)", got, ok)
	}
	if got, ok := utils.ToFloat64Safe(counter(3)); !ok || got != 3 {
		t.Errorf("ToFloat64Safe(counter(3)) = (%v, %v)", got, ok)
	}
	if got := utils.ToFloat64(celsius(36.6)); math.Abs(got-36.6) > 1e-12 {
		t.Errorf("ToFloat64(celsius(36.6)) = %v", got)
	}
}
