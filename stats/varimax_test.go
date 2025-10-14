package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestVarimaxRotmat(t *testing.T) {
	// Simple test data
	data := make([]*insyra.DataList, 3)
	for i := 0; i < 3; i++ {
		vals := []any{1.0, 2.0, 3.0, 4.0, 5.0}
		dl := insyra.NewDataList(vals...)
		data[i] = dl
	}

	dt := insyra.NewDataTable(data...)

	opt := stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{Standardize: true},
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
		Extraction: stats.FactorExtractionPCA,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationVarimax},
		Scoring:    stats.FactorScoreNone,
	}

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatalf("FactorAnalysis returned nil")
	}

	// Check if Phi is identity for orthogonal rotation
	if model.Phi != nil {
		model.Phi.AtomicDo(func(table *insyra.DataTable) {
			r, c := table.Size()
			t.Logf("Phi dims: %d x %d", r, c)
			for i := 0; i < r; i++ {
				for j := 0; j < c; j++ {
					row := table.GetRow(i)
					v, _ := row.Get(j).(float64)
					expected := 0.0
					if i == j {
						expected = 1.0
					}
					if (v - expected) > 1e-6 {
						t.Errorf("Phi[%d,%d] = %f, expected %f (orthogonal rotation should have identity Phi)", i, j, v, expected)
					}
				}
			}
		})
	}
}
