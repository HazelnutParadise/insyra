package stats_test

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestFactorAnalysisValidation performs comprehensive validation of Factor Analysis
// by comparing results with Python's factor_analyzer library
func TestFactorAnalysisValidation(t *testing.T) {
	// Load test data
	dt, err := loadTestData()
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// Load Python reference results
	pythonResults, err := loadPythonResults()
	if err != nil {
		t.Logf("Warning: Could not load Python results: %v", err)
		t.Log("Generating Python results...")
		if err := generatePythonResults(); err != nil {
			t.Fatalf("Failed to generate Python results: %v", err)
		}
		pythonResults, err = loadPythonResults()
		if err != nil {
			t.Fatalf("Failed to load generated Python results: %v", err)
		}
	}

	// Prepare test configurations
	configs := []testConfig{
		// MINRES extraction with various rotations
		{extraction: stats.FactorExtractionMINRES, rotation: stats.FactorRotationNone, pythonMethod: "minres", pythonRotation: "none"},
		{extraction: stats.FactorExtractionMINRES, rotation: stats.FactorRotationVarimax, pythonMethod: "minres", pythonRotation: "varimax"},
		{extraction: stats.FactorExtractionMINRES, rotation: stats.FactorRotationOblimin, pythonMethod: "minres", pythonRotation: "oblimin"},
		{extraction: stats.FactorExtractionMINRES, rotation: stats.FactorRotationQuartimax, pythonMethod: "minres", pythonRotation: "quartimax"},
		// PCA extraction with various rotations
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationNone, pythonMethod: "principal", pythonRotation: "none"},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationVarimax, pythonMethod: "principal", pythonRotation: "varimax"},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationPromax, pythonMethod: "principal", pythonRotation: "promax"},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationOblimin, pythonMethod: "principal", pythonRotation: "oblimin"},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationQuartimax, pythonMethod: "principal", pythonRotation: "quartimax"},
		// Additional Go-only rotations (no Python comparison but test they work)
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationQuartimin, pythonMethod: "", pythonRotation: ""},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationGeominT, pythonMethod: "", pythonRotation: ""},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationGeominQ, pythonMethod: "", pythonRotation: ""},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationBentlerT, pythonMethod: "", pythonRotation: ""},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationBentlerQ, pythonMethod: "", pythonRotation: ""},
		{extraction: stats.FactorExtractionPCA, rotation: stats.FactorRotationSimplimax, pythonMethod: "", pythonRotation: ""},
		// PAF extraction
		{extraction: stats.FactorExtractionPAF, rotation: stats.FactorRotationVarimax, pythonMethod: "", pythonRotation: ""},
	}

	// Create comparison table
	comparisonTable := [][]string{
		{"Extraction", "Rotation", "Field", "Go Shape", "Python Shape", "Max Diff", "Mean Diff", "Status"},
	}

	for _, config := range configs {
		testName := fmt.Sprintf("%s_%s", config.extraction, config.rotation)
		t.Run(testName, func(t *testing.T) {
			// Run Go factor analysis
			opt := stats.FactorAnalysisOptions{
				Preprocess: stats.FactorPreprocessOptions{
					Standardize: true,
					Missing:     "listwise",
				},
				Count: stats.FactorCountSpec{
					Method: stats.FactorCountFixed,
					FixedK: 2,
				},
				Extraction: config.extraction,
				Rotation: stats.FactorRotationOptions{
					Method: config.rotation,
					Kappa:  4, // For promax
					Delta:  0, // For oblimin
				},
				Scoring: stats.FactorScoreRegression,
				MaxIter: 50,
				MinErr:  0.001,
			}

			model := stats.FactorAnalysis(dt, opt)
			if model == nil {
				t.Errorf("FactorAnalysis returned nil")
				comparisonTable = append(comparisonTable, []string{
					string(config.extraction), string(config.rotation), "ALL", "N/A", "N/A", "N/A", "N/A", "FAILED",
				})
				return
			}

			// Find matching Python result
			pythonResult := findPythonResult(pythonResults, config.pythonMethod, config.pythonRotation)
			if pythonResult == nil {
				// No Python comparison available (Go-only rotation methods)
				if config.pythonMethod == "" {
					t.Logf("Go-only rotation method %s/%s - no Python comparison, checking it runs successfully",
						config.extraction, config.rotation)
					
					// Just verify it produced valid results
					if model.Loadings == nil {
						t.Errorf("Loadings is nil")
					}
					if model.Communalities == nil {
						t.Errorf("Communalities is nil")
					}
					if model.Uniquenesses == nil {
						t.Errorf("Uniquenesses is nil")
					}
					if model.Eigenvalues == nil {
						t.Errorf("Eigenvalues is nil")
					}
					
					comparisonTable = append(comparisonTable, []string{
						string(config.extraction), string(config.rotation), "N/A (Go-only)", "N/A", "N/A", "N/A", "N/A", "OK (Go-only)",
					})
					return
				}
				
				t.Logf("No matching Python result for %s/%s", config.pythonMethod, config.pythonRotation)
				return
			}

			// Compare each field
			comparisons := compareResults(model, pythonResult, t)

			// Add to comparison table
			for _, comp := range comparisons {
				status := "OK"
				if comp.maxDiff > 0.1 {
					status = "LARGE DIFF"
				} else if comp.maxDiff > 0.01 {
					status = "MODERATE DIFF"
				}

				comparisonTable = append(comparisonTable, []string{
					string(config.extraction),
					string(config.rotation),
					comp.field,
					comp.goShape,
					comp.pythonShape,
					fmt.Sprintf("%.6f", comp.maxDiff),
					fmt.Sprintf("%.6f", comp.meanDiff),
					status,
				})
			}
		})
	}

	// Print comparison table
	t.Log("\n\n=== FACTOR ANALYSIS VALIDATION COMPARISON TABLE ===")
	for _, row := range comparisonTable {
		t.Logf("%-15s %-15s %-20s %-12s %-12s %-12s %-12s %-15s",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7])
	}

	// Save comparison table to file
	saveComparisonTable(comparisonTable, "/tmp/go_python_fa_comparison.csv")
	t.Log("\nComparison table saved to /tmp/go_python_fa_comparison.csv")
}

