package stats_test

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

var spssLoadings = [][]float64{
	{0.356343, 0.761905, -0.016447},
	{0.410697, 0.569859, 0.255471},
	{0.503369, 0.533348, 0.07185},
	{0.695961, 0.112845, -0.485805},
	{0.731736, -0.294728, -0.443574},
	{0.681167, -0.001737, -0.311898},
	{0.516291, -0.160115, 0.611136},
	{0.684009, -0.33903, 0.28828},
	{0.652246, -0.167228, 0.320202},
}

var spssRotmat = [][]float64{
	{0.987413, 0.149994, 0.050166},
	{-0.132469, 0.989933, -0.049839},
	{0.017541, -0.037383, -0.999147},
}

var spssPhi = [][]float64{
	{1.0, 0.015183, 0.03841},
	{0.015183, 1.0, -0.010466},
	{0.03841, -0.010466, 1.0},
}

func readFactorAnalysisSampleCSV(t *testing.T) *insyra.DataTable {
	file, err := os.Open("../local/factor_analysis_sample.csv")
	if err != nil {
		t.Skip("Test data not found")
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < 2 {
		t.Fatal("CSV insufficient data")
	}

	header := records[0]
	cols := make([]*insyra.DataList, len(header))
	for j := range header {
		values := make([]any, len(records)-1)
		for i := 1; i < len(records); i++ {
			val, _ := strconv.ParseFloat(records[i][j], 64)
			values[i-1] = val
		}
		dl := insyra.NewDataList(values...)
		dl.SetName(header[j])
		cols[j] = dl
	}

	return insyra.NewDataTable(cols...)
}

func extractMatDense(table insyra.IDataTable) *mat.Dense {
	var result *mat.Dense
	table.AtomicDo(func(dt *insyra.DataTable) {
		r, c := dt.Size()
		m := mat.NewDense(r, c, nil)
		for i := 0; i < r; i++ {
			row := dt.GetRow(i)
			for j := 0; j < c; j++ {
				v, _ := row.Get(j).(float64)
				m.Set(i, j, v)
			}
		}
		result = m
	})
	return result
}

func compareMatrices(actual, expected *mat.Dense) (maxAbs, rmse float64) {
	if actual == nil || expected == nil {
		return math.NaN(), math.NaN()
	}
	r1, c1 := actual.Dims()
	r2, c2 := expected.Dims()
	if r1 != r2 || c1 != c2 {
		return math.NaN(), math.NaN()
	}
	sumSq, maxAbs, n := 0.0, 0.0, 0
	for i := 0; i < r1; i++ {
		for j := 0; j < c1; j++ {
			diff := math.Abs(actual.At(i, j) - expected.At(i, j))
			if diff > maxAbs {
				maxAbs = diff
			}
			sumSq += diff * diff
			n++
		}
	}
	rmse = math.Sqrt(sumSq / float64(n))
	return
}

func alignFactors(ref, actual *mat.Dense) ([]int, []float64, *mat.Dense) {
	if ref == nil || actual == nil {
		return nil, nil, nil
	}
	r, c := ref.Dims()
	r2, c2 := actual.Dims()
	if r != r2 || c != c2 {
		return nil, nil, nil
	}
	bestRMSE := math.Inf(1)
	var bestPerm []int
	var bestSigns []float64
	var bestAligned *mat.Dense
	perm, used := make([]int, c), make([]bool, c)
	var gen func(int)
	gen = func(pos int) {
		if pos == c {
			for s := 0; s < (1 << c); s++ {
				signs := make([]float64, c)
				for i := 0; i < c; i++ {
					signs[i] = 1.0
					if (s>>i)&1 == 1 {
						signs[i] = -1.0
					}
				}
				aligned := mat.NewDense(r, c, nil)
				for i := 0; i < r; i++ {
					for j := 0; j < c; j++ {
						aligned.Set(i, j, actual.At(i, perm[j])*signs[j])
					}
				}
				_, rmse := compareMatrices(aligned, ref)
				if rmse < bestRMSE {
					bestRMSE = rmse
					bestPerm = append([]int{}, perm...)
					bestSigns = append([]float64{}, signs...)
					bestAligned = mat.DenseCopyOf(aligned)
				}
			}
			return
		}
		for i := 0; i < c; i++ {
			if !used[i] {
				used[i] = true
				perm[pos] = i
				gen(pos + 1)
				used[i] = false
			}
		}
	}
	gen(0)
	return bestPerm, bestSigns, bestAligned
}

func applyPermSignsToRotmat(rotmat *mat.Dense, perm []int, signs []float64) *mat.Dense {
	if rotmat == nil {
		return nil
	}
	r, c := rotmat.Dims()
	result := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, rotmat.At(perm[i], perm[j])*signs[i]*signs[j])
		}
	}
	return result
}

