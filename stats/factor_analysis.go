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
	Method FactorRotationMethod
	Kappa  float64 // Optional: Promax power (default 4)
	Delta  float64 // Optional: default 0 for Oblimin
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
	Phi                  insyra.IDataTable // Factor correlation matrix (m × m), nil for orthogonal
	RotationMatrix       insyra.IDataTable // Rotation matrix (m × m), nil if no rotation
	Eigenvalues          insyra.IDataTable // Eigenvalues vector (p × 1)
	ExplainedProportion  insyra.IDataTable // Proportion explained by each factor (m × 1)
	CumulativeProportion insyra.IDataTable // Cumulative proportion explained (m × 1)
	Scores               insyra.IDataTable // Factor scores (n × m), nil if not computed

	Converged  bool
	Iterations int
	CountUsed  int
	Messages   []string
}

const (
	tableNameFactorLoadings       = "FactorLoadings"
	tableNameFactorStructure      = "FactorStructure"
	tableNameUniqueness           = "Uniqueness"
	tableNameCommunalities        = "Communalities"
	tableNamePhiMatrix            = "PhiMatrix"
	tableNameRotationMatrix       = "RotationMatrix"
	tableNameEigenvalues          = "Eigenvalues"
	tableNameExplainedProportion  = "ExplainedProportion"
	tableNameCumulativeProportion = "CumulativeProportion"
	tableNameFactorScores         = "FactorScores"
)

