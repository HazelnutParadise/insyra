package quant

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestValueAtRiskHistoricalAndParametric(t *testing.T) {
	returns := []float64{-0.05, -0.02, 0, 0.01, 0.03}
	got, err := ValueAtRisk(toDL(returns...), 0.8, VaRHistorical)
	if err != nil {
		t.Fatalf("ValueAtRisk returned unexpected error: %v", err)
	}
	want := -toDL(returns...).Percentile(20)
	assertClose(t, got, want, 1e-12)

	parametric := []float64{0.001 - 0.02, 0.001, 0.001 + 0.02}
	got, err = ValueAtRisk(toDL(parametric...), 0.95, VaRParametric)
	if err != nil {
		t.Fatalf("parametric ValueAtRisk returned unexpected error: %v", err)
	}
	z, err := stats.NormPPF(0.05)
	if err != nil {
		t.Fatalf("NormPPF returned unexpected error: %v", err)
	}
	assertClose(t, got, -(0.001 + z*0.02), 1e-12)
}

func TestConditionalValueAtRiskIsAtLeastValueAtRisk(t *testing.T) {
	returns := toDL(-0.08, -0.04, -0.01, 0.01, 0.03, 0.06)
	for _, method := range []VaRMethod{VaRHistorical, VaRParametric} {
		var_, err := ValueAtRisk(returns, 0.8, method)
		if err != nil {
			t.Fatalf("ValueAtRisk returned unexpected error: %v", err)
		}
		cvar, err := ConditionalValueAtRisk(returns, 0.8, method)
		if err != nil {
			t.Fatalf("ConditionalValueAtRisk returned unexpected error: %v", err)
		}
		if cvar+1e-12 < var_ {
			t.Errorf("method %d: CVaR = %.17g, VaR = %.17g; CVaR must be at least VaR", method, cvar, var_)
		}
	}
}

func TestValueAtRiskRejectsInvalidInput(t *testing.T) {
	valid := toDL(-0.02, 0.01)
	for _, confidence := range []float64{0, 1, 1.5} {
		t.Run(fmt.Sprintf("confidence_%v", confidence), func(t *testing.T) {
			if _, err := ValueAtRisk(valid, confidence, VaRHistorical); err == nil || !strings.Contains(err.Error(), "confidence") {
				t.Errorf("ValueAtRisk error = %v, want confidence error", err)
			}
		})
	}
	if _, err := ValueAtRisk(valid, 0.95, VaRMethod(99)); err == nil || !strings.Contains(err.Error(), "method") {
		t.Errorf("unknown method error = %v, want method error", err)
	}
	if _, err := ConditionalValueAtRisk(toDL(0.01), 0.95, VaRHistorical); err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Errorf("too-short CVaR error = %v, want at-least-2 error", err)
	}

	for _, tc := range []struct {
		name string
		bad  any
	}{
		{name: "unreadable", bad: "n/a"},
		{name: "nan", bad: math.NaN()},
		{name: "inf", bad: math.Inf(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := insyra.NewDataList(0.01, tc.bad, 0.03)
			for _, method := range []VaRMethod{VaRHistorical, VaRParametric} {
				if _, err := ValueAtRisk(bad, 0.95, method); err == nil || !strings.Contains(err.Error(), "row 2") {
					t.Errorf("ValueAtRisk error = %v, want row 2", err)
				}
				if _, err := ConditionalValueAtRisk(bad, 0.95, method); err == nil || !strings.Contains(err.Error(), "row 2") {
					t.Errorf("ConditionalValueAtRisk error = %v, want row 2", err)
				}
			}
		})
	}
}

func TestSortinoRatioUsesAllPeriodsInDownsideDeviation(t *testing.T) {
	returns := toDL(0.02, -0.01, 0.03, -0.02)
	got, err := SortinoRatio(returns, 0, 1)
	if err != nil {
		t.Fatalf("SortinoRatio returned unexpected error: %v", err)
	}
	want := 0.005 / math.Sqrt((0.0001+0.0004)/4)
	assertClose(t, got, want, 1e-12)
}

