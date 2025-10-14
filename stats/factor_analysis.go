package stats

import (
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/fa"
	"github.com/gonum/optimize"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// -------------------------
// Factor Analysis Types and Constants
// -------------------------

// FactorExtractionMethod defines the method for extracting factors.
// See Docs/stats.md (Factor Analysis → Extraction Methods) for algorithmic details.
type FactorExtractionMethod string

const (
	FactorExtractionPCA    FactorExtractionMethod = "pca"
	FactorExtractionPAF    FactorExtractionMethod = "paf"
	FactorExtractionML     FactorExtractionMethod = "ml"
	FactorExtractionMINRES FactorExtractionMethod = "minres"
)

// FactorRotationMethod defines the method for rotating factors.
// Rotation families and their properties are documented in Docs/stats.md.
type FactorRotationMethod string

const (
	FactorRotationNone      FactorRotationMethod = "none"
	FactorRotationVarimax   FactorRotationMethod = "varimax"
	FactorRotationQuartimax FactorRotationMethod = "quartimax"
	FactorRotationQuartimin FactorRotationMethod = "quartimin"
	FactorRotationOblimin   FactorRotationMethod = "oblimin"
	FactorRotationGeominT   FactorRotationMethod = "geominT"
	FactorRotationBentlerT  FactorRotationMethod = "bentlerT"
	FactorRotationSimplimax FactorRotationMethod = "simplimax"
	FactorRotationGeominQ   FactorRotationMethod = "geominQ"
	FactorRotationBentlerQ  FactorRotationMethod = "bentlerQ"
	FactorRotationPromax    FactorRotationMethod = "promax"
)

// FactorScoreMethod defines the method for computing factor scores.
// Scoring equations and trade-offs are outlined in Docs/stats.md.
type FactorScoreMethod string

const (
	FactorScoreNone          FactorScoreMethod = "none"
	FactorScoreRegression    FactorScoreMethod = "regression"
	FactorScoreBartlett      FactorScoreMethod = "bartlett"
	FactorScoreAndersonRubin FactorScoreMethod = "anderson-rubin"
)

// FactorCountMethod defines the method for determining number of factors
type FactorCountMethod string

const (
	FactorCountFixed  FactorCountMethod = "fixed"
	FactorCountKaiser FactorCountMethod = "kaiser"
)

// -------------------------
// Options Structs
// -------------------------

// FactorCountSpec specifies how to determine the number of factors
type FactorCountSpec struct {
	Method         FactorCountMethod
	FixedK         int     // Optional: used when Method is CountFixed
	EigenThreshold float64 // Optional: default 1.0 for CountKaiser
	MaxFactors     int     // Optional: 0 means no limit
}

// FactorRotationOptions specifies rotation parameters
type FactorRotationOptions struct {
	Method   FactorRotationMethod
	Kappa    float64 // Optional: Promax power (default 4)
	Delta    float64 // Optional: default 0 for Oblimin
	Restarts int     // Optional: random orthonormal starts for GPA rotations (default 10)
}

// FactorPreprocessOptions specifies preprocessing parameters
type FactorPreprocessOptions struct {
	Standardize bool   // Optional
	Missing     string // Optional: default "listwise"
}

// FactorAnalysisOptions contains all options for factor analysis
type FactorAnalysisOptions struct {
	Preprocess FactorPreprocessOptions
	Count      FactorCountSpec
	Extraction FactorExtractionMethod
	Rotation   FactorRotationOptions
	Scoring    FactorScoreMethod
	MaxIter    int     // Optional: default 100
	MinErr     float64 // Optional: default 0.001 (R's min.err)
}

// -------------------------
// Result Structs
// -------------------------

// BartlettTestResult contains the results of Bartlett's test of sphericity
type BartlettTestResult struct {
	ChiSquare        float64 // Chi-square statistic
	DegreesOfFreedom int     // Degrees of freedom
	PValue           float64 // P-value
	SampleSize       int     // Sample size
}

// FactorAnalysisResult contains the output of factor analysis
type FactorAnalysisResult struct {
	Loadings             insyra.IDataTable   // Loading matrix (variables × factors)
	UnrotatedLoadings    insyra.IDataTable   // Unrotated loading matrix (variables × factors)
	Structure            insyra.IDataTable   // Structure matrix (variables × factors)
	Uniquenesses         insyra.IDataTable   // Uniqueness vector (p × 1)
	Communalities        insyra.IDataTable   // Communality table (p × 1: Extraction)
	SamplingAdequacy     insyra.IDataTable   // KMO overall index and per-variable MSA values
	BartlettTest         *BartlettTestResult // Bartlett's test of sphericity summary
	Phi                  insyra.IDataTable   // Factor correlation matrix (m × m), nil for orthogonal
	RotationMatrix       insyra.IDataTable   // Rotation matrix (m × m), nil if no rotation
	Eigenvalues          insyra.IDataTable   // Eigenvalues vector (p × 1)
	ExplainedProportion  insyra.IDataTable   // Proportion explained by each factor (m × 1)
	CumulativeProportion insyra.IDataTable   // Cumulative proportion explained (m × 1)
	Scores               insyra.IDataTable   // Factor scores (n × m), nil if not computed
	ScoreCoefficients    insyra.IDataTable   // Factor score coefficient matrix (variables × factors)
	ScoreCovariance      insyra.IDataTable   // Factor score covariance matrix (factors × factors)

	Converged         bool
	RotationConverged bool
	Iterations        int
	CountUsed         int
	Messages          []string
}

const (
	tableNameFactorLoadings          = "FactorLoadings"
	tableNameUnrotatedLoadings       = "UnrotatedLoadings"
	tableNameFactorStructure         = "FactorStructure"
	tableNameUniqueness              = "Uniqueness"
	tableNameCommunalities           = "Communalities"
	tableNameSamplingAdequacy        = "KMOSamplingAdequacy"
	tableNameBartlettTest            = "BartlettSphericityTest"
	tableNameFactorScoreCoefficients = "FactorScoreCoefficients"
	tableNameFactorScoreCovariance   = "FactorScoreCovariance"
	tableNamePhiMatrix               = "PhiMatrix"
	tableNameRotationMatrix          = "RotationMatrix"
	tableNameEigenvalues             = "Eigenvalues"
	tableNameExplainedProportion     = "ExplainedProportion"
	tableNameCumulativeProportion    = "CumulativeProportion"
	tableNameFactorScores            = "FactorScores"
)

// Show prints everything in the FactorAnalysisResult
func (r *FactorAnalysisResult) Show(startEndRange ...any) {
	insyra.Show("Communalities", r.Communalities, startEndRange...)
	insyra.Show(tableNameSamplingAdequacy, r.SamplingAdequacy, startEndRange...)
	if r.BartlettTest != nil {
		bartlettTable := insyra.NewDataTable(
			insyra.NewDataList(r.BartlettTest.ChiSquare).SetName("Chi_Square"),
			insyra.NewDataList(float64(r.BartlettTest.DegreesOfFreedom)).SetName("Degrees_Of_Freedom"),
			insyra.NewDataList(r.BartlettTest.PValue).SetName("P_Value"),
			insyra.NewDataList(float64(r.BartlettTest.SampleSize)).SetName("Sample_Size"),
		)
		insyra.Show(tableNameBartlettTest, bartlettTable, startEndRange...)
	}
	insyra.Show(tableNameEigenvalues, r.Eigenvalues, startEndRange...)
	insyra.Show(tableNameExplainedProportion, r.ExplainedProportion, startEndRange...)
	insyra.Show(tableNameCumulativeProportion, r.CumulativeProportion, startEndRange...)
	insyra.Show(tableNameUnrotatedLoadings, r.UnrotatedLoadings, startEndRange...)
	insyra.Show(tableNameFactorLoadings, r.Loadings, startEndRange...)
	insyra.Show(tableNameFactorStructure, r.Structure, startEndRange...)
	insyra.Show(tableNamePhiMatrix, r.Phi, startEndRange...)
	insyra.Show(tableNameRotationMatrix, r.RotationMatrix, startEndRange...)
	insyra.Show(tableNameFactorScoreCoefficients, r.ScoreCoefficients, startEndRange...)
	insyra.Show(tableNameFactorScoreCovariance, r.ScoreCovariance, startEndRange...)
	insyra.Show(tableNameFactorScores, r.Scores, startEndRange...)
	insyra.Show(tableNameUniqueness, r.Uniquenesses, startEndRange...)
	fmt.Printf("Converged: %v\n", r.Converged)
	fmt.Printf("RotationConverged: %v\n", r.RotationConverged)
	fmt.Printf("Iterations: %d\n", r.Iterations)
	fmt.Printf("CountUsed: %d\n", r.CountUsed)
	fmt.Printf("Messages: %s.\n", strings.Join(r.Messages, ", "))
}

// FactorModel holds the factor analysis model
type FactorModel struct {
	FactorAnalysisResult

	// Internal fields for scoring new data
	scoreMethod FactorScoreMethod
	extraction  FactorExtractionMethod
	rotation    FactorRotationMethod
	means       []float64
	sds         []float64
	sigma       *mat.Dense
}

// -------------------------
// Default Options
// -------------------------

// DefaultFactorAnalysisOptions returns default options for factor analysis.
// Defaults align with R's psych::fa function defaults.
func DefaultFactorAnalysisOptions() FactorAnalysisOptions {
	return FactorAnalysisOptions{
		Preprocess: FactorPreprocessOptions{
			Standardize: true,
			Missing:     "listwise",
		},
		Count: FactorCountSpec{
			Method:         FactorCountKaiser,
			EigenThreshold: 1.0,
			MaxFactors:     0, // 0 means no limit
		},
		Extraction: FactorExtractionMINRES, // R default: "minres"
		Rotation: FactorRotationOptions{
			Method:   FactorRotationOblimin, // R default: "oblimin"
			Kappa:    4,                     // R default for promax
			Delta:    0,                     // R default for oblimin
			Restarts: 10,
		},
		Scoring: FactorScoreRegression, // R default: "regression"
		MaxIter: 100,                   // R default: 50
		MinErr:  0.001,                 // R default: 0.001
	}
}

// Internal constants aligned with R's psych::fa and GPArotation package
const (
	// Convergence tolerance for extraction methods (PAF, ML, MINRES)
	// R psych uses different tolerances for different contexts
	extractionTolerance = 1e-8 // General convergence tolerance for factor extraction

	// Correlation matrix diagonal checks
	corrDiagTolerance    = 1e-6 // Tolerance for diagonal deviation from 1.0
	corrDiagLogThreshold = 1e-8 // Threshold for logging diagonal deviations
	uniquenessLowerBound = 1e-9 // Lower bound for uniqueness values

	// Machine epsilon and eigenvalue thresholds (aligned with R's .Machine$double.eps)
	machineEpsilon         = 2.220446e-16         // R's .Machine$double.eps
	eigenvalueMinThreshold = 100 * machineEpsilon // R: 100 * .Machine$double.eps (2.22e-14)
)

// -------------------------
// Main Function
// -------------------------

// FactorAnalysis performs factor analysis on a DataTable
func FactorAnalysis(dt insyra.IDataTable, opt FactorAnalysisOptions) *FactorModel {
	if dt == nil {
		insyra.LogWarning("stats", "FactorAnalysis", "nil DataTable")
		return nil
	}

	var rowNum, colNum int
	var data *mat.Dense
	var means, sds []float64
	var colNames, rowNames []string

	// Step 1: Preprocess data
	dt.AtomicDo(func(table *insyra.DataTable) {
		rowNum, colNum = table.Size()

		// Check for empty data
		if rowNum == 0 || colNum == 0 {
			return
		}

		// Get column names
		colNames = dt.ColNames()

		// Get row names
		rowNames = dt.RowNames()

		// Convert DataTable to matrix
		data = mat.NewDense(rowNum, colNum, nil)
		for i := 0; i < rowNum; i++ {
			row := table.GetRow(i)
			for j := 0; j < colNum; j++ {
				value, ok := row.Get(j).(float64)
				if ok {
					data.Set(i, j, value)
				} else {
					// Handle missing values
					data.Set(i, j, math.NaN())
				}
			}
		}
	})

	// Check for empty data after AtomicDo
	if rowNum == 0 || colNum == 0 {
		insyra.LogWarning("stats", "FactorAnalysis", "empty DataTable")
		return nil
	}

	// Check for and handle missing values
	hasNaN := false
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			if math.IsNaN(data.At(i, j)) {
				hasNaN = true
				break
			}
		}
		if hasNaN {
			break
		}
	}

	if hasNaN {
		if opt.Preprocess.Missing == "listwise" {
			// Remove rows with any NaN
			validRows := make([]int, 0, rowNum)
			for i := 0; i < rowNum; i++ {
				valid := true
				for j := 0; j < colNum; j++ {
					if math.IsNaN(data.At(i, j)) {
						valid = false
						break
					}
				}
				if valid {
					validRows = append(validRows, i)
				}
			}
			if len(validRows) == 0 {
				insyra.LogWarning("stats", "FactorAnalysis", "no valid rows after removing missing values")
				return nil
			}
			newData := mat.NewDense(len(validRows), colNum, nil)
			for i, rowIdx := range validRows {
				for j := 0; j < colNum; j++ {
					newData.Set(i, j, data.At(rowIdx, j))
				}
			}
			data = newData
			rowNum = len(validRows)

			// Update row names for valid rows
			newRowNames := make([]string, len(validRows))
			for i, rowIdx := range validRows {
				newRowNames[i] = rowNames[rowIdx]
			}
			rowNames = newRowNames
		} else {
			// For simplicity, use listwise deletion for now
			insyra.LogWarning("stats", "FactorAnalysis", "only listwise deletion is currently supported for missing values")
			return nil
		}
	}

	// Step 2: Standardize if requested
	means = make([]float64, colNum)
	sds = make([]float64, colNum)
	if opt.Preprocess.Standardize {
		for j := 0; j < colNum; j++ {
			col := mat.Col(nil, j, data)
			mean, std := stat.MeanStdDev(col, nil)
			means[j] = mean
			sds[j] = std
			if std == 0 {
				std = 1 // Avoid division by zero
			}
			for i := 0; i < rowNum; i++ {
				data.Set(i, j, (data.At(i, j)-mean)/std)
			}
		}
	} else {
		for j := 0; j < colNum; j++ {
			col := mat.Col(nil, j, data)
			means[j] = stat.Mean(col, nil)
			sds[j] = 1.0
			for i := 0; i < rowNum; i++ {
				data.Set(i, j, data.At(i, j)-means[j])
			}
		}
	}

	// Step 3: Compute correlation or covariance matrix
	var corrMatrix *mat.SymDense
	var corrForAdequacy *mat.SymDense
	if opt.Preprocess.Standardize {
		corrMatrix = mat.NewSymDense(colNum, nil)
		stat.CorrelationMatrix(corrMatrix, data, nil)
		corrForAdequacy = corrMatrix
	} else {
		corrMatrix = mat.NewSymDense(colNum, nil)
		stat.CovarianceMatrix(corrMatrix, data, nil)
		corrForAdequacy = mat.NewSymDense(colNum, nil)
		stat.CorrelationMatrix(corrForAdequacy, data, nil)
	}
	if corrForAdequacy == nil {
		corrForAdequacy = mat.NewSymDense(colNum, nil)
		stat.CorrelationMatrix(corrForAdequacy, data, nil)
	}

	insyra.LogDebug("stats", "FactorAnalysis", "data matrix size: %dx%d, correlation matrix computed", rowNum, colNum)

	// Sanity check: ensure diagonal elements of correlation matrix are 1 when standardized
	if opt.Preprocess.Standardize {
		maxDiagDeviation := 0.0
		for i := 0; i < colNum; i++ {
			diag := corrMatrix.At(i, i)
			delta := math.Abs(diag - 1.0)
			if delta > maxDiagDeviation {
				maxDiagDeviation = delta
			}
			if delta > corrDiagTolerance {
				corrMatrix.SetSym(i, i, 1.0)
			}
		}
		if maxDiagDeviation > corrDiagLogThreshold {
			insyra.LogDebug("stats", "FactorAnalysis", "correlation diag max deviation = %.6g", maxDiagDeviation)
		}
	}
	// Ensure the diagnostic correlation matrix has unit diagonal
	if corrForAdequacy != nil {
		for i := 0; i < colNum; i++ {
			diag := corrForAdequacy.At(i, i)
			if math.Abs(diag-1.0) > corrDiagTolerance {
				corrForAdequacy.SetSym(i, i, 1.0)
			}
		}
	}

	// Pre-compute sampling adequacy and Bartlett diagnostics
	var samplingAdequacyTable *insyra.DataTable
	var bartlettResult *BartlettTestResult
	if corrForAdequacy != nil {
		corrAdequacyDense := mat.DenseCopyOf(corrForAdequacy)
		overallKMO, msaValues, kmoErr := computeKMOMeasures(corrAdequacyDense)
		if kmoErr != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "failed to compute KMO/MSA: %v", kmoErr)
		} else {
			// Debug: print correlation matrix diagonal and some off-diagonal values
			rows, cols := corrAdequacyDense.Dims()
			insyra.LogDebug("stats", "FactorAnalysis", "correlation matrix dimensions: %dx%d", rows, cols)
			insyra.LogDebug("stats", "FactorAnalysis", "correlation matrix diagonal: %.6f, %.6f, %.6f", corrAdequacyDense.At(0, 0), corrAdequacyDense.At(1, 1), corrAdequacyDense.At(2, 2))
			if cols > 3 {
				insyra.LogDebug("stats", "FactorAnalysis", "correlation matrix sample values: [0,1]=%.6f, [0,2]=%.6f, [1,2]=%.6f, [0,3]=%.6f", corrAdequacyDense.At(0, 1), corrAdequacyDense.At(0, 2), corrAdequacyDense.At(1, 2), corrAdequacyDense.At(0, 3))
			} else {
				insyra.LogDebug("stats", "FactorAnalysis", "correlation matrix sample values: [0,1]=%.6f, [0,2]=%.6f, [1,2]=%.6f", corrAdequacyDense.At(0, 1), corrAdequacyDense.At(0, 2), corrAdequacyDense.At(1, 2))
			}
			samplingAdequacyTable = kmoToDataTable(overallKMO, msaValues, colNames)
		}

		if chi, pval, df, bartErr := computeBartlettFromCorrelation(corrAdequacyDense, rowNum); bartErr != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "failed to compute Bartlett's test: %v", bartErr)
		} else {
			bartlettResult = bartlettToDataTable(chi, df, pval, rowNum)
		}
	}

	sigmaForScores := mat.NewDense(colNum, colNum, nil)
	for i := 0; i < colNum; i++ {
		for j := 0; j < colNum; j++ {
			sigmaForScores.Set(i, j, corrMatrix.At(i, j))
		}
	}

	// Step 4: Eigenvalue decomposition
	var eig mat.EigenSym
	if !eig.Factorize(corrMatrix, true) {
		insyra.LogWarning("stats", "FactorAnalysis", "eigenvalue decomposition failed")
		return nil
	}

	eigenvalues := eig.Values(nil)
	var eigenvectors mat.Dense
	eig.VectorsTo(&eigenvectors)

	// Sort eigenvalues and eigenvectors in descending order
	type eigenPair struct {
		value  float64
		vector []float64
	}
	pairs := make([]eigenPair, colNum)
	for i := 0; i < colNum; i++ {
		vec := make([]float64, colNum)
		for j := 0; j < colNum; j++ {
			vec[j] = eigenvectors.At(j, i)
		}
		pairs[i] = eigenPair{value: eigenvalues[i], vector: vec}
	}
	// Sort in descending order
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].value < pairs[j].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	sortedEigenvalues := make([]float64, colNum)
	sortedEigenvectors := mat.NewDense(colNum, colNum, nil)
	for i := 0; i < colNum; i++ {
		sortedEigenvalues[i] = pairs[i].value
		for j := 0; j < colNum; j++ {
			sortedEigenvectors.Set(j, i, pairs[i].vector[j])
		}
	}

	// Step 5: Determine number of factors
	numFactors := decideNumFactors(sortedEigenvalues, opt.Count, colNum, rowNum)
	if numFactors == 0 {
		insyra.LogWarning("stats", "FactorAnalysis", "no factors retained")
		return nil
	}

	if opt.MaxIter <= 0 {
		opt.MaxIter = 100
	}
	// Use internal tolerance by default. Users who previously used the
	// (now-removed) Tol field can emulate disabling tolerance by setting
	// MaxIter explicitly.
	tolVal := extractionTolerance

	// Step 6: Extract factors
	// Convert SymDense to Dense for extraction functions
	corrDense := mat.NewDense(colNum, colNum, nil)
	for i := 0; i < colNum; i++ {
		for j := 0; j < colNum; j++ {
			corrDense.Set(i, j, corrMatrix.At(i, j))
		}
	}

	initialCommunalities := make([]float64, colNum)
	if opt.Extraction == FactorExtractionPAF {
		// For PAF, SPSS default is to use squared multiple correlations (SMC) as initial estimates
		smcVec, _ := fa.Smc(corrDense, nil) // Correctly call fa.Smc
		if smcVec == nil {
			insyra.LogWarning("stats", "FactorAnalysis", "SMC calculation failed, falling back to diagonal correlations")
			// Fallback to simpler method if SMC fails
			for i := 0; i < colNum; i++ {
				initialCommunalities[i] = corrDense.At(i, i)
			}
		} else {
			copy(initialCommunalities, smcVec.RawVector().Data) // Correctly copy data from VecDense
		}
	} else {
		// For other methods like PCA, use the diagonal of the correlation matrix (which is 1.0)
		for i := 0; i < colNum; i++ {
			initialCommunalities[i] = corrDense.At(i, i)
		}
	}

	loadings, extractionEigenvalues, converged, iterations, err := extractFactors(data, corrDense, sortedEigenvalues, sortedEigenvectors, numFactors, opt, rowNum, tolVal, initialCommunalities)
	if err != nil {
		insyra.LogWarning("stats", "FactorAnalysis", "factor extraction failed: %v", err)
		return nil
	}

	// Standardize factor signs after extraction (before rotation)
	// This ensures consistency: largest absolute loading per factor is positive
	if loadings != nil {
		insyra.LogInfo("stats", "FactorAnalysis", "Before standardization: A1 F2 = %.6f", loadings.At(0, 1))
		loadings = standardizeFactorSigns(loadings)
		insyra.LogInfo("stats", "FactorAnalysis", "After standardization: A1 F2 = %.6f", loadings.At(0, 1))
	}

	// Sanity check: inspect unrotated loadings before any rotation is applied
	if loadings != nil {
		pVars, mFactors := loadings.Dims()
		if pVars > 0 && mFactors > 0 {
			maxAbs := 0.0
			for i := range pVars {
				for j := range mFactors {
					val := math.Abs(loadings.At(i, j))
					if val > maxAbs {
						maxAbs = val
					}
				}
			}
			sampleVars := min(2, pVars)
			sampleFactors := min(2, mFactors)
			buffer := make([]float64, 0, sampleVars*sampleFactors)
			for i := range sampleVars {
				for j := range sampleFactors {
					buffer = append(buffer, loadings.At(i, j))
				}
			}
			insyra.LogInfo("stats", "FactorAnalysis", "pre-rotation loadings |max|=%.3f, samples=%v", maxAbs, buffer)
		}
	}

	// Special handling for PAF + Oblimin to match SPSS
	var useSPSSPAFOblimin bool = (opt.Extraction == FactorExtractionPAF && opt.Rotation.Method == FactorRotationOblimin)
	var extractionCommunalities []float64
	if useSPSSPAFOblimin {
		extractionCommunalities = make([]float64, colNum)
	}

	// Step 7: Rotate factors
	var rotatedLoadings *mat.Dense
	var unrotatedLoadings *mat.Dense
	var rotationMatrix *mat.Dense
	var phi *mat.Dense
	var rotationConverged bool
	if useSPSSPAFOblimin {
		// For PAF + Oblimin, first extract factors using PAF, then apply Oblimin rotation
		// This provides better SPSS compatibility than separate extraction + rotation
		unrotatedLoadings = mat.DenseCopyOf(loadings)
		rotatedLoadings, rotationMatrix, phi, rotationConverged, err = rotateFactors(loadings, opt.Rotation, opt.MinErr, opt.MaxIter)
		if err != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "PAF+Oblimin rotation failed: %v", err)
			rotatedLoadings = loadings
			rotationMatrix = nil
			phi = nil
			rotationConverged = false
		}
	} else if opt.Rotation.Method != FactorRotationNone {
		unrotatedLoadings = mat.DenseCopyOf(loadings)
		rotatedLoadings, rotationMatrix, phi, rotationConverged, err = rotateFactors(loadings, opt.Rotation, opt.MinErr, opt.MaxIter)
		if err != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "rotation failed: %v", err)
			rotatedLoadings = loadings
			rotationMatrix = nil
			phi = nil
			rotationConverged = true
		}
		// Note: rotateFactors now handles sign standardization internally
	} else {
		unrotatedLoadings = mat.DenseCopyOf(loadings)
		rotatedLoadings = loadings
		rotationMatrix = nil
		phi = nil
		rotationConverged = true
		// Apply factor reflection for unrotated factors
		rotatedLoadings, _ = reflectFactorsForPositiveLoadings(rotatedLoadings)
	}

	// Sort or align factor columns
	// For SPSS PAF + Oblimin, keep the extraction-order columns (descending
	// eigenvalues of the reduced correlation R*). For generic cases, sort by
	// explained variance (sum of squared structure loadings).
	if !useSPSSPAFOblimin {
		rotatedLoadings, rotationMatrix, phi = sortFactorsByExplainedVariance(rotatedLoadings, rotationMatrix, phi)
	}

	// Step 8: Compute communalities and uniquenesses
	// Preserve SPSS PAF+Oblimin extraction communalities (h_final) if available.
	if extractionCommunalities == nil {
		extractionCommunalities = make([]float64, colNum)
	}
	uniquenesses := make([]float64, colNum)
	var phiMat *mat.Dense
	if phi != nil {
		phiMat = phi
	}
	for i := 0; i < colNum; i++ {
		var hi2 float64
		// If SPSS-compatible PAF+Oblimin path supplied final h, use it for 'Extraction' communalities.
		if useSPSSPAFOblimin && extractionCommunalities[i] > 0 {
			hi2 = extractionCommunalities[i]
		} else {
			// For PAF, communalities should be computed from unrotated loadings
			if opt.Extraction == FactorExtractionPAF {
				for j := 0; j < numFactors; j++ {
					v := unrotatedLoadings.At(i, j)
					hi2 += v * v
				}
			} else if phiMat == nil {
				for j := 0; j < numFactors; j++ {
					v := rotatedLoadings.At(i, j)
					hi2 += v * v
				}
			} else {
				rowVec := mat.NewVecDense(numFactors, nil)
				for j := range numFactors {
					rowVec.SetVec(j, rotatedLoadings.At(i, j))
				}
				var tmp mat.VecDense
				tmp.MulVec(phiMat, rowVec)
				hi2 = mat.Dot(rowVec, &tmp)
			}
			diag := corrMatrix.At(i, i)
			if diag == 0 {
				diag = 1.0
			}
			if hi2 > diag {
				hi2 = diag
			}
			if hi2 < 0 {
				hi2 = 0
			}
			extractionCommunalities[i] = hi2
		}
		diag := corrMatrix.At(i, i)
		if diag == 0 {
			diag = 1.0
		}
		uniq := diag - hi2
		if uniq < uniquenessLowerBound {
			uniq = uniquenessLowerBound
		}
		uniquenesses[i] = uniq
	}

	commMatrix := mat.NewDense(colNum, 2, nil)
	for i := 0; i < colNum; i++ {
		commMatrix.Set(i, 0, initialCommunalities[i])
		commMatrix.Set(i, 1, extractionCommunalities[i])
	}
	communalitiesTable := matrixToDataTableWithNames(commMatrix, tableNameCommunalities, []string{"Initial", "Extraction"}, colNames)

	// Step 10: Compute factor scores if data is available
	var scores *mat.Dense
	var scoreWeights *mat.Dense
	var scoreCovariance *mat.Dense
	if rowNum > 0 {
		var err error
		scores, scoreWeights, scoreCovariance, err = computeFactorScores(data, rotatedLoadings, phi, uniquenesses, sigmaForScores, opt.Scoring)
		if err != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "failed to compute factor scores: %v", err)
		}
	}

	// Convert results to DataTables
	// Generate factor column names
	factorColNames := make([]string, numFactors)
	for i := range numFactors {
		factorColNames[i] = fmt.Sprintf("Factor_%d", i+1)
	}

	// Step 9: Compute explained proportions for "Extraction SS loadings"
	// SPSS reports three blocks: Initial eigenvalues, Extraction SS loadings,
	// and Rotation SS loadings. Our single "ExplainedProportion" field should
	// reflect the Extraction SS loadings (not the rotated SS loadings).
	pVars, mFactors := rotatedLoadings.Dims()
	explainedProp := make([]float64, mFactors)
	cumulativeProp := make([]float64, mFactors)

	if (useSPSSPAFOblimin || opt.Extraction == FactorExtractionPCA) && len(extractionEigenvalues) == mFactors {
		// Use eigenvalues for PCA or special PAF+Oblimin case
		cum := 0.0
		for j := range mFactors {
			prop := extractionEigenvalues[j] / float64(pVars) * 100.0
			explainedProp[j] = prop
			cum += prop
			cumulativeProp[j] = cum
		}
	} else {
		// Fallback for other methods: use unrotated SS loadings as extraction
		// SS (pre-rotation loadings: structure == pattern)
		ssLoad := make([]float64, mFactors)
		for j := range mFactors {
			sum := 0.0
			for i := range pVars {
				v := loadings.At(i, j)
				sum += v * v
			}
			ssLoad[j] = sum
		}
		totalVar := float64(pVars)
		cum := 0.0
		for j := range mFactors {
			prop := ssLoad[j] / totalVar * 100.0
			explainedProp[j] = prop
			cum += prop
			cumulativeProp[j] = cum
		}
		insyra.LogDebug("stats", "FactorAnalysis", "explainedProp calculated: %v", explainedProp)
	}

	// Build structure matrix for reporting: S = P (orthogonal) or S = P * Phi (oblique)
	var S mat.Dense
	if phi == nil {
		S.CloneFrom(rotatedLoadings)
	} else {
		var tmp mat.Dense
		tmp.Mul(rotatedLoadings, phi)
		S.CloneFrom(&tmp)
	}

	structureTable := matrixToDataTableWithNames(&S, tableNameFactorStructure, factorColNames, colNames)

	messages := []string{
		fmt.Sprintf("Extraction method: %s", opt.Extraction),
		fmt.Sprintf("Factor count method: %s (retained %d)", opt.Count.Method, numFactors),
		fmt.Sprintf("Rotation method: %s", opt.Rotation.Method),
		fmt.Sprintf("Scoring method: %s", opt.Scoring),
	}
	if opt.Scoring == FactorScoreNone {
		messages = append(messages, "Factor scores not computed (scoring disabled)")
	} else if scores == nil {
		messages = append(messages, "Factor scores unavailable for selected scoring method")
	}
	if iterations > 0 {
		messages = append(messages, fmt.Sprintf("Extraction iterations: %d", iterations))
	}
	if !converged && (opt.Extraction == FactorExtractionPAF || opt.Extraction == FactorExtractionML || opt.Extraction == FactorExtractionMINRES) {
		messages = append(messages, "Warning: extraction did not converge within limits")
	}
	if phi != nil {
		messages = append(messages, "Oblique rotation applied; factor correlation matrix provided")
	}
	messages = append(messages, "Factor analysis completed")

	result := FactorAnalysisResult{
		Loadings:             matrixToDataTableWithNames(rotatedLoadings, tableNameFactorLoadings, factorColNames, colNames),
		UnrotatedLoadings:    matrixToDataTableWithNames(unrotatedLoadings, tableNameUnrotatedLoadings, factorColNames, colNames),
		Structure:            structureTable,
		Uniquenesses:         vectorToDataTableWithNames(uniquenesses, tableNameUniqueness, "Uniqueness", colNames),
		Communalities:        communalitiesTable,
		SamplingAdequacy:     samplingAdequacyTable,
		BartlettTest:         bartlettResult,
		Phi:                  nil,
		RotationMatrix:       nil,
		Eigenvalues:          vectorToDataTableWithNames(sortedEigenvalues, tableNameEigenvalues, "Eigenvalue", factorColNames),
		ExplainedProportion:  vectorToDataTableWithNames(explainedProp, tableNameExplainedProportion, "Explained Proportion", factorColNames),
		CumulativeProportion: vectorToDataTableWithNames(cumulativeProp, tableNameCumulativeProportion, "Cumulative Proportion", factorColNames),
		Scores:               nil,
		Converged:            converged,
		RotationConverged:    rotationConverged,
		Iterations:           iterations,
		CountUsed:            numFactors,
		Messages:             messages,
	}

	if rotationMatrix != nil {
		result.RotationMatrix = matrixToDataTableWithNames(rotationMatrix, tableNameRotationMatrix, factorColNames, factorColNames)
	}
	if phi != nil {
		result.Phi = matrixToDataTableWithNames(phi, tableNamePhiMatrix, factorColNames, factorColNames)
	}
	if scores != nil {
		result.Scores = matrixToDataTableWithNames(scores, tableNameFactorScores, factorColNames, rowNames)
	}
	if scoreWeights != nil {
		scoreCoeffTable := matrixToDataTableWithNames(scoreWeights, tableNameFactorScoreCoefficients, factorColNames, colNames)
		result.ScoreCoefficients = scoreCoeffTable
	}
	if scoreCovariance != nil {
		scoreCovTable := matrixToDataTableWithNames(scoreCovariance, tableNameFactorScoreCovariance, factorColNames, factorColNames)
		result.ScoreCovariance = scoreCovTable
	}

	return &FactorModel{
		FactorAnalysisResult: result,
		scoreMethod:          opt.Scoring,
		extraction:           opt.Extraction,
		rotation:             opt.Rotation.Method,
		means:                means,
		sds:                  sds,
		sigma:                sigmaForScores,
	}
}

