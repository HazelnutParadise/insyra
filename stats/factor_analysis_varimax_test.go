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

	if len(goCommunalities) != len(expectedCommunalities) {
		t.Fatalf("communalities length mismatch: expected %d got %d", len(expectedCommunalities), len(goCommunalities))
	}
	for i := range expectedCommunalities {
		if math.Abs(goCommunalities[i]-expectedCommunalities[i]) > 1e-3 {
			t.Fatalf("communalities[%d] mismatch: got %.6f want %.6f", i, goCommunalities[i], expectedCommunalities[i])
		}
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