func TestSortinoRatioRejectsZeroDownsideAndInvalidPeriods(t *testing.T) {
	for _, periods := range []float64{0, -1} {
		if _, err := SortinoRatio(toDL(0.01, 0.02), 0, periods); err == nil || !strings.Contains(err.Error(), "periodsPerYear") {
			t.Errorf("periodsPerYear=%v error = %v, want periodsPerYear error", periods, err)
		}
	}
	if _, err := SortinoRatio(toDL(0.01, 0.02), 0, 1); err == nil || !strings.Contains(err.Error(), "downside deviation") {
		t.Errorf("zero-downside error = %v, want downside deviation error", err)
	}
}

func TestCalmarRatioComposesAnnualizedReturnAndMaxDrawdown(t *testing.T) {
	equity := toDL(100, 110, 105, 120)
	got, err := CalmarRatio(equity, 365)
	if err != nil {
		t.Fatalf("CalmarRatio returned unexpected error: %v", err)
	}
	annualized, err := AnnualizedReturn(equity, 365)
	if err != nil {
		t.Fatalf("AnnualizedReturn returned unexpected error: %v", err)
	}
	drawdown, err := MaxDrawdown(equity)
	if err != nil {
		t.Fatalf("MaxDrawdown returned unexpected error: %v", err)
	}
	assertClose(t, got, annualized/drawdown, 1e-12)

	if _, err := CalmarRatio(toDL(100, 101, 102), 365); err == nil || !strings.Contains(err.Error(), "drawdown") {
		t.Errorf("zero-drawdown error = %v, want drawdown error", err)
	}
}

func TestInformationRatioHandComputed(t *testing.T) {
	returns := toDL(0.03, 0.01, 0.02, 0)
	benchmark := toDL(0.01, 0, 0.01, 0.01)
	got, err := InformationRatio(returns, benchmark, 1)
	if err != nil {
		t.Fatalf("InformationRatio returned unexpected error: %v", err)
	}
	mean := 0.0075
	sd := math.Sqrt(0.000475 / 3)
	assertClose(t, got, mean/sd, 1e-12)

	if _, err := InformationRatio(returns, returns, 1); err == nil || !strings.Contains(err.Error(), "tracking error") {
		t.Errorf("zero-tracking-error error = %v, want tracking error error", err)
	}
	if _, err := InformationRatio(returns, toDL(0.01, 0, 0.01), 1); err == nil || !strings.Contains(err.Error(), "length") {
		t.Errorf("length mismatch error = %v, want length error", err)
	}
}

func TestDrawdownSeriesMatchesMaxDrawdown(t *testing.T) {
	equity := toDL(100, 110, 105, 120, 90)
	got, err := DrawdownSeries(equity)
	if err != nil {
		t.Fatalf("DrawdownSeries returned unexpected error: %v", err)
	}
	values := got.Data()
	want := []any{0.0, 0.0, 5.0 / 110.0, 0.0, 0.25}
	if len(values) != len(want) {
		t.Fatalf("DrawdownSeries length = %d, want %d", len(values), len(want))
	}
	for i := range want {
		assertClose(t, values[i].(float64), want[i].(float64), 1e-12)
	}
	maxDD, err := MaxDrawdown(equity)
	if err != nil {
		t.Fatalf("MaxDrawdown returned unexpected error: %v", err)
	}
	assertClose(t, values[0].(float64), 0, 1e-12)
	assertClose(t, values[4].(float64), maxDD, 1e-12)
}

func TestDrawdownSeriesMonotoneAndNonPositivePeaks(t *testing.T) {
	monotone, err := DrawdownSeries(toDL(1, 2, 3))
	if err != nil {
		t.Fatalf("monotone DrawdownSeries returned unexpected error: %v", err)
	}
	for i, value := range monotone.Data() {
		if value != 0.0 {
			t.Errorf("monotone drawdown[%d] = %v, want 0", i, value)
		}
	}

	withNonPositive, err := DrawdownSeries(toDL(-1, 0, 2, 1))
	if err != nil {
		t.Fatalf("non-positive DrawdownSeries returned unexpected error: %v", err)
	}
	values := withNonPositive.Data()
	if values[0] != nil || values[1] != nil {
		t.Errorf("drawdowns before positive peak = %#v, want nil, nil", values[:2])
	}
	assertClose(t, values[2].(float64), 0, 1e-12)
	assertClose(t, values[3].(float64), 0.5, 1e-12)

	if _, err := DrawdownSeries(insyra.NewDataList()); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty DrawdownSeries error = %v, want empty error", err)
	}
}