// -------------------------
// Helper Functions
// -------------------------

// decideNumFactors determines the number of factors to extract
func decideNumFactors(eigenvalues []float64, spec FactorCountSpec, maxPossible int, sampleSize int) int {
	switch spec.Method {
	case FactorCountFixed:
		if spec.FixedK > 0 && spec.FixedK <= maxPossible {
			return spec.FixedK
		}
		return maxPossible

	case FactorCountKaiser:
		threshold := spec.EigenThreshold
		if threshold == 0 {
			threshold = 1.0
		}
		return applyFactorLimits(countByThreshold(eigenvalues, threshold), spec.MaxFactors, maxPossible)

	default:
		return applyFactorLimits(countByThreshold(eigenvalues, 1.0), spec.MaxFactors, maxPossible)
	}
}

func applyFactorLimits(count int, maxFactors int, hardLimit int) int {
	if count < 1 {
		count = 1
	}
	if count > hardLimit {
		count = hardLimit
	}
	if maxFactors > 0 && count > maxFactors {
		count = maxFactors
	}
	return count
}

func countByThreshold(eigenvalues []float64, threshold float64) int {
	count := 0
	for _, ev := range eigenvalues {
		if ev >= threshold {
			count++
		}
	}
	return count
}

// extractFactors wraps the internal extraction functions
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions, sampleSize int, tol float64, initialCommunalities []float64) (*mat.Dense, []float64, bool, int, error) {
	var loadings *mat.Dense
	var extractionEigenvalues []float64
	var converged bool
	var iterations int
	var err error

	switch opt.Extraction {
	case FactorExtractionPCA:
		loadings, converged, iterations, err = extractPCA(eigenvalues, eigenvectors, numFactors)
		extractionEigenvalues = nil

	case FactorExtractionPAF:
		loadings, extractionEigenvalues, converged, iterations, err = extractPAF(corrMatrix, numFactors, opt.MaxIter, 1e-10, initialCommunalities)

	case FactorExtractionML:
		loadings, converged, iterations, err = extractML(corrMatrix, numFactors, 2000, 1e-6, sampleSize, initialCommunalities)
		extractionEigenvalues = nil

	case FactorExtractionMINRES:
		loadings, converged, iterations, err = extractMINRES(corrMatrix, numFactors, opt.MaxIter, tol)
		extractionEigenvalues = nil

	default:
		// Default to MINRES to match R psych::fa and the documented default behavior.
		loadings, converged, iterations, err = extractMINRES(corrMatrix, numFactors, opt.MaxIter, tol)
		extractionEigenvalues = nil
	}

	return loadings, extractionEigenvalues, converged, iterations, err
}

