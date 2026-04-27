package stats_test

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestDumpAllFieldsForCompareWithR(t *testing.T) {
	rows := [][]float64{
		{1.0, 1.1, 0.9, 5.0, 5.2, 4.8},
		{1.2, 1.0, 1.1, 4.8, 5.0, 4.9},
		{0.8, 0.9, 1.0, 5.3, 5.1, 5.2},
		{4.9, 5.1, 5.0, 1.1, 1.0, 1.2},
		{5.2, 4.8, 5.1, 0.9, 1.2, 1.0},
		{5.0, 5.2, 4.9, 1.0, 0.8, 1.1},
		{2.6, 2.7, 2.5, 3.2, 3.1, 3.3},
		{3.1, 3.0, 3.2, 2.6, 2.5, 2.7},
		{1.5, 1.4, 1.6, 4.6, 4.4, 4.5},
		{4.5, 4.6, 4.4, 1.5, 1.6, 1.4},
	}
	dt := insyra.NewDataTable()
	for c := 0; c < 6; c++ {
		col := insyra.NewDataList()
		for r := 0; r < len(rows); r++ {
			col.Append(rows[r][c])
		}
		dt.AppendCols(col)
	}
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionMINRES
	opt.Rotation.Method = stats.FactorRotationOblimin
	opt.Scoring = stats.FactorScoreRegression
	res, err := stats.FactorAnalysis(dt, opt)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	dump := func(name string, dt insyra.IDataTable) {
		if dt == nil {
			fmt.Printf("%s: nil\n", name)
			return
		}
		nr, nc := dt.Size()
		fmt.Printf("%s (%dx%d):\n", name, nr, nc)
		for i := 0; i < nr; i++ {
			fmt.Printf("  ")
			for j := 0; j < nc; j++ {
				v := dt.GetElementByNumberIndex(i, j)
				if f, ok := v.(float64); ok {
					fmt.Printf("% .10g  ", f)
				} else {
					fmt.Printf("%v  ", v)
				}
			}
			fmt.Println()
		}
	}
	dump("Loadings", res.Loadings)
	dump("UnrotatedLoadings", res.UnrotatedLoadings)
	dump("Structure", res.Structure)
	dump("Uniquenesses", res.Uniquenesses)
	dump("Communalities", res.Communalities)
	dump("SamplingAdequacy", res.SamplingAdequacy)
	dump("Phi", res.Phi)
	dump("RotationMatrix", res.RotationMatrix)
	dump("Eigenvalues", res.Eigenvalues)
	dump("ExplainedProportion", res.ExplainedProportion)
	dump("CumulativeProportion", res.CumulativeProportion)
	dump("ScoreCoefficients", res.ScoreCoefficients)
	dump("ScoreCovariance", res.ScoreCovariance)
	if res.BartlettTest != nil {
		fmt.Printf("Bartlett: chi=%g df=%d p=%g n=%d\n",
			res.BartlettTest.ChiSquare, res.BartlettTest.DegreesOfFreedom,
			res.BartlettTest.PValue, res.BartlettTest.SampleSize)
	}
}
