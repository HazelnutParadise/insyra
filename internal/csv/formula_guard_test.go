package csv

import "testing"

func TestGuardFormulaNumericSpellings(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"-1e309", "-1e309"}, {"+Inf", "'+Inf"}, {"-NaN", "'-NaN"}, {"+0x1p2", "'+0x1p2"},
	} {
		if got := GuardFormula(tc.in); got != tc.want {
			t.Errorf("GuardFormula(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
