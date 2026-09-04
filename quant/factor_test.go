package quant

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func factorTable(names []string, columns ...insyra.IDataList) *insyra.DataTable {
	dataLists := make([]*insyra.DataList, len(columns))
	for i, column := range columns {
		dataLists[i] = column.(*insyra.DataList)
	}
	return insyra.NewDataTable(dataLists...).SetColNames(names)
}

func assertSliceClose(t *testing.T, got, want []float64, tolerance float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length = %d, want %d", len(got), len(want))
	}
	for i := range got {
		assertClose(t, got[i], want[i], tolerance)
	}
}

func TestFactorModelAgreesWithLinearRegression(t *testing.T) {
	rng := rand.New(rand.NewSource(20260904))
	asset := make([]float64, 64)
	factors := make([][]float64, 3)
	for j := range factors {
		factors[j] = make([]float64, len(asset))
	}
	for i := range asset {
		for j := range factors {
			factors[j][i] = (rng.Float64() - 0.5) * 0.04
		}
		asset[i] = 0.0007 + 1.25*factors[0][i] - 0.7*factors[1][i] + 0.4*factors[2][i] + (rng.Float64()-0.5)*0.01
	}

	assetDL := toDL(asset...)
	factorDLs := []insyra.IDataList{toDL(factors[0]...), toDL(factors[1]...), toDL(factors[2]...)}
	got, err := FactorModel(assetDL, factorTable([]string{"MKT", "SMB", "HML"}, factorDLs...), 0)
	if err != nil {
		t.Fatalf("FactorModel returned unexpected error: %v", err)
	}
	want, err := stats.LinearRegression(assetDL, factorDLs[0], factorDLs[1], factorDLs[2])
	if err != nil {
		t.Fatalf("LinearRegression returned unexpected error: %v", err)
	}

	assertClose(t, got.Alpha, want.Coefficients[0], 1e-12)
	assertClose(t, got.AlphaStdErr, want.StandardErrors[0], 1e-12)
	assertClose(t, got.AlphaTValue, want.TValues[0], 1e-12)
	assertClose(t, got.AlphaPValue, want.PValues[0], 1e-12)
	assertSliceClose(t, got.Exposures, want.Coefficients[1:], 1e-12)
	assertSliceClose(t, got.StdErrs, want.StandardErrors[1:], 1e-12)
	assertSliceClose(t, got.TValues, want.TValues[1:], 1e-12)
	assertSliceClose(t, got.PValues, want.PValues[1:], 1e-12)
	assertClose(t, got.RSquared, want.RSquared, 1e-12)
	assertClose(t, got.AdjustedRSquared, want.AdjustedRSquared, 1e-12)
	assertSliceClose(t, got.Residuals, want.Residuals, 1e-12)
	if got.N != len(asset) {
		t.Errorf("N = %d, want %d", got.N, len(asset))
	}
}

func TestFactorModelOneFactorAgreesWithCAPM(t *testing.T) {
	rng := rand.New(rand.NewSource(20260904))
	const riskFreeRate = 0.0003
	marketExcess := make([]float64, 64)
	asset := make([]float64, len(marketExcess))
	marketRaw := make([]float64, len(marketExcess))
	for i := range marketExcess {
		marketExcess[i] = (rng.Float64() - 0.5) * 0.04
		marketRaw[i] = marketExcess[i] + riskFreeRate
		asset[i] = 0.001 + 1.4*marketExcess[i] + (rng.Float64()-0.5)*0.01
	}

	got, err := FactorModel(toDL(asset...), factorTable([]string{"MKT"}, toDL(marketExcess...)), riskFreeRate)
	if err != nil {
		t.Fatalf("FactorModel returned unexpected error: %v", err)
	}
	want, err := CAPM(toDL(asset...), toDL(marketRaw...), riskFreeRate)
	if err != nil {
		t.Fatalf("CAPM returned unexpected error: %v", err)
	}
	assertClose(t, got.Exposures[0], want.Beta, 1e-12)
	assertClose(t, got.Alpha, want.Alpha, 1e-12)
	assertClose(t, got.StdErrs[0], want.BetaStdErr, 1e-12)
	assertClose(t, got.AlphaStdErr, want.AlphaStdErr, 1e-12)
}

func TestFactorModelExposureLookup(t *testing.T) {
	asset := toDL(0.01, -0.004, 0.012, 0.003, -0.008, 0.006)
	factors := factorTable(
		[]string{"MKT", "SMB", "HML"},
		toDL(0.006, -0.002, 0.008, 0.001, -0.005, 0.004),
		toDL(0.002, -0.001, 0.003, -0.002, 0.001, 0.004),
		toDL(-0.003, 0.002, -0.001, 0.004, -0.002, 0.001),
	)

	got, err := FactorModel(asset, factors, 0)
	if err != nil {
		t.Fatalf("FactorModel returned unexpected error: %v", err)
	}
	if len(got.FactorNames) != 3 || got.FactorNames[0] != "MKT" || got.FactorNames[1] != "SMB" || got.FactorNames[2] != "HML" {
		t.Errorf("FactorNames = %v, want [MKT SMB HML]", got.FactorNames)
	}
	if exposure, ok := got.Exposure("SMB"); !ok {
		t.Error("Exposure(SMB) was not found")
	} else {
		assertClose(t, exposure, got.Exposures[1], 1e-12)
	}
	if exposure, ok := got.Exposure("MOM"); ok || exposure != 0 {
		t.Errorf("Exposure(MOM) = %v, %v, want 0, false", exposure, ok)
	}
}

