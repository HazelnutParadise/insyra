package stats_test

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Helper function to check if two floats are approximately equal
func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

// TestFactorAnalysisBasic tests basic factor analysis with default MINRES extraction
func TestFactorAnalysisBasic(t *testing.T) {
	// Create a simple test dataset (4 variables, 10 observations)
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0))
	dt.AppendCols(insyra.NewDataList(2.0, 4.0, 6.0, 8.0, 10.0, 12.0, 14.0, 16.0, 18.0, 20.0))
	dt.AppendCols(insyra.NewDataList(1.5, 3.0, 4.5, 6.0, 7.5, 9.0, 10.5, 12.0, 13.5, 15.0))
	dt.AppendCols(insyra.NewDataList(3.0, 6.0, 9.0, 12.0, 15.0, 18.0, 21.0, 24.0, 27.0, 30.0))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}

	// Check that loadings exist
	if model.Result.Loadings == nil {
		t.Fatal("Expected non-nil loadings")
	}

	// Check dimensions
	var loadingsRows, loadingsCols int
	model.Result.Loadings.AtomicDo(func(table *insyra.DataTable) {
		loadingsRows, loadingsCols = table.Size()
	})

	if loadingsRows != 4 {
		t.Errorf("Expected 4 rows in loadings, got %d", loadingsRows)
	}
	if loadingsCols != 2 {
		t.Errorf("Expected 2 columns in loadings, got %d", loadingsCols)
	}

	// Check that model reports correct number of factors
	if model.Result.CountUsed != 2 {
		t.Errorf("Expected 2 factors, got %d", model.Result.CountUsed)
	}

	// Check communalities are between 0 and 1
	if model.Result.Communalities != nil {
		model.Result.Communalities.AtomicDo(func(table *insyra.DataTable) {
			rows, _ := table.Size()
			for i := 0; i < rows; i++ {
				row := table.GetRow(i)
				val, ok := row.Get(0).(float64)
				if !ok {
					t.Errorf("Expected float64 communality at row %d", i)
					continue
				}
				if val < 0 || val > 1.01 { // Allow small numerical error
					t.Errorf("Communality at row %d out of range [0,1]: %f", i, val)
				}
			}
		})
	}

	// Check uniquenesses are between 0 and 1
	if model.Result.Uniquenesses != nil {
		model.Result.Uniquenesses.AtomicDo(func(table *insyra.DataTable) {
			rows, _ := table.Size()
			for i := 0; i < rows; i++ {
				row := table.GetRow(i)
				val, ok := row.Get(0).(float64)
				if !ok {
					t.Errorf("Expected float64 uniqueness at row %d", i)
					continue
				}
				if val < -0.01 || val > 1.01 { // Allow small numerical error
					t.Errorf("Uniqueness at row %d out of range [0,1]: %f", i, val)
				}
			}
		})
	}

	// Check eigenvalues are in descending order
	if model.Result.Eigenvalues != nil {
		model.Result.Eigenvalues.AtomicDo(func(table *insyra.DataTable) {
			rows, _ := table.Size()
			var prevEigen = math.Inf(1)
			for i := 0; i < rows; i++ {
				row := table.GetRow(i)
				val, ok := row.Get(0).(float64)
				if !ok {
					continue
				}
				if val > prevEigen+1e-6 {
					t.Errorf("Eigenvalues not in descending order at position %d: %f > %f", i, val, prevEigen)
				}
				prevEigen = val
			}
		})
	}
}

// TestFactorAnalysisKaiserCriterion tests Kaiser criterion for factor selection
func TestFactorAnalysisKaiserCriterion(t *testing.T) {
	// Create test data
	dt := insyra.NewDataTable()
	for i := 0; i < 5; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 20; j++ {
			col.Append(float64(j+i*10) + float64(i))
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountKaiser
	opt.Count.EigenThreshold = 1.0

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}

	// At least one factor should be retained with Kaiser criterion
	if model.Result.CountUsed < 1 {
		t.Errorf("Expected at least 1 factor with Kaiser criterion, got %d", model.Result.CountUsed)
	}
}

// TestFactorAnalysisPAF tests Principal Axis Factoring extraction
func TestFactorAnalysisPAF(t *testing.T) {
	// Create test data
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0))
	dt.AppendCols(insyra.NewDataList(2.0, 4.0, 6.0, 8.0, 10.0, 12.0, 14.0, 16.0))
	dt.AppendCols(insyra.NewDataList(1.5, 3.0, 4.5, 6.0, 7.5, 9.0, 10.5, 12.0))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionPAF
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 1
	opt.MaxIter = 50
	opt.Tol = 1e-4

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}

	// Check loadings exist
	if model.Result.Loadings == nil {
		t.Fatal("Expected non-nil loadings")
	}

	// PAF should report convergence status
	if model.Result.Iterations == 0 && opt.Extraction == stats.FactorExtractionPAF {
		t.Log("Warning: PAF reported 0 iterations")
	}
}