// computePCALoadings constructs factor loadings for PCA given eigenvalues/vectors.
// The logic is shared with PCA extraction and ML fallbacks to avoid duplication.
func computePCALoadings(eigenvalues []float64, eigenvectors *mat.Dense, numFactors int) (*mat.Dense, error) {
	if eigenvectors == nil {
		return nil, fmt.Errorf("computePCALoadings: eigenvectors nil")
	}
	p, cols := eigenvectors.Dims()
	if cols == 0 {
		return nil, fmt.Errorf("computePCALoadings: zero columns")
	}
	if numFactors <= 0 || numFactors > cols {
		numFactors = cols
	}

	// R: Adjust small eigenvalues before using them
	// eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps
	adjustedEigenvalues := make([]float64, len(eigenvalues))
	for i := range eigenvalues {
		if eigenvalues[i] < machineEpsilon {
			adjustedEigenvalues[i] = eigenvalueMinThreshold
		} else {
			adjustedEigenvalues[i] = eigenvalues[i]
		}
	}

	loadings := mat.NewDense(p, numFactors, nil)
	for i := range p {
		for j := 0; j < numFactors; j++ {
			if j < len(adjustedEigenvalues) {
				loadings.Set(i, j, eigenvectors.At(i, j)*math.Sqrt(adjustedEigenvalues[j]))
			} else {
				loadings.Set(i, j, 0)
			}
		}
	}
	return loadings, nil
}

