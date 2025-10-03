package stats_test

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

const (
	uniquenessLowerBoundTest = 1e-9
	epsilonSmallTest         = 1e-10
)

func TestVarimaxRotationMatchesRReference(t *testing.T) {
	insyra.Config.SetLogLevel(insyra.LogLevelWarning)

	data := isr.DT.From(isr.CSV{
		FilePath: testDataPath(t, "local", "fa_test_dataset.csv"),
		InputOpts: isr.CSV_inOpts{
			FirstRow2ColNames: true,
			FirstCol2RowNames: false,
		},
	})

	opt := stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{
			Standardize: true,
			Missing:     "listwise",
		},
		Count: stats.FactorCountSpec{
			Method: stats.FactorCountFixed,
			FixedK: 2,
		},
		Extraction: stats.FactorExtractionPAF,
		Rotation: stats.FactorRotationOptions{
			Method: stats.FactorRotationVarimax,
		},
		Scoring: stats.FactorScoreRegression,
		MaxIter: 200,
		MinErr:  1e-3,
	}

	model := stats.FactorAnalysis(data, opt)
	if model == nil {
		t.Fatal("factor analysis returned nil")
	}

	goLoadings, err := dataTableToMatrix(model.Loadings)
	if err != nil {
		t.Fatalf("convert go loadings: %v", err)
	}
	goCommunalities, err := dataTableToVector(model.Communalities)
	if err != nil {
		t.Fatalf("convert go communalities: %v", err)
	}

	expectedLoadings, err := readMatrixCSV(testDataPath(t, "local", "fa_r_paf_varimax_loadings.csv"))
	if err != nil {
		t.Fatalf("read expected loadings: %v", err)
	}
	expectedCommunalities, err := readVectorCSV(testDataPath(t, "local", "fa_r_paf_varimax_communalities.csv"))
	if err != nil {
		t.Fatalf("read expected communalities: %v", err)
	}

	if len(expectedLoadings) != len(goLoadings) {
		t.Fatalf("row mismatch: expected %d rows, got %d", len(expectedLoadings), len(goLoadings))
	}
	if len(expectedLoadings) == 0 {
		t.Fatal("expected loadings empty")
	}

	m := len(goLoadings[0])
	bestDiff := math.MaxFloat64
	bestPerm := make([]int, m)
	bestSign := make([]float64, m)
	perms := permutations(m)
	signs := signCombos(m)
	for _, perm := range perms {
		for _, sign := range signs {
			diff := maxAbsDiff(goLoadings, expectedLoadings, perm, sign)
			if diff < bestDiff {
				bestDiff = diff
				copy(bestPerm, perm)
				copy(bestSign, sign)
			}
		}
	}
	if bestDiff > 1e-3 {
		aligned := applyAlignment(goLoadings, bestPerm, bestSign)
		var builder strings.Builder
		builder.WriteString("aligned loadings (Go):\n")
		for i := range aligned {
			for j := range aligned[i] {
				builder.WriteString(fmt.Sprintf(" %0.6f", aligned[i][j]))
			}
			builder.WriteByte('\n')
		}
		t.Fatalf("loadings mismatch: best max abs diff %.6f exceeds tolerance\nperm=%v sign=%v\n%s", bestDiff, bestPerm, bestSign, builder.String())
	}
	t.Logf("max abs diff (loadings after alignment): %.6f", bestDiff)

	if len(goCommunalities) != len(expectedCommunalities) {
		t.Fatalf("communalities length mismatch: expected %d got %d", len(expectedCommunalities), len(goCommunalities))
	}
	for i := range expectedCommunalities {
		if math.Abs(goCommunalities[i]-expectedCommunalities[i]) > 1e-3 {
			t.Fatalf("communalities[%d] mismatch: got %.6f want %.6f", i, goCommunalities[i], expectedCommunalities[i])
		}
	}
	t.Logf("max abs diff (communalities): %.6f", maxAbsDiffVectors(goCommunalities, expectedCommunalities))

	if model.Scores == nil {
		t.Fatal("model scores are nil")
	}
	goScores, err := dataTableToMatrix(model.Scores)
	if err != nil {
		t.Fatalf("convert go scores: %v", err)
	}
	if len(goScores) == 0 {
		t.Fatal("go scores matrix empty")
	}

	expectedScores := computeExpectedRegressionScores(t, data, expectedLoadings, expectedCommunalities)
	if len(expectedScores) != len(goScores) {
		t.Fatalf("scores row mismatch: expected %d got %d", len(expectedScores), len(goScores))
	}

	maxScoreDiff := maxAbsDiff(goScores, expectedScores, bestPerm, bestSign)
	t.Logf("max abs diff (scores after alignment): %.6f", maxScoreDiff)
	if maxScoreDiff > 5e-3 {
		t.Fatalf("factor scores mismatch: max abs diff %.6f exceeds tolerance", maxScoreDiff)
	}
}

