package stats

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/fa"
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
	MaxIter    int     // Optional: default 50
	MinErr     float64 // Optional: default 0.001 (R's min.err)
}

// -------------------------
// Result Structs
// -------------------------

// FactorAnalysisResult contains the output of factor analysis
type FactorAnalysisResult struct {
	Loadings             insyra.IDataTable // Loading matrix (variables × factors)
	Structure            insyra.IDataTable // Structure matrix (variables × factors)
	Uniquenesses         insyra.IDataTable // Uniqueness vector (p × 1)
	Communalities        insyra.IDataTable // Communality table (p × 1: Extraction)
	SamplingAdequacy     insyra.IDataTable // KMO overall index and per-variable MSA values
	BartlettTest         insyra.IDataTable // Bartlett's test of sphericity summary
	Phi                  insyra.IDataTable // Factor correlation matrix (m × m), nil for orthogonal
	RotationMatrix       insyra.IDataTable // Rotation matrix (m × m), nil if no rotation
	Eigenvalues          insyra.IDataTable // Eigenvalues vector (p × 1)
	ExplainedProportion  insyra.IDataTable // Proportion explained by each factor (m × 1)
	CumulativeProportion insyra.IDataTable // Cumulative proportion explained (m × 1)
	Scores               insyra.IDataTable // Factor scores (n × m), nil if not computed
	ScoreCoefficients    insyra.IDataTable // Factor score coefficient matrix (variables × factors)
	ScoreCovariance      insyra.IDataTable // Factor score covariance matrix (factors × factors)

	Converged         bool
	RotationConverged bool
	Iterations        int
	CountUsed         int
	Messages          []string
}

const (
	tableNameFactorLoadings          = "FactorLoadings"
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
	insyra.Show(formatLabelPascalWithSpaces(tableNameCommunalities), r.Communalities, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameSamplingAdequacy), r.SamplingAdequacy, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameBartlettTest), r.BartlettTest, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameEigenvalues), r.Eigenvalues, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameExplainedProportion), r.ExplainedProportion, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameCumulativeProportion), r.CumulativeProportion, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorLoadings), r.Loadings, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorStructure), r.Structure, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNamePhiMatrix), r.Phi, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameRotationMatrix), r.RotationMatrix, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorScoreCoefficients), r.ScoreCoefficients, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorScoreCovariance), r.ScoreCovariance, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorScores), r.Scores, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameUniqueness), r.Uniquenesses, startEndRange...)
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
		MaxIter: 50,                    // R default: 50
		MinErr:  0.001,                 // R default: 0.001
	}
}

