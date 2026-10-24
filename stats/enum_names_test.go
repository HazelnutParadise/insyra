package stats

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
		{PearsonCorrelation, "PearsonCorrelation"},
		{KendallCorrelation, "KendallCorrelation"},
		{SpearmanCorrelation, "SpearmanCorrelation"},
		{CorrelationMethod(99), "CorrelationMethod(99)"},
		{KurtosisG2, "KurtosisG2"},
		{KurtosisAdjusted, "KurtosisAdjusted"},
		{KurtosisBiasAdjusted, "KurtosisBiasAdjusted"},
		{KurtosisMethod(99), "KurtosisMethod(99)"},
		{SkewnessG1, "SkewnessG1"},
		{SkewnessAdjusted, "SkewnessAdjusted"},
		{SkewnessBiasAdjusted, "SkewnessBiasAdjusted"},
		{SkewnessMethod(99), "SkewnessMethod(99)"},
		{PAdjustHolm, "PAdjustHolm"},
		{PAdjustBonferroni, "PAdjustBonferroni"},
		{PAdjustHochberg, "PAdjustHochberg"},
		{PAdjustBH, "PAdjustBH"},
		{PAdjustBY, "PAdjustBY"},
		{PAdjustMethod(99), "PAdjustMethod(99)"},
		{CovOLS, "CovOLS"},
		{CovHC0, "CovHC0"},
		{CovHC1, "CovHC1"},
		{CovHC2, "CovHC2"},
		{CovHC3, "CovHC3"},
		{CovHAC, "CovHAC"},
		{CovType(99), "CovType(99)"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(c.got); got != c.want {
			t.Errorf("%T: got %q, want %q", c.got, got, c.want)
		}
	}
}