func TestFactorModelRiskFreeRateShiftsOnlyAlpha(t *testing.T) {
	asset := toDL(0.01, -0.004, 0.012, 0.003, -0.008, 0.006, 0.002)
	factors := factorTable(
		[]string{"MKT", "SMB"},
		toDL(0.006, -0.002, 0.008, 0.001, -0.005, 0.004, 0.003),
		toDL(0.002, -0.001, 0.003, -0.002, 0.001, 0.004, -0.003),
	)
	withoutRF, err := FactorModel(asset, factors, 0)
	if err != nil {
		t.Fatalf("FactorModel without risk-free rate returned unexpected error: %v", err)
	}
	const riskFreeRate = 0.0002
	withRF, err := FactorModel(asset, factors, riskFreeRate)
	if err != nil {
		t.Fatalf("FactorModel with risk-free rate returned unexpected error: %v", err)
	}
	assertClose(t, withRF.Alpha, withoutRF.Alpha-riskFreeRate, 1e-12)
	assertSliceClose(t, withRF.Exposures, withoutRF.Exposures, 1e-12)
	assertSliceClose(t, withRF.StdErrs, withoutRF.StdErrs, 1e-12)
	assertSliceClose(t, withRF.TValues, withoutRF.TValues, 1e-12)
	assertSliceClose(t, withRF.PValues, withoutRF.PValues, 1e-12)
	assertClose(t, withRF.RSquared, withoutRF.RSquared, 1e-12)
	assertClose(t, withRF.AdjustedRSquared, withoutRF.AdjustedRSquared, 1e-12)
	assertSliceClose(t, withRF.Residuals, withoutRF.Residuals, 1e-12)
}

func TestFactorModelRejectsInvalidInput(t *testing.T) {
	validAsset := toDL(0.01, -0.004, 0.012, 0.003, -0.008, 0.006)
	validFactors := factorTable(
		[]string{"MKT", "SMB"},
		toDL(0.006, -0.002, 0.008, 0.001, -0.005, 0.004),
		toDL(0.002, -0.001, 0.003, -0.002, 0.001, 0.004),
	)

	cases := []struct {
		name    string
		asset   insyra.IDataList
		factors insyra.IDataTable
		want    string
	}{
		{"nil asset", nil, validFactors, "asset is nil"},
		{"nil factors", validAsset, nil, "factors is nil"},
		{"zero factor columns", validAsset, insyra.NewDataTable(), "no factor columns"},
		{"length mismatch", toDL(0.01, -0.004, 0.012, 0.003, -0.008, 0.006, 0.002), factorTable([]string{"SMB"}, toDL(0.002, -0.001, 0.003, -0.002, 0.001)), "SMB"},
		{"too few observations", toDL(0.01, 0.02, 0.03, 0.04), factorTable([]string{"MKT", "SMB", "HML"}, toDL(0.01, 0.02, 0.03, 0.04), toDL(0.02, 0.03, 0.04, 0.05), toDL(0.03, 0.04, 0.05, 0.06)), "at least 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := FactorModel(tc.asset, tc.factors, 0); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("FactorModel error = %v, want mention %q", err, tc.want)
			}
		})
	}

	badCases := []struct {
		name    string
		asset   insyra.IDataList
		factors insyra.IDataTable
		series  string
		row     string
	}{
		{"unreadable factor cell", insyra.NewDataList(0.01, -0.004, 0.012, 0.003, -0.008, 0.006, 0.002), factorTable(
			[]string{"MKT", "HML"},
			toDL(0.006, -0.002, 0.008, 0.001, -0.005, 0.004, 0.003),
			insyra.NewDataList(0.002, -0.001, 0.003, 0.004, 0.001, 0.004, "n/a"),
		), "HML", "row 7"},
		{"NaN factor cell", validAsset, factorTable(
			[]string{"MKT", "SMB"},
			toDL(0.006, -0.002, 0.008, 0.001, -0.005, 0.004),
			insyra.NewDataList(0.002, math.NaN(), 0.003, -0.002, 0.001, 0.004),
		), "SMB", "row 2"},
		{"Inf asset cell", insyra.NewDataList(0.01, -0.004, math.Inf(1), 0.003, -0.008, 0.006), validFactors, "asset", "row 3"},
	}
	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			err := func() error {
				_, err := FactorModel(tc.asset, tc.factors, 0)
				return err
			}()
			if err == nil {
				t.Fatal("FactorModel returned nil error")
			}
			if !strings.Contains(err.Error(), tc.series) || !strings.Contains(err.Error(), tc.row) {
				t.Errorf("FactorModel error = %q, want %s and %s", err, tc.series, tc.row)
			}
		})
	}

	collinear := factorTable(
		[]string{"MKT", "SMB"},
		toDL(0.01, -0.02, 0.015, 0.004, -0.008),
		toDL(0.01, -0.02, 0.015, 0.004, -0.008),
	)
	if _, err := FactorModel(toDL(0.02, -0.01, 0.03, 0.01, -0.02), collinear, 0); err == nil {
		t.Error("FactorModel returned a result for collinear factors")
	}
}