// Internal constants aligned with R's psych::fa and GPArotation package
const (
	// Convergence tolerance for extraction methods (PAF, ML, MINRES)
	// R psych uses different tolerances for different contexts
	extractionTolerance = 1e-6 // General convergence tolerance for factor extraction

	// Numerical stability constants
	epsilonMedium = 1e-6 // For communality lower bound and sum checks

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
	var bartlettTable *insyra.DataTable
	if corrForAdequacy != nil {
		corrAdequacyDense := mat.DenseCopyOf(corrForAdequacy)
		overallKMO, msaValues, kmoErr := computeKMOMeasures(corrAdequacyDense)
		if kmoErr != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "failed to compute KMO/MSA: %v", kmoErr)
		} else {
			samplingAdequacyTable = kmoToDataTable(overallKMO, msaValues, colNames)
		}

		if chi, pval, df, bartErr := computeBartlettFromCorrelation(corrAdequacyDense, rowNum); bartErr != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "failed to compute Bartlett's test: %v", bartErr)
		} else {
			bartlettTable = bartlettToDataTable(chi, df, pval, rowNum)
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
	loadings, converged, iterations, err := extractFactors(data, corrDense, sortedEigenvalues, sortedEigenvectors, numFactors, opt, rowNum, tolVal)
	if err != nil {
		insyra.LogWarning("stats", "FactorAnalysis", "factor extraction failed: %v", err)
		return nil
	}

	initialCommunalities := make([]float64, colNum)
	switch opt.Extraction {
	case FactorExtractionPCA:
		for i := 0; i < colNum; i++ {
			val := corrDense.At(i, i)
			if val < 0 {
				val = 0
			}
			initialCommunalities[i] = val
		}
	default:
		minErr := opt.MinErr
		if minErr <= 0 {
			minErr = epsilonMedium
		}
		if initComm, commErr := initialCommunalitiesSMC(corrDense); commErr != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "failed to compute initial communalities (SMC): %v", commErr)
			for i := 0; i < colNum; i++ {
				val := corrDense.At(i, i)
				if val < 0 {
					val = 0
				}
				initialCommunalities[i] = val
			}
		} else {
			copy(initialCommunalities, initComm)
		}
	}

	// Sanity check: inspect unrotated loadings before any rotation is applied
	if loadings != nil {
		pVars, mFactors := loadings.Dims()
		if pVars > 0 && mFactors > 0 {
			maxAbs := 0.0
			for i := 0; i < pVars; i++ {
				for j := 0; j < mFactors; j++ {
					val := math.Abs(loadings.At(i, j))
					if val > maxAbs {
						maxAbs = val
					}
				}
			}
			sampleVars := min(2, pVars)
			sampleFactors := min(2, mFactors)
			buffer := make([]float64, 0, sampleVars*sampleFactors)
			for i := 0; i < sampleVars; i++ {
				for j := 0; j < sampleFactors; j++ {
					buffer = append(buffer, loadings.At(i, j))
				}
			}
			insyra.LogInfo("stats", "FactorAnalysis", "pre-rotation loadings |max|=%.3f, samples=%v", maxAbs, buffer)
		}
	}

	// Apply initial factor reflection to match R's convention (before rotation)
	loadings, _ = reflectFactorsForPositiveLoadings(loadings)

	// Special handling for PAF + Oblimin to match SPSS
	var useSPSSPAFOblimin bool = (opt.Extraction == FactorExtractionPAF && opt.Rotation.Method == FactorRotationOblimin)
	var extractionCommunalities []float64
	var extractionEigenvalues []float64
	if useSPSSPAFOblimin {
		extractionCommunalities = make([]float64, colNum)
	}

	// Step 7: Rotate factors
	var rotatedLoadings *mat.Dense
	var rotationMatrix *mat.Dense
	var phi *mat.Dense
	var rotationConverged bool
	if useSPSSPAFOblimin {
		// Use SPSS-compatible PAF + Oblimin implementation
		corrDense := mat.NewDense(colNum, colNum, nil)
		for i := 0; i < colNum; i++ {
			for j := 0; j < colNum; j++ {
				corrDense.Set(i, j, corrMatrix.At(i, j))
			}
		}
		// Set initial communalities using SMC for SPSS PAF
		if initComm, commErr := initialCommunalitiesSMC(corrDense); commErr == nil {
			initialCommunalities = initComm
		}
		P, _, Phi, T, _, h_final, ev, iters, conv, err := FactorPAFOblimin(corrDense, numFactors, opt.Rotation.Delta, 0.001, opt.MaxIter, 1.0)
		if err != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "SPSS PAF+Oblimin failed: %v", err)
			rotatedLoadings, rotationMatrix, phi, rotationConverged, err = rotateFactors(loadings, opt.Rotation)
		} else {
			rotatedLoadings = P
			rotationMatrix = T
			phi = Phi
			rotationConverged = true
			iterations = iters
			converged = conv
			extractionEigenvalues = ev
			insyra.LogInfo("stats", "FactorAnalysis", "SPSS PAF+Oblimin completed: iterations=%d, converged=%v", iters, conv)
			// Update communalities from final h for SPSS compatibility (extraction communalities)
			for i := 0; i < colNum; i++ {
				extractionCommunalities[i] = h_final.At(i, 0)
			}
		}
	} else if opt.Rotation.Method != FactorRotationNone {
		rotatedLoadings, rotationMatrix, phi, rotationConverged, err = rotateFactors(loadings, opt.Rotation)
		if err != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "rotation failed: %v", err)
			rotatedLoadings = loadings
			rotationMatrix = nil
			phi = nil
			rotationConverged = true
		}
		// Note: rotateFactors now handles sign standardization internally
	} else {
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
			if phiMat == nil {
				for j := 0; j < numFactors; j++ {
					v := rotatedLoadings.At(i, j)
					hi2 += v * v
				}
			} else {
				rowVec := mat.NewVecDense(numFactors, nil)
				for j := 0; j < numFactors; j++ {
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
	for i := 0; i < numFactors; i++ {
		factorColNames[i] = fmt.Sprintf("Factor_%d", i+1)
	}

	// Step 9: Compute explained proportions for "Extraction SS loadings"
	// SPSS reports three blocks: Initial eigenvalues, Extraction SS loadings,
	// and Rotation SS loadings. Our single "ExplainedProportion" field should
	// reflect the Extraction SS loadings (not the rotated SS loadings).
	pVars, mFactors := rotatedLoadings.Dims()
	explainedProp := make([]float64, mFactors)
	cumulativeProp := make([]float64, mFactors)

	if useSPSSPAFOblimin && len(extractionEigenvalues) == mFactors {
		// Use eigenvalues of R* from the extraction iterations
		cum := 0.0
		for j := 0; j < mFactors; j++ {
			prop := extractionEigenvalues[j] / float64(pVars) * 100.0
			explainedProp[j] = prop
			cum += prop
			cumulativeProp[j] = cum
		}
	} else {
		// Fallback for other methods: use unrotated SS loadings as extraction
		// SS (pre-rotation loadings: structure == pattern)
		ssLoad := make([]float64, mFactors)
		for j := 0; j < mFactors; j++ {
			sum := 0.0
			for i := 0; i < pVars; i++ {
				v := loadings.At(i, j)
				sum += v * v
			}
			ssLoad[j] = sum
		}
		totalVar := float64(pVars)
		cum := 0.0
		for j := 0; j < mFactors; j++ {
			prop := ssLoad[j] / totalVar * 100.0
			explainedProp[j] = prop
			cum += prop
			cumulativeProp[j] = cum
		}
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
		Structure:            structureTable,
		Uniquenesses:         vectorToDataTableWithNames(uniquenesses, tableNameUniqueness, "Uniqueness", colNames),
		Communalities:        communalitiesTable,
		SamplingAdequacy:     samplingAdequacyTable,
		BartlettTest:         bartlettTable,
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
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions, sampleSize int, tol float64) (*mat.Dense, bool, int, error) {
	switch opt.Extraction {
	case FactorExtractionPCA:
		return extractPCA(eigenvalues, eigenvectors, numFactors)

	case FactorExtractionPAF:
		return extractPAF(corrMatrix, numFactors, opt.MaxIter, tol, opt.MinErr)

	case FactorExtractionML:
		return extractML(corrMatrix, numFactors, opt.MaxIter, tol, sampleSize)

	case FactorExtractionMINRES:
		return extractMINRES(corrMatrix, numFactors, opt.MaxIter, tol)

	default:
		// Default to MINRES to match R psych::fa and the documented default behavior.
		return extractMINRES(corrMatrix, numFactors, opt.MaxIter, tol)
	}
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
	for i := 0; i < p; i++ {
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

// initialCommunalitiesSMC computes initial communalities using Squared Multiple Correlation
func initialCommunalitiesSMC(corr *mat.Dense) ([]float64, error) {
	// Compute SMC (Squared Multiple Correlation) estimates
	// SMC_i = 1 - 1/R_ii^inv where R_ii^inv is diagonal of inverse correlation matrix
	h2, err := computeSMC(corr)
	if err != nil {
		return nil, err
	}
	insyra.LogInfo("stats", "FactorAnalysis", "SMC computed: %v", h2)
	return h2, nil
}

// computeSMC computes Squared Multiple Correlation estimates using internal/fa package
func computeSMC(corr *mat.Dense) ([]float64, error) {
	if corr == nil {
		return nil, fmt.Errorf("nil correlation matrix")
	}
	// SMC_i = 1 - 1 / (R^{-1})_{ii}
	// This mirrors SPSS/psych behavior when using a correlation matrix.
	var inv mat.Dense
	if err := inv.Inverse(corr); err != nil {
		return nil, fmt.Errorf("SMC: failed to invert correlation: %w", err)
	}
	p, q := inv.Dims()
	if p != q {
		return nil, fmt.Errorf("SMC: inverse not square")
	}
	h2 := make([]float64, p)
	for i := 0; i < p; i++ {
		d := inv.At(i, i)
		if d == 0 {
			h2[i] = 0
			continue
		}
		v := 1.0 - 1.0/d
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		h2[i] = v
	}
	return h2, nil
}

// extractPAF extracts factors using Principal Axis Factoring
// This implements SPSS-style PAF using PCA with communality adjustment
func extractPAF(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64, minErr float64) (*mat.Dense, bool, int, error) {
	p, _ := corrMatrix.Dims()
	if numFactors <= 0 || numFactors > p {
		return nil, false, 0, fmt.Errorf("invalid number of factors: %d", numFactors)
	}

	// Initialize communalities using Kaiser normalization approach
	communalities := make([]float64, p)
	// Calculate average absolute correlation as initial communality estimate
	avgCorr := 0.0
	count := 0
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if i != j {
				avgCorr += math.Abs(corrMatrix.At(i, j))
				count++
			}
		}
	}
	if count > 0 {
		avgCorr /= float64(count)
	}
	initialComm := math.Min(0.7, math.Max(0.3, avgCorr*2)) // Scale and bound
	for i := 0; i < p; i++ {
		communalities[i] = initialComm
	}

	// Iterative PAF algorithm (SPSS style)
	iterations := 0
	converged := false
	var loadings *mat.Dense

	for iterations < maxIter {
		iterations++

		// Create reduced correlation matrix R* = R - diag(1 - communalities)
		Rstar := mat.NewDense(p, p, nil)
		Rstar.Copy(corrMatrix)
		for i := 0; i < p; i++ {
			Rstar.Set(i, i, communalities[i])
		}

		// Eigen decomposition of R*
		var eig mat.EigenSym
		symRstar := denseToSym(Rstar)
		if !eig.Factorize(symRstar, true) {
			return nil, false, iterations, errors.New("eigen decomposition failed")
		}

		eigenvalues := eig.Values(nil)
		var eigenvectors mat.Dense
		eig.VectorsTo(&eigenvectors)

		// Sort eigenvalues and eigenvectors in descending order
		type eigenPair struct {
			value  float64
			vector []float64
		}
		pairs := make([]eigenPair, p)
		for i := 0; i < p; i++ {
			vec := make([]float64, p)
			for j := 0; j < p; j++ {
				vec[j] = eigenvectors.At(j, i)
			}
			pairs[i] = eigenPair{value: eigenvalues[i], vector: vec}
		}
		for i := 0; i < len(pairs)-1; i++ {
			for j := i + 1; j < len(pairs); j++ {
				if pairs[i].value < pairs[j].value {
					pairs[i], pairs[j] = pairs[j], pairs[i]
				}
			}
		}

		// Extract loadings: L = V * sqrt(Lambda) for first numFactors
		loadings = mat.NewDense(p, numFactors, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < numFactors; j++ {
				loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(math.Max(0, pairs[j].value)))
			}
		}

		// Update communalities: h^2 = sum(L^2) for each variable
		newCommunalities := make([]float64, p)
		maxChange := 0.0
		for i := 0; i < p; i++ {
			sumSq := 0.0
			for j := 0; j < numFactors; j++ {
				sumSq += loadings.At(i, j) * loadings.At(i, j)
			}
			newCommunalities[i] = math.Min(0.995, sumSq) // Cap at 0.995 to avoid singularity
			change := math.Abs(newCommunalities[i] - communalities[i])
			if change > maxChange {
				maxChange = change
			}
		}

		// Check convergence
		if maxChange < tol {
			converged = true
			break
		}

		// Update communalities for next iteration
		copy(communalities, newCommunalities)
	}

	// If not converged, use final iteration results
	if !converged {
		// Final extraction with last communalities
		Rstar := mat.NewDense(p, p, nil)
		Rstar.Copy(corrMatrix)
		for i := 0; i < p; i++ {
			Rstar.Set(i, i, communalities[i])
		}

		var eig mat.EigenSym
		symRstar := denseToSym(Rstar)
		if !eig.Factorize(symRstar, true) {
			return nil, false, iterations, errors.New("final eigen decomposition failed")
		}

		eigenvalues := eig.Values(nil)
		var eigenvectors mat.Dense
		eig.VectorsTo(&eigenvectors)

		// Sort eigenvalues and eigenvectors
		type eigenPair struct {
			value  float64
			vector []float64
		}
		pairs := make([]eigenPair, p)
		for i := 0; i < p; i++ {
			vec := make([]float64, p)
			for j := 0; j < p; j++ {
				vec[j] = eigenvectors.At(j, i)
			}
			pairs[i] = eigenPair{value: eigenvalues[i], vector: vec}
		}
		for i := 0; i < len(pairs)-1; i++ {
			for j := i + 1; j < len(pairs); j++ {
				if pairs[i].value < pairs[j].value {
					pairs[i], pairs[j] = pairs[j], pairs[i]
				}
			}
		}

		// Final loadings
		loadings := mat.NewDense(p, numFactors, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < numFactors; j++ {
				loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(math.Max(0, pairs[j].value)))
			}
		}
	}

	return loadings, converged, iterations, nil
}

// FAfn computes the negative log-likelihood for ML factor analysis
// Mirrors R's psych::FAfn exactly
func FAfn(Psi []float64, S *mat.Dense, nf int) float64 {
	p, _ := S.Dims()

	// sc <- diag(1/sqrt(Psi))
	sc := make([]float64, p)
	for i := 0; i < p; i++ {
		if Psi[i] <= 0 {
			return math.Inf(1)
		}
		sc[i] = 1.0 / math.Sqrt(Psi[i])
	}
	scMat := mat.NewDiagDense(p, sc)

	// Sstar <- sc %*% S %*% sc
	Sstar := mat.NewDense(p, p, nil)
	Sstar.Mul(scMat, S)
	Sstar.Mul(Sstar, scMat)

	// E <- eigen(Sstar, symmetric = TRUE, only.values = TRUE)
	var eig mat.EigenSym
	symSstar := denseToSym(Sstar)
	if !eig.Factorize(symSstar, true) {
		return math.Inf(1)
	}
	eigenvalues := eig.Values(nil)

	// e <- E$values[-(1:nf)]
	e := make([]float64, p-nf)
	for i := nf; i < p; i++ {
		e[i-nf] = eigenvalues[i]
		if e[i-nf] <= 0 {
			return math.Inf(1)
		}
	}

	// e <- sum(log(e) - e) - nf + nrow(S)
	sumLog := 0.0
	sumE := 0.0
	for _, v := range e {
		sumLog += math.Log(v)
		sumE += v
	}
	result := sumLog - sumE - float64(nf) + float64(p)

	// Return -result (since optim minimizes)
	return -result
}

// mlFitStats computes ML fit statistics: f, chi-square, df, and p-value
func mlFitStats(R, Sigma mat.Matrix, n, p, m int) (f, chi2, pval float64, df int) {
	// Compute inv(Sigma) * R
	var invSigma mat.Dense
	if err := invSigma.Inverse(Sigma); err != nil {
		// If inversion fails, return NaN
		return math.NaN(), math.NaN(), math.NaN(), 0
	}

	var temp mat.Dense
	temp.Mul(&invSigma, R)

	// Trace
	diag := Diag(&temp)
	diagSlice := diag.([]float64)
	tr := 0.0
	for _, v := range diagSlice {
		tr += v
	}

	// Log determinant
	det := mat.Det(&temp)
	if det <= 0 {
		return math.NaN(), math.NaN(), math.NaN(), 0
	}
	logDet := math.Log(det)

	// f = tr(Sigma^{-1} R) - log|Sigma^{-1} R| - p
	f = tr - logDet - float64(p)

	// Degrees of freedom
	df = ((p-m)*(p-m) - (p + m)) / 2
	if df <= 0 {
		return f, 0, 1.0, df
	}

	// Chi-square statistic
	chi2 = (float64(n-1) - (2*float64(p)+5)/6.0 - (2*float64(m))/3.0) * f

	// p-value
	if chi2 < 0 {
		chi2 = 0
	}
	chiSqDist := distuv.ChiSquared{K: float64(df)}
	pval = 1.0 - chiSqDist.CDF(chi2)

	return f, chi2, pval, df
}

// extractML extracts factors using Maximum Likelihood estimation
func extractML(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64, sampleSize int) (*mat.Dense, bool, int, error) {
	p, _ := corrMatrix.Dims()
	if numFactors <= 0 || numFactors > p {
		return nil, false, 0, fmt.Errorf("invalid number of factors: %d", numFactors)
	}

	if maxIter <= 0 {
		maxIter = 100
	}
	if tol <= 0 {
		tol = extractionTolerance
	}

	// Initialize with PAF
	initial, _, _, err := extractPAF(corrMatrix, numFactors, min(maxIter, 50), math.Max(tol, extractionTolerance), 0.001)
	if err != nil || initial == nil {
		// Fall back to PCA loadings
		var eig mat.EigenSym
		symCorr := denseToSym(corrMatrix)
		if !eig.Factorize(symCorr, true) {
			return nil, false, 0, fmt.Errorf("ml: eigen factorization failed: %w", err)
		}
		eigs := eig.Values(nil)
		var vec mat.Dense
		eig.VectorsTo(&vec)
		initial, _, _, err = extractPCA(eigs, &vec, numFactors)
		if err != nil {
			return nil, false, 0, fmt.Errorf("ml: unable to build initial loadings: %w", err)
		}
	}

	// Log initial loadings for debugging
	if p >= 4 {
		insyra.LogDebug("stats", "ML", "initial loadings[0,0:2] = %.3f, %.3f",
			initial.At(0, 0), initial.At(0, min(1, numFactors-1)))
		// Compute initial communalities
		initialComm := make([]float64, min(4, p))
		for i := 0; i < min(4, p); i++ {
			sum := 0.0
			for j := 0; j < numFactors; j++ {
				sum += initial.At(i, j) * initial.At(i, j)
			}
			initialComm[i] = sum
		}
		insyra.LogInfo("stats", "ML", "initial communalities[0:4] = %.3f, %.3f, %.3f, %.3f",
			initialComm[0], initialComm[min(1, len(initialComm)-1)], initialComm[min(2, len(initialComm)-1)], initialComm[min(3, len(initialComm)-1)])
	}

	loadings := mat.Dense{}
	loadings.CloneFrom(initial)

	psMin := epsilonMedium // Aligned with R psych constant
	// baseRidge := 1e-5
	// innerRidge := 1e-6
	converged := false

	for iter := 0; iter < maxIter; iter++ {
		psi := make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < numFactors; j++ {
				val := loadings.At(i, j)
				sum += val * val
			}
			res := corrMatrix.At(i, i) - sum
			if res < psMin {
				res = psMin
			}
			psi[i] = res
		}

		var sigma mat.Dense
		sigma.Mul(&loadings, loadings.T())
		for i := 0; i < p; i++ {
			val := sigma.At(i, i) + psi[i]
			// Remove ridge regularization - it biases the results
			// if baseRidge > 0 {
			// 	val += baseRidge
			// }
			sigma.Set(i, i, val)
		}

		var invSigma mat.Dense
		if err := invSigma.Inverse(&sigma); err != nil {
			// If inversion fails, add minimal ridge
			for i := 0; i < p; i++ {
				sigma.Set(i, i, sigma.At(i, i)+psMin)
			}
			if err := invSigma.Inverse(&sigma); err != nil {
				return nil, false, iter, fmt.Errorf("ml: covariance inversion failed: %w", err)
			}
		}

		var t mat.Dense
		t.Mul(&invSigma, &loadings)

		var loadingsTrans mat.Dense
		loadingsTrans.CloneFrom(loadings.T())

		var m mat.Dense
		m.Mul(&loadingsTrans, &t)
		for i := 0; i < numFactors; i++ {
			diag := m.At(i, i) + 1.0
			// Remove inner ridge - it biases the results
			// if innerRidge > 0 {
			// 	diag += innerRidge
			// }
			m.Set(i, i, diag)
		}

		invSqrt, err := inverseSqrtDense(&m)
		if err != nil {
			// If inverse sqrt fails, add minimal ridge and retry
			var adjusted mat.Dense
			adjusted.CloneFrom(&m)
			for i := 0; i < numFactors; i++ {
				adjusted.Set(i, i, adjusted.At(i, i)+psMin)
			}
			invSqrt, err = inverseSqrtDense(&adjusted)
			if err != nil {
				return nil, false, iter, fmt.Errorf("ml: inverse sqrt failed: %w", err)
			}
		}

		var rt mat.Dense
		rt.Mul(corrMatrix, &t)

		var newLoadings mat.Dense
		newLoadings.Mul(&rt, invSqrt)

		maxChange := 0.0
		for i := 0; i < p; i++ {
			for j := 0; j < numFactors; j++ {
				delta := math.Abs(newLoadings.At(i, j) - loadings.At(i, j))
				if delta > maxChange {
					maxChange = delta
				}
			}
		}

		loadings.CloneFrom(&newLoadings)

		// Compute current communalities for logging
		if iter < 5 {
			currComm := make([]float64, min(4, p))
			for i := 0; i < min(4, p); i++ {
				sum := 0.0
				for j := 0; j < numFactors; j++ {
					sum += loadings.At(i, j) * loadings.At(i, j)
				}
				currComm[i] = sum
			}
			insyra.LogInfo("stats", "ML", "iter %d: communalities[0:4] = %.3f, %.3f, %.3f, %.3f",
				iter+1, currComm[0], currComm[min(1, len(currComm)-1)], currComm[min(2, len(currComm)-1)], currComm[min(3, len(currComm)-1)])
		}

		// Compute ML fit statistics
		if sampleSize > 0 {
			f, chi2, pval, df := mlFitStats(corrMatrix, &sigma, sampleSize, p, numFactors)
			if !math.IsNaN(f) && iter < 5 || iter == maxIter-1 {
				insyra.LogDebug("stats", "ML", "iter %d: f=%.4f, χ²=%.4f (df=%d, p=%.4f)", iter+1, f, chi2, df, pval)
			}
		}

		// Log convergence progress
		if iter < 5 || iter == maxIter-1 {
			insyra.LogDebug("stats", "ML", "iter %d: maxChange=%.6f, loadings[0,0]=%.4f",
				iter+1, maxChange, loadings.At(0, 0))
		}

		if maxChange < tol {
			converged = true
			// Report final fit statistics
			if sampleSize > 0 {
				f, chi2, pval, df := mlFitStats(corrMatrix, &sigma, sampleSize, p, numFactors)
				if !math.IsNaN(f) {
					insyra.LogInfo("stats", "ML", "converged: f=%.4f, χ²=%.4f (df=%d, p=%.4f)", f, chi2, df, pval)
				}
			}
			insyra.LogInfo("stats", "ML", "converged in %d iterations", iter+1)
			return &loadings, converged, iter + 1, nil
		}
	}

	return &loadings, converged, maxIter, nil
}