func applyPermSignsToPhi(phi *mat.Dense, perm []int, signs []float64) *mat.Dense {
	if phi == nil {
		return nil
	}
	r, c := phi.Dims()
	result := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, phi.At(perm[i], perm[j]))
		}
	}
	return result
}

func TestFactorAnalysis_SPSS_Complete(t *testing.T) {
	dt := readFactorAnalysisSampleCSV(t)

	result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{Standardize: true},
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
		Extraction: stats.FactorExtractionPAF,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationOblimin, Kappa: 0},
		MinErr:     1e-9,
		MaxIter:    1000,
	})

	if result == nil {
		t.Fatal("Factor analysis result is nil")
	}

	loadingsMat := extractMatDense(result.Loadings)
	spssLoadingsMat := mat.NewDense(9, 3, nil)
	for i := 0; i < 9; i++ {
		for j := 0; j < 3; j++ {
			spssLoadingsMat.Set(i, j, spssLoadings[i][j])
		}
	}
	perm, signs, alignedLoadings := alignFactors(spssLoadingsMat, loadingsMat)
	maxAbsL, rmseL := compareMatrices(alignedLoadings, spssLoadingsMat)

	t.Logf("Loadings: maxAbs=%.6f rmse=%.6f (perm=%v signs=%v)", maxAbsL, rmseL, perm, signs)

	if result.RotationMatrix != nil {
		rotmatMat := extractMatDense(result.RotationMatrix)
		spssRotmatMat := mat.NewDense(3, 3, nil)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				spssRotmatMat.Set(i, j, spssRotmat[i][j])
			}
		}
		alignedRotmat := applyPermSignsToRotmat(rotmatMat, perm, signs)
		maxAbsR, rmseR := compareMatrices(alignedRotmat, spssRotmatMat)
		t.Logf("Rotmat:   maxAbs=%.6f rmse=%.6f", maxAbsR, rmseR)
	}

	if result.Phi != nil {
		phiMat := extractMatDense(result.Phi)
		spssPhiMat := mat.NewDense(3, 3, nil)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				spssPhiMat.Set(i, j, spssPhi[i][j])
			}
		}
		alignedPhi := applyPermSignsToPhi(phiMat, perm, signs)
		maxAbsP, rmseP := compareMatrices(alignedPhi, spssPhiMat)
		t.Logf("Phi:      maxAbs=%.6f rmse=%.6f", maxAbsP, rmseP)
	}

	fmt.Printf("\n Factor Analysis SPSS Comparison Complete\n")
	fmt.Printf("  Loadings: maxAbs=%.6f rmse=%.6f\n", maxAbsL, rmseL)
}

func TestFactorAnalysis_RotationMethods(t *testing.T) {
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1).SetName("V4"),
	}
	dt := insyra.NewDataTable(data...)

	methods := []stats.FactorRotationMethod{
		stats.FactorRotationNone,
		stats.FactorRotationVarimax,
		stats.FactorRotationOblimin,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
				Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
				Extraction: stats.FactorExtractionPCA,
				Rotation:   stats.FactorRotationOptions{Method: method},
				Scoring:    stats.FactorScoreNone,
			})
			if result == nil || result.Loadings == nil {
				t.Errorf("Failed for method %s", method)
			}
		})
	}
}

func TestFactorAnalysis_ExtractionMethods(t *testing.T) {
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.2, 3.1, 4.5, 5.3, 6.2).SetName("A1"),
		insyra.NewDataList(1.5, 2.4, 3.6, 4.0, 5.7, 6.5).SetName("A2"),
		insyra.NewDataList(2.1, 3.0, 3.8, 4.9, 5.8, 6.9).SetName("A3"),
		insyra.NewDataList(0.9, 1.8, 2.9, 4.2, 5.0, 5.8).SetName("A4"),
	}
	dt := insyra.NewDataTable(data...)

	methods := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionMINRES,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
				Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
				Extraction: method,
				Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationVarimax},
				Scoring:    stats.FactorScoreNone,
			})
			if result == nil || result.Loadings == nil {
				t.Errorf("Failed for method %s", method)
			}
		})
	}
}