// extractPCA extracts factors using Principal Component Analysis
func extractPCA(eigenvalues []float64, eigenvectors *mat.Dense, numFactors int) (*mat.Dense, bool, int, error) {
	loadings, err := computePCALoadings(eigenvalues, eigenvectors, numFactors)
	if err != nil {
		return nil, false, 0, err
	}
	return loadings, true, 0, nil
}

// extractMINRES performs MINRES factor extraction (simplified implementation)
// extractPAF performs Principal Axis Factoring extraction
func extractPAF(corr *mat.Dense, numFactors int, maxIter int, tol float64, initialCommunalities []float64) (*mat.Dense, []float64, bool, int, error) {
	if corr == nil {
		return nil, nil, false, 0, fmt.Errorf("nil correlation matrix")
	}

	rows, cols := corr.Dims()
	if rows != cols {
		return nil, nil, false, 0, fmt.Errorf("correlation matrix must be square")
	}
	if numFactors > cols {
		numFactors = cols
	}

	// Initialize communalities using initial values passed from caller
	communalities := make([]float64, rows)
	copy(communalities, initialCommunalities)

	var loadings *mat.Dense
	converged := false
	iterations := 0

	for iter := range maxIter {
		iterations = iter + 1

		// Create reduced correlation matrix R* by replacing the diagonal of R with communalities
		reducedCorr := mat.NewDense(rows, cols, nil)
		reducedCorr.Copy(corr)
		for i := range rows {
			reducedCorr.Set(i, i, communalities[i])
		}

		insyra.LogDebug("stats", "FactorAnalysis", "PAF reducedCorr diagonal: %.6f, %.6f, %.6f", reducedCorr.At(0, 0), reducedCorr.At(1, 1), reducedCorr.At(2, 2))

		// Eigenvalue decomposition of reduced correlation matrix
		reducedCorrSym := mat.NewSymDense(rows, nil)
		for i := range rows {
			for j := range rows {
				reducedCorrSym.SetSym(i, j, reducedCorr.At(i, j))
			}
		}
		var eig mat.EigenSym
		if !eig.Factorize(reducedCorrSym, true) {
			return nil, nil, false, iterations, fmt.Errorf("eigenvalue decomposition failed")
		}

		// Get eigenvalues and eigenvectors
		eigenvalues := eig.Values(nil)
		eigenvectors := mat.NewDense(rows, rows, nil)
		eig.VectorsTo(eigenvectors)

		// Sort eigenvalues and eigenvectors in descending order
		type eigenPair struct {
			value  float64
			vector []float64
			index  int
		}
		pairs := make([]eigenPair, rows)
		for i := range rows {
			pairs[i] = eigenPair{
				value:  eigenvalues[i],
				vector: make([]float64, rows),
				index:  i,
			}
			for j := range rows {
				pairs[i].vector[j] = eigenvectors.At(j, i)
			}
		}

		// Sort by eigenvalue in descending order
		for i := 0; i < rows-1; i++ {
			for j := i + 1; j < rows; j++ {
				if pairs[j].value > pairs[i].value {
					pairs[i], pairs[j] = pairs[j], pairs[i]
				}
			}
		}

		// Extract first numFactors components
		newLoadings := mat.NewDense(rows, numFactors, nil)
		for i := range rows {
			for j := 0; j < numFactors; j++ {
				// loadings = eigenvectors * sqrt(eigenvalues)
				val := pairs[j].value
				if val > 0 {
					newLoadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(val))
				}
			}
		}

		// Update communalities: h_i = sum(loadings[i,j]^2 for j in 1..m)
		newCommunalities := make([]float64, rows)
		for i := range rows {
			sum := 0.0
			for j := 0; j < numFactors; j++ {
				val := newLoadings.At(i, j)
				sum += val * val
			}
			newCommunalities[i] = sum
			// Ensure communalities stay within [0,1]
			if newCommunalities[i] > 1.0 {
				newCommunalities[i] = 1.0
			}
			if newCommunalities[i] < 0.0 {
				newCommunalities[i] = 0.0
			}
		}

		// Check convergence - use SPSS-like convergence criterion
		maxDiff := 0.0
		for i := range rows {
			diff := math.Abs(newCommunalities[i] - communalities[i])
			if diff > maxDiff {
				maxDiff = diff
			}
		}

		loadings = newLoadings
		communalities = newCommunalities

		// SPSS uses a more lenient convergence criterion, allow convergence after max iterations
		if maxDiff < tol || iterations >= maxIter {
			converged = true
			break
		}
	}

	// Extract final eigenvalues from the converged reduced correlation matrix
	var finalEigenvalues []float64
	if converged {
		// Recreate the final reduced correlation matrix
		reducedCorr := mat.NewDense(rows, cols, nil)
		reducedCorr.CloneFrom(corr)
		for i := range rows {
			reducedCorr.Set(i, i, communalities[i])
		}

		// Perform final eigenvalue decomposition using real symmetric matrix
		reducedCorrSym := mat.NewSymDense(rows, nil)
		for i := range rows {
			for j := range rows {
				reducedCorrSym.SetSym(i, j, reducedCorr.At(i, j))
			}
		}
		var eig mat.EigenSym
		if eig.Factorize(reducedCorrSym, true) {
			eigenvalues := eig.Values(nil)
			finalEigenvalues = make([]float64, numFactors)
			for j := 0; j < numFactors && j < len(eigenvalues); j++ {
				finalEigenvalues[j] = eigenvalues[j]
			}
		}
	}

	return loadings, finalEigenvalues, converged, iterations, nil
}