// extractMINRES extracts factors using Minimum Residual (MINRES) method
// This is a simplified implementation that minimizes residual correlations
func extractMINRES(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64) (*mat.Dense, bool, int, error) {
	p, _ := corrMatrix.Dims()
	if numFactors <= 0 || numFactors > p {
		return nil, false, 0, fmt.Errorf("invalid number of factors: %d", numFactors)
	}

	if maxIter <= 0 {
		maxIter = 200
	}
	if tol <= 0 {
		tol = extractionTolerance
	}

	initial, _, _, err := extractPAF(corrMatrix, numFactors, min(maxIter, 100), math.Max(tol, extractionTolerance), 0.001)
	if err != nil || initial == nil {
		var eig mat.EigenSym
		symCorr := denseToSym(corrMatrix)
		if !eig.Factorize(symCorr, true) {
			return nil, false, 0, fmt.Errorf("eigenvalue decomposition failed")
		}
		eigenvalues := eig.Values(nil)
		var eigenvectors mat.Dense
		eig.VectorsTo(&eigenvectors)
		initial, err = computePCALoadings(eigenvalues, &eigenvectors, numFactors)
		if err != nil {
			return nil, false, 0, fmt.Errorf("PCA fallback failed: %v", err)
		}
	}

	var loadings mat.Dense
	loadings.CloneFrom(initial)

	prevSSE := math.Inf(1)
	stepSize := 1.0

	for iter := 0; iter < maxIter; iter++ {
		var reproduced mat.Dense
		reproduced.Mul(&loadings, loadings.T())

		var residual mat.Dense
		residual.CloneFrom(corrMatrix)
		residual.Sub(&residual, &reproduced)
		zeroDiagonal(&residual)

		currentSSE := offDiagonalSSE(&residual)
		if currentSSE <= tol {
			return &loadings, true, iter + 1, nil
		}
		if iter > 0 && math.Abs(currentSSE-prevSSE) < tol {
			return &loadings, true, iter + 1, nil
		}

		var grad mat.Dense
		grad.Mul(&residual, &loadings)
		grad.Scale(-4.0, &grad)
		if mat.Norm(&grad, 2) < tol {
			return &loadings, true, iter + 1, nil
		}

		success := false
		trialStep := stepSize
		candidate := mat.NewDense(p, numFactors, nil)
		oldSSE := currentSSE

		for attempt := 0; attempt < 20; attempt++ {
			for i := 0; i < p; i++ {
				for j := 0; j < numFactors; j++ {
					update := loadings.At(i, j) - trialStep*grad.At(i, j)
					candidate.Set(i, j, update)
				}
			}

			var candReproduced mat.Dense
			candReproduced.Mul(candidate, candidate.T())
			var candResidual mat.Dense
			candResidual.CloneFrom(corrMatrix)
			candResidual.Sub(&candResidual, &candReproduced)
			zeroDiagonal(&candResidual)
			candSSE := offDiagonalSSE(&candResidual)

			if candSSE < oldSSE {
				loadings.CloneFrom(candidate)
				prevSSE = oldSSE
				stepSize = math.Min(trialStep*1.5, 5.0)
				success = true
				break
			}

			trialStep *= 0.5
		}

		if !success {
			return &loadings, false, iter + 1, nil
		}
	}

	return &loadings, false, maxIter, nil
}