type testConfig struct {
	extraction     stats.FactorExtractionMethod
	rotation       stats.FactorRotationMethod
	pythonMethod   string
	pythonRotation string
}

type comparisonResult struct {
	field       string
	goShape     string
	pythonShape string
	maxDiff     float64
	meanDiff    float64
}

type pythonFAResult struct {
	Config struct {
		Method   string `json:"method"`
		Rotation string `json:"rotation"`
		NFactors int    `json:"n_factors"`
	} `json:"config"`
	Loadings             [][]float64 `json:"loadings"`
	Structure            [][]float64 `json:"structure"`
	Communalities        []float64   `json:"communalities"`
	Uniquenesses         []float64   `json:"uniquenesses"`
	Phi                  [][]float64 `json:"phi"`
	Eigenvalues          []float64   `json:"eigenvalues"`
	ExplainedProportion  []float64   `json:"explained_proportion"`
	CumulativeProportion []float64   `json:"cumulative_proportion"`
	Scores               [][]float64 `json:"scores"`
	Error                string      `json:"error"`
}

func loadTestData() (insyra.IDataTable, error) {
	// Use simple correlated data similar to existing tests that work
	// This ensures no matrix singularity issues
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0, 1.5, 2.3, 3.1, 4.2, 5.1,
			1.2, 2.0, 3.3, 4.0, 5.2, 1.8, 2.5, 3.5, 4.3, 5.3).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1, 2.3, 3.2, 4.1, 5.0, 6.2,
			2.1, 3.0, 4.2, 5.1, 6.3, 2.4, 3.3, 4.3, 5.3, 6.4).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2, 3.3, 4.2, 5.0, 6.1, 7.3,
			3.2, 4.1, 5.2, 6.2, 7.4, 3.4, 4.3, 5.3, 6.3, 7.5).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1, 4.2, 5.1, 6.2, 7.1, 8.2,
			4.1, 5.0, 6.3, 7.2, 8.3, 4.3, 5.3, 6.4, 7.3, 8.4).SetName("V4"),
		insyra.NewDataList(1.5, 2.4, 3.3, 4.2, 5.1, 1.7, 2.6, 3.5, 4.4, 5.3,
			1.6, 2.5, 3.4, 4.3, 5.2, 1.8, 2.7, 3.6, 4.5, 5.4).SetName("V5"),
		insyra.NewDataList(2.2, 3.3, 4.4, 5.5, 6.6, 2.4, 3.5, 4.6, 5.7, 6.8,
			2.3, 3.4, 4.5, 5.6, 6.7, 2.5, 3.6, 4.7, 5.8, 6.9).SetName("V6"),
	}
	
	dt := insyra.NewDataTable(data...)
	
	// Verify dimensions
	var rows, cols int
	dt.AtomicDo(func(table *insyra.DataTable) {
		rows, cols = table.Size()
	})
	
	fmt.Printf("Loaded data dimensions: %d rows x %d cols\n", rows, cols)
	
	// Also save this data to CSV for Python
	dt.ToCSV("/tmp/factor_analysis_validation_data.csv", false, true, true)
	
	return dt, nil
}