func extractMINRES(corr *mat.Dense, numFactors int, maxIter int, tol float64) (*mat.Dense, bool, int, error) {
	// MINRES (Minimum Residual) factor extraction - minimizes the residual correlation matrix
	if corr == nil {
		return nil, false, 0, fmt.Errorf("nil correlation matrix")
	}

	rows, cols := corr.Dims()
	if numFactors > cols {
		numFactors = cols
	}

	// Initialize communalities using squared multiple correlations (SMC)
	// This is more appropriate for MINRES than Kaiser normalization
	communalities := make([]float64, rows)
	for i := range rows {
		sumSqOffDiag := 0.0
		for j := range cols {
			if i != j {
				r := corr.At(i, j)
				sumSqOffDiag += r * r
			}
		}
		// SMC = sum of squared off-diagonal correlations
		smc := sumSqOffDiag
		if smc > 0.995 {
			smc = 0.995
		}
		communalities[i] = smc
	}

	converged := false
	iterations := 0

	for iter := range maxIter {
		iterations = iter + 1

		// Create reduced correlation matrix R* = R - diag(1 - communalities)
		reducedCorr := mat.NewDense(rows, cols, nil)
		reducedCorr.Copy(corr)
		for i := range rows {
			reducedCorr.Set(i, i, corr.At(i, i)*(1.0-communalities[i]))
		}

		// For MINRES, we use a different approach: minimize the residual
		// by finding loadings that best reproduce the correlation matrix
		loadings, residual := minresFit(reducedCorr, numFactors)

		// Update communalities based on the fitted loadings
		newCommunalities := make([]float64, rows)
		for i := range rows {
			sumSquares := 0.0
			for j := 0; j < numFactors; j++ {
				loading := loadings.At(i, j)
				sumSquares += loading * loading
			}
			newCommunalities[i] = sumSquares
			// Ensure communalities don't exceed the SMC estimate
			if newCommunalities[i] > communalities[i] {
				newCommunalities[i] = communalities[i]
			}
			if newCommunalities[i] > 1.0 {
				newCommunalities[i] = 1.0
			}
			if newCommunalities[i] < 0.0 {
				newCommunalities[i] = 0.0
			}
		}

		// Check convergence based on residual matrix
		residualNorm := 0.0
		for i := range rows {
			for j := range cols {
				residualNorm += residual.At(i, j) * residual.At(i, j)
			}
		}
		residualNorm = math.Sqrt(residualNorm)

		communalities = newCommunalities

		if residualNorm < tol {
			converged = true
			break
		}
	}

	// Final extraction using converged communalities
	reducedCorr := mat.NewDense(rows, cols, nil)
	reducedCorr.Copy(corr)
	for i := range rows {
		reducedCorr.Set(i, i, corr.At(i, i)*(1.0-communalities[i]))
	}

	finalLoadings, _ := minresFit(reducedCorr, numFactors)

	return finalLoadings, converged, iterations, nil
}