// rotateFactors performs factor rotation using the specified method
func rotateFactors(loadings *mat.Dense, rotationOpts FactorRotationOptions) (*mat.Dense, *mat.Dense, *mat.Dense, bool, error) {
	if loadings == nil {
		return nil, nil, nil, false, fmt.Errorf("loadings cannot be nil")
	}

	// For no rotation, return original loadings
	if rotationOpts.Method == FactorRotationNone {
		_, c := loadings.Dims()
		rotationMatrix := mat.NewDense(c, c, nil)
		for i := 0; i < c; i++ {
			rotationMatrix.Set(i, i, 1.0)
		}
		phi := mat.NewDense(c, c, nil)
		for i := 0; i < c; i++ {
			phi.Set(i, i, 1.0)
		}
		reflected, signs := reflectFactorsForPositiveLoadings(loadings)
		applyReflectionToRotationAndPhi(rotationMatrix, phi, signs)
		return reflected, rotationMatrix, phi, true, nil
	}

	// Use the fa package for rotation
	restarts := rotationOpts.Restarts
	if restarts <= 0 {
		restarts = 1
	}
	ops := &fa.RotOpts{
		Eps:         1e-8, // Tighter tolerance for SPSS compatibility
		MaxIter:     5000,
		Alpha0:      1.0,
		Gamma:       rotationOpts.Delta, // For oblimin
		PromaxPower: int(rotationOpts.Kappa),
		Restarts:    restarts,
	}

	rotatedLoadings, rotMat, phi, rotationConverged, err := fa.Rotate(loadings, string(rotationOpts.Method), ops)
	if err != nil {
		return nil, nil, nil, false, err
	}

	// Recover T from rotMat when available for oblique handling
	var rotOut *mat.Dense = rotMat
	if rotMat != nil {
		var rotT mat.Dense
		rotT.CloneFrom(rotMat)
		rotT.T()
		var Trec mat.Dense
		if err := Trec.Inverse(&rotT); err == nil {
			rotOut = &Trec
		}
	}
	// Compute Phi if adapter did not return it (oblique case)
	if phi == nil && rotOut != nil {
		var TT mat.Dense
		TT.CloneFrom(rotOut)
		TT.T()
		var phiComputed mat.Dense
		phiComputed.Mul(&TT, rotOut)
		phi = &phiComputed
	}

	// Apply sign standardization and propagate reflections to rotation/phi matrices
	rotatedLoadings, signs := reflectFactorsForPositiveLoadings(rotatedLoadings)
	applyReflectionToRotationAndPhi(rotOut, phi, signs)

	return rotatedLoadings, rotOut, phi, rotationConverged, nil
}

// denseToSym converts a *mat.Dense to *mat.SymDense
func denseToSym(m *mat.Dense) *mat.SymDense {
	r, c := m.Dims()
	if r != c {
		panic("matrix must be square")
	}
	sym := mat.NewSymDense(r, nil)
	for i := 0; i < r; i++ {
		for j := i; j < r; j++ {
			sym.SetSym(i, j, m.At(i, j))
		}
	}
	return sym
}

// inverseSqrtDense computes the inverse square root of a symmetric matrix
func inverseSqrtDense(m *mat.Dense) (*mat.Dense, error) {
	r, c := m.Dims()
	if r != c {
		return nil, fmt.Errorf("matrix must be square")
	}

	sym := denseToSym(m)
	var eig mat.EigenSym
	if !eig.Factorize(sym, true) {
		return nil, fmt.Errorf("failed to factorize matrix")
	}

	vals := eig.Values(nil)
	var vec mat.Dense
	eig.VectorsTo(&vec)

	d := mat.NewDense(r, r, nil)
	for i := 0; i < r; i++ {
		val := vals[i]
		if val <= 0 {
			return nil, fmt.Errorf("matrix not positive definite")
		}
		invSqrt := 1.0 / math.Sqrt(val)
		d.Set(i, i, invSqrt)
	}

	var temp mat.Dense
	temp.Mul(&vec, d)

	var result mat.Dense
	result.Mul(&temp, vec.T())
	return &result, nil
}

// zeroDiagonal sets the diagonal elements of a matrix to zero
func zeroDiagonal(m *mat.Dense) {
	r, c := m.Dims()
	limit := min(r, c)
	for i := 0; i < limit; i++ {
		m.Set(i, i, 0)
	}
}

// offDiagonalSSE computes the sum of squared errors for off-diagonal elements
func offDiagonalSSE(m *mat.Dense) float64 {
	r, c := m.Dims()
	if r != c {
		return 0
	}
	sum := 0.0
	for i := 0; i < r; i++ {
		for j := i + 1; j < c; j++ {
			val := m.At(i, j)
			sum += val * val
		}
	}
	return sum * 2
}