func dataTableToMatrix(dt insyra.IDataTable) ([][]float64, error) {
	rows, cols := dt.Size()
	if rows == 0 || cols == 0 {
		return nil, nil
	}
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
	}
	var err error
	dt.AtomicDo(func(table *insyra.DataTable) {
		for j := 0; j < cols; j++ {
			column := table.GetColByNumber(j)
			for i := 0; i < rows; i++ {
				val, ok := column.Data()[i].(float64)
				if !ok {
					err = fmt.Errorf("value (%d,%d) is not float64", i, j)
					return
				}
				matrix[i][j] = val
			}
		}
	})
	return matrix, err
}

func dataTableToVector(dt insyra.IDataTable) ([]float64, error) {
	rows, _ := dt.Size()
	vector := make([]float64, rows)
	var err error
	dt.AtomicDo(func(table *insyra.DataTable) {
		column := table.GetColByNumber(0)
		for i := 0; i < rows; i++ {
			val, ok := column.Data()[i].(float64)
			if !ok {
				err = fmt.Errorf("value %d is not float64", i)
				return
			}
			vector[i] = val
		}
	})
	return vector, err
}

func readMatrixCSV(path string) ([][]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("matrix csv must have header and rows")
	}

	cols := len(records[0]) - 1
	rows := len(records) - 1
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
	}

	for i := 1; i < len(records); i++ {
		for j := 1; j < len(records[i]); j++ {
			val, err := strconv.ParseFloat(records[i][j], 64)
			if err != nil {
				return nil, fmt.Errorf("parse float at row %d col %d: %w", i, j, err)
			}
			matrix[i-1][j-1] = val
		}
	}
	return matrix, nil
}

func readVectorCSV(path string) ([]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("vector csv must have header and rows")
	}

	vector := make([]float64, len(records)-1)
	for i := 1; i < len(records); i++ {
		val, err := strconv.ParseFloat(records[i][1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse float at row %d: %w", i, err)
		}
		vector[i-1] = val
	}
	return vector, nil
}

func computeExpectedRegressionScores(t *testing.T, data insyra.IDataTable, loadings [][]float64, communalities []float64) [][]float64 {
	t.Helper()
	rawData, err := dataTableToMatrix(data)
	if err != nil {
		t.Fatalf("convert raw data: %v", err)
	}
	if len(rawData) == 0 || len(rawData[0]) == 0 {
		t.Fatal("raw data matrix is empty")
	}
	n := len(rawData)
	p := len(rawData[0])

	standardized := mat.NewDense(n, p, nil)
	col := make([]float64, n)
	for j := 0; j < p; j++ {
		for i := 0; i < n; i++ {
			col[i] = rawData[i][j]
		}
		mean, std := stat.MeanStdDev(col, nil)
		if std == 0 {
			std = 1
		}
		for i := 0; i < n; i++ {
			standardized.Set(i, j, (rawData[i][j]-mean)/std)
		}
	}

	if len(loadings) != p {
		t.Fatalf("loadings row mismatch for score check: expected %d got %d", p, len(loadings))
	}
	m := len(loadings[0])
	lambda := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			lambda.Set(i, j, loadings[i][j])
		}
	}

	phi := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		phi.Set(i, i, 1)
	}

	var lambdaPhi mat.Dense
	lambdaPhi.Mul(lambda, phi)

	psiVals := make([]float64, p)
	for i := 0; i < p; i++ {
		uniq := 1.0
		if i < len(communalities) {
			uniq = 1.0 - communalities[i]
		}
		if uniq < uniquenessLowerBoundTest {
			uniq = uniquenessLowerBoundTest
		}
		psiVals[i] = uniq
	}

	sigma := mat.NewDense(p, p, nil)
	sigma.Mul(&lambdaPhi, lambda.T())
	for i := 0; i < p; i++ {
		sigma.Set(i, i, sigma.At(i, i)+psiVals[i])
	}

	var sigmaInv mat.Dense
	if err := safeInvertTest(&sigmaInv, sigma, 1e-6); err != nil {
		t.Fatalf("invert sigma: %v", err)
	}

	var sigmaInvLambdaPhi mat.Dense
	sigmaInvLambdaPhi.Mul(&sigmaInv, &lambdaPhi)

	var ltSigInvL mat.Dense
	ltSigInvL.Mul(lambda.T(), &sigmaInvLambdaPhi)

	var weightsInnerInv mat.Dense
	if err := safeInvertTest(&weightsInnerInv, &ltSigInvL, 1e-6); err != nil {
		t.Fatalf("invert weights inner: %v", err)
	}

	var weights mat.Dense
	weights.Mul(&sigmaInvLambdaPhi, &weightsInnerInv)

	var expectedDense mat.Dense
	expectedDense.Mul(standardized, &weights)

	expected := make([][]float64, n)
	for i := 0; i < n; i++ {
		expected[i] = make([]float64, m)
		for j := 0; j < m; j++ {
			expected[i][j] = expectedDense.At(i, j)
		}
	}
	return expected
}