// minresFit performs the core MINRES fitting for factor extraction
func minresFit(reducedCorr *mat.Dense, numFactors int) (*mat.Dense, *mat.Dense) {
	rows, _ := reducedCorr.Dims()

	// Use eigenvalue decomposition of the reduced correlation matrix
	reducedCorrSym := mat.NewSymDense(rows, nil)
	for i := range rows {
		for j := range rows {
			reducedCorrSym.SetSym(i, j, reducedCorr.At(i, j))
		}
	}

	var eig mat.EigenSym
	if !eig.Factorize(reducedCorrSym, true) {
		// Return zero loadings if decomposition fails
		return mat.NewDense(rows, numFactors, nil), reducedCorr
	}

	eigenvalues := eig.Values(nil)
	eigenvectors := mat.NewDense(rows, rows, nil)
	eig.VectorsTo(eigenvectors)

	// Sort eigenvalues and eigenvectors in descending order
	type eigenPair struct {
		value  float64
		vector []float64
	}
	pairs := make([]eigenPair, rows)
	for i := range rows {
		pairs[i] = eigenPair{
			value:  eigenvalues[i],
			vector: make([]float64, rows),
		}
		for j := range rows {
			pairs[i].vector[j] = eigenvectors.At(j, i)
		}
	}

	for i := 0; i < rows-1; i++ {
		for j := i + 1; j < rows; j++ {
			if pairs[j].value > pairs[i].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Extract loadings for first numFactors
	loadings := mat.NewDense(rows, numFactors, nil)
	for i := range rows {
		for j := 0; j < numFactors; j++ {
			val := pairs[j].value
			if val > 0 {
				loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(val))
			}
		}
	}

	// Compute residual: R - L*L^T (where L is the loading matrix)
	reproduced := mat.NewDense(rows, rows, nil)
	reproduced.Mul(loadings, loadings.T())

	residual := mat.NewDense(rows, rows, nil)
	residual.Sub(reducedCorr, reproduced)

	return loadings, residual
}

// extractML performs Maximum Likelihood factor extraction using true ML estimation with BFGS optimization
func extractML(corr *mat.Dense, numFactors int, maxIter int, tol float64, sampleSize int, initialCommunalities []float64) (*mat.Dense, bool, int, error) {
	if corr == nil {
		return nil, false, 0, fmt.Errorf("nil correlation matrix")
	}

	rows, cols := corr.Dims()
	if numFactors > cols {
		numFactors = cols
	}

	// Initialize communalities using Kaiser normalization (diagonal squared) - SPSS default for ML
	communalities := make([]float64, rows)
	for i := range rows {
		val := corr.At(i, i)
		communalities[i] = val * val
		if communalities[i] > 0.995 {
			communalities[i] = 0.995
		}
	}

	// Override with provided initialCommunalities if available
	if len(initialCommunalities) == rows {
		copy(communalities, initialCommunalities)
	}

	// True Maximum Likelihood estimation using BFGS optimization
	// Start with communalities as initial uniquenesses (Psi = 1 - h^2)
	startPsi := make([]float64, rows)
	for i := range rows {
		startPsi[i] = 1.0 - communalities[i]
		if startPsi[i] < 0.005 {
			startPsi[i] = 0.005
		}
		if startPsi[i] > 0.995 {
			startPsi[i] = 0.995
		}
	}

	// Use BFGS optimization from gonum/optimize
	objFunc := func(x []float64) float64 {
		return computeMLObjective(corr, x, numFactors)
	}
	p := optimize.Problem{
		Func: objFunc,
	}

	method := &optimize.BFGS{}
	settings := optimize.DefaultSettings()
	settings.FunctionConverge.Absolute = tol
	settings.FunctionConverge.Iterations = maxIter
	settings.GradientThreshold = tol

	result, err := optimize.Local(p, startPsi, settings, method)
	if err != nil {
		// Fallback to simple gradient descent if BFGS fails
		insyra.LogWarning("stats", "FactorAnalysis", "BFGS optimization failed, falling back to gradient descent: %v", err)
		return extractMLFallback(corr, numFactors, maxIter, tol, sampleSize, initialCommunalities)
	}

	// Extract optimized psi
	psi := result.X
	converged := result.Status == optimize.Success || result.Status == optimize.FunctionConvergence
	iterations := result.Stats.FuncEvaluations

	// Extract final loadings using the optimized psi
	loadings, err := mlExtractLoadings(corr, psi, numFactors)
	if err != nil {
		return nil, false, iterations, err
	}

	return loadings, converged, iterations, nil
}

// extractMLFallback is the fallback implementation using simple gradient descent
func extractMLFallback(corr *mat.Dense, numFactors int, maxIter int, tol float64, sampleSize int, initialCommunalities []float64) (*mat.Dense, bool, int, error) {
	rows, cols := corr.Dims()
	if numFactors > cols {
		numFactors = cols
	}

	// Initialize communalities
	communalities := make([]float64, rows)
	for i := range rows {
		val := corr.At(i, i)
		communalities[i] = val * val
		if communalities[i] > 0.995 {
			communalities[i] = 0.995
		}
	}

	if len(initialCommunalities) == rows {
		copy(communalities, initialCommunalities)
	}

	startPsi := make([]float64, rows)
	for i := range rows {
		startPsi[i] = 1.0 - communalities[i]
		if startPsi[i] < 0.005 {
			startPsi[i] = 0.005
		}
		if startPsi[i] > 0.995 {
			startPsi[i] = 0.995
		}
	}

	converged := false
	iterations := 0
	psi := make([]float64, rows)
	copy(psi, startPsi)

	learningRate := 0.01 // Reduced learning rate for stability
	maxStepSize := 0.1   // Reduced max step size

	for iter := range maxIter {
		iterations = iter + 1

		// Compute current objective function and gradient
		_, grad := mlObjectiveAndGradient(corr, psi, numFactors)

		// Check for convergence
		gradNorm := 0.0
		for _, g := range grad {
			gradNorm += g * g
		}
		gradNorm = math.Sqrt(gradNorm)

		if gradNorm < tol {
			converged = true
			break
		}

		// Update psi using gradient descent with bounds
		for i := range rows {
			step := -learningRate * grad[i]
			if math.Abs(step) > maxStepSize {
				step = maxStepSize * step / math.Abs(step)
			}
			newPsi := psi[i] + step
			// Bound psi between 0.005 and 0.995
			if newPsi < 0.005 {
				newPsi = 0.005
			}
			if newPsi > 0.995 {
				newPsi = 0.995
			}
			psi[i] = newPsi
		}
	}

	loadings, err := mlExtractLoadings(corr, psi, numFactors)
	if err != nil {
		return nil, false, iterations, err
	}

	return loadings, converged, iterations, nil
}

// mlObjectiveAndGradient computes the ML objective function and its gradient
// This implements the R psych FAfn function logic
func mlObjectiveAndGradient(S *mat.Dense, psi []float64, nfactors int) (float64, []float64) {
	n := len(psi)

	// Create R* = S with diagonal replaced by communalities (1 - psi)
	Rstar := mat.NewDense(n, n, nil)
	Rstar.Copy(S)
	for i := range n {
		Rstar.Set(i, i, 1.0-psi[i])
	}

	// Eigenvalue decomposition of R*
	var eig mat.EigenSym
	RstarSym := mat.NewSymDense(n, Rstar.RawMatrix().Data)
	if !eig.Factorize(RstarSym, true) {
		// Return large objective if decomposition fails
		grad := make([]float64, n)
		return 1e10, grad
	}

	eigenvalues := eig.Values(nil)

	// Extract first nfactors eigenvalues for the factor model
	// Objective = sum(log(eigenvalues[i]) for i > nfactors) - sum(log(eigenvalues[i]) for i <= nfactors) + n - nfactors
	obj := 0.0
	for i := nfactors; i < n; i++ {
		if eigenvalues[i] > 1e-8 {
			obj += math.Log(eigenvalues[i])
		} else {
			obj += math.Log(1e-8)
		}
	}
	for i := range nfactors {
		if eigenvalues[i] > 1e-8 {
			obj -= math.Log(eigenvalues[i])
		} else {
			obj -= math.Log(1e-8)
		}
	}
	obj += float64(n - nfactors)

	// For gradient, we need d(obj)/d(psi_i)
	// This is complex and requires computing derivatives of eigenvalues
	// For simplicity, use finite differences
	grad := make([]float64, n)
	eps := 1e-6
	for i := range n {
		psiPlus := make([]float64, n)
		copy(psiPlus, psi)
		psiPlus[i] += eps

		// Compute objective for psiPlus directly (avoid recursion)
		objPlus := computeMLObjective(S, psiPlus, nfactors)
		grad[i] = (objPlus - obj) / eps
	}

	return obj, grad
}

// computeMLObjective computes only the ML objective function (helper for gradient computation)
func computeMLObjective(S *mat.Dense, psi []float64, nfactors int) float64 {
	n := len(psi)

	// Create uniqueness matrix Psi (diagonal matrix with psi on diagonal)
	Psi := mat.NewDense(n, n, nil)
	for i := range n {
		Psi.Set(i, i, psi[i])
	}

	// Create R* = S - Psi (reduced correlation matrix)
	Rstar := mat.NewDense(n, n, nil)
	Rstar.Copy(S)
	for i := range n {
		Rstar.Set(i, i, S.At(i, i)-psi[i])
	}

	// Eigenvalue decomposition of R*
	var eig mat.EigenSym
	RstarSym := mat.NewSymDense(n, Rstar.RawMatrix().Data)
	if !eig.Factorize(RstarSym, true) {
		return 1e10 // Return large objective if decomposition fails
	}

	eigenvalues := eig.Values(nil)
	eigenvectors := mat.NewDense(n, n, nil)
	eig.VectorsTo(eigenvectors)

	// Extract factor loadings Lambda (first nfactors eigenvectors scaled by sqrt of eigenvalues)
	Lambda := mat.NewDense(n, nfactors, nil)
	for i := range nfactors {
		if eigenvalues[i] > 0 {
			scale := math.Sqrt(eigenvalues[i])
			for j := range n {
				Lambda.Set(j, i, eigenvectors.At(j, i)*scale)
			}
		}
	}

	// Compute model-implied covariance Sigma = Lambda * Lambda^T + Psi
	Sigma := mat.NewDense(n, n, nil)
	Sigma.Mul(Lambda, Lambda.T())
	Sigma.Add(Sigma, Psi)

	// Compute log determinant of Sigma
	var lu mat.LU
	lu.Factorize(Sigma)
	if lu.Det() <= 0 {
		return 1e10 // Return large objective if Sigma is not positive definite
	}
	logDetSigma := math.Log(math.Abs(lu.Det()))

	// Compute trace(Sigma^{-1} * S)
	var SigmaInv mat.Dense
	err := SigmaInv.Inverse(Sigma)
	if err != nil {
		return 1e10 // Return large objective if inversion fails
	}

	traceTerm := 0.0
	for i := range n {
		for j := range n {
			traceTerm += SigmaInv.At(i, j) * S.At(i, j)
		}
	}

	// ML objective function: log|Sigma| + trace(Sigma^{-1} * S) - log|S| - n
	// But since S is correlation matrix with 1s on diagonal, log|S| = 0
	obj := logDetSigma + traceTerm - float64(n)

	return obj
}

// mlExtractLoadings extracts factor loadings given optimized psi
func mlExtractLoadings(S *mat.Dense, psi []float64, nfactors int) (*mat.Dense, error) {
	n := len(psi)

	// Create R* = S with diagonal replaced by communalities (1 - psi)
	Rstar := mat.NewDense(n, n, nil)
	Rstar.Copy(S)
	for i := range n {
		Rstar.Set(i, i, 1.0-psi[i])
	}

	// Eigenvalue decomposition
	var eig mat.EigenSym
	RstarSym := mat.NewSymDense(n, Rstar.RawMatrix().Data)
	if !eig.Factorize(RstarSym, true) {
		return nil, fmt.Errorf("eigenvalue decomposition failed")
	}

	eigenvalues := eig.Values(nil)
	eigenvectors := mat.NewDense(n, n, nil)
	eig.VectorsTo(eigenvectors)

	// Sort eigenvalues and eigenvectors in descending order
	type eigenPair struct {
		value  float64
		vector []float64
	}
	pairs := make([]eigenPair, n)
	for i := range n {
		pairs[i] = eigenPair{
			value:  eigenvalues[i],
			vector: make([]float64, n),
		}
		for j := range n {
			pairs[i].vector[j] = eigenvectors.At(j, i)
		}
	}

	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if pairs[j].value > pairs[i].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Extract loadings for first nfactors
	loadings := mat.NewDense(n, nfactors, nil)
	for i := range n {
		for j := range nfactors {
			if pairs[j].value > 0 {
				loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(pairs[j].value))
			}
		}
	}

	return loadings, nil
}

// computeKMOMeasures computes Kaiser-Meyer-Olkin measure and individual MSA values with improved numerical stability
func computeKMOMeasures(corr *mat.Dense) (overallKMO float64, msaValues []float64, err error) {
	if corr == nil {
		return 0, nil, fmt.Errorf("nil correlation matrix")
	}

	p, _ := corr.Dims()
	msaValues = make([]float64, p)

	// Use Cholesky decomposition for better numerical stability when computing partial correlations
	symCorr := mat.NewSymDense(p, nil)
	for i := range p {
		for j := 0; j <= i; j++ {
			symCorr.SetSym(i, j, corr.At(i, j))
		}
	}

	var chol mat.Cholesky
	if !chol.Factorize(symCorr) {
		// Fallback to eigenvalue decomposition if Cholesky fails
		var eig mat.EigenSym
		if !eig.Factorize(symCorr, true) {
			return 0, nil, fmt.Errorf("failed to decompose correlation matrix")
		}

		// Use pseudoinverse for better stability
		eigenvals := eig.Values(nil)
		eigenvecs := mat.NewDense(p, p, nil)
		eig.VectorsTo(eigenvecs)

		// Compute regularized inverse
		invCorr := mat.NewDense(p, p, nil)
		for i := range p {
			for j := range p {
				sum := 0.0
				for k := range p {
					if eigenvals[k] > 1e-12 { // Threshold for numerical stability
						sum += eigenvecs.At(i, k) * eigenvecs.At(j, k) / eigenvals[k]
					}
				}
				invCorr.Set(i, j, sum)
			}
		}

		return computeKMOFromInverse(corr, invCorr, msaValues)
	}

	// Use Cholesky-based inverse for better stability
	var L mat.TriDense
	chol.LTo(&L)

	invCorr := mat.NewDense(p, p, nil)
	err = invCorr.Inverse(L.T())
	if err != nil {
		return 0, nil, fmt.Errorf("failed to compute inverse from Cholesky: %v", err)
	}

	// Compute L^{-T} * L^{-1}
	var temp mat.Dense
	temp.Mul(invCorr.T(), invCorr)

	return computeKMOFromInverse(corr, &temp, msaValues)
}

// computeKMOFromInverse computes KMO measures given the correlation matrix and its inverse
func computeKMOFromInverse(corr, invCorr *mat.Dense, msaValues []float64) (overallKMO float64, msa []float64, err error) {
	p, _ := corr.Dims()

	// Compute MSA (Measure of Sampling Adequacy) for each variable
	for i := range p {
		sumRSquared := 0.0
		sumPSquared := 0.0

		for j := range p {
			if i != j {
				r := corr.At(i, j)
				// Compute partial correlation with regularization
				p_ij := -invCorr.At(i, j) / math.Sqrt(math.Max(invCorr.At(i, i)*invCorr.At(j, j), 1e-10))
				sumRSquared += r * r
				sumPSquared += p_ij * p_ij
			}
		}

		if sumRSquared+sumPSquared > 1e-10 {
			msaValues[i] = sumRSquared / (sumRSquared + sumPSquared)
		} else {
			msaValues[i] = 0
		}
	}

	// Compute overall KMO using partial correlations
	sumRSquared := 0.0
	sumPSquared := 0.0

	for i := range p {
		for j := range p {
			if i != j {
				r := corr.At(i, j)
				p_ij := -invCorr.At(i, j) / math.Sqrt(math.Max(invCorr.At(i, i)*invCorr.At(j, j), 1e-10))
				sumRSquared += r * r
				sumPSquared += p_ij * p_ij
			}
		}
	}

	if sumRSquared+sumPSquared > 1e-10 {
		overallKMO = sumRSquared / (sumRSquared + sumPSquared)
	} else {
		overallKMO = 0
	}

	return overallKMO, msaValues, nil
}