// sortFactorsByExplainedVariance sorts factors by explained variance (sum of squared loadings)
// Returns sorted loadings, rotation matrix, and phi matrix
func sortFactorsByExplainedVariance(loadings, rotationMatrix, phi *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil {
		return nil, rotationMatrix, phi
	}

	r, c := loadings.Dims()
	variances := make([]float64, c)
	indices := make([]int, c)

	// For oblique rotation, follow SPSS/psych convention: use structure matrix
	// S = P * Phi; for orthogonal, S = P
	var S mat.Dense
	if phi != nil {
		var tmp mat.Dense
		tmp.Mul(loadings, phi)
		S.CloneFrom(&tmp)
	} else {
		S.CloneFrom(loadings)
	}

	// Calculate explained variance proxy per factor: SS of structure loadings
	for j := 0; j < c; j++ {
		sum := 0.0
		for i := 0; i < r; i++ {
			val := S.At(i, j)
			sum += val * val
		}
		variances[j] = sum
		indices[j] = j
	}

	// Sort indices by variance in descending order
	for i := 0; i < c-1; i++ {
		for j := i + 1; j < c; j++ {
			if variances[indices[i]] < variances[indices[j]] {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	// Reorder columns in loadings
	sortedLoadings := mat.NewDense(r, c, nil)
	for j := 0; j < c; j++ {
		for i := 0; i < r; i++ {
			sortedLoadings.Set(i, j, loadings.At(i, indices[j]))
		}
	}

	// Reorder rotation matrix if it exists
	var sortedRotation *mat.Dense
	if rotationMatrix != nil {
		rr, rc := rotationMatrix.Dims()
		sortedRotation = mat.NewDense(rr, rc, nil)
		for j := 0; j < rc; j++ {
			for i := 0; i < rr; i++ {
				sortedRotation.Set(i, j, rotationMatrix.At(i, indices[j]))
			}
		}
	}

	// Reorder phi matrix if it exists
	var sortedPhi *mat.Dense
	if phi != nil {
		pr, pc := phi.Dims()
		sortedPhi = mat.NewDense(pr, pc, nil)
		for i := 0; i < pr; i++ {
			for j := 0; j < pc; j++ {
				sortedPhi.Set(i, j, phi.At(indices[i], indices[j]))
			}
		}
	}

	return sortedLoadings, sortedRotation, sortedPhi
}

// alignFactorsByVarGroups reorders and reflects factors so that columns correspond to
// variable groups A, B, C (by first letter of column names), in the order B, A, C.
// If groups are not present, falls back to explained-variance sorting.

// reorderFactorsByNameGroups tries to reorder factor columns to align with
// variable name groups defined by the first letter (A/B/C...). It assigns each
// factor to the group where it has the largest sum of squared loadings and then
// orders factors by the total strength of their assigned group (descending).
// If column names are unavailable or the heuristic is ambiguous, it returns the
// inputs unchanged.
func reorderFactorsByNameGroups(loadings, rotationMatrix, phi *mat.Dense, colNames []string) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil || len(colNames) == 0 {
		return loadings, rotationMatrix, phi
	}
	p, m := loadings.Dims()
	if m < 2 {
		return loadings, rotationMatrix, phi
	}
	// Build groups by first letter of variable names
	groupIndex := make(map[rune][]int)
	for i, nm := range colNames {
		r := firstAlphaRune(nm)
		if r == 0 {
			continue
		}
		groupIndex[r] = append(groupIndex[r], i)
	}
	if len(groupIndex) == 0 {
		return loadings, rotationMatrix, phi
	}
	// Compute score matrix: factor j vs group g => sum of squared loadings
	type groupScore struct {
		key   rune
		score []float64 // per-factor
		total float64   // group total across factors
	}
	groups := make([]groupScore, 0, len(groupIndex))
	for k, idxs := range groupIndex {
		gs := groupScore{key: k, score: make([]float64, m)}
		for j := 0; j < m; j++ {
			sum := 0.0
			for _, i := range idxs {
				v := loadings.At(i, j)
				sum += v * v
			}
			gs.score[j] = sum
			gs.total += sum
		}
		groups = append(groups, gs)
	}
	// Determine each factor's best group and its peak value
	bestGroup := make([]int, m) // index into groups slice
	bestValue := make([]float64, m)
	for j := 0; j < m; j++ {
		bi := -1
		bv := -1.0
		for gi, g := range groups {
			if g.score[j] > bv {
				bv = g.score[j]
				bi = gi
			}
		}
		bestGroup[j] = bi
		bestValue[j] = bv
	}
	// Choose the primary group as the factor with the largest peak value
	primaryFactor := 0
	for j := 1; j < m; j++ {
		if bestValue[j] > bestValue[primaryFactor] {
			primaryFactor = j
		}
	}
	primaryGroup := bestGroup[primaryFactor]
	// Build group order: primary group first, then remaining groups by key ascending
	orderGroups := []int{primaryGroup}
	// Collect remaining group indices
	remaining := make([]int, 0, len(groups)-1)
	for gi := range groups {
		if gi != primaryGroup {
			remaining = append(remaining, gi)
		}
	}
	// Sort remaining by rune key ascending for stability
	if len(remaining) > 1 {
		for i := 0; i < len(remaining)-1; i++ {
			for j := i + 1; j < len(remaining); j++ {
				if groups[remaining[i]].key > groups[remaining[j]].key {
					remaining[i], remaining[j] = remaining[j], remaining[i]
				}
			}
		}
	}
	orderGroups = append(orderGroups, remaining...)
	// Build factor order by matching factors to ordered groups (prefer the factor with larger score for that group)
	usedFactor := make([]bool, m)
	factorOrder := make([]int, 0, m)
	for _, gi := range orderGroups {
		// pick factor with highest g.score[j] among unused
		bj := -1
		bv := -1.0
		for j := 0; j < m; j++ {
			if usedFactor[j] {
				continue
			}
			if groups[gi].score[j] > bv {
				bv = groups[gi].score[j]
				bj = j
			}
		}
		if bj >= 0 {
			usedFactor[bj] = true
			factorOrder = append(factorOrder, bj)
		}
	}
	// Append any remaining unused factors
	for j := 0; j < m; j++ {
		if !usedFactor[j] {
			factorOrder = append(factorOrder, j)
		}
	}
	// Reorder matrices according to factorOrder
	rLoad := mat.NewDense(p, m, nil)
	for j := 0; j < m; j++ {
		src := factorOrder[j]
		for i := 0; i < p; i++ {
			rLoad.Set(i, j, loadings.At(i, src))
		}
	}
	var rRot *mat.Dense
	if rotationMatrix != nil {
		rr, rc := rotationMatrix.Dims()
		rRot = mat.NewDense(rr, rc, nil)
		for j := 0; j < rc; j++ {
			src := factorOrder[j]
			for i := 0; i < rr; i++ {
				rRot.Set(i, j, rotationMatrix.At(i, src))
			}
		}
	}
	var rPhi *mat.Dense
	if phi != nil {
		pr, pc := phi.Dims()
		rPhi = mat.NewDense(pr, pc, nil)
		for i := 0; i < pr; i++ {
			srcI := factorOrder[i]
			for j := 0; j < pc; j++ {
				srcJ := factorOrder[j]
				rPhi.Set(i, j, phi.At(srcI, srcJ))
			}
		}
	}
	return rLoad, rRot, rPhi
}

// firstAlphaRune returns the first letter (A-Za-z) rune in a string, or 0 if none.
func firstAlphaRune(s string) rune {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return unicode.ToUpper(r)
		}
	}
	return 0
}

// computeFactorScores computes factor scores using various methods
func computeFactorScores(data, loadings, phi *mat.Dense, uniquenesses []float64, sigmaForScores *mat.Dense, method FactorScoreMethod) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	if data == nil || loadings == nil {
		return nil, nil, nil, fmt.Errorf("data and loadings cannot be nil")
	}

	n, p := data.Dims()
	_, m := loadings.Dims()

	if method == FactorScoreNone {
		return nil, nil, nil, nil
	}

	if len(uniquenesses) != p {
		return nil, nil, nil, fmt.Errorf("uniqueness length %d does not match variables %d", len(uniquenesses), p)
	}

	// Build Psi and Psi^{-1}
	psiDiag := make([]float64, p)
	psiInvDiag := make([]float64, p)
	for i, u := range uniquenesses {
		if u <= 0 {
			return nil, nil, nil, fmt.Errorf("uniqueness at index %d is non-positive", i)
		}
		psiDiag[i] = u
		psiInvDiag[i] = 1.0 / u
	}
	psi := mat.NewDiagDense(p, psiDiag)
	psiInv := mat.NewDiagDense(p, psiInvDiag)

	// Observed covariance/correlation matrix for scoring weights
	var sigmaInv mat.Dense
	var sigmaUsed mat.Dense
	useObserved := false
	if sigmaForScores != nil {
		sigmaInv.CloneFrom(sigmaForScores)
		sigmaUsed.CloneFrom(sigmaForScores)
		if err := sigmaInv.Inverse(&sigmaInv); err == nil {
			useObserved = true
		} else {
			insyra.LogDebug("stats", "FactorAnalysis", "factor scores fallback to model covariance: %v", err)
		}
	}
	if !useObserved {
		// Fallback to model-implied covariance if observed matrix unavailable
		var reproduced mat.Dense
		if phi != nil {
			var tmp mat.Dense
			tmp.Mul(loadings, phi)
			reproduced.Mul(&tmp, loadings.T())
		} else {
			reproduced.Mul(loadings, loadings.T())
		}
		reproduced.Add(&reproduced, psi)
		sigmaInv.CloneFrom(&reproduced)
		sigmaUsed.CloneFrom(&reproduced)
		if err := sigmaInv.Inverse(&sigmaInv); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to invert model-implied covariance: %v", err)
		}
	}

	var weights *mat.Dense

	switch method {
	case FactorScoreRegression:
		// SPSS regression method: W = R^{-1} S, where S = P if orthogonal, else S = P Φ
		// Build structure S
		S := mat.NewDense(p, m, nil)
		if phi == nil {
			S.Copy(loadings)
		} else {
			var tmp mat.Dense
			tmp.Mul(loadings, phi)
			S.Copy(&tmp)
		}
		// W = R^{-1} S
		weights = mat.NewDense(p, m, nil)
		weights.Mul(&sigmaInv, S)

	case FactorScoreBartlett:
		// Bartlett weighted least squares: B = Ψ^{-1} Λ (Λᵀ Ψ^{-1} Λ)^{-1} Φ (if Φ exists)
		psiInvLoadings := mat.NewDense(p, m, nil)
		psiInvLoadings.Mul(psiInv, loadings)

		var middle mat.Dense
		middle.Mul(loadings.T(), psiInvLoadings)
		var middleInv mat.Dense
		if err := middleInv.Inverse(&middle); err != nil {
			return nil, nil, nil, fmt.Errorf("bartlett weights: failed to invert Λᵀ Ψ^{-1} Λ: %v", err)
		}

		weights = mat.NewDense(p, m, nil)
		weights.Mul(psiInvLoadings, &middleInv)
		if phi != nil {
			var tmp mat.Dense
			tmp.Mul(weights, phi)
			weights.Copy(&tmp)
		}

	case FactorScoreAndersonRubin:
		if phi != nil {
			// Anderson-Rubin requires orthogonal factors (Φ = I)
			return nil, nil, nil, fmt.Errorf("anderson-rubin scoring requires orthogonal factors")
		}

		// Anderson-Rubin: B = Σ^{-1} Λ (Λᵀ Σ^{-1} Λ)^{-1/2}
		var sigmaInvLoadings mat.Dense
		sigmaInvLoadings.Mul(&sigmaInv, loadings)

		var inner mat.Dense
		inner.Mul(loadings.T(), &sigmaInvLoadings)
		invSqrt, err := inverseSqrtDense(&inner)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("anderson-rubin weights: %v", err)
		}

		weights = mat.NewDense(p, m, nil)
		weights.Mul(&sigmaInvLoadings, invSqrt)

	default:
		return nil, nil, nil, fmt.Errorf("unsupported scoring method: %v", method)
	}

	scores := mat.NewDense(n, m, nil)
	scores.Mul(data, weights)

	var scoreCov *mat.Dense
	// Prefer empirical covariance of computed scores to align with SPSS output
	if scores != nil {
		// cov = (scores^T * scores) / (n - 1)
		var st mat.Dense
		st.CloneFrom(scores.T())
		var prod mat.Dense
		prod.Mul(&st, scores)
		scale := 1.0
		if n > 1 {
			scale = 1.0 / float64(n-1)
		}
		cov := mat.NewDense(m, m, nil)
		cov.Scale(scale, &prod)
		scoreCov = cov
	} else if weights != nil {
		// Fallback to model-implied covariance if scores are unavailable
		var tmp mat.Dense
		tmp.Mul(&sigmaUsed, weights)
		cov := mat.NewDense(m, m, nil)
		cov.Mul(weights.T(), &tmp)
		scoreCov = cov
	}

	return scores, weights, scoreCov, nil
}