func TestFactorAnalysisPCAExtraction(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0))
	dt.AppendCols(insyra.NewDataList(1.2, 2.4, 3.6, 4.8, 6.0))
	dt.AppendCols(insyra.NewDataList(0.8, 1.6, 2.4, 3.2, 4.0))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionPCA
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected model for PCA extraction")
	}
	if model.Result.Loadings == nil {
		t.Fatal("Expected loadings for PCA extraction")
	}
	joined := strings.ToLower(strings.Join(model.Result.Messages, " "))
	if !strings.Contains(joined, "pca") {
		t.Errorf("Expected messages to mention PCA extraction, got %v", model.Result.Messages)
	}
}

// TestFactorAnalysisNoRotation tests factor analysis without rotation
func TestFactorAnalysisNoRotation(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0))
	dt.AppendCols(insyra.NewDataList(2.0, 4.0, 6.0, 8.0, 10.0))
	dt.AppendCols(insyra.NewDataList(3.0, 6.0, 9.0, 12.0, 15.0))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 1

	model := stats.FactorAnalysis(dt, opt)

	// Rotation matrix should be nil when no rotation is applied
	if model.Result.RotationMatrix != nil {
		t.Error("Expected nil rotation matrix when rotation is disabled")
	}
}

// TestFactorAnalysisVarimaxRotation tests Varimax rotation
func TestFactorAnalysisVarimaxRotation(t *testing.T) {
	dt := insyra.NewDataTable()
	// Create data with two underlying factors
	for i := 0; i < 6; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 15; j++ {
			// First 3 variables load on factor 1, last 3 on factor 2
			if i < 3 {
				col.Append(float64(j) + float64(i)*2)
			} else {
				col.Append(float64(j)*1.5 + float64(i-3)*3)
			}
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2

	model := stats.FactorAnalysis(dt, opt)

	// Rotation matrix should exist for Varimax
	if model.Result.RotationMatrix == nil {
		t.Error("Expected non-nil rotation matrix for Varimax rotation")
	}

	// Phi should be nil for orthogonal rotation
	if model.Result.Phi != nil {
		t.Log("Note: Phi is set for orthogonal rotation (should be nil or identity)")
	}
}

func TestFactorAnalysisMLExtraction(t *testing.T) {
	dt := insyra.NewDataTable()
	for i := 0; i < 4; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 60; j++ {
			col.Append(float64(j) + float64(i)*0.4 + float64(j%5)*0.2)
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.MaxIter = 120
	opt.Tol = 1e-5

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected model for ML extraction")
	}
	if model.Result.Loadings == nil {
		t.Fatal("Expected loadings for ML extraction")
	}
	if !model.Result.Converged {
		t.Log("ML extraction did not report convergence; verify tolerance settings")
	}
	joined := strings.ToLower(strings.Join(model.Result.Messages, " "))
	if !strings.Contains(joined, "ml") {
		t.Errorf("Expected messages to mention ML extraction, got %v", model.Result.Messages)
	}
}

func TestFactorAnalysisMINRESExtraction(t *testing.T) {
	dt := insyra.NewDataTable()
	for i := 0; i < 4; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 80; j++ {
			col.Append(float64(j) + float64(i)*0.25 + float64(j%3))
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionMINRES
	opt.MaxIter = 80
	opt.Tol = 1e-5

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected model for MINRES extraction")
	}
	if model.Result.Loadings == nil {
		t.Fatal("Expected loadings for MINRES extraction")
	}
	joined := strings.ToLower(strings.Join(model.Result.Messages, " "))
	if !strings.Contains(joined, "minres") {
		t.Errorf("Expected messages to mention MINRES extraction, got %v", model.Result.Messages)
	}
}

func TestFactorAnalysisPromaxRotation(t *testing.T) {
	dt := insyra.NewDataTable()
	for i := 0; i < 6; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 40; j++ {
			value := float64(j)
			if i < 3 {
				value += float64(i) * 0.8
			} else {
				value += float64(j) * 0.4
			}
			col.Append(value)
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Rotation.Method = stats.FactorRotationPromax
	opt.Rotation.ForceOblique = true
	opt.Rotation.Kappa = 4

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected model for Promax rotation")
	}
	if model.Result.Phi == nil {
		t.Fatal("Expected Phi matrix for oblique rotation")
	}
	joined := strings.ToLower(strings.Join(model.Result.Messages, " "))
	if !strings.Contains(joined, "oblique rotation") {
		t.Errorf("Expected oblique rotation message, got %v", model.Result.Messages)
	}
}

func TestFactorAnalysisObliminOptionalOrthogonal(t *testing.T) {
	dt := insyra.NewDataTable()
	for i := 0; i < 5; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 50; j++ {
			col.Append(float64(j) + float64(i)*0.5)
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Rotation.Method = stats.FactorRotationOblimin
	opt.Rotation.ForceOblique = false

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected model for Oblimin rotation")
	}
	if model.Result.RotationMatrix == nil {
		t.Fatal("Expected rotation matrix for oblimin rotation")
	}
	if model.Result.Phi != nil {
		t.Error("Expected Phi to be nil when ForceOblique is false")
	}
}

func TestFactorAnalysisParallelAnalysisCount(t *testing.T) {
	dt := insyra.NewDataTable()
	for i := 0; i < 6; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 120; j++ {
			col.Append(float64(j) + float64(i)*0.3 + float64(j%4))
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountParallelAnalysis
	opt.Count.MaxFactors = 3
	opt.Count.ParallelReplications = 20
	opt.Count.ParallelPercentile = 0.95
	opt.Rotation.Method = stats.FactorRotationNone

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected model for parallel analysis")
	}
	if model.Result.CountUsed < 1 {
		t.Errorf("Expected at least one factor from parallel analysis, got %d", model.Result.CountUsed)
	}
	if model.Result.CountUsed > opt.Count.MaxFactors {
		t.Errorf("Parallel analysis exceeded MaxFactors (%d) with %d", opt.Count.MaxFactors, model.Result.CountUsed)
	}
	joined := strings.ToLower(strings.Join(model.Result.Messages, " "))
	if !strings.Contains(joined, "parallel analysis") {
		t.Errorf("Expected messages to mention parallel analysis, got %v", model.Result.Messages)
	}
}

// TestFactorScoring tests different factor scoring methods
func TestFactorScoring(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0))
	dt.AppendCols(insyra.NewDataList(2.0, 4.0, 6.0, 8.0, 10.0))
	dt.AppendCols(insyra.NewDataList(1.5, 3.0, 4.5, 6.0, 7.5))

	methods := []stats.FactorScoreMethod{
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			opt := stats.DefaultFactorAnalysisOptions()
			opt.Scoring = method
			opt.Count.Method = stats.FactorCountFixed
			opt.Count.FixedK = 1

			model := stats.FactorAnalysis(dt, opt)

			if model.Result.Scores == nil {
				t.Errorf("Expected non-nil scores for %s method", method)
			}

			// Check scores dimensions
			model.Result.Scores.AtomicDo(func(table *insyra.DataTable) {
				rows, cols := table.Size()
				if rows != 5 {
					t.Errorf("Expected 5 rows in scores, got %d", rows)
				}
				if cols != 1 {
					t.Errorf("Expected 1 column in scores, got %d", cols)
				}
			})
		})
	}
}