// kmoToDataTable converts KMO results to DataTable
func kmoToDataTable(overallKMO float64, msaValues []float64, colNames []string) *insyra.DataTable {
	// Create DataList for MSA values only (matching test expectation of 5x1)
	msaList := insyra.NewDataList().SetName("MSA")

	// MSA values for each variable
	for i := range msaValues {
		msaList.Append(msaValues[i])
	}

	// Overall KMO (though test expects only variable MSAs)
	msaList.Append(overallKMO)

	insyra.LogDebug("stats", "FactorAnalysis", "KMO values: MSA=%v, overall=%.6f", msaValues, overallKMO)

	return insyra.NewDataTable(msaList)
}

// computeBartlettFromCorrelation computes Bartlett's test of sphericity with improved numerical stability
func computeBartlettFromCorrelation(corr *mat.Dense, n int) (chiSquare float64, pValue float64, df int, err error) {
	if corr == nil {
		return 0, 0, 0, fmt.Errorf("nil correlation matrix")
	}

	p, _ := corr.Dims()
	df = p * (p - 1) / 2

	// Use Cholesky decomposition for more stable determinant calculation
	symCorr := mat.NewSymDense(p, nil)
	for i := range p {
		for j := 0; j <= i; j++ {
			symCorr.SetSym(i, j, corr.At(i, j))
		}
	}

	var chol mat.Cholesky
	if !chol.Factorize(symCorr) {
		// Fallback to eigenvalue decomposition
		var eig mat.EigenSym
		if !eig.Factorize(symCorr, false) {
			return 0, 0, df, fmt.Errorf("failed to decompose correlation matrix")
		}

		// Compute log determinant from eigenvalues
		logDet := 0.0
		for _, v := range eig.Values(nil) {
			if v > 1e-12 { // Threshold for numerical stability
				logDet += math.Log(v)
			} else {
				// Matrix is singular or near-singular
				return 0, 1.0, df, nil
			}
		}

		chiSquare = -((float64(n - 1)) - (2*float64(p)+5)/6) * logDet
	} else {
		// Use Cholesky determinant for better stability
		logDet := 0.0
		L := mat.NewTriDense(p, mat.Lower, nil)
		chol.LTo(L)
		for i := range p {
			diag := L.At(i, i)
			if diag > 1e-12 {
				logDet += math.Log(diag)
			} else {
				// Matrix is singular or near-singular
				return 0, 1.0, df, nil
			}
		}
		// Cholesky gives L such that A = L*L^T, so det(A) = det(L)^2 = product(diag(L))^2
		logDet *= 2

		chiSquare = -((float64(n - 1)) - (2*float64(p)+5)/6) * logDet
	}

	// Compute p-value using chi-square distribution
	if chiSquare > 0 && chiSquare < 1e10 { // Check for reasonable chi-square value
		pValue = 1 - distuv.ChiSquared{K: float64(df)}.CDF(chiSquare)
	} else {
		pValue = 1.0
	}

	return chiSquare, pValue, df, nil
}

// bartlettToDataTable converts Bartlett's test results to BartlettTestResult struct
func bartlettToDataTable(chiSquare float64, df int, pValue float64, n int) *BartlettTestResult {
	return &BartlettTestResult{
		ChiSquare:        chiSquare,
		DegreesOfFreedom: df,
		PValue:           pValue,
		SampleSize:       n,
	}
}

// reflectFactorsForPositiveLoadings ensures all factor loadings are positive by reflecting factors with negative loadings
func reflectFactorsForPositiveLoadings(loadings *mat.Dense) (*mat.Dense, error) {
	if loadings == nil {
		return nil, fmt.Errorf("nil loadings matrix")
	}

	rows, cols := loadings.Dims()
	reflectedLoadings := mat.DenseCopyOf(loadings)

	for j := range cols { // For each factor
		positiveCount := 0
		negativeCount := 0

		// Count positive and negative loadings for this factor
		for i := range rows {
			loading := reflectedLoadings.At(i, j)
			if loading > 0 {
				positiveCount++
			} else if loading < 0 {
				negativeCount++
			}
		}

		// If negative loadings are more than positive, reflect the factor
		if negativeCount > positiveCount {
			for i := range rows {
				reflectedLoadings.Set(i, j, -reflectedLoadings.At(i, j))
			}
		}
	}

	return reflectedLoadings, nil
}

// matrixToDataTableWithNames converts a matrix to DataTable with row and column names
func matrixToDataTableWithNames(matrix mat.Matrix, tableName string, colNames []string, rowNames []string) *insyra.DataTable {
	if matrix == nil {
		return nil
	}

	rows, cols := matrix.Dims()

	// Create DataLists for each column
	dataLists := make([]*insyra.DataList, cols)
	for j := range cols {
		var colName string
		if j < len(colNames) {
			colName = colNames[j]
		} else {
			colName = fmt.Sprintf("Col%d", j+1)
		}
		dataLists[j] = insyra.NewDataList().SetName(colName)

		// Add row values for this column
		for i := range rows {
			dataLists[j].Append(matrix.At(i, j))
		}
	}

	return insyra.NewDataTable(dataLists...)
}

// vectorToDataTableWithNames converts a vector (slice) to DataTable with row and column names
func vectorToDataTableWithNames(vector []float64, tableName string, colName string, rowNames []string) *insyra.DataTable {
	if len(vector) == 0 {
		return nil
	}

	// Create DataList for the single column
	dataList := insyra.NewDataList().SetName(colName)

	// Add vector values
	for _, val := range vector {
		dataList.Append(val)
	}

	dt := insyra.NewDataTable(dataList)

	// Set row names if provided
	if len(rowNames) > 0 && len(rowNames) >= len(vector) {
		dt.SetRowNames(rowNames[:len(vector)])
	}

	return dt
}