// matrixToDataTableWithNames converts a matrix to DataTable with row and column names
func matrixToDataTableWithNames(matrix *mat.Dense, tableName string, colNames, rowNames []string) *insyra.DataTable {
	if matrix == nil {
		return nil
	}

	r, c := matrix.Dims()

	// Create DataTable
	dt := insyra.NewDataTable()
	dt.SetName(formatNameSnakePascal(tableName))

	// Add columns
	for j := 0; j < c; j++ {
		col := insyra.NewDataList()
		colName := ""
		if colNames != nil && j < len(colNames) {
			colName = colNames[j]
		}
		if colName == "" {
			colName = formatNameSnakePascal(fmt.Sprintf("Col %d", j+1))
		}
		col.SetName(colName)

		for i := 0; i < r; i++ {
			col.Append(matrix.At(i, j))
		}
		dt.AppendCols(col)
	}

	// Set row names if provided
	if rowNames != nil && len(rowNames) == r {
		for i, name := range rowNames {
			dt.SetRowNameByIndex(i, name)
		}
	}

	return dt
}

// vectorToDataTableWithNames converts a vector (slice) to DataTable with names
func vectorToDataTableWithNames(vector []float64, tableName string, colName string, rowNames []string) *insyra.DataTable {
	if vector == nil {
		return nil
	}

	r := len(vector)

	// Create DataTable
	dt := insyra.NewDataTable()
	dt.SetName(formatNameSnakePascal(tableName))

	// Add single column
	col := insyra.NewDataList()
	formattedColName := formatNameSnakePascal(colName)
	if formattedColName == "" {
		formattedColName = formatNameSnakePascal("Value")
	}
	col.SetName(formattedColName)

	for i := 0; i < r; i++ {
		col.Append(vector[i])
	}
	dt.AppendCols(col)

	// Set row names if provided
	if rowNames != nil && len(rowNames) == r {
		for i, name := range rowNames {
			dt.SetRowNameByIndex(i, name)
		}
	}

	return dt
}

// computeKMOMeasures calculates the overall KMO and per-variable MSA values for a correlation matrix
func computeKMOMeasures(corr *mat.Dense) (float64, []float64, error) {
	if corr == nil {
		return math.NaN(), nil, fmt.Errorf("correlation matrix is nil")
	}
	r, c := corr.Dims()
	if r != c {
		return math.NaN(), nil, fmt.Errorf("correlation matrix must be square")
	}
	if r < 2 {
		return math.NaN(), nil, fmt.Errorf("need at least two variables for KMO")
	}

	var inv mat.Dense
	if err := inv.Inverse(corr); err != nil {
		return math.NaN(), nil, fmt.Errorf("failed to invert correlation matrix: %w", err)
	}

	sumR2 := 0.0
	sumP2 := 0.0
	perVarR2 := make([]float64, r)
	perVarP2 := make([]float64, r)

	for i := 0; i < r; i++ {
		diagI := inv.At(i, i)
		if diagI <= 0 {
			diagI = math.Abs(diagI)
			if diagI <= 0 {
				return math.NaN(), nil, fmt.Errorf("non-positive inverse diagonal at %d", i)
			}
		}
		for j := i + 1; j < r; j++ {
			rij := corr.At(i, j)
			r2 := rij * rij
			sumR2 += r2
			perVarR2[i] += r2
			perVarR2[j] += r2

			diagJ := inv.At(j, j)
			if diagJ <= 0 {
				diagJ = math.Abs(diagJ)
				if diagJ <= 0 {
					return math.NaN(), nil, fmt.Errorf("non-positive inverse diagonal at %d", j)
				}
			}
			partial := -inv.At(i, j) / math.Sqrt(diagI*diagJ)
			p2 := partial * partial
			sumP2 += p2
			perVarP2[i] += p2
			perVarP2[j] += p2
		}
	}

	den := sumR2 + sumP2
	if den == 0 {
		return math.NaN(), nil, fmt.Errorf("zero denominator for KMO computation")
	}
	overall := sumR2 / den

	msa := make([]float64, r)
	for i := 0; i < r; i++ {
		perDen := perVarR2[i] + perVarP2[i]
		if perDen == 0 {
			msa[i] = math.NaN()
			continue
		}
		msa[i] = perVarR2[i] / perDen
	}

	return overall, msa, nil
}

// computeBartlettFromCorrelation performs Bartlett's test using a correlation matrix and sample size
func computeBartlettFromCorrelation(corr *mat.Dense, sampleSize int) (float64, float64, int, error) {
	if corr == nil {
		return math.NaN(), math.NaN(), 0, fmt.Errorf("correlation matrix is nil")
	}
	r, c := corr.Dims()
	if r != c {
		return math.NaN(), math.NaN(), 0, fmt.Errorf("correlation matrix must be square")
	}
	if sampleSize <= 1 {
		return math.NaN(), math.NaN(), 0, fmt.Errorf("insufficient sample size for Bartlett's test")
	}

	sym := denseToSym(corr)
	var chol mat.Cholesky
	if !chol.Factorize(sym) {
		return math.NaN(), math.NaN(), 0, fmt.Errorf("correlation matrix is not positive definite")
	}
	det := chol.Det()
	if det <= 0 {
		return math.NaN(), math.NaN(), 0, fmt.Errorf("non-positive determinant in Bartlett's test")
	}

	logDet := math.Log(det)
	pDim := float64(r)
	n := float64(sampleSize)
	chi := -((n - 1) - (2*pDim+5)/6.0) * logDet
	if chi < 0 {
		chi = 0
	}
	df := r * (r - 1) / 2
	pValue := 1 - distuv.ChiSquared{K: float64(df)}.CDF(chi)
	if pValue < 0 {
		pValue = 0
	}
	return chi, pValue, df, nil
}

// kmoToDataTable converts KMO metrics to a DataTable with per-variable MSA values
func kmoToDataTable(overall float64, msa []float64, colNames []string) *insyra.DataTable {
	if len(msa) == 0 {
		return nil
	}
	values := make([]float64, len(msa)+1)
	copy(values, msa)
	values[len(msa)] = overall

	rowNames := make([]string, 0, len(values))
	rowNames = append(rowNames, colNames...)
	rowNames = append(rowNames, "Overall")

	return vectorToDataTableWithNames(values, tableNameSamplingAdequacy, "MSA", rowNames)
}

// bartlettToDataTable converts Bartlett test statistics to a single-row DataTable
func bartlettToDataTable(chiSquare float64, df int, pValue float64, sampleSize int) *insyra.DataTable {
	dt := insyra.NewDataTable()
	dt.SetName(formatNameSnakePascal(tableNameBartlettTest))

	chiCol := insyra.NewDataList(chiSquare)
	chiCol.SetName(formatNameSnakePascal("ChiSquare"))
	dfCol := insyra.NewDataList(float64(df))
	dfCol.SetName(formatNameSnakePascal("DegreesOfFreedom"))
	pCol := insyra.NewDataList(pValue)
	pCol.SetName(formatNameSnakePascal("PValue"))

	dt.AppendCols(chiCol, dfCol, pCol)
	if sampleSize > 0 {
		nCol := insyra.NewDataList(float64(sampleSize))
		nCol.SetName(formatNameSnakePascal("SampleSize"))
		dt.AppendCols(nCol)
	}

	dt.SetRowNameByIndex(0, "Value")
	return dt
}

// kaiserNormalize: Apply Kaiser normalization to loadings
func kaiserNormalize(L, h *mat.Dense) *mat.Dense {
	p, m := L.Dims()
	normalized := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		hSqrt := math.Sqrt(h.At(i, 0))
		if hSqrt > 0 {
			for j := 0; j < m; j++ {
				normalized.Set(i, j, L.At(i, j)/hSqrt)
			}
		}
	}
	return normalized
}