// TestFactorScoresDT tests computing scores for new data
func TestFactorScoresDT(t *testing.T) {
	// Training data
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0))
	dt.AppendCols(insyra.NewDataList(2.0, 4.0, 6.0, 8.0, 10.0))
	dt.AppendCols(insyra.NewDataList(1.5, 3.0, 4.5, 6.0, 7.5))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 1

	model := stats.FactorAnalysis(dt, opt)

	// New data
	newDt := insyra.NewDataTable()
	newDt.AppendCols(insyra.NewDataList(6.0, 7.0))
	newDt.AppendCols(insyra.NewDataList(12.0, 14.0))
	newDt.AppendCols(insyra.NewDataList(9.0, 10.5))

	scores, err := model.FactorScores(newDt, nil)
	if err != nil {
		t.Fatalf("FactorScoresDT failed: %v", err)
	}

	if scores == nil {
		t.Fatal("Expected non-nil scores")
	}

	// Check dimensions
	scores.AtomicDo(func(table *insyra.DataTable) {
		rows, cols := table.Size()
		if rows != 2 {
			t.Errorf("Expected 2 rows in scores, got %d", rows)
		}
		if cols != 1 {
			t.Errorf("Expected 1 column in scores, got %d", cols)
		}
	})
}

// TestScreeDataDT tests scree plot data generation
func TestScreeDataDT(t *testing.T) {
	dt := insyra.NewDataTable()
	for i := 0; i < 4; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 10; j++ {
			col.Append(float64(j) + float64(i)*5)
		}
		dt.AppendCols(col)
	}

	eigenDT, cumDT, err := stats.ScreePlotData(dt, true)
	if err != nil {
		t.Fatalf("ScreeDataDT failed: %v", err)
	}

	if eigenDT == nil {
		t.Fatal("Expected non-nil eigenvalue DataTable")
	}
	if cumDT == nil {
		t.Fatal("Expected non-nil cumulative DataTable")
	}

	// Check dimensions
	eigenDT.AtomicDo(func(table *insyra.DataTable) {
		rows, _ := table.Size()
		if rows != 4 {
			t.Errorf("Expected 4 eigenvalues, got %d", rows)
		}
	})

	cumDT.AtomicDo(func(table *insyra.DataTable) {
		rows, _ := table.Size()
		if rows != 4 {
			t.Errorf("Expected 4 cumulative values, got %d", rows)
		}

		// Check that cumulative proportions are monotonically increasing
		var prev = -1.0
		for i := 0; i < rows; i++ {
			row := table.GetRow(i)
			val, ok := row.Get(0).(float64)
			if ok {
				if val < prev {
					t.Errorf("Cumulative proportions not monotonically increasing at position %d", i)
				}
				if val < 0 || val > 1.01 {
					t.Errorf("Cumulative proportion at position %d out of range [0,1]: %f", i, val)
				}
				prev = val
			}
		}

		// Last cumulative proportion should be close to 1.0
		lastRow := table.GetRow(rows - 1)
		lastVal, ok := lastRow.Get(0).(float64)
		if ok && !approxEqual(lastVal, 1.0, 0.01) {
			t.Errorf("Last cumulative proportion should be ~1.0, got %f", lastVal)
		}
	})
}

