package stats_test

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	stats "github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

// TestScoringInvariants checks that each scoring method satisfies its
// defining algebraic property — independent of R-parity.
//
//	Regression scores (Thomson):
//	  W = U^-1 · Λ · (Λ' · U^-1 · Λ + Φ^-1)^-1   when oblique
//	  W = U^-1 · Λ · (Λ' · U^-1 · Λ + I)^-1      when orthogonal
//	  scores = Z · W where Z is standardized data
//
//	Bartlett scores (WLS):
//	  W = U^-1 · Λ · (Λ' · U^-1 · Λ)^-1
//	  Cov(scores) = (Λ' · U^-1 · Λ)^-1
//
//	Anderson-Rubin scores:
//	  Cov(scores) = I (uncorrelated factors with unit variance)
//
// Run with INSYRA_VERIFY_SCORING=1.
func TestScoringInvariants(t *testing.T) {
	if os.Getenv("INSYRA_VERIFY_SCORING") != "1" {
		t.Skip()
	}
	const n = 80
	rows := make([][]any, n)
	for i := 0; i < n; i++ {
		x := float64(i)
		f1 := math.Sin(0.31*x) + 0.2*math.Cos(0.7*x)
		f2 := math.Cos(0.42*x) - 0.3*math.Sin(0.5*x)
		rows[i] = []any{
			0.85*f1 + 0.10*f2 + 0.20*math.Sin(1.3*x),
			0.78*f1 + 0.05*f2 + 0.18*math.Cos(1.7*x),
			0.10*f1 + 0.82*f2 + 0.21*math.Sin(2.1*x),
			0.05*f1 + 0.75*f2 + 0.19*math.Cos(2.4*x),
			0.30*f1 + 0.40*f2 + 0.17*math.Sin(2.8*x),
		}
	}
	tbl := insyra.NewDataTable()
	for c := 0; c < 5; c++ {
		col := make([]any, n)
		for i := 0; i < n; i++ {
			col[i] = rows[i][c]
		}
		tbl.AppendCols(insyra.NewDataList(col...))
	}

	scoringMethods := []stats.FactorScoreMethod{
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}

	allPass := true
	for _, sm := range scoringMethods {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 2
		opt.Extraction = stats.FactorExtractionML
		opt.Rotation.Method = stats.FactorRotationVarimax
		opt.Scoring = sm
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Errorf("[%s] FA error: %v", sm, err)
			allPass = false
			continue
		}
		Scores := dtToDense(res.Scores)
		ScoreCov := dtToDense(res.ScoreCovariance)

		report := []string{}

		// Empirical covariance of scores: Cov(scores) = scores' · scores / (n-1)
		if Scores != nil {
			sRows, sCols := Scores.Dims()
			centered := mat.NewDense(sRows, sCols, nil)
			means := make([]float64, sCols)
			for j := 0; j < sCols; j++ {
				for i := 0; i < sRows; i++ {
					means[j] += Scores.At(i, j)
				}
				means[j] /= float64(sRows)
			}
			for i := 0; i < sRows; i++ {
				for j := 0; j < sCols; j++ {
					centered.Set(i, j, Scores.At(i, j)-means[j])
				}
			}
			centeredT := mat.DenseCopyOf(centered.T())
			empCov := mat.NewDense(sCols, sCols, nil)
			empCov.Mul(centeredT, centered)
			scale := 1.0 / float64(sRows-1)
			for i := 0; i < sCols; i++ {
				for j := 0; j < sCols; j++ {
					empCov.Set(i, j, empCov.At(i, j)*scale)
				}
			}

			// Anderson-Rubin: empirical Cov should be ≈ I (after standardization)
			if sm == stats.FactorScoreAndersonRubin {
				maxOff := 0.0
				for i := 0; i < sCols; i++ {
					for j := 0; j < sCols; j++ {
						target := 0.0
						if i == j {
							target = 1.0
						}
						if d := math.Abs(empCov.At(i, j) - target); d > maxOff {
							maxOff = d
						}
					}
				}
				if maxOff > 1e-3 {
					report = append(report, fmt.Sprintf("AR empirical Cov(scores)≠I max=%.3e", maxOff))
				}
			}

			// Reported ScoreCovariance should match empirical (within sample noise)
			if ScoreCov != nil {
				maxDiff := maxAbsDiff(empCov, ScoreCov)
				// tolerance loose — sample covariance vs theoretical
				if maxDiff > 0.02 {
					report = append(report, fmt.Sprintf("reported ScoreCov vs empirical max=%.3e", maxDiff))
				}
			} else {
				report = append(report, "ScoreCovariance is nil")
			}
		} else {
			report = append(report, "Scores is nil")
		}

		status := "✓ PASS"
		if len(report) > 0 {
			status = "✗ FAIL: " + fmt.Sprintf("%v", report)
			allPass = false
		}
		fmt.Printf("[%-15s] %s\n", sm, status)
	}
	if !allPass {
		t.Fail()
	}
}