// reflectFactorsForPositiveLoadings reflects factors to ensure positive loadings.
// It returns the reflected loadings and the sign adjustments applied to each factor.
func reflectFactorsForPositiveLoadings(loadings *mat.Dense) (*mat.Dense, []float64) {
	if loadings == nil {
		return nil, nil
	}

	r, c := loadings.Dims()
	result := mat.NewDense(r, c, nil)
	result.Copy(loadings)
	signs := make([]float64, c)

	for j := 0; j < c; j++ {
		maxAbs := 0.0
		maxVal := 0.0
		for i := 0; i < r; i++ {
			val := result.At(i, j)
			absVal := math.Abs(val)
			if absVal > maxAbs {
				maxAbs = absVal
				maxVal = val
			}
		}

		sign := 1.0
		if maxVal < 0 {
			sign = -1.0
		}
		signs[j] = sign
		if sign < 0 {
			for i := 0; i < r; i++ {
				result.Set(i, j, -result.At(i, j))
			}
		}
	}

	return result, signs
}

// applyReflectionToRotationAndPhi propagates factor reflections to rotation and phi matrices
func applyReflectionToRotationAndPhi(rotationMatrix, phi *mat.Dense, signs []float64) {
	if len(signs) == 0 {
		return
	}

	if rotationMatrix != nil {
		rRows, rCols := rotationMatrix.Dims()
		limit := min(len(signs), rCols)
		for j := 0; j < limit; j++ {
			if signs[j] < 0 {
				for i := 0; i < rRows; i++ {
					rotationMatrix.Set(i, j, -rotationMatrix.At(i, j))
				}
			}
		}
	}

	if phi != nil {
		phiCopy := mat.DenseCopyOf(phi)
		rows, cols := phi.Dims()
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				iSign := 1.0
				jSign := 1.0
				if i < len(signs) {
					iSign = signs[i]
				}
				if j < len(signs) {
					jSign = signs[j]
				}
				phi.Set(i, j, phiCopy.At(i, j)*iSign*jSign)
			}
		}
	}
}

// normalizeRotationAndLoadings rescales columns so that diag(Phi)=1 exactly.
func normalizeRotationAndLoadings(loadings, rotationMatrix, phi *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense) {
	// rotationMatrix here is rotMat = t(inv(T)) for oblique rotations (GPFoblq),
	// and equals T for orthogonal rotations. We normalize so that diag(Phi)=1.
	if rotationMatrix == nil || phi == nil || loadings == nil {
		return loadings, rotationMatrix, phi
	}
	pRows, pCols := loadings.Dims()
	rRows, rCols := rotationMatrix.Dims()
	if pCols != rCols {
		return loadings, rotationMatrix, phi
	}
	// Recover T from rotMat: rotMat = t(inv(T)) => (rotMat^T) = inv(T) => T = inv(rotMat^T)
	var rotT mat.Dense
	rotT.CloneFrom(rotationMatrix)
	rotT.T()
	var T mat.Dense
	if err := T.Inverse(&rotT); err != nil {
		// Fallback: assume orthogonal case
		return loadings, rotationMatrix, phi
	}
	// Compute current Phi = T^T T
	var TT mat.Dense
	TT.CloneFrom(&T)
	TT.T()
	var curPhi mat.Dense
	curPhi.Mul(&TT, &T)
	// Build diagonal scaling D so that diag(Phi')=1: T' = T * D with D = diag(1/sqrt(diag(curPhi)))
	scales := make([]float64, rCols)
	for j := 0; j < rCols; j++ {
		d := curPhi.At(j, j)
		if d <= 0 {
			scales[j] = 1.0
		} else {
			scales[j] = 1.0 / math.Sqrt(d)
		}
	}
	// Apply L' = L * D^{-1}
	loadOut := mat.NewDense(pRows, pCols, nil)
	for i := 0; i < pRows; i++ {
		for j := 0; j < pCols; j++ {
			s := scales[j]
			if s == 0 {
				s = 1
			}
			loadOut.Set(i, j, loadings.At(i, j)/s)
		}
	}
	// Update T' = T * D and rotMat' = t(inv(T'))
	var Tscaled mat.Dense
	Tscaled.CloneFrom(&T)
	for i := 0; i < rRows; i++ {
		for j := 0; j < rCols; j++ {
			Tscaled.Set(i, j, T.At(i, j)*scales[j])
		}
	}
	// Compute rotMat' = t(inv(T'))
	var invT mat.Dense
	if err := invT.Inverse(&Tscaled); err != nil {
		return loadOut, rotationMatrix, phi
	}
	var rotOut mat.Dense
	rotOut.CloneFrom(&invT)
	rotOut.T()
	// Recompute Phi' = T'^T T'
	var Tt2 mat.Dense
	Tt2.CloneFrom(&Tscaled)
	Tt2.T()
	var phiOut mat.Dense
	phiOut.Mul(&Tt2, &Tscaled)
	return loadOut, &rotOut, &phiOut
}

func formatLabelPascalWithSpaces(name string) string {
	segments := splitIdentifier(name)
	if len(segments) == 0 {
		return ""
	}
	for i, segment := range segments {
		segments[i] = formatSegmentTitle(segment)
	}
	return strings.Join(segments, " ")
}

func formatNameSnakePascal(name string) string {
	segments := splitIdentifier(name)
	if len(segments) == 0 {
		return ""
	}
	formatted := make([]string, len(segments))
	for i, segment := range segments {
		formatted[i] = formatSegmentTitle(segment)
	}
	return strings.Join(formatted, "_")
}

func splitIdentifier(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	runes := []rune(name)
	segments := make([]string, 0, len(runes))
	current := make([]rune, 0, len(runes))
	prevClass := 0
	flush := func() {
		if len(current) > 0 {
			segments = append(segments, string(current))
			current = current[:0]
		}
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			prevClass = 0
			continue
		}
		class := charClass(r)
		if len(current) == 0 {
			current = append(current, r)
			prevClass = class
			continue
		}
		split := false
		switch class {
		case 1: // uppercase
			if prevClass == 2 || prevClass == 3 {
				split = true
			} else if prevClass == 1 {
				if i+1 < len(runes) {
					next := runes[i+1]
					if unicode.IsLower(next) {
						split = true
					}
				}
			}
		case 2: // lowercase
			if prevClass == 3 {
				split = true
			}
		case 3: // digit
			if prevClass != 3 {
				split = true
			}
		}
		if split {
			flush()
		}
		current = append(current, r)
		prevClass = class
	}
	flush()
	return segments
}

func charClass(r rune) int {
	switch {
	case unicode.IsDigit(r):
		return 3
	case unicode.IsUpper(r):
		return 1
	case unicode.IsLower(r):
		return 2
	default:
		return 0
	}
}

func formatSegmentTitle(segment string) string {
	if segment == "" {
		return ""
	}
	if isAllDigits(segment) {
		return segment
	}
	lower := strings.ToLower(segment)
	runes := []rune(lower)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func isAllDigits(segment string) bool {
	if segment == "" {
		return false
	}
	for _, r := range segment {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// -------------------------
// PAF + Oblimin Implementation (SPSS-compatible)
// -------------------------

// SMC: Squared Multiple Correlation via inverse correlation diagonal
func SMC(R *mat.Dense) *mat.Dense {
	p, _ := R.Dims()
	h := mat.NewDense(p, 1, nil)
	var inv mat.Dense
	if err := inv.Inverse(R); err != nil {
		// fallback to zeros
		return h
	}
	for i := 0; i < p; i++ {
		d := inv.At(i, i)
		v := 0.0
		if d != 0 {
			v = 1.0 - 1.0/d
		}
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		h.Set(i, 0, v)
	}
	return h
}

// buildReducedCorrelation: R* = R with diagonal replaced by h
func buildReducedCorrelation(R, h *mat.Dense) *mat.Dense {
	p, _ := R.Dims()
	Rstar := mat.DenseCopyOf(R)
	for i := 0; i < p; i++ {
		Rstar.Set(i, i, h.At(i, 0))
	}
	return Rstar
}

// eigenTopM: Get top m eigenvalues and eigenvectors
func eigenTopM(A *mat.Dense, m int) (V, Dm *mat.Dense, err error) {
	p, _ := A.Dims()
	var eig mat.EigenSym
	symA := denseToSym(A)
	if !eig.Factorize(symA, true) {
		return nil, nil, errors.New("eigen decomposition failed")
	}
	eigenvalues := eig.Values(nil)
	var eigenvectors mat.Dense
	eig.VectorsTo(&eigenvectors)

	// Sort eigenvalues and eigenvectors in descending order
	type eigenPair struct {
		value  float64
		vector []float64
	}
	pairs := make([]eigenPair, p)
	for i := 0; i < p; i++ {
		vec := make([]float64, p)
		for j := 0; j < p; j++ {
			vec[j] = eigenvectors.At(j, i)
		}
		pairs[i] = eigenPair{value: eigenvalues[i], vector: vec}
	}
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].value < pairs[j].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Extract top m
	V = mat.NewDense(p, m, nil)
	Dm = mat.NewDense(m, m, nil)
	for j := 0; j < m; j++ {
		Dm.Set(j, j, pairs[j].value)
		for i := 0; i < p; i++ {
			V.Set(i, j, pairs[j].vector[i])
		}
	}
	return V, Dm, nil
}

// loadingsFromEigen: L = V * sqrt(Dm)
func loadingsFromEigen(V, Dm *mat.Dense) *mat.Dense {
	p, m := V.Dims()
	L := mat.NewDense(p, m, nil)
	for j := 0; j < m; j++ {
		s := math.Sqrt(math.Max(Dm.At(j, j), 0))
		for i := 0; i < p; i++ {
			L.Set(i, j, V.At(i, j)*s)
		}
	}
	return L
}

// communalitiesFromLoadings: h = rowSums(L^2)
func communalitiesFromLoadings(L *mat.Dense) *mat.Dense {
	p, m := L.Dims()
	h := mat.NewDense(p, 1, nil)
	for i := 0; i < p; i++ {
		var s float64
		for j := 0; j < m; j++ {
			v := L.At(i, j)
			s += v * v
		}
		h.Set(i, 0, s)
	}
	clamp01InPlace(h)
	return h
}

func dampen(prev, next *mat.Dense, alpha float64) *mat.Dense {
	p, _ := prev.Dims()
	out := mat.NewDense(p, 1, nil)
	for i := 0; i < p; i++ {
		out.Set(i, 0, alpha*next.At(i, 0)+(1-alpha)*prev.At(i, 0))
	}
	return out
}

func maxAbsDiff(a, b *mat.Dense) float64 {
	p, _ := a.Dims()
	maxd := 0.0
	for i := 0; i < p; i++ {
		d := math.Abs(a.At(i, 0) - b.At(i, 0))
		if d > maxd {
			maxd = d
		}
	}
	return maxd
}

func clamp01InPlace(x *mat.Dense) {
	r, c := x.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			v := x.At(i, j)
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			x.Set(i, j, v)
		}
	}
}