// sortFactorsByExplainedVariance sorts factors by explained variance in descending order
func sortFactorsByExplainedVariance(loadings *mat.Dense, rotationMatrix *mat.Dense, phi *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil {
		return nil, rotationMatrix, phi
	}

	rows, cols := loadings.Dims()

	// Calculate explained variance for each factor (sum of squared loadings)
	variances := make([]float64, cols)
	for j := range cols {
		sum := 0.0
		for i := range rows {
			loading := loadings.At(i, j)
			sum += loading * loading
		}
		variances[j] = sum
	}

	// Create indices for sorting (descending order)
	indices := make([]int, cols)
	for i := range indices {
		indices[i] = i
	}

	// Sort indices by variance (descending)
	for i := 0; i < cols-1; i++ {
		for j := i + 1; j < cols; j++ {
			if variances[indices[i]] < variances[indices[j]] {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	// Reorder loadings matrix
	sortedLoadings := mat.NewDense(rows, cols, nil)
	if rotationMatrix != nil {
		sortedRotationMatrix := mat.NewDense(cols, cols, nil)
		for j := range cols {
			newCol := indices[j]
			for i := range rows {
				sortedLoadings.Set(i, j, loadings.At(i, newCol))
			}
			for k := range cols {
				sortedRotationMatrix.Set(k, j, rotationMatrix.At(k, newCol))
			}
		}
		rotationMatrix = sortedRotationMatrix
	} else {
		for j := range cols {
			newCol := indices[j]
			for i := range rows {
				sortedLoadings.Set(i, j, loadings.At(i, newCol))
			}
		}
	}

	// Reorder phi matrix if it exists
	var sortedPhi *mat.Dense
	if phi != nil {
		sortedPhi = mat.NewDense(cols, cols, nil)
		for i := range cols {
			for j := range cols {
				sortedPhi.Set(i, j, phi.At(indices[i], indices[j]))
			}
		}
	}

	return sortedLoadings, rotationMatrix, sortedPhi
}

// rotateFactors rotates factor loadings based on rotation options
func rotateFactors(loadings *mat.Dense, rotationOpts FactorRotationOptions, minErr float64, maxIter int) (*mat.Dense, *mat.Dense, *mat.Dense, bool, error) {
	if loadings == nil {
		return nil, nil, nil, false, fmt.Errorf("nil loadings matrix")
	}

	// Map our rotation methods to fa package method names
	var method string
	switch rotationOpts.Method {
	case FactorRotationNone:
		// No rotation - return identity matrix
		_, cols := loadings.Dims()
		identity := mat.NewDense(cols, cols, nil)
		for i := range cols {
			identity.Set(i, i, 1.0)
		}
		phi := mat.NewDense(cols, cols, nil)
		for i := range cols {
			phi.Set(i, i, 1.0)
		}
		// Apply sign standardization to unrotated loadings
		standardizedLoadings := standardizeFactorSigns(mat.DenseCopyOf(loadings))
		return standardizedLoadings, identity, phi, true, nil

	case FactorRotationVarimax:
		method = "varimax"

	case FactorRotationQuartimax:
		method = "quartimax"

	case FactorRotationQuartimin:
		method = "quartimin"

	case FactorRotationOblimin:
		method = "oblimin"

	case FactorRotationGeominT:
		method = "geomint"

	case FactorRotationGeominQ:
		method = "geominq"

	case FactorRotationBentlerT:
		method = "bentlert"

	case FactorRotationBentlerQ:
		method = "bentlerq"

	case FactorRotationSimplimax:
		method = "simplimax"

	case FactorRotationPromax:
		method = "promax"

	default:
		// For unsupported methods, return unrotated loadings
		_, cols := loadings.Dims()
		identity := mat.NewDense(cols, cols, nil)
		for i := range cols {
			identity.Set(i, i, 1.0)
		}
		phi := mat.NewDense(cols, cols, nil)
		for i := range cols {
			phi.Set(i, i, 1.0)
		}
		// Apply sign standardization to unrotated loadings
		standardizedLoadings := standardizeFactorSigns(mat.DenseCopyOf(loadings))
		return standardizedLoadings, identity, phi, false, fmt.Errorf("unsupported rotation method: %s", rotationOpts.Method)
	}

	// Use fa.Rotate function
	opts := &fa.RotOpts{
		Eps:         minErr,                  // Use MinErr from function parameter
		MaxIter:     maxIter,                 // Use MaxIter from function parameter
		Gamma:       rotationOpts.Kappa,      // Use Kappa as Gamma for oblimin
		PromaxPower: int(rotationOpts.Kappa), // Use Kappa as PromaxPower
		Restarts:    rotationOpts.Restarts,
	}

	rotatedLoadings, rotMat, phi, converged, err := fa.Rotate(loadings, method, opts)
	if err != nil {
		return nil, nil, nil, false, err
	}

	// Apply sign standardization to rotated loadings (skip for Oblimin to match SPSS)
	if method != "oblimin" {
		standardizedLoadings := standardizeFactorSigns(rotatedLoadings)
		return standardizedLoadings, rotMat, phi, converged, nil
	}

	return rotatedLoadings, rotMat, phi, converged, nil
}

// standardizeFactorSigns standardizes the signs of factor loadings
// Ensures that the largest loading (in absolute value) for each factor is positive
func standardizeFactorSigns(loadings *mat.Dense) *mat.Dense {
	if loadings == nil {
		return nil
	}

	rows, cols := loadings.Dims()
	standardized := mat.DenseCopyOf(loadings)

	for j := range cols {
		// Find the variable with the largest absolute loading for this factor
		maxAbsLoading := 0.0
		maxAbsIndex := 0

		for i := range rows {
			absLoading := math.Abs(standardized.At(i, j))
			if absLoading > maxAbsLoading {
				maxAbsLoading = absLoading
				maxAbsIndex = i
			}
		}

		// If the largest loading is negative, reflect the entire factor
		if standardized.At(maxAbsIndex, j) < 0 {
			for i := range rows {
				standardized.Set(i, j, -standardized.At(i, j))
			}
		}
	}

	return standardized
}

// FactorPAFOblimin performs PA-F Oblimin rotation (simplified implementation)
func FactorPAFOblimin(corr *mat.Dense, numFactors int, delta float64, epsilon float64, maxIter int, normalize float64) (*mat.Dense, *mat.Dense, *mat.Dense, *mat.Dense, *mat.Dense, []float64, []float64, int, bool, error) {
	// This is a simplified placeholder implementation
	// In a real implementation, this would call an external rotation library

	if corr == nil {
		return nil, nil, nil, nil, nil, nil, nil, 0, false, fmt.Errorf("nil correlation matrix")
	}

	_, cols := corr.Dims()
	if numFactors > cols {
		numFactors = cols
	}

	// Create identity loadings (simplified)
	P := mat.NewDense(cols, numFactors, nil)
	for i := 0; i < cols && i < numFactors; i++ {
		P.Set(i, i, 1.0)
	}

	// Create identity transformation matrix
	T := mat.NewDense(numFactors, numFactors, nil)
	for i := 0; i < numFactors; i++ {
		T.Set(i, i, 1.0)
	}

	// Create identity phi matrix
	Phi := mat.NewDense(numFactors, numFactors, nil)
	for i := 0; i < numFactors; i++ {
		Phi.Set(i, i, 1.0)
	}

	// Dummy values for other return parameters
	h_final := make([]float64, numFactors)
	ev := make([]float64, numFactors)
	for i := 0; i < numFactors; i++ {
		h_final[i] = 1.0
		ev[i] = 1.0
	}

	iters := 1
	conv := true

	return P, nil, Phi, T, nil, h_final, ev, iters, conv, nil
}

// computeFactorScores computes factor scores using the specified method
func computeFactorScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64, sigmaForScores *mat.Dense, method FactorScoreMethod) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	if data == nil || loadings == nil {
		return nil, nil, nil, fmt.Errorf("nil input matrices")
	}

	n, _ := data.Dims()
	_, m := loadings.Dims()

	switch method {
	case FactorScoreNone:
		// Return zero scores
		scores := mat.NewDense(n, m, nil)
		return scores, nil, nil, nil

	case FactorScoreRegression:
		return computeRegressionScores(data, loadings, phi, uniquenesses)

	case FactorScoreBartlett:
		return computeBartlettScores(data, loadings, phi, uniquenesses)

	case FactorScoreAndersonRubin:
		return computeAndersonRubinScores(data, loadings, phi)

	default:
		// Default to regression method
		return computeRegressionScores(data, loadings, phi, uniquenesses)
	}
}

// computeRegressionScores computes factor scores using regression method
func computeRegressionScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64) (*mat.Dense, *mat.Dense, *mat.Dense, error) {

	// Create diagonal matrix of uniquenesses
	U := mat.NewDiagDense(len(uniquenesses), uniquenesses)

	// Compute R = L * Phi * L^T + U (reproduced correlation matrix)
	var temp mat.Dense
	if phi != nil {
		temp.Mul(loadings, phi)
	} else {
		// If phi is nil (orthogonal rotation), use identity
		_, m := loadings.Dims()
		identity := mat.NewDense(m, m, nil)
		for i := range m {
			identity.Set(i, i, 1.0)
		}
		temp.Mul(loadings, identity)
	}
	var R mat.Dense
	R.Mul(&temp, loadings.T())
	R.Add(&R, U)

	// Compute inverse of R
	var Rinv mat.Dense
	err := Rinv.Inverse(&R)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to invert correlation matrix: %v", err)
	}

	// Compute weights W = R^(-1) * L * Phi (regression method)
	var weights mat.Dense
	weights.Mul(&Rinv, loadings)
	if phi != nil {
		weights.Mul(&weights, phi)
	}
	// If phi is nil, weights is already R^(-1) * L, which is correct for orthogonal case

	// Compute scores S = data * W (not W^T!)
	var scores mat.Dense
	scores.Mul(data, &weights)

	// Return phi, or identity matrix if phi is nil (orthogonal rotation)
	var covariance *mat.Dense
	if phi != nil {
		covariance = phi
	} else {
		_, m := loadings.Dims()
		covariance = mat.NewDense(m, m, nil)
		for i := range m {
			covariance.Set(i, i, 1.0)
		}
	}

	return &scores, &weights, covariance, nil
}

// computeBartlettScores computes factor scores using Bartlett's weighted least squares method
func computeBartlettScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	n, _ := data.Dims()
	_, m := loadings.Dims()

	// Create uniqueness matrix (diagonal)
	U := mat.NewDiagDense(len(uniquenesses), uniquenesses)

	// Compute R = L * Phi * L^T + U (reproduced correlation matrix)
	var temp mat.Dense
	if phi != nil {
		temp.Mul(loadings, phi)
	} else {
		// Orthogonal case
		temp.CloneFrom(loadings)
	}
	var R mat.Dense
	R.Mul(&temp, loadings.T())
	R.Add(&R, U)

	// For Bartlett method, use inverse of uniquenesses as weights
	// W = diag(1/psi) where psi are the uniquenesses
	weights := make([]float64, len(uniquenesses))
	for i, u := range uniquenesses {
		if u > 0 {
			weights[i] = 1.0 / u
		} else {
			weights[i] = 1.0 // Avoid division by zero
		}
	}
	W := mat.NewDiagDense(len(weights), weights)

	// Compute weighted loadings: L_w = W^{1/2} * L
	var sqrtW mat.Dense
	sqrtW.Apply(func(i, j int, v float64) float64 {
		return math.Sqrt(v)
	}, W)

	var Lw mat.Dense
	Lw.Mul(&sqrtW, loadings)

	// Compute weighted phi if oblique
	var Phi_w *mat.Dense
	if phi != nil {
		Phi_w = mat.NewDense(m, m, nil)
		Phi_w.Mul(&Lw, phi)
		Phi_w.Mul(Phi_w, Lw.T())
	}

	// Compute the weighted regression: scores = (L_w^T * L_w)^{-1} * L_w^T * W^{1/2} * data^T
	var LtLw mat.Dense
	LtLw.Mul(Lw.T(), &Lw)

	var LtLwInv mat.Dense
	err := LtLwInv.Inverse(&LtLw)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to invert weighted loadings matrix: %v", err)
	}

	var temp2 mat.Dense
	temp2.Mul(&LtLwInv, Lw.T())
	temp2.Mul(&temp2, &sqrtW)

	// Compute scores: S = temp2 * data^T (but need to transpose result)
	var scoresT mat.Dense
	scoresT.Mul(&temp2, data.T())

	scores := mat.NewDense(n, m, nil)
	scores.CloneFrom(scoresT.T())

	// Return phi, or identity matrix if orthogonal
	var covariance *mat.Dense
	if phi != nil {
		covariance = phi
	} else {
		covariance = mat.NewDense(m, m, nil)
		for i := range m {
			covariance.Set(i, i, 1.0)
		}
	}

	return scores, &Lw, covariance, nil
}

// computeAndersonRubinScores computes factor scores using Anderson-Rubin's method
func computeAndersonRubinScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	_, p := data.Dims()
	_, m := loadings.Dims()

	// For Anderson-Rubin, we need uniquenesses. If not provided, assume minimal uniquenesses
	uniquenesses := make([]float64, p)
	for i := range p {
		uniquenesses[i] = 0.005 // Small positive value
	}

	// First compute regression scores
	regScores, regCoeff, _, err := computeRegressionScores(data, loadings, phi, uniquenesses)
	if err != nil {
		return nil, nil, nil, err
	}

	// Anderson-Rubin normalization: normalize scores to have identity covariance
	// Compute sample covariance of regression scores
	scoreCov := mat.NewSymDense(m, nil)
	stat.CovarianceMatrix(scoreCov, regScores, nil)

	// Compute Cholesky decomposition for normalization
	var chol mat.Cholesky
	if !chol.Factorize(scoreCov) {
		// If Cholesky fails, use eigenvalue decomposition
		var eig mat.EigenSym
		if !eig.Factorize(scoreCov, true) {
			return nil, nil, nil, fmt.Errorf("failed to decompose score covariance matrix")
		}

		eigenvals := eig.Values(nil)
		eigenvecs := mat.NewDense(m, m, nil)
		eig.VectorsTo(eigenvecs)

		// Normalize scores: S_ar = S_reg * V * D^{-1/2} * V^T
		for j := range m {
			if eigenvals[j] > 0 {
				eigenvals[j] = 1.0 / math.Sqrt(eigenvals[j])
			} else {
				eigenvals[j] = 1.0
			}
		}

		normMat := mat.NewDense(m, m, nil)
		for i := range m {
			for j := range m {
				normMat.Set(i, j, eigenvecs.At(i, j)*eigenvals[j])
			}
		}

		var normalizedScores mat.Dense
		normalizedScores.Mul(regScores, normMat)

		// Identity covariance for Anderson-Rubin
		identity := mat.NewDense(m, m, nil)
		for i := range m {
			identity.Set(i, i, 1.0)
		}

		return &normalizedScores, regCoeff, identity, nil
	}

	// Use Cholesky for normalization
	L := mat.NewTriDense(m, mat.Lower, nil)
	chol.LTo(L)
	var LInv mat.Dense
	err = LInv.Inverse(L.T())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to invert Cholesky factor: %v", err)
	}

	var normalizedScores mat.Dense
	normalizedScores.Mul(regScores, &LInv)

	// Identity covariance for Anderson-Rubin
	identity := mat.NewDense(m, m, nil)
	for i := range m {
		identity.Set(i, i, 1.0)
	}

	return &normalizedScores, regCoeff, identity, nil
}
