package finance

import (
	"fmt"
	"testing"
)

// TestEnumsPrintTheirNames pins enum-conventions: every numeric enum
// prints a name, and a value outside its constants prints TypeName(n).
func TestEnumsPrintTheirNames(t *testing.T) {
	cases := []struct {
		got  fmt.Stringer
		want string
	}{
		{Basis30_360US, "Basis30_360US"},
		{BasisActualActual, "BasisActualActual"},
		{BasisActual360, "BasisActual360"},
		{BasisActual365, "BasisActual365"},
		{Basis30_360EU, "Basis30_360EU"},
		{DayCountBasis(99), "DayCountBasis(99)"},
		{PaymentEnd, "PaymentEnd"},
		{PaymentBegin, "PaymentBegin"},
		{PaymentTiming(99), "PaymentTiming(99)"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(c.got); got != c.want {
			t.Errorf("%T: got %q, want %q", c.got, got, c.want)
		}
	}
}