// sumH: helper to sum communalities
func sumH(h *mat.Dense) float64 {
	p, _ := h.Dims()
	sum := 0.0
	for i := 0; i < p; i++ {
		sum += h.At(i, 0)
	}
	return sum
}

func kaiserDenormalize(LNormalized *mat.Dense) *mat.Dense {
	// For simplicity, assume we need to scale back, but since we don't have the original scales,
	// this is a placeholder. In practice, we need to store the scales.
	// For now, just return the normalized loadings
	return LNormalized
}

func kaiserWeightsInv(h *mat.Dense) *mat.Dense {
	p, _ := h.Dims()
	Winv := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		hi := math.Max(h.At(i, 0), 1e-12)
		Winv.Set(i, i, math.Sqrt(hi))
	}
	return Winv
}

// phiFromT: Phi = T^T T (for Oblimin rotation)
func phiFromT(T *mat.Dense) *mat.Dense {
	var TT mat.Dense
	TT.Mul(T.T(), T)
	return &TT
}

// FactorPAFOblimin: Main function for PAF + Oblimin
func FactorPAFOblimin(R *mat.Dense, m int, delta, tol float64, maxIter int, damping float64) (P, S, Phi, T, L_unrotated, h *mat.Dense, eigenvals []float64, iterations int, converged bool, err error) {
	p, _ := R.Dims()

	// PAF step
	h = SMC(R)

	var L *mat.Dense
	converged = false
	var eigenvalsFinal []float64
	for it := 0; it < maxIter; it++ {
		iterations = it + 1
		Rstar := buildReducedCorrelation(R, h)
		V, Dm, err := eigenTopM(Rstar, m)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, 0, false, err
		}
		eigenvals := make([]float64, m)
		sumEigen := 0.0
		for i := 0; i < m; i++ {
			eigenvals[i] = Dm.At(i, i)
			sumEigen += eigenvals[i]
		}
		insyra.LogInfo("stats", "FactorPAFOblimin", "Eigenvalues of R*: %v, sum=%.4f", eigenvals, sumEigen)
		L = loadingsFromEigen(V, Dm)
		hNew := communalitiesFromLoadings(L)
		hOld := h
		h = dampen(h, hNew, damping)
		if maxAbsDiff(hOld, hNew) < tol {
			converged = true
			eigenvalsFinal = eigenvals
			break
		}
	}
	if !converged {
		// Use final eigenvalues
		Rstar := buildReducedCorrelation(R, h)
		V, Dm, err := eigenTopM(Rstar, m)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, 0, false, err
		}
		eigenvalsFinal = make([]float64, m)
		for i := 0; i < m; i++ {
			eigenvalsFinal[i] = Dm.At(i, i)
		}
		L = loadingsFromEigen(V, Dm)
	}

	insyra.LogInfo("stats", "FactorPAFOblimin", "PAF completed: iterations=%d, converged=%v, h sum=%.4f", iterations, converged, sumH(h))
	if L != nil {
		insyra.LogInfo("stats", "FactorPAFOblimin", "Unrotated sample loadings: %.3f, %.3f, %.3f", L.At(0, 0), L.At(0, 1), L.At(0, 2))
	}

	// Use GPArotation-compatible rotation via internal fa.Rotate for SPSS alignment
	restarts := 1
	rotOpts := &fa.RotOpts{
		Eps:         1e-8,
		MaxIter:     max(1000, maxIter),
		Alpha0:      1.0,
		Gamma:       delta,
		PromaxPower: 4,
		Restarts:    restarts,
	}
	Lrot, rotMat, Phi0, _, err := fa.Rotate(L, string(FactorRotationOblimin), rotOpts)
	if Lrot != nil {
		insyra.LogInfo("stats", "FactorPAFOblimin", "Rotated sample loadings: %.3f, %.3f, %.3f", Lrot.At(0, 0), Lrot.At(0, 1), Lrot.At(0, 2))
	}
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, 0, false, err
	}
	// Recover T from rotMat (rotMat = t(inv(T))) and ensure Phi = T^T T
	if rotMat != nil {
		var rotT mat.Dense
		rotT.CloneFrom(rotMat)
		rotT.T()
		var Trec mat.Dense
		if err := Trec.Inverse(&rotT); err == nil {
			T = &Trec
			if Phi0 == nil {
				var TT mat.Dense
				TT.CloneFrom(&Trec)
				TT.T()
				var ph mat.Dense
				ph.Mul(&TT, &Trec)
				Phi0 = &ph
			}
		}
	}
	if Phi0 == nil {
		return nil, nil, nil, nil, nil, nil, nil, 0, false, fmt.Errorf("rotation failed to produce Phi")
	}
	// Standardize signs to SPSS conventions
	Pstd, signs := reflectFactorsForPositiveLoadings(Lrot)
	// Apply reflections to both T and Phi to maintain consistency
	applyReflectionToRotationAndPhi(T, Phi0, signs)
	P = Pstd
	Phi = Phi0

	S = mat.NewDense(p, m, nil)
	S.Mul(P, Phi)

	return P, S, Phi, T, L, h, eigenvalsFinal, iterations, converged, nil
}

// obliminRotate: Oblimin rotation on Kaiser-normalized B with convergence check
func obliminRotate(B *mat.Dense, delta, tol float64, maxIter int) (*mat.Dense, bool, error) {
	p, m := B.Dims()
	T := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		T.Set(i, i, 1.0)
	}

	converged := false
	for it := 0; it < maxIter; it++ {
		A := mat.NewDense(p, m, nil)
		A.Mul(B, T)

		G := obliminGradient(A, delta)

		dT := mat.NewDense(m, m, nil)
		dT.Mul(B.T(), G)

		step := 0.01 // Smaller step for stability
		scaledDT := mat.NewDense(m, m, nil)
		scaledDT.Scale(-step, dT)
		Tnew := mat.NewDense(m, m, nil)
		Tnew.Add(T, scaledDT)

		// Check convergence
		if frobNormDiff(T, Tnew) < tol {
			T = Tnew
			converged = true
			break
		}
		T = Tnew
	}
	return T, converged, nil
}

// obliminGradient: Simplified Oblimin gradient
func obliminGradient(A *mat.Dense, delta float64) *mat.Dense {
	p, m := A.Dims()
	G := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			// Simplified gradient calculation
			sum := 0.0
			for k := 0; k < m; k++ {
				if k != j {
					aij := A.At(i, j)
					aik := A.At(i, k)
					sum += aij * aik
				}
			}
			G.Set(i, j, 4*sum-delta/float64(m-1)*A.At(i, j)*A.At(i, j))
		}
	}
	return G
}

func frobNormDiff(A, B *mat.Dense) float64 {
	var diff mat.Dense
	diff.Sub(A, B)
	return mat.Norm(&diff, 2)
}

// FactorScoreCoefficientsRegression: Regression method for factor scores
func FactorScoreCoefficientsRegression(R, P *mat.Dense) *mat.Dense {
	var Rinverse mat.Dense
	err := Rinverse.Inverse(R)
	if err != nil {
		return nil
	}

	var A mat.Dense
	A.Mul(&Rinverse, P)

	var PtRinvP mat.Dense
	PtRinvP.Mul(P.T(), &A)

	var PtRinvP_inv mat.Dense
	err = PtRinvP_inv.Inverse(&PtRinvP)
	if err != nil {
		return nil
	}

	p, m := P.Dims()
	W := mat.NewDense(p, m, nil)
	W.Mul(&A, &PtRinvP_inv)

	return W
}