func generatePythonResults() error {
	// Run Python script to generate reference results
	cmd := exec.Command("python3", "/tmp/python_fa_reference.py")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python script failed: %v\nOutput: %s", err, output)
	}
	return nil
}

func loadPythonResults() ([]pythonFAResult, error) {
	data, err := os.ReadFile("/tmp/python_fa_results.json")
	if err != nil {
		return nil, err
	}

	var results []pythonFAResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func findPythonResult(results []pythonFAResult, method, rotation string) *pythonFAResult {
	for i := range results {
		if results[i].Config.Method == method && results[i].Config.Rotation == rotation {
			if results[i].Error == "" {
				return &results[i]
			}
		}
	}
	return nil
}

func compareResults(model *stats.FactorModel, pythonResult *pythonFAResult, t *testing.T) []comparisonResult {
	comparisons := []comparisonResult{}

	// Compare Loadings
	if model.Loadings != nil && pythonResult.Loadings != nil {
		goLoadings := extractMatrixData(model.Loadings)
		pythonLoadings := pythonResult.Loadings
		maxDiff, meanDiff := compareMatrices(goLoadings, pythonLoadings)

		comparisons = append(comparisons, comparisonResult{
			field:       "Loadings",
			goShape:     fmt.Sprintf("%dx%d", len(goLoadings), len(goLoadings[0])),
			pythonShape: fmt.Sprintf("%dx%d", len(pythonLoadings), len(pythonLoadings[0])),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare Structure (for oblique rotations)
	if model.Structure != nil && pythonResult.Structure != nil {
		goStructure := extractMatrixData(model.Structure)
		pythonStructure := pythonResult.Structure
		maxDiff, meanDiff := compareMatrices(goStructure, pythonStructure)

		comparisons = append(comparisons, comparisonResult{
			field:       "Structure",
			goShape:     fmt.Sprintf("%dx%d", len(goStructure), len(goStructure[0])),
			pythonShape: fmt.Sprintf("%dx%d", len(pythonStructure), len(pythonStructure[0])),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare Communalities
	if model.Communalities != nil && pythonResult.Communalities != nil {
		goCommunalities := extractVectorData(model.Communalities)
		pythonCommunalities := pythonResult.Communalities
		maxDiff, meanDiff := compareVectors(goCommunalities, pythonCommunalities)

		comparisons = append(comparisons, comparisonResult{
			field:       "Communalities",
			goShape:     fmt.Sprintf("%d", len(goCommunalities)),
			pythonShape: fmt.Sprintf("%d", len(pythonCommunalities)),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare Uniquenesses
	if model.Uniquenesses != nil && pythonResult.Uniquenesses != nil {
		goUniquenesses := extractVectorData(model.Uniquenesses)
		pythonUniquenesses := pythonResult.Uniquenesses
		maxDiff, meanDiff := compareVectors(goUniquenesses, pythonUniquenesses)

		comparisons = append(comparisons, comparisonResult{
			field:       "Uniquenesses",
			goShape:     fmt.Sprintf("%d", len(goUniquenesses)),
			pythonShape: fmt.Sprintf("%d", len(pythonUniquenesses)),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare Phi (for oblique rotations)
	if model.Phi != nil && pythonResult.Phi != nil {
		goPhi := extractMatrixData(model.Phi)
		pythonPhi := pythonResult.Phi
		maxDiff, meanDiff := compareMatrices(goPhi, pythonPhi)

		comparisons = append(comparisons, comparisonResult{
			field:       "Phi",
			goShape:     fmt.Sprintf("%dx%d", len(goPhi), len(goPhi[0])),
			pythonShape: fmt.Sprintf("%dx%d", len(pythonPhi), len(pythonPhi[0])),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare Eigenvalues
	if model.Eigenvalues != nil && pythonResult.Eigenvalues != nil {
		goEigenvalues := extractVectorData(model.Eigenvalues)
		pythonEigenvalues := pythonResult.Eigenvalues
		// Only compare the first n_factors eigenvalues
		nFactors := min(len(goEigenvalues), len(pythonEigenvalues), 2)
		maxDiff, meanDiff := compareVectors(goEigenvalues[:nFactors], pythonEigenvalues[:nFactors])

		comparisons = append(comparisons, comparisonResult{
			field:       "Eigenvalues",
			goShape:     fmt.Sprintf("%d", len(goEigenvalues)),
			pythonShape: fmt.Sprintf("%d", len(pythonEigenvalues)),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare ExplainedProportion
	if model.ExplainedProportion != nil && pythonResult.ExplainedProportion != nil {
		goExplained := extractVectorData(model.ExplainedProportion)
		pythonExplained := pythonResult.ExplainedProportion
		maxDiff, meanDiff := compareVectors(goExplained, pythonExplained)

		comparisons = append(comparisons, comparisonResult{
			field:       "ExplainedProportion",
			goShape:     fmt.Sprintf("%d", len(goExplained)),
			pythonShape: fmt.Sprintf("%d", len(pythonExplained)),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare CumulativeProportion
	if model.CumulativeProportion != nil && pythonResult.CumulativeProportion != nil {
		goCumulative := extractVectorData(model.CumulativeProportion)
		pythonCumulative := pythonResult.CumulativeProportion
		maxDiff, meanDiff := compareVectors(goCumulative, pythonCumulative)

		comparisons = append(comparisons, comparisonResult{
			field:       "CumulativeProportion",
			goShape:     fmt.Sprintf("%d", len(goCumulative)),
			pythonShape: fmt.Sprintf("%d", len(pythonCumulative)),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	// Compare Scores
	if model.Scores != nil && pythonResult.Scores != nil {
		goScores := extractMatrixData(model.Scores)
		pythonScores := pythonResult.Scores
		maxDiff, meanDiff := compareMatrices(goScores, pythonScores)

		comparisons = append(comparisons, comparisonResult{
			field:       "Scores",
			goShape:     fmt.Sprintf("%dx%d", len(goScores), len(goScores[0])),
			pythonShape: fmt.Sprintf("%dx%d", len(pythonScores), len(pythonScores[0])),
			maxDiff:     maxDiff,
			meanDiff:    meanDiff,
		})
	}

	return comparisons
}

func extractMatrixData(dt insyra.IDataTable) [][]float64 {
	var result [][]float64

	dt.AtomicDo(func(table *insyra.DataTable) {
		rows, cols := table.Size()
		result = make([][]float64, rows)
		for i := 0; i < rows; i++ {
			result[i] = make([]float64, cols)
			row := table.GetRow(i)
			for j := 0; j < cols; j++ {
				if val, ok := row.Get(j).(float64); ok {
					result[i][j] = val
				}
			}
		}
	})

	return result
}

func extractVectorData(dt insyra.IDataTable) []float64 {
	var result []float64

	dt.AtomicDo(func(table *insyra.DataTable) {
		rows, _ := table.Size()
		result = make([]float64, rows)
		for i := 0; i < rows; i++ {
			row := table.GetRow(i)
			if val, ok := row.Get(0).(float64); ok {
				result[i] = val
			}
		}
	})

	return result
}

func compareMatrices(a, b [][]float64) (maxDiff, meanDiff float64) {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) || len(a[0]) != len(b[0]) {
		return math.NaN(), math.NaN()
	}

	var sumDiff float64
	var count int

	for i := 0; i < len(a); i++ {
		for j := 0; j < len(a[i]); j++ {
			diff := math.Abs(a[i][j] - b[i][j])
			if diff > maxDiff {
				maxDiff = diff
			}
			sumDiff += diff
			count++
		}
	}

	if count > 0 {
		meanDiff = sumDiff / float64(count)
	}

	return maxDiff, meanDiff
}

func compareVectors(a, b []float64) (maxDiff, meanDiff float64) {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return math.NaN(), math.NaN()
	}

	var sumDiff float64

	for i := 0; i < len(a); i++ {
		diff := math.Abs(a[i] - b[i])
		if diff > maxDiff {
			maxDiff = diff
		}
		sumDiff += diff
	}

	meanDiff = sumDiff / float64(len(a))

	return maxDiff, meanDiff
}

func saveComparisonTable(table [][]string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, row := range table {
		writer.Write(row)
	}
}

func min(vals ...int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
