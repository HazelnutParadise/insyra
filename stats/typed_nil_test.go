package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
	"github.com/HazelnutParadise/insyra/stats"
)

// A nil list, bare or wrapped, is an error, never a panic in the caller's
// goroutine or in one the function starts itself.
func TestNilListsAreErrorsNotPanics(t *testing.T) {
	var typedNil *insyra.DataList
	good := insyra.NewDataList(1.0, 2.0, 3.0, 4.0)
	table := insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0), insyra.NewDataList(2.0, 1.0, 4.0, 3.0))
	for name, nilList := range map[string]insyra.IDataList{"typed nil": typedNil, "wrapper": isr.DL} {
		cases := map[string]func() error{
			"LinearRegression y": func() error { _, e := stats.LinearRegression(nilList, good); return e },
			"LinearRegression x": func() error { _, e := stats.LinearRegression(good, nilList); return e },
			"PoissonRegression":  func() error { _, e := stats.PoissonRegression(nilList, good); return e },
			"LogisticRegression": func() error { _, e := stats.LogisticRegression(nilList, good); return e },
			"Correlation":        func() error { _, e := stats.Correlation(nilList, good, stats.PearsonCorrelation); return e },
			"Covariance":         func() error { _, e := stats.Covariance(good, nilList); return e },
			"ChiSquareGoF": func() error {
				_, e := stats.ChiSquareGoodnessOfFit(nilList, map[string]float64{"a": 0.5, "b": 0.5}, false)
				return e
			},
			"ChiSquareIndep": func() error { _, e := stats.ChiSquareIndependenceTest(good, nilList); return e },
			"Silhouette":     func() error { _, e := stats.Silhouette(table, nilList); return e },
			"KNNClassify":    func() error { _, e := stats.KNNClassify(table, nilList, table, 1); return e },
			"KNNRegress":     func() error { _, e := stats.KNNRegress(table, nilList, table, 1); return e },
			"Skewness":       func() error { _, e := stats.Skewness(nilList); return e },
			"Kurtosis":       func() error { _, e := stats.Kurtosis(nilList); return e },
		}
		for cname, f := range cases {
			t.Run(name+"/"+cname, func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panicked: %v", r)
					}
				}()
				if err := f(); err == nil {
					t.Fatal("want an error, got nil")
				}
			})
		}
	}
}
