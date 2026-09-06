package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestNilInterfaceInputsDoNotPanic(t *testing.T) {
	good := insyra.NewDataList(1.0, 2.0, 3.0, 4.0)
	calls := map[string]func() error{
		"Correlation":    func() error { _, err := stats.Correlation(nil, good, stats.PearsonCorrelation); return err },
		"Covariance":     func() error { _, err := stats.Covariance(good, nil); return err },
		"PairedTTest":    func() error { _, err := stats.PairedTTest(nil, good); return err },
		"MannWhitneyU":   func() error { _, err := stats.MannWhitneyU(nil, good, stats.TwoSided); return err },
		"PairedWilcoxon": func() error { _, err := stats.PairedWilcoxon(good, nil, stats.TwoSided); return err },
		"ExponentialReg": func() error { _, err := stats.ExponentialRegression(nil, good); return err },
		"LogarithmicReg": func() error { _, err := stats.LogarithmicRegression(good, nil); return err },
		"PolynomialReg":  func() error { _, err := stats.PolynomialRegression(nil, good, 2); return err },
	}
	for name, f := range calls {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked: %v", name, r)
				}
			}()
			if err := f(); err == nil {
				t.Fatalf("%s: expected error for nil input", name)
			}
		}()
	}
}