// TestFactorAnalysisNilInput tests error handling for nil input
func TestFactorAnalysisNilInput(t *testing.T) {
	opt := stats.DefaultFactorAnalysisOptions()
	model := stats.FactorAnalysis(nil, opt)
	if model != nil {
		t.Error("Expected nil model for nil DataTable input")
	}
}

// TestFactorAnalysisEmptyData tests error handling for empty data
func TestFactorAnalysisEmptyData(t *testing.T) {
	dt := insyra.NewDataTable()
	opt := stats.DefaultFactorAnalysisOptions()
	model := stats.FactorAnalysis(dt, opt)
	if model != nil {
		t.Error("Expected nil model for empty DataTable")
	}
}

// TestFactorAnalysisSingleVariable tests with single variable
func TestFactorAnalysisSingleVariable(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 1

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}

	// Should extract exactly 1 factor
	if model.Result.CountUsed != 1 {
		t.Errorf("Expected 1 factor, got %d", model.Result.CountUsed)
	}
}

// TestDefaultOptions tests that default options are reasonable
func TestDefaultOptions(t *testing.T) {
	opt := stats.DefaultFactorAnalysisOptions()

	if opt.Preprocess.Standardize != true {
		t.Error("Default should standardize data")
	}

	if opt.Count.Method != stats.FactorCountKaiser {
		t.Error("Default should use Kaiser criterion")
	}

	if opt.Extraction != stats.FactorExtractionMINRES {
		t.Error("Default should use MINRES extraction")
	}

	if opt.Rotation.Method != stats.FactorRotationVarimax {
		t.Error("Default should use Varimax rotation")
	}

	if opt.Scoring != stats.FactorScoreRegression {
		t.Error("Default should use regression scoring")
	}

	if opt.MaxIter != 100 {
		t.Error("Default MaxIter should be 100")
	}

	if opt.Tol != 1e-6 {
		t.Error("Default tolerance should be 1e-6")
	}
}

// TestFactorAnalysisWithStandardizedData tests that results are reasonable
func TestFactorAnalysisWithStandardizedData(t *testing.T) {
	// Create correlated data
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList(10.0, 20.0, 30.0, 40.0, 50.0))
	dt.AppendCols(insyra.NewDataList(15.0, 30.0, 45.0, 60.0, 75.0))  // Highly correlated with first
	dt.AppendCols(insyra.NewDataList(100.0, 90.0, 80.0, 70.0, 60.0)) // Different scale and pattern

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Preprocess.Standardize = true
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2

	model := stats.FactorAnalysis(dt, opt)

	// Sum of communalities should be reasonable (between 0 and number of variables)
	var sumComm float64
	if model.Result.Communalities != nil {
		model.Result.Communalities.AtomicDo(func(table *insyra.DataTable) {
			rows, _ := table.Size()
			for i := 0; i < rows; i++ {
				row := table.GetRow(i)
				val, ok := row.Get(0).(float64)
				if ok {
					sumComm += val
				}
			}
		})

		if sumComm < 0 || sumComm > 3.1 {
			t.Errorf("Sum of communalities out of reasonable range: %f", sumComm)
		}
	}

	// Explained proportions should sum to something reasonable
	var sumExplained float64
	if model.Result.ExplainedProportion != nil {
		model.Result.ExplainedProportion.AtomicDo(func(table *insyra.DataTable) {
			rows, _ := table.Size()
			for i := 0; i < rows; i++ {
				row := table.GetRow(i)
				val, ok := row.Get(0).(float64)
				if ok {
					sumExplained += val
				}
			}
		})

		if sumExplained < 0 || sumExplained > 1.01 {
			t.Errorf("Sum of explained proportions out of range [0,1]: %f", sumExplained)
		}
	}
}