// Show prints everything in the FactorAnalysisResult
func (r *FactorAnalysisResult) Show(startEndRange ...any) {
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorLoadings), r.Loadings, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorStructure), r.Structure, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameUniqueness), r.Uniquenesses, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameCommunalities), r.Communalities, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNamePhiMatrix), r.Phi, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameRotationMatrix), r.RotationMatrix, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameEigenvalues), r.Eigenvalues, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameExplainedProportion), r.ExplainedProportion, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameCumulativeProportion), r.CumulativeProportion, startEndRange...)
	insyra.Show(formatLabelPascalWithSpaces(tableNameFactorScores), r.Scores, startEndRange...)
	fmt.Printf("Converged: %v\n", r.Converged)
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
			Method: FactorRotationOblimin, // R default: "oblimin"
			Kappa:  4,                     // R default for promax
			Delta:  0,                     // R default for oblimin
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
	if opt.Preprocess.Standardize {
		corrMatrix = mat.NewSymDense(colNum, nil)
		stat.CorrelationMatrix(corrMatrix, data, nil)
	} else {
		corrMatrix = mat.NewSymDense(colNum, nil)
		stat.CovarianceMatrix(corrMatrix, data, nil)
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
		if initComm, commErr := initialCommunalitiesSMC(corrDense, minErr); commErr != nil {
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
	loadings = reflectFactorsForPositiveLoadings(loadings)

	// Step 7: Rotate factors
	var rotatedLoadings *mat.Dense
	var rotationMatrix *mat.Dense
	var phi *mat.Dense
	if opt.Rotation.Method != FactorRotationNone {
		rotatedLoadings, rotationMatrix, phi, err = rotateFactors(loadings, opt.Rotation)
		if err != nil {
			insyra.LogWarning("stats", "FactorAnalysis", "rotation failed: %v", err)
			rotatedLoadings = loadings
			rotationMatrix = nil
			phi = nil
		}
		// Note: rotateFactors now handles sign standardization internally
	} else {
		rotatedLoadings = loadings
		rotationMatrix = nil
		phi = nil
		// Apply factor reflection for unrotated factors
		rotatedLoadings = reflectFactorsForPositiveLoadings(rotatedLoadings)
	}

	// Sort factors by explained variance (following R's psych package)
	rotatedLoadings, rotationMatrix, phi = sortFactorsByExplainedVariance(rotatedLoadings, rotationMatrix, phi)

	// Step 8: Compute communalities and uniquenesses
	extractionCommunalities := make([]float64, colNum)
	uniquenesses := make([]float64, colNum)
	var phiMat *mat.Dense
	if phi != nil {
		phiMat = phi
	}
	for i := 0; i < colNum; i++ {
		var hi2 float64
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
		uniq := diag - hi2
		if uniq < uniquenessLowerBound {
			uniq = uniquenessLowerBound
		}
		uniquenesses[i] = uniq
	}

	commMatrix := mat.NewDense(colNum, 1, nil)
	for i := 0; i < colNum; i++ {
		commMatrix.Set(i, 0, extractionCommunalities[i])
	}
	communalitiesTable := matrixToDataTableWithNames(commMatrix, tableNameCommunalities, []string{"Extraction"}, colNames)

	// Step 9: Compute explained proportions using structure matrix (SS loadings / number of variables)
	// Following R's psych package: SS loadings = sum of squared structure loadings
	pVars, mFactors := rotatedLoadings.Dims()
	S := mat.NewDense(pVars, mFactors, nil)
	if phi == nil {
		// Orthogonal rotation: structure = pattern
		S.Copy(rotatedLoadings)
	} else {
		// Oblique rotation: structure = pattern * phi
		var tmp mat.Dense
		tmp.Mul(rotatedLoadings, phi)
		S.Copy(&tmp)
	}

	// Calculate SS loadings for each factor using structure squared
	ssLoad := make([]float64, mFactors)
	for j := 0; j < mFactors; j++ {
		sum := 0.0
		for i := 0; i < pVars; i++ {
			// For oblique: use structure squared
			// For orthogonal: structure = loadings, so this is loadings squared
			val := S.At(i, j)
			sum += val * val
		}
		ssLoad[j] = sum
	}

	// Proportion of variance explained
	totalVar := float64(pVars) // Total variance in correlation matrix is p
	explainedProp := make([]float64, mFactors)
	cumulativeProp := make([]float64, mFactors)

	cumSum := 0.0
	for j := 0; j < mFactors; j++ {
		prop := ssLoad[j] / totalVar
		explainedProp[j] = prop
		cumSum += prop
		cumulativeProp[j] = cumSum
	}

	// Step 10: Compute factor scores if data is available
	var scores *mat.Dense
	if rowNum > 0 {
		var err error
		scores, err = computeFactorScores(data, rotatedLoadings, phi, uniquenesses, sigmaForScores, opt.Scoring)
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
	structureTable := matrixToDataTableWithNames(S, tableNameFactorStructure, factorColNames, colNames)

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
		Phi:                  nil,
		RotationMatrix:       nil,
		Eigenvalues:          vectorToDataTableWithNames(sortedEigenvalues, tableNameEigenvalues, "Eigenvalue", factorColNames),
		ExplainedProportion:  vectorToDataTableWithNames(explainedProp, tableNameExplainedProportion, "Explained Proportion", factorColNames),
		CumulativeProportion: vectorToDataTableWithNames(cumulativeProp, tableNameCumulativeProportion, "Cumulative Proportion", factorColNames),
		Scores:               nil,
		Converged:            converged,
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
func initialCommunalitiesSMC(corr *mat.Dense, minErr float64) ([]float64, error) {
	// Compute SMC (Squared Multiple Correlation) estimates
	// SMC_i = 1 - 1/R_ii^inv where R_ii^inv is diagonal of inverse correlation matrix
	h2, err := computeSMC(corr, minErr)
	if err != nil {
		return nil, err
	}
	return h2, nil
}

// computeSMC computes Squared Multiple Correlation estimates using internal/fa package
func computeSMC(corr *mat.Dense, minErr float64) ([]float64, error) {
	smcVec, err := fa.SMC(corr, false) // false for correlation matrix
	if err != nil {
		return nil, fmt.Errorf("SMC computation failed: %w", err)
	}

	p := smcVec.Len()
	h2 := make([]float64, p)
	for i := 0; i < p; i++ {
		smc := smcVec.AtVec(i)
		// Apply bounds as in original implementation
		if smc < 0 {
			smc = 0.0
		}
		if smc < minErr {
			smc = minErr
		}
		if smc > 1.0 {
			smc = 1.0
		}
		h2[i] = smc
	}
	return h2, nil
}

// extractPAF extracts factors using Principal Axis Factoring
func extractPAF(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64, minErr float64) (*mat.Dense, bool, int, error) {
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

	// Initialize communalities with SMC (Squared Multiple Correlation)
	communalities, err := initialCommunalitiesSMC(corrMatrix, minErr)
	if err != nil {
		return nil, false, 0, fmt.Errorf("failed to compute initial communalities: %w", err)
	}

	// Keep history
	commHistory := make([][]float64, 0, maxIter)
	converged := false
	var loadings *mat.Dense

	// main iterative loop
	for iter := 0; iter < maxIter; iter++ {
		// build reduced correlation matrix with communalities on diagonal
		p, _ := corrMatrix.Dims()
		reducedCorr := mat.NewDense(p, p, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < p; j++ {
				if i == j {
					reducedCorr.Set(i, j, communalities[i])
				} else {
					reducedCorr.Set(i, j, corrMatrix.At(i, j))
				}
			}
		}

		// eigen decomposition of reducedCorr
		var eig mat.EigenSym
		symReduced := denseToSym(reducedCorr)

		if !eig.Factorize(symReduced, true) {
			return nil, false, iter, errors.New("eigenvalue decomposition failed in PAF")
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
		// Sort in descending order
		for i := 0; i < len(pairs)-1; i++ {
			for j := i + 1; j < len(pairs); j++ {
				if pairs[i].value < pairs[j].value {
					pairs[i], pairs[j] = pairs[j], pairs[i]
				}
			}
		}

		// R: Adjust small eigenvalues before using them
		// eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps
		adjustedEigenvalues := make([]float64, len(pairs))
		for i := range pairs {
			if pairs[i].value < machineEpsilon {
				adjustedEigenvalues[i] = eigenvalueMinThreshold
			} else {
				adjustedEigenvalues[i] = pairs[i].value
			}
		}

		// Extract loadings using adjusted eigenvalues
		// R: if (nf > 1) { loadings <- eigens$vectors[, 1:nf] %*% diag(sqrt(eigens$values[1:nf])) }
		loadings = mat.NewDense(p, numFactors, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < numFactors; j++ {
				loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(adjustedEigenvalues[j]))
			}
		}

		if iter == 0 && p >= 4 {
			insyra.LogInfo("stats", "PAF", "iter %d pre-update loadings[0,0:2]=%.3f, %.3f", iter+1,
				loadings.At(0, 0), loadings.At(0, min(1, numFactors-1)))
		}

		// Update communalities
		newCommunalities := make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < numFactors; j++ {
				sum += loadings.At(i, j) * loadings.At(i, j)
			}
			// In R's PAF, communalities can equal the diagonal
			diag := corrMatrix.At(i, i)
			if sum < epsilonMedium {
				sum = epsilonMedium
			}
			if sum > diag {
				sum = diag // Allow communalities to reach diagonal
			}
			newCommunalities[i] = sum
		}

		if iter == 0 && p >= 4 {
			insyra.LogInfo("stats", "PAF", "iter %d post-loading communalities[0:4]=%.3f, %.3f, %.3f, %.3f", iter+1,
				newCommunalities[0], newCommunalities[min(1, p-1)], newCommunalities[min(2, p-1)], newCommunalities[min(3, p-1)])
		}

		// record history (copy slice)
		tmp := make([]float64, p)
		copy(tmp, newCommunalities)
		commHistory = append(commHistory, tmp)

		// Apply minimum error constraint (R's min.err parameter)
		for i := 0; i < p; i++ {
			if newCommunalities[i] < minErr {
				newCommunalities[i] = minErr
			}
		}

		// Check convergence
		maxChange := 0.0
		for i := 0; i < p; i++ {
			change := math.Abs(newCommunalities[i] - communalities[i])
			if change > maxChange {
				maxChange = change
			}
		}

		communalities = newCommunalities

		// Log convergence progress
		if iter < 5 || iter == maxIter-1 {
			insyra.LogDebug("stats", "PAF", "iter %d: maxChange=%.6f, h2[0]=%.4f",
				iter+1, maxChange, communalities[0])
		}

		if maxChange < tol {
			converged = true
			insyra.LogInfo("stats", "PAF", "converged in %d iterations", iter+1)
			return loadings, converged, iter + 1, nil
		}
	}

	// If we reach here, did not converge within maxIter
	// Log initial SMC and history summary (first 10 iterations and final)
	if len(commHistory) > 0 {
		// initial SMC = first  value prior to iteration (communalities passed in initially)
		insyra.LogInfo("stats", "PAF", "Initial SMC (first 4) = %.3f, %.3f, %.3f, %.3f",
			commHistory[0][0], commHistory[0][min(1, p-1)], commHistory[0][min(2, p-1)], commHistory[0][min(3, p-1)])
		maxShow := 10
		if len(commHistory) < maxShow {
			maxShow = len(commHistory)
		}
		for i := 0; i < maxShow; i++ {
			// print first 4 entries per iteration for brevity
			h := commHistory[i]
			insyra.LogInfo("stats", "PAF", "iter %d h2 (first 4) = %.4f, %.4f, %.4f, %.4f", i+1, h[0], h[min(1, p-1)], h[min(2, p-1)], h[min(3, p-1)])
		}
		// print final
		last := commHistory[len(commHistory)-1]
		insyra.LogInfo("stats", "PAF", "final h2 (first 4) = %.4f, %.4f, %.4f, %.4f", last[0], last[min(1, p-1)], last[min(2, p-1)], last[min(3, p-1)])
	}

	return loadings, converged, maxIter, nil
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
func rotateFactors(loadings *mat.Dense, rotationOpts FactorRotationOptions) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	if loadings == nil {
		return nil, nil, nil, fmt.Errorf("loadings cannot be nil")
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
		return reflectFactorsForPositiveLoadings(loadings), rotationMatrix, phi, nil
	}

	// Use the fa package for rotation
	opts := &fa.RotOpts{
		Eps:         1e-5,
		MaxIter:     1000,
		Alpha0:      1.0,
		Gamma:       rotationOpts.Delta, // For oblimin
		PromaxPower: int(rotationOpts.Kappa),
	}

	rotatedLoadings, rotMat, phi, err := fa.Rotate(loadings, string(rotationOpts.Method), opts)
	if err != nil {
		return nil, nil, nil, err
	}

	// Apply sign standardization
	rotatedLoadings = reflectFactorsForPositiveLoadings(rotatedLoadings)

	return rotatedLoadings, rotMat, phi, nil
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

	// Calculate explained variance for each factor (sum of squared loadings)
	for j := 0; j < c; j++ {
		sum := 0.0
		for i := 0; i < r; i++ {
			val := loadings.At(i, j)
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
		for j := 0; j < pc; j++ {
			for i := 0; i < pr; i++ {
				sortedPhi.Set(i, j, phi.At(i, indices[j]))
			}
		}
		// Also reorder rows
		tempPhi := mat.NewDense(pc, pc, nil)
		for j := 0; j < pc; j++ {
			for i := 0; i < pc; i++ {
				tempPhi.Set(j, i, sortedPhi.At(indices[j], indices[i]))
			}
		}
		sortedPhi = tempPhi
	}

	return sortedLoadings, sortedRotation, sortedPhi
}

// computeFactorScores computes factor scores using various methods
func computeFactorScores(data, loadings, phi *mat.Dense, uniquenesses []float64, sigmaForScores *mat.Dense, method FactorScoreMethod) (*mat.Dense, error) {
	if data == nil || loadings == nil {
		return nil, fmt.Errorf("data and loadings cannot be nil")
	}

	n, p := data.Dims()
	_, m := loadings.Dims()

	if method == FactorScoreNone {
		return nil, nil
	}

	if len(uniquenesses) != p {
		return nil, fmt.Errorf("uniqueness length %d does not match variables %d", len(uniquenesses), p)
	}

	// Build Psi and Psi^{-1}
	psiDiag := make([]float64, p)
	psiInvDiag := make([]float64, p)
	for i, u := range uniquenesses {
		if u <= 0 {
			return nil, fmt.Errorf("uniqueness at index %d is non-positive", i)
		}
		psiDiag[i] = u
		psiInvDiag[i] = 1.0 / u
	}
	psi := mat.NewDiagDense(p, psiDiag)
	psiInv := mat.NewDiagDense(p, psiInvDiag)

	// Observed covariance/correlation matrix for scoring weights
	var sigmaInv mat.Dense
	useObserved := false
	if sigmaForScores != nil {
		sigmaInv.CloneFrom(sigmaForScores)
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
		if err := sigmaInv.Inverse(&sigmaInv); err != nil {
			return nil, fmt.Errorf("failed to invert model-implied covariance: %v", err)
		}
	}

	switch method {
	case FactorScoreRegression:
		// Thurstone regression method: B = Σ^{-1} Λ Φ
		var weights mat.Dense
		weights.Mul(&sigmaInv, loadings)
		if phi != nil {
			var tmp mat.Dense
			tmp.Mul(&weights, phi)
			weights.Copy(&tmp)
		}
		scores := mat.NewDense(n, m, nil)
		scores.Mul(data, &weights)
		return scores, nil

	case FactorScoreBartlett:
		// Bartlett weighted least squares: B = Ψ^{-1} Λ (Λᵀ Ψ^{-1} Λ)^{-1} Φ (if Φ exists)
		psiInvLoadings := mat.NewDense(p, m, nil)
		psiInvLoadings.Mul(psiInv, loadings)

		var middle mat.Dense
		middle.Mul(loadings.T(), psiInvLoadings)
		var middleInv mat.Dense
		if err := middleInv.Inverse(&middle); err != nil {
			return nil, fmt.Errorf("bartlett weights: failed to invert Λᵀ Ψ^{-1} Λ: %v", err)
		}

		weights := mat.NewDense(p, m, nil)
		weights.Mul(psiInvLoadings, &middleInv)
		if phi != nil {
			var tmp mat.Dense
			tmp.Mul(weights, phi)
			weights.Copy(&tmp)
		}

		scores := mat.NewDense(n, m, nil)
		scores.Mul(data, weights)
		return scores, nil

	case FactorScoreAndersonRubin:
		if phi != nil {
			// Anderson-Rubin requires orthogonal factors (Φ = I)
			return nil, fmt.Errorf("anderson-rubin scoring requires orthogonal factors")
		}

		// Anderson-Rubin: B = Σ^{-1} Λ (Λᵀ Σ^{-1} Λ)^{-1/2}
		var sigmaInvLoadings mat.Dense
		sigmaInvLoadings.Mul(&sigmaInv, loadings)

		var inner mat.Dense
		inner.Mul(loadings.T(), &sigmaInvLoadings)
		invSqrt, err := inverseSqrtDense(&inner)
		if err != nil {
			return nil, fmt.Errorf("anderson-rubin weights: %v", err)
		}

		var weights mat.Dense
		weights.Mul(&sigmaInvLoadings, invSqrt)

		scores := mat.NewDense(n, m, nil)
		scores.Mul(data, &weights)
		return scores, nil

	default:
		return nil, fmt.Errorf("unsupported scoring method: %v", method)
	}
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

// reflectFactorsForPositiveLoadings reflects factors to ensure positive loadings
// This makes factor interpretations more consistent by ensuring the largest loading is positive
func reflectFactorsForPositiveLoadings(loadings *mat.Dense) *mat.Dense {
	if loadings == nil {
		return nil
	}

	r, c := loadings.Dims()
	result := mat.NewDense(r, c, nil)
	result.Copy(loadings)

	for j := 0; j < c; j++ {
		// Find the maximum absolute value in this column
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

		// If the maximum absolute value is negative, reflect the factor
		if maxVal < 0 {
			for i := 0; i < r; i++ {
				result.Set(i, j, -result.At(i, j))
			}
		}
	}

	return result
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
