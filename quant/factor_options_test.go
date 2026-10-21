package quant

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// robustFactorInputs is a seeded factor regression whose residuals are AR(1)
// and grow with the market move, the case HAC standard errors exist for.
func robustFactorInputs(n int) (insyra.IDataList, *insyra.DataTable, []insyra.IDataList) {
	rng := rand.New(rand.NewSource(478))
	asset := make([]float64, n)
	mkt := make([]float64, n)
	smb := make([]float64, n)
	prevE := 0.0
	for i := range asset {
		mkt[i] = (rng.Float64() - 0.5) * 0.04
		smb[i] = (rng.Float64() - 0.5) * 0.02
		e := 0.6*prevE + (rng.Float64()-0.5)*0.01*(1+50*math.Abs(mkt[i]))
		prevE = e
		asset[i] = 0.0004 + 1.1*mkt[i] + 0.3*smb[i] + e
	}
	columns := []insyra.IDataList{toDL(mkt...), toDL(smb...)}
	return toDL(asset...), factorTable([]string{"MKT", "SMB"}, columns...), columns
}

func TestFactorModelHACAgreesWithRobustRegression(t *testing.T) {
	asset, factors, columns := robustFactorInputs(120)
	got, err := FactorModel(asset, factors, 0, FactorModelOptions{CovType: stats.CovHAC, Lag: 3})
	if err != nil {
		t.Fatal(err)
	}
	want, err := stats.LinearRegressionWithOptions(stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: 3}, asset, columns...)
	if err != nil {
		t.Fatal(err)
	}
	if got.AlphaStdErr != want.StandardErrors[0] || got.AlphaTValue != want.TValues[0] || got.AlphaPValue != want.PValues[0] {
		t.Errorf("alpha inference %v %v %v, want %v %v %v", got.AlphaStdErr, got.AlphaTValue, got.AlphaPValue, want.StandardErrors[0], want.TValues[0], want.PValues[0])
	}
	assertSliceClose(t, got.StdErrs, want.StandardErrors[1:], 0)
	assertSliceClose(t, got.TValues, want.TValues[1:], 0)
	assertSliceClose(t, got.PValues, want.PValues[1:], 0)
	if got.CovType != stats.CovHAC || got.Lag != 3 {
		t.Errorf("recorded %v lag %d, want CovHAC lag 3", got.CovType, got.Lag)
	}
	plain, err := FactorModel(asset, factors, 0)
	if err != nil {
		t.Fatal(err)
	}
	if plain.AlphaStdErr == got.AlphaStdErr || plain.Alpha != got.Alpha {
		t.Errorf("HAC should change the standard error and keep alpha: %v/%v, %v/%v", plain.AlphaStdErr, got.AlphaStdErr, plain.Alpha, got.Alpha)
	}
}

func TestFactorModelWithoutOptionsIsUnchanged(t *testing.T) {
	asset, factors, columns := robustFactorInputs(60)
	plain, err := FactorModel(asset, factors, 0)
	if err != nil {
		t.Fatal(err)
	}
	zero, err := FactorModel(asset, factors, 0, FactorModelOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// %#v prints every float in a form that parses back to the same bits.
	if fmt.Sprintf("%#v", plain) != fmt.Sprintf("%#v", zero) {
		t.Errorf("zero options differ from no options")
	}
	if plain.CovType != stats.CovOLS || plain.Lag != 0 {
		t.Errorf("recorded %v lag %d, want CovOLS lag 0", plain.CovType, plain.Lag)
	}
	ols, err := stats.LinearRegression(asset, columns...)
	if err != nil {
		t.Fatal(err)
	}
	if plain.AlphaStdErr != ols.StandardErrors[0] {
		t.Errorf("default alpha standard error %v, want the OLS one %v", plain.AlphaStdErr, ols.StandardErrors[0])
	}
}

func TestFactorModelHACLagSettings(t *testing.T) {
	asset, factors, _ := robustFactorInputs(120)
	auto, err := FactorModel(asset, factors, 0, FactorModelOptions{CovType: stats.CovHAC})
	if err != nil {
		t.Fatal(err)
	}
	if auto.Lag != stats.NeweyWestLag(120) {
		t.Errorf("automatic lag %d, want %d", auto.Lag, stats.NeweyWestLag(120))
	}
	if _, err := FactorModel(asset, factors, 0, FactorModelOptions{CovType: stats.CovHAC, Lag: -1}); err == nil {
		t.Error("a negative lag should be refused")
	}
	if _, err := FactorModel(asset, factors, 0, FactorModelOptions{CovType: stats.CovHAC, Lag: 120}); err == nil {
		t.Error("a lag equal to n should be refused")
	}
	if _, err := FactorModel(asset, factors, 0, FactorModelOptions{}, FactorModelOptions{}); err == nil || !strings.Contains(err.Error(), "at most one") {
		t.Errorf("two options: %v", err)
	}
	hc1, err := FactorModel(asset, factors, 0, FactorModelOptions{CovType: stats.CovHC1, Lag: 7})
	if err != nil || hc1.CovType != stats.CovHC1 || hc1.Lag != 0 {
		t.Errorf("HC1 ignores the lag: %+v, %v", hc1, err)
	}
}
