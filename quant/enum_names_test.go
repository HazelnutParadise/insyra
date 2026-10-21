package quant

import (
	"fmt"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// TestEnumsPrintTheirNames pins enum-conventions: every numeric enum
// prints a name, and a value outside its constants prints TypeName(n).
func TestEnumsPrintTheirNames(t *testing.T) {
	cases := []struct {
		got  fmt.Stringer
		want string
	}{
		{OptionCall, "OptionCall"},
		{OptionPut, "OptionPut"},
		{OptionType(99), "OptionType(99)"},
		{MinimumVariance, "MinimumVariance"},
		{TargetReturn, "TargetReturn"},
		{MaximumSharpe, "MaximumSharpe"},
		{PortfolioObjective(99), "PortfolioObjective(99)"},
		{VaRHistorical, "VaRHistorical"},
		{VaRParametric, "VaRParametric"},
		{VaRMethod(99), "VaRMethod(99)"},
		{RSIWilder, "RSIWilder"},
		{RSISimple, "RSISimple"},
		{RSIMethod(99), "RSIMethod(99)"},
		{RSIMissingError, "RSIMissingError"},
		{RSIMissingFlat, "RSIMissingFlat"},
		{RSIMissingBreak, "RSIMissingBreak"},
		{RSIMissing(99), "RSIMissing(99)"},
		{ResampleStationary, "ResampleStationary"},
		{ResampleMovingBlock, "ResampleMovingBlock"},
		{ResamplingScheme(99), "ResamplingScheme(99)"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(c.got); got != c.want {
			t.Errorf("%T: got %q, want %q", c.got, got, c.want)
		}
	}
}

// TestEnumErrorsNameTheValue pins that an error reporting an unknown enum
// value names it instead of printing a bare number.
func TestEnumErrorsNameTheValue(t *testing.T) {
	r := insyra.NewDataList(-0.02, 0.01, 0.03, -0.01, 0.02)
	_, err := ValueAtRisk(r, 0.95, VaRMethod(3))
	wantErrContains(t, err, "VaRMethod(3)")
	_, err = ConditionalValueAtRisk(r, 0.95, VaRMethod(3))
	wantErrContains(t, err, "VaRMethod(3)")
	_, err = BlackScholes(BSInput{Spot: 100, Strike: 100, Volatility: 0.2, TimeToExpiry: 1, Type: OptionType(7)})
	wantErrContains(t, err, "OptionType(7)")
	_, err = OptimizePortfolioMoments([]float64{0.1, 0.2}, [][]float64{{0.04, 0}, {0, 0.09}}, nil, PortfolioConfig{Objective: PortfolioObjective(9)})
	wantErrContains(t, err, "PortfolioObjective(9)")
}

func wantErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error %v does not contain %q", err, want)
	}
}