func safeInvertTest(dst *mat.Dense, src mat.Matrix, ridge float64) error {
	var a mat.Dense
	a.CloneFrom(src)
	r, c := a.Dims()
	if r == c && ridge > 0 {
		for i := 0; i < r; i++ {
			a.Set(i, i, a.At(i, i)+ridge)
		}
	}
	if err := dst.Inverse(&a); err != nil {
		return pseudoInverseTest(dst, &a)
	}
	return nil
}

func pseudoInverseTest(dst *mat.Dense, src mat.Matrix) error {
	var svd mat.SVD
	if !svd.Factorize(src, mat.SVDThin) {
		return fmt.Errorf("pseudo-inverse failed")
	}
	vals := svd.Values(nil)
	var u, vt mat.Dense
	svd.UTo(&u)
	svd.VTo(&vt)
	diag := mat.NewDense(len(vals), len(vals), nil)
	for i, val := range vals {
		if val > epsilonSmallTest {
			diag.Set(i, i, 1/val)
		}
	}
	var v mat.Dense
	v.CloneFrom(vt.T())
	var temp mat.Dense
	temp.Mul(&v, diag)
	var uT mat.Dense
	uT.CloneFrom(u.T())
	dst.Mul(&temp, &uT)
	return nil
}

func permutations(n int) [][]int {
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	var result [][]int
	var generate func(int)
	generate = func(k int) {
		if k == n {
			perm := make([]int, n)
			copy(perm, indices)
			result = append(result, perm)
			return
		}
		for i := k; i < n; i++ {
			indices[k], indices[i] = indices[i], indices[k]
			generate(k + 1)
			indices[k], indices[i] = indices[i], indices[k]
		}
	}
	generate(0)
	return result
}

func signCombos(n int) [][]float64 {
	combos := make([][]float64, 0, 1<<n)
	var helper func(int, []float64)
	helper = func(idx int, current []float64) {
		if idx == n {
			combo := make([]float64, n)
			copy(combo, current)
			combos = append(combos, combo)
			return
		}
		current[idx] = 1
		helper(idx+1, current)
		current[idx] = -1
		helper(idx+1, current)
	}
	helper(0, make([]float64, n))
	return combos
}

func maxAbsDiff(goMatrix, expectedMatrix [][]float64, perm []int, sign []float64) float64 {
	maxDiff := 0.0
	for i := range expectedMatrix {
		for j := range expectedMatrix[i] {
			val := goMatrix[i][perm[j]] * sign[j]
			diff := math.Abs(val - expectedMatrix[i][j])
			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}
	return maxDiff
}

func maxAbsDiffVectors(a, b []float64) float64 {
	maxDiff := 0.0
	for i := range a {
		diff := math.Abs(a[i] - b[i])
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

func applyAlignment(src [][]float64, perm []int, sign []float64) [][]float64 {
	rows := len(src)
	cols := len(perm)
	aligned := make([][]float64, rows)
	for i := range aligned {
		aligned[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			aligned[i][j] = src[i][perm[j]] * sign[j]
		}
	}
	return aligned
}

func testDataPath(t *testing.T, parts ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime caller failed")
	}
	base := filepath.Dir(filename)
	joined := append([]string{base, ".."}, parts...)
	return filepath.Join(joined...)
}
