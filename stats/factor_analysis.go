package stats

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra"
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
	FactorRotationPromax    FactorRotationMethod = "promax"
	FactorRotationOblimin   FactorRotationMethod = "oblimin"
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
	Tol        float64 // Optional: default 1e-4
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
	Communalities        insyra.IDataTable // Communality vector (p × 1)
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

// Show prints everything in the FactorAnalysisResult
func (r *FactorAnalysisResult) Show(startEndRange ...any) {
	insyra.Show("Loadings", r.Loadings, startEndRange...)
	insyra.Show("Structure", r.Structure, startEndRange...)
	insyra.Show("Uniquenesses", r.Uniquenesses, startEndRange...)
	insyra.Show("Communalities", r.Communalities, startEndRange...)
	insyra.Show("Phi", r.Phi, startEndRange...)
	insyra.Show("RotationMatrix", r.RotationMatrix, startEndRange...)
	insyra.Show("Eigenvalues", r.Eigenvalues, startEndRange...)
	insyra.Show("ExplainedProportion", r.ExplainedProportion, startEndRange...)
	insyra.Show("CumulativeProportion", r.CumulativeProportion, startEndRange...)
	insyra.Show("Scores", r.Scores, startEndRange...)
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
			Method: FactorRotationNone, // R default: "none"
			Kappa:  4,                  // R default for promax
			Delta:  0,
		},
		Scoring: FactorScoreNone, // R default: "none"
		MaxIter: 50,              // R default: 50
		Tol:     1e-4,            // R default: 1e-4
		MinErr:  0.001,           // R default: 0.001
	}
}

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
			if delta > 1e-6 {
				corrMatrix.SetSym(i, i, 1.0)
			}
		}
		if maxDiagDeviation > 1e-8 {
			insyra.LogDebug("stats", "FactorAnalysis", "correlation diag max deviation = %.6g", maxDiagDeviation)
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
	// Treat Tol < 0 as unspecified/default and set to 1e-6.
	// If Tol == 0, respect it as an explicit request to disable tol-based convergence
	// (i.e. rely solely on MaxIter), which mirrors R's behavior when tol is very small
	// or when the user wants to force iteration limits.
	if opt.Tol < 0 {
		opt.Tol = 1e-6
	}

	// Step 6: Extract factors
	// Convert SymDense to Dense for extraction functions
	corrDense := mat.NewDense(colNum, colNum, nil)
	for i := 0; i < colNum; i++ {
		for j := 0; j < colNum; j++ {
			corrDense.Set(i, j, corrMatrix.At(i, j))
		}
	}
	loadings, converged, iterations, err := extractFactors(data, corrDense, sortedEigenvalues, sortedEigenvectors, numFactors, opt, rowNum)
	if err != nil {
		insyra.LogWarning("stats", "FactorAnalysis", "factor extraction failed: %v", err)
		return nil
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
		rotatedLoadings, rotationMatrix, phi = rotateFactors(loadings, opt.Rotation)
	} else {
		rotatedLoadings = loadings
		rotationMatrix = nil
		phi = nil
	}

	// Apply factor reflection to ensure positive loadings (match R's convention)
	var preReflection *mat.Dense
	if rotatedLoadings != nil {
		copyLoadings := mat.NewDense(rotatedLoadings.RawMatrix().Rows, rotatedLoadings.RawMatrix().Cols, nil)
		copyLoadings.Copy(rotatedLoadings)
		preReflection = copyLoadings
		rotatedLoadings = reflectFactorsForPositiveLoadings(rotatedLoadings)
	}
	if rotationMatrix != nil && preReflection != nil {
		rotationMatrix = reflectRotationMatrix(rotationMatrix, rotatedLoadings, preReflection)
	}
	if phi != nil && preReflection != nil {
		phi = reflectPhiMatrix(phi, rotatedLoadings, preReflection)
	}

	// Sort factors by explained variance (following R's psych package)
	rotatedLoadings, rotationMatrix, phi = sortFactorsByExplainedVariance(rotatedLoadings, rotationMatrix, phi)

	// Step 8: Compute communalities and uniquenesses
	communalities := make([]float64, colNum)
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
		communalities[i] = hi2
		uniq := diag - hi2
		if uniq < 1e-9 {
			uniq = 1e-9
		}
		uniquenesses[i] = uniq
	}

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
		scores = computeFactorScores(data, rotatedLoadings, phi, uniquenesses, opt.Scoring)
	}

	// Convert results to DataTables
	// Generate factor column names
	factorColNames := make([]string, numFactors)
	for i := 0; i < numFactors; i++ {
		factorColNames[i] = fmt.Sprintf("Factor_%d", i+1)
	}
	structureTable := matrixToDataTableWithNames(S, "Factor Structure", factorColNames, colNames)

	messages := []string{
		fmt.Sprintf("Extraction method: %s", opt.Extraction),
		fmt.Sprintf("Factor count method: %s (retained %d)", opt.Count.Method, numFactors),
		fmt.Sprintf("Rotation method: %s", opt.Rotation.Method),
		fmt.Sprintf("Scoring method: %s", opt.Scoring),
	}
	if iterations > 0 {
		messages = append(messages, fmt.Sprintf("Extraction iterations: %d (tol %.2g)", iterations, opt.Tol))
	}
	if !converged && (opt.Extraction == FactorExtractionPAF || opt.Extraction == FactorExtractionML || opt.Extraction == FactorExtractionMINRES) {
		messages = append(messages, "Warning: extraction did not converge within limits")
	}
	if phi != nil {
		messages = append(messages, "Oblique rotation applied; factor correlation matrix provided")
	}
	messages = append(messages, "Factor analysis completed")

	result := FactorAnalysisResult{
		Loadings:             matrixToDataTableWithNames(rotatedLoadings, "Factor Loadings", factorColNames, colNames),
		Structure:            structureTable,
		Uniquenesses:         vectorToDataTableWithNames(uniquenesses, "Uniqueness", "Uniqueness", colNames),
		Communalities:        vectorToDataTableWithNames(communalities, "Communality", "Communality", colNames),
		Phi:                  nil,
		RotationMatrix:       nil,
		Eigenvalues:          vectorToDataTableWithNames(sortedEigenvalues, "Eigenvalue", "Eigenvalue", factorColNames),
		ExplainedProportion:  vectorToDataTableWithNames(explainedProp, "Explained Proportion", "Explained Proportion", factorColNames),
		CumulativeProportion: vectorToDataTableWithNames(cumulativeProp, "Cumulative Proportion", "Cumulative Proportion", factorColNames),
		Scores:               nil,
		Converged:            converged,
		Iterations:           iterations,
		CountUsed:            numFactors,
		Messages:             messages,
	}

	if rotationMatrix != nil {
		result.RotationMatrix = matrixToDataTableWithNames(rotationMatrix, "Rotation", factorColNames, factorColNames)
	}
	if phi != nil {
		result.Phi = matrixToDataTableWithNames(phi, "Phi", factorColNames, factorColNames)
	}
	if scores != nil {
		result.Scores = matrixToDataTableWithNames(scores, "Scores", factorColNames, rowNames)
	}

	return &FactorModel{
		FactorAnalysisResult: result,
		scoreMethod:          opt.Scoring,
		extraction:           opt.Extraction,
		rotation:             opt.Rotation.Method,
		means:                means,
		sds:                  sds,
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

// extractFactors extracts factors using the specified method
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions, sampleSize int) (*mat.Dense, bool, int, error) {
	switch opt.Extraction {
	case FactorExtractionPCA:
		return extractPCA(eigenvalues, eigenvectors, numFactors)

	case FactorExtractionPAF:
		return extractPAF(corrMatrix, numFactors, opt.MaxIter, opt.Tol, opt.MinErr)

	case FactorExtractionML:
		return extractML(corrMatrix, numFactors, opt.MaxIter, opt.Tol, sampleSize)

	case FactorExtractionMINRES:
		return extractMINRES(corrMatrix, numFactors, opt.MaxIter, opt.Tol)

	default:
		// Default to MINRES to match R psych::fa and the documented default behavior.
		return extractMINRES(corrMatrix, numFactors, opt.MaxIter, opt.Tol)
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
	loadings := mat.NewDense(p, numFactors, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < numFactors; j++ {
			if j < len(eigenvalues) && eigenvalues[j] > 0 {
				loadings.Set(i, j, eigenvectors.At(i, j)*math.Sqrt(eigenvalues[j]))
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
func initialCommunalitiesSMC(corr *mat.Dense, minErr float64) []float64 {
	// Compute SMC (Squared Multiple Correlation) estimates
	// SMC_i = 1 - 1/R_ii^inv where R_ii^inv is diagonal of inverse correlation matrix
	h2 := computeSMC(corr, minErr)
	return h2
}

// computeSMC computes Squared Multiple Correlation estimates using R's approximation
func computeSMC(corr *mat.Dense, minErr float64) []float64 {
	p, _ := corr.Dims()
	h2 := make([]float64, p)

	// Convert to SymDense for inversion
	symCorr := mat.NewSymDense(p, nil)
	for i := 0; i < p; i++ {
		for j := i; j < p; j++ {
			symCorr.SetSym(i, j, corr.At(i, j))
		}
	}

	// Compute inverse of the full correlation matrix
	var invCorr mat.Dense
	err := invCorr.Inverse(symCorr)
	if err != nil {
		// If inversion fails, add small ridge and try again
		for i := 0; i < p; i++ {
			symCorr.SetSym(i, i, symCorr.At(i, i)+1e-6)
		}
		err = invCorr.Inverse(symCorr)
	}
	if err != nil {
		// If still fails, use default communalities of 0.5
		insyra.LogWarning("stats", "computeSMC", "correlation matrix inversion failed, using default h2=0.5")
		for i := 0; i < p; i++ {
			h2[i] = 0.5
		}
		return h2
	}

	// SMC_i = 1 - 1/R^{-1}_{ii}
	for i := 0; i < p; i++ {
		invDiag := invCorr.At(i, i)
		if invDiag <= 0 {
			h2[i] = 0.5 // fallback
		} else {
			smc := 1.0 - 1.0/invDiag
			// SMC can be negative or > 1 in theory, so clamp to reasonable range
			if smc < minErr {
				smc = minErr
			}
			if smc > 1.0 {
				smc = 1.0
			}
			h2[i] = smc
		}
	}

	return h2
} // extractPAF extracts factors using Principal Axis Factoring
func extractPAF(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64, minErr float64) (*mat.Dense, bool, int, error) {
	p := corrMatrix.RawMatrix().Rows

	// Initialize communalities with SMC (Squared Multiple Correlation)
	communalities := initialCommunalitiesSMC(corrMatrix, minErr)

	// Log initial communalities for verification (per issue #xxx)
	if p >= 4 {
		insyra.LogInfo("stats", "PAF", "init h2 (first 4) = %.3f, %.3f, %.3f, %.3f",
			communalities[0], communalities[1], communalities[2], communalities[3])
	}

	var loadings *mat.Dense
	converged := false

	// Keep history of communalities for debugging/comparison with R
	commHistory := make([][]float64, 0, maxIter)

	for iter := 0; iter < maxIter; iter++ {
		// Create reduced correlation matrix with communalities on diagonal
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

		// Eigenvalue decomposition - ensure matrix is symmetric
		var eig mat.EigenSym
		symReduced := mat.NewSymDense(p, nil)
		for i := 0; i < p; i++ {
			for j := i; j < p; j++ {
				// Ensure perfect symmetry
				val := reducedCorr.At(i, j)
				symReduced.SetSym(i, j, val)
			}
		}

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

		// Extract loadings
		loadings = mat.NewDense(p, numFactors, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < numFactors; j++ {
				if pairs[j].value > 0 {
					loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(pairs[j].value))
				} else {
					loadings.Set(i, j, 0)
				}
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
			if sum < 1e-6 {
				sum = 1e-6
			}
			if sum > diag {
				sum = diag // Allow communalities to reach diagonal
			}
			newCommunalities[i] = sum
		}

		if iter == 0 && p >= 4 {
			insyra.LogInfo("stats", "PAF", "iter %d post-loading communalities[0:4]=%.3f, %.3f, %.3f, %.3f", iter+1,
				newCommunalities[0], newCommunalities[min(1,p-1)], newCommunalities[min(2,p-1)], newCommunalities[min(3,p-1)])
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

	// If we reach here, did not converge within maxIter. Log a concise history
	insyra.LogWarning("stats", "PAF", "Did not converge after %d iterations", maxIter)

	// Log initial SMC and history summary (first 10 iterations and final)
	if len(commHistory) > 0 {
		// initial SMC = first value prior to iteration (communalities passed in initially)
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
	tr := 0.0
	for i := 0; i < p; i++ {
		tr += temp.At(i, i)
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
		tol = 1e-6
	}

	initial, _, _, err := extractPAF(corrMatrix, numFactors, min(maxIter, 50), math.Max(tol, 1e-6), 0.001)
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
			initialComm[0], initialComm[min(1,len(initialComm)-1)], initialComm[min(2,len(initialComm)-1)], initialComm[min(3,len(initialComm)-1)])
	}

	loadings := mat.Dense{}
	loadings.CloneFrom(initial)

	psMin := 1e-6
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
				iter+1, currComm[0], currComm[min(1,len(currComm)-1)], currComm[min(2,len(currComm)-1)], currComm[min(3,len(currComm)-1)])
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

	insyra.LogWarning("stats", "ML", "did not converge after %d iterations", maxIter)
	return &loadings, converged, maxIter, nil
}

// extractMINRES extracts factors using the minimum residual approach (akin to ULS)
func extractMINRES(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64) (*mat.Dense, bool, int, error) {
	p, _ := corrMatrix.Dims()
	if numFactors <= 0 || numFactors > p {
		return nil, false, 0, fmt.Errorf("invalid number of factors: %d", numFactors)
	}

	if maxIter <= 0 {
		maxIter = 500
	}
	if tol <= 0 {
		tol = 1e-6
	}

	initial, _, _, err := extractPAF(corrMatrix, numFactors, min(maxIter, 100), math.Max(tol, 1e-6), 0.001)
	if err != nil || initial == nil {
		// Fall back to PCA loadings if PAF initialization fails
		var eig mat.EigenSym
		symCorr := denseToSym(corrMatrix)
		if !eig.Factorize(symCorr, true) {
			return nil, false, 0, fmt.Errorf("minres: eigen factorization failed: %w", err)
		}
		eigs := eig.Values(nil)
		var vec mat.Dense
		eig.VectorsTo(&vec)
		initial, _, _, err = extractPCA(eigs, &vec, numFactors)
		if err != nil {
			return nil, false, 0, fmt.Errorf("minres: unable to build initial loadings: %w", err)
		}
	}

	var loadings mat.Dense
	loadings.CloneFrom(initial)
	clampLoadingsToDiag(&loadings, corrMatrix)

	if p >= 4 {
		insyra.LogInfo("stats", "MINRES", "initial loadings[0,0:2] = %.3f, %.3f",
			loadings.At(0, 0), loadings.At(0, min(1, numFactors-1)))
	}

	var approx mat.Dense
	var residual mat.Dense
	var grad mat.Dense
	prevSSE := math.Inf(1)
	converged := false

	for iter := 0; iter < maxIter; iter++ {
		// Compute approximation: L L^T
		approx.Mul(&loadings, loadings.T())
		residual.Sub(corrMatrix, &approx)
		zeroDiagonal(&residual)

		currentSSE := offDiagonalSSE(&residual)
		if currentSSE < tol {
			converged = true
			insyra.LogInfo("stats", "MINRES", "converged in %d iterations (SSE=%.6f)", iter+1, currentSSE)
			return &loadings, converged, iter + 1, nil
		}
		if math.Abs(prevSSE-currentSSE) < tol {
			converged = true
			insyra.LogInfo("stats", "MINRES", "converged by delta in %d iterations (SSE=%.6f)", iter+1, currentSSE)
			return &loadings, converged, iter + 1, nil
		}

		// Compute gradient: -4 * (R - L L^T) * L
		grad.Mul(&residual, &loadings)
		grad.Scale(-4.0, &grad)

		// Line search for optimal step size
		eta := 1.0
		improved := false
		var candidateSSE float64
		var newLoadings mat.Dense

		for trial := 0; trial < 10; trial++ {
			// L_new = L - eta * grad
			var scaled mat.Dense
			scaled.Scale(eta, &grad)
			newLoadings.Sub(&loadings, &scaled)
			clampLoadingsToDiag(&newLoadings, corrMatrix)

			var candidateApprox mat.Dense
			candidateApprox.Mul(&newLoadings, newLoadings.T())
			var candidateResidual mat.Dense
			candidateResidual.Sub(corrMatrix, &candidateApprox)
			zeroDiagonal(&candidateResidual)
			candidateSSE = offDiagonalSSE(&candidateResidual)

			if candidateSSE < currentSSE {
				loadings.CloneFrom(&newLoadings)
				prevSSE = currentSSE
				improved = true
				break
			}
			eta *= 0.5
		}

		if !improved {
			prevSSE = currentSSE
			break
		}

		if iter < 5 || iter == maxIter-1 {
			insyra.LogDebug("stats", "MINRES", "iter %d: SSE=%.6f, eta=%.4f", iter+1, candidateSSE, eta)
		}

		if math.Abs(prevSSE-candidateSSE) < tol {
			converged = true
			insyra.LogInfo("stats", "MINRES", "converged by delta in %d iterations (SSE=%.6f)", iter+1, candidateSSE)
			return &loadings, converged, iter + 1, nil
		}
	}

	if !converged {
		insyra.LogWarning("stats", "MINRES", "did not converge after %d iterations (SSE=%.6f)", maxIter, prevSSE)
	}
	return &loadings, converged, maxIter, nil
}

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

func denseToSym(m *mat.Dense) *mat.SymDense {
	r, c := m.Dims()
	if r != c {
		panic("denseToSym: matrix not square")
	}
	sym := mat.NewSymDense(r, nil)
	for i := 0; i < r; i++ {
		for j := i; j < c; j++ {
			val := 0.5 * (m.At(i, j) + m.At(j, i))
			sym.SetSym(i, j, val)
		}
	}
	return sym
}

func zeroDiagonal(m *mat.Dense) {
	r, c := m.Dims()
	limit := min(r, c)
	for i := 0; i < limit; i++ {
		m.Set(i, i, 0)
	}
}

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

func clampLoadingsToDiag(loadings *mat.Dense, corrMatrix *mat.Dense) {
	rows, cols := loadings.Dims()
	for i := 0; i < rows; i++ {
		diag := corrMatrix.At(i, i)
		if diag <= 0 {
			diag = 1.0
		}
		sum := 0.0
		for j := 0; j < cols; j++ {
			val := loadings.At(i, j)
			sum += val * val
		}
		limit := diag - 1e-6
		if limit <= 1e-6 {
			limit = 1e-6
		}
		if sum > limit {
			scale := math.Sqrt(limit / sum)
			for j := 0; j < cols; j++ {
				loadings.Set(i, j, loadings.At(i, j)*scale)
			}
		}
	}
}

// normalizeToCorrelation normalizes a matrix to have diagonal elements equal to 1
// This converts a covariance-like matrix to a correlation matrix
func normalizeToCorrelation(m *mat.Dense) *mat.Dense {
	r, c := m.Dims()
	if r != c {
		return m
	}
	d := make([]float64, r)
	hasZeroDiag := false
	for i := 0; i < r; i++ {
		v := m.At(i, i)
		if v <= 1e-10 {
			// Diagonal is essentially zero - this factor is degenerate
			// Just return identity correlation for this factor
			d[i] = 1.0
			hasZeroDiag = true
		} else {
			d[i] = math.Sqrt(v)
		}
	}

	// If we have degenerate factors, return a properly formed correlation matrix
	if hasZeroDiag {
		out := mat.NewDense(r, r, nil)
		for i := 0; i < r; i++ {
			for j := 0; j < r; j++ {
				if i == j {
					out.Set(i, j, 1.0)
				} else if d[i] > 1e-10 && d[j] > 1e-10 {
					out.Set(i, j, m.At(i, j)/(d[i]*d[j]))
				} else {
					out.Set(i, j, 0.0)
				}
			}
		}
		return out
	}

	// Normal case - no degenerate factors
	out := mat.NewDense(r, r, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < r; j++ {
			out.Set(i, j, m.At(i, j)/(d[i]*d[j]))
		}
	}
	return out
}

// rotateFactors rotates the factor loadings according to the provided options.
// It returns the rotated loadings, the transformation matrix, and the factor correlation matrix (phi) when available.
func rotateFactors(loadings *mat.Dense, opt FactorRotationOptions) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil {
		return nil, nil, nil
	}

	_, m := loadings.Dims()
	if m <= 1 || opt.Method == FactorRotationNone {
		return loadings, nil, nil
	}

	rotMaxIter := 200
	rotTol := 1e-6
	switch opt.Method {
	case FactorRotationVarimax:
		rotated, rotMatrix, err := rotateOrthomax(loadings, 1.0, rotMaxIter, rotTol)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "varimax rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, nil

	case FactorRotationQuartimax:
		rotated, rotMatrix, err := rotateOrthomax(loadings, 0.0, rotMaxIter, rotTol)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "quartimax rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, nil

	case FactorRotationPromax:
		kappa := opt.Kappa
		if kappa == 0 {
			kappa = 4
		}
		rotated, rotMatrix, phi, err := rotatePromax(loadings, kappa, rotMaxIter, rotTol)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "promax rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, phi

	case FactorRotationOblimin:
		delta := opt.Delta
		rotated, rotMatrix, phi, err := rotateOblimin(loadings, delta, rotMaxIter, rotTol)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "oblimin rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, phi

	default:
		return loadings, nil, nil
	}
}

// kaiserNormalize applies Kaiser normalization to the loading matrix by scaling rows to unit length.
func kaiserNormalize(L *mat.Dense) (*mat.Dense, *mat.VecDense) {
	r, c := L.Dims()
	out := mat.NewDense(r, c, nil)
	w := mat.NewVecDense(r, nil)
	for i := 0; i < r; i++ {
		var s float64
		for j := 0; j < c; j++ {
			v := L.At(i, j)
			s += v * v
		}
		if s <= 0 {
			s = 1e-12
		}
		wi := math.Sqrt(s)
		w.SetVec(i, wi)
		for j := 0; j < c; j++ {
			out.Set(i, j, L.At(i, j)/wi)
		}
	}
	return out, w
}

// kaiserDenorm rescales a Kaiser-normalized loading matrix back to its original scale.
func kaiserDenorm(Lnorm *mat.Dense, w *mat.VecDense) *mat.Dense {
	r, c := Lnorm.Dims()
	out := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		wi := w.AtVec(i)
		for j := 0; j < c; j++ {
			out.Set(i, j, Lnorm.At(i, j)*wi)
		}
	}
	return out
}

// rotateOrthomax performs orthogonal rotation (varimax, quartimax, etc.)
// Returns the rotated loadings (denormalized) and rotation matrix.
func rotateOrthomax(loadings *mat.Dense, gamma float64, maxIter int, tol float64) (*mat.Dense, *mat.Dense, error) {
	rotatedNorm, rotation, weights, err := rotateOrthomaxNormalized(loadings, gamma, maxIter, tol)
	if err != nil {
		return nil, nil, err
	}
	return kaiserDenorm(rotatedNorm, weights), rotation, nil
}

// rotateOrthomaxNormalized performs orthogonal rotation on Kaiser-normalized loadings.
// Returns the rotated normalized loadings, rotation matrix, and Kaiser weights.
// This is used internally by Promax to avoid double normalization/denormalization.
func rotateOrthomaxNormalized(loadings *mat.Dense, gamma float64, maxIter int, tol float64) (*mat.Dense, *mat.Dense, *mat.VecDense, error) {
	Lnorm, w := kaiserNormalize(loadings)

	p, m := Lnorm.Dims()
	rotation := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		rotation.Set(i, i, 1)
	}

	rotated := mat.NewDense(p, m, nil)
	rotated.Copy(Lnorm)

	prevObj := orthomaxObjective(rotated, gamma)

	for iter := 0; iter < maxIter; iter++ {
		rowNorms := make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < m; j++ {
				v := rotated.At(i, j)
				sum += v * v
			}
			rowNorms[i] = sum
		}

		term := mat.NewDense(p, m, nil)
		for i := 0; i < p; i++ {
			rowScale := gamma * rowNorms[i] / float64(p)
			for j := 0; j < m; j++ {
				v := rotated.At(i, j)
				term.Set(i, j, 4*v*(v*v-rowScale))
			}
		}

		var grad mat.Dense
		grad.Mul(rotated.T(), term)

		var gradT mat.Dense
		gradT.CloneFrom(grad.T())

		var skew mat.Dense
		skew.Sub(&grad, &gradT)
		skew.Scale(0.5, &skew)

		if frobeniusNormDense(&skew) < tol {
			return rotated, rotation, w, nil
		}

		step := 1.0
		improved := false
		for attempt := 0; attempt < 6; attempt++ {
			var delta mat.Dense
			delta.Mul(rotation, &skew)
			delta.Scale(step, &delta)

			var trial mat.Dense
			trial.Add(rotation, &delta)

			var qr mat.QR
			qr.Factorize(&trial)
			var q mat.Dense
			qr.QTo(&q)

			var trialRotated mat.Dense
			trialRotated.Mul(Lnorm, &q)

			obj := orthomaxObjective(&trialRotated, gamma)
			if obj > prevObj+1e-10 {
				rotation.Copy(&q)
				rotated.CloneFrom(&trialRotated)
				prevObj = obj
				improved = true
				break
			}
			step *= 0.5
		}

		if !improved {
			break
		}
	}

	return rotated, rotation, w, nil
}

func rotatePromax(loadings *mat.Dense, kappa float64, maxIter int, tol float64) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	if kappa == 0 {
		kappa = 4
	}

	p, m := loadings.Dims()

	// Step 1: Kaiser normalize the loadings and perform varimax on normalized loadings
	// This is the KEY FIX - R's psych::fa with promax ALWAYS uses Kaiser normalization
	rotatedNorm, rotation, weights, err := rotateOrthomaxNormalized(loadings, 1.0, maxIter, tol)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("promax: varimax failed: %w", err)
	}

	// Step 2: Build target matrix Q = sign(L0_norm) * |L0_norm|^kappa
	// Important: target is computed on NORMALIZED loadings
	target := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			val := rotatedNorm.At(i, j)
			sign := 1.0
			if val < 0 {
				sign = -1.0
			} else if val == 0 {
				sign = 0.0
			}
			target.Set(i, j, sign*math.Pow(math.Abs(val), kappa))
		}
	}

	// Step 3: Solve for transformation T = (L0_norm' L0_norm)^-1 L0_norm' Q
	var L0t mat.Dense
	L0t.CloneFrom(rotatedNorm.T())

	var L0tL0 mat.Dense
	L0tL0.Mul(&L0t, rotatedNorm)

	var L0tL0Inv mat.Dense
	if err := safeInvert(&L0tL0Inv, &L0tL0, 1e-10); err != nil {
		return nil, nil, nil, fmt.Errorf("promax: unable to invert L0'L0: %w", err)
	}

	var L0tQ mat.Dense
	L0tQ.Mul(&L0t, target)

	var trans mat.Dense
	trans.Mul(&L0tL0Inv, &L0tQ)

	// Step 4: Compute pattern loadings on NORMALIZED space FIRST: P_norm = L0_norm * T
	// Do this BEFORE normalizing T!
	var patternNorm mat.Dense
	patternNorm.Mul(rotatedNorm, &trans)

	// Step 5: DENORMALIZE the pattern to get back to original scale
	// Pattern = unnormalize(P_norm) using the Kaiser weights
	pattern := kaiserDenorm(&patternNorm, weights)

	// Step 6: Compute Phi from unnormalized T first
	// Phi = cov2cor( (T' * T)^-1 )
	var TTt mat.Dense
	TTt.Mul(trans.T(), &trans)

	var phiInv mat.Dense
	if err := safeInvert(&phiInv, &TTt, 1e-10); err != nil {
		return nil, nil, nil, fmt.Errorf("promax: unable to compute Phi: %w", err)
	}

	// Convert Phi to correlation matrix (cov2cor equivalent)
	r, c := phiInv.Dims()
	diagSqrt := make([]float64, r)
	for i := 0; i < r; i++ {
		diagSqrt[i] = math.Sqrt(phiInv.At(i, i))
	}
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			phiInv.Set(i, j, phiInv.At(i, j)/(diagSqrt[i]*diagSqrt[j]))
		}
	}

	// Now normalize T for pattern computation
	// Calculate scales
	scales := make([]float64, m)
	for j := 0; j < m; j++ {
		diag := TTt.At(j, j)
		if diag > 1e-12 {
			scales[j] = 1.0 / math.Sqrt(diag)
		} else {
			scales[j] = 1.0
		}
	}

	// Apply scales
	for j := 0; j < m; j++ {
		scale := scales[j]
		for i := 0; i < m; i++ {
			trans.Set(i, j, trans.At(i, j)*scale)
		}
	}

	// Step 8: Compute combined rotation
	var combined mat.Dense
	combined.Mul(rotation, &trans)

	return pattern, &combined, &phiInv, nil
}

func rotateOblimin(loadings *mat.Dense, delta float64, maxIter int, tol float64) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	p, m := loadings.Dims()
	trans := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		trans.Set(i, i, 1)
	}

	rotated := mat.NewDense(p, m, nil)
	rotated.Copy(loadings)

	prevObj := obliminObjective(rotated, delta)

	for iter := 0; iter < maxIter; iter++ {
		rowNorms := make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < m; j++ {
				val := rotated.At(i, j)
				sum += val * val
			}
			rowNorms[i] = sum
		}

		term := mat.NewDense(p, m, nil)
		for i := 0; i < p; i++ {
			rowScale := delta * rowNorms[i] / float64(p)
			for j := 0; j < m; j++ {
				val := rotated.At(i, j)
				term.Set(i, j, 4*val*(val*val-rowScale))
			}
		}

		var grad mat.Dense
		grad.Mul(loadings.T(), term)

		norm := frobeniusNormDense(&grad)
		if norm < tol {
			break
		}

		step := 1.0
		improved := false
		for attempt := 0; attempt < 6; attempt++ {
			var scaledGrad mat.Dense
			scaledGrad.Scale(step, &grad)

			var trial mat.Dense
			trial.Add(trans, &scaledGrad)

			var trialRot mat.Dense
			trialRot.Mul(loadings, &trial)

			obj := obliminObjective(&trialRot, delta)
			if obj > prevObj+1e-10 {
				trans.CloneFrom(&trial)
				rotated.CloneFrom(&trialRot)
				prevObj = obj
				improved = true
				break
			}
			step *= 0.5
		}

		if !improved {
			break
		}
	}

	var transInv mat.Dense
	if err := invertWithFallback(&transInv, trans); err != nil {
		return rotated, trans, nil, fmt.Errorf("oblimin: transformation not invertible: %w", err)
	}

	var transInvT mat.Dense
	transInvT.CloneFrom(transInv.T())

	var phi mat.Dense
	phi.Mul(&transInv, &transInvT)

	// Normalize Phi to have diagonal elements equal to 1 (correlation matrix)
	diagScales := make([]float64, m)
	for j := 0; j < m; j++ {
		diagVal := phi.At(j, j)
		if diagVal <= 1e-12 {
			diagScales[j] = 1.0
		} else {
			diagScales[j] = math.Sqrt(diagVal)
		}
	}
	phiNorm := normalizeToCorrelation(&phi)

	rotatedScaled := mat.NewDense(p, m, nil)
	rotatedScaled.Copy(rotated)
	for j := 0; j < m; j++ {
		scale := diagScales[j]
		if scale == 0 {
			scale = 1.0
		}
		for i := 0; i < p; i++ {
			rotatedScaled.Set(i, j, rotatedScaled.At(i, j)*scale)
		}
	}

	return rotatedScaled, trans, phiNorm, nil
}

func invertWithFallback(dst *mat.Dense, src mat.Matrix) error {
	if m, ok := src.(*mat.Dense); ok {
		if err := dst.Inverse(m); err == nil {
			return nil
		}
	}
	var dense mat.Dense
	dense.CloneFrom(src)
	if err := dst.Inverse(&dense); err == nil {
		return nil
	}
	return pseudoInverse(dst, &dense)
}

// safeInvert inverts a matrix with optional regularization for numerical stability
func safeInvert(dst *mat.Dense, src mat.Matrix, ridge float64) error {
	var a mat.Dense
	a.CloneFrom(src)
	r, c := a.Dims()
	if r == c && ridge > 0 {
		for i := 0; i < r; i++ {
			a.Set(i, i, a.At(i, i)+ridge)
		}
	}
	if err := dst.Inverse(&a); err != nil {
		// Still failed, try pseudo-inverse
		return pseudoInverse(dst, &a)
	}
	return nil
}

func pseudoInverse(dst *mat.Dense, src mat.Matrix) error {
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
		if val > 1e-10 {
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

func matrixToString(m mat.Matrix) string {
	r, c := m.Dims()
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < r; i++ {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString("[")
		for j := 0; j < c; j++ {
			if j > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%.4f", m.At(i, j)))
		}
		sb.WriteString("]")
	}
	sb.WriteString("]")
	return sb.String()
}

func frobeniusNormDense(m *mat.Dense) float64 {
	r, c := m.Dims()
	sum := 0.0
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			val := m.At(i, j)
			sum += val * val
		}
	}
	return math.Sqrt(sum)
}

func orthomaxObjective(loadings *mat.Dense, gamma float64) float64 {
	p, m := loadings.Dims()
	sum := 0.0
	for i := 0; i < p; i++ {
		rowSum := 0.0
		for j := 0; j < m; j++ {
			val := loadings.At(i, j)
			sum += math.Pow(val, 4)
			rowSum += val * val
		}
		sum -= gamma / float64(p) * rowSum * rowSum
	}
	return sum
}

func obliminObjective(loadings *mat.Dense, delta float64) float64 {
	p, m := loadings.Dims()
	sum := 0.0
	for i := 0; i < p; i++ {
		rowSum := 0.0
		for j := 0; j < m; j++ {
			val := loadings.At(i, j)
			sum += math.Pow(val, 4)

			rowSum += val * val
		}
		sum -= 2 * delta / float64(p) * rowSum * rowSum
	}
	return sum
}

// computeFactorScores computes factor scores
func computeFactorScores(data, loadings, phi *mat.Dense, uniquenesses []float64, method FactorScoreMethod) *mat.Dense {
	if loadings == nil {
		return nil
	}
	n, p := data.Dims()
	_, m := loadings.Dims()

	scores := mat.NewDense(n, m, nil)

	switch method {
	case FactorScoreNone:
		// No scores computed
		return nil
	case FactorScoreRegression:
		if phi != nil {
			if obliqueScores := computeFactorScoresRegressionOblique(data, loadings, phi, uniquenesses); obliqueScores != nil {
				return obliqueScores
			}
			insyra.LogWarning("stats", "FactorScores", "regression scoring oblique fallback to orthogonal weights")
		}
		// Regression method (orthogonal): Scores = X * Loadings * (Loadings' * Loadings)^-1
		var loadingsTrans mat.Dense
		loadingsTrans.CloneFrom(loadings.T())

		var prod mat.Dense
		prod.Mul(&loadingsTrans, loadings)

		var inv mat.Dense
		if err := safeInvert(&inv, &prod, 1e-6); err != nil {
			// If inversion fails, return zero scores
			return scores
		}

		var temp mat.Dense
		temp.Mul(loadings, &inv)

		scores.Mul(data, &temp)

	case FactorScoreBartlett:
		// Bartlett method: Uses inverse of uniqueness matrix
		// Scores = X * Psi^-1 * Loadings * (Loadings' * Psi^-1 * Loadings)^-1
		psiInvData := make([]float64, p)
		for i := 0; i < p; i++ {
			if i < len(uniquenesses) && uniquenesses[i] > 0 {
				psiInvData[i] = 1.0 / uniquenesses[i]
			} else {
				psiInvData[i] = 1.0
			}
		}
		psiInv := mat.NewDiagDense(p, psiInvData)

		var temp1 mat.Dense
		temp1.Mul(psiInv, loadings)

		var loadingsTrans mat.Dense
		loadingsTrans.CloneFrom(loadings.T())

		var temp2 mat.Dense
		temp2.Mul(&loadingsTrans, &temp1)

		var inv mat.Dense
		if err := safeInvert(&inv, &temp2, 1e-6); err != nil {
			// Fallback to regression method
			return computeFactorScores(data, loadings, phi, uniquenesses, FactorScoreRegression)
		}

		var temp3 mat.Dense
		temp3.Mul(&temp1, &inv)

		scores.Mul(data, &temp3)

	case FactorScoreAndersonRubin:
		// Anderson-Rubin method: Produces uncorrelated scores
		// Similar to regression but with additional normalization
		var loadingsTrans mat.Dense
		loadingsTrans.CloneFrom(loadings.T())

		var prod mat.Dense
		prod.Mul(&loadingsTrans, loadings)

		var inv mat.Dense
		if err := safeInvert(&inv, &prod, 1e-6); err != nil {
			return scores
		}

		var temp mat.Dense
		temp.Mul(loadings, &inv)

		scores.Mul(data, &temp)

		// Normalize scores to have unit variance
		for j := 0; j < m; j++ {
			col := mat.Col(nil, j, scores)
			_, std := stat.MeanStdDev(col, nil)
			if std > 0 {
				for i := 0; i < n; i++ {
					scores.Set(i, j, scores.At(i, j)/std)
				}
			}
		}

	default:
		// Default to regression method
		return computeFactorScores(data, loadings, phi, uniquenesses, FactorScoreRegression)
	}

	return scores
}

func computeFactorScoresRegressionOblique(data, loadings, phi *mat.Dense, uniquenesses []float64) *mat.Dense {
	if loadings == nil || phi == nil {
		return nil
	}
	n, p := data.Dims()
	loadRows, loadCols := loadings.Dims()
	m := loadCols
	phiRows, phiCols := phi.Dims()
	insyra.LogInfo("stats", "FactorScores", "oblique scoring dims: loadings %dx%d, phi %dx%d, data %dx%d", loadRows, loadCols, phiRows, phiCols, n, p)
	if phiRows != m || phiCols != m {
		insyra.LogWarning("stats", "FactorScores", "phi dimension mismatch: expected %dx%d, got %dx%d", m, m, phiRows, phiCols)
		return nil
	}

	psiVals := make([]float64, p)
	for i := 0; i < p; i++ {
		if i < len(uniquenesses) && uniquenesses[i] > 0 {
			psiVals[i] = uniquenesses[i]
		} else {
			psiVals[i] = 1.0
		}
	}
	lambdaPhi := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			sum := 0.0
			for k := 0; k < m; k++ {
				sum += loadings.At(i, k) * phi.At(k, j)
			}
			lambdaPhi.Set(i, j, sum)
		}
	}
	common := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			sum := 0.0
			for k := 0; k < m; k++ {
				sum += lambdaPhi.At(i, k) * loadings.At(j, k)
			}
			common.Set(i, j, sum)
		}
	}

	sigma := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			val := common.At(i, j)
			if i == j {
				val += psiVals[i]
			}
			sigma.Set(i, j, val)
		}
	}

	var sigmaInv mat.Dense
	if err := safeInvert(&sigmaInv, sigma, 1e-6); err != nil {
		insyra.LogWarning("stats", "FactorScores", "sigma inversion failed: %v", err)
		return nil
	}

	sigmaInvLambdaPhi := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			sum := 0.0
			for k := 0; k < p; k++ {
				sum += sigmaInv.At(i, k) * lambdaPhi.At(k, j)
			}
			sigmaInvLambdaPhi.Set(i, j, sum)
		}
	}

	ltSigInvL := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			sum := 0.0
			for k := 0; k < p; k++ {
				sum += loadings.At(k, i) * sigmaInvLambdaPhi.At(k, j)
			}
			ltSigInvL.Set(i, j, sum)
		}
	}

	var b mat.Dense
	if err := safeInvert(&b, ltSigInvL, 1e-6); err != nil {
		insyra.LogWarning("stats", "FactorScores", "weight inversion failed: %v", err)
		return nil
	}

	w := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			sum := 0.0
			for k := 0; k < m; k++ {
				sum += sigmaInvLambdaPhi.At(i, k) * b.At(k, j)
			}
			w.Set(i, j, sum)
		}
	}

	scores := mat.NewDense(n, m, nil)
	scores.Mul(data, w)

	return scores
}

// matrixToDataTableWithNames converts a gonum matrix to a DataTable with specified column and row names
func matrixToDataTableWithNames(m *mat.Dense, baseName string, colNames []string, rowNames []string) insyra.IDataTable {
	if m == nil {
		return nil
	}

	r, c := m.Dims()
	dt := insyra.NewDataTable()

	for j := 0; j < c; j++ {
		col := insyra.NewDataList()
		for i := 0; i < r; i++ {
			col.Append(m.At(i, j))
		}
		dt.AppendCols(col)
	}

	// Set column names
	for j := 0; j < c && j < len(colNames); j++ {
		dt.SetColNameByNumber(j, colNames[j])
	}

	// Set row names
	for i := 0; i < r && i < len(rowNames); i++ {
		dt.SetRowNameByIndex(i, rowNames[i])
	}

	return dt.SetName(baseName)
}

// vectorToDataTableWithNames converts a float slice to a single-column DataTable with names
func vectorToDataTableWithNames(v []float64, baseName string, colName string, rowNames []string) insyra.IDataTable {
	dt := insyra.NewDataTable()
	col := insyra.NewDataList(v)
	dt.AppendCols(col)
	dt.SetColNameByNumber(0, colName)
	for i, rowName := range rowNames {
		if i < len(v) {
			dt.SetRowNameByIndex(i, rowName)
		}
	}
	return dt.SetName(baseName)
}

// FactorScores computes factor scores for new data
func (m *FactorModel) FactorScores(dt insyra.IDataTable, method *FactorScoreMethod) (insyra.IDataTable, error) {
	if dt == nil {
		return nil, errors.New("nil DataTable")
	}

	chosen := m.scoreMethod
	if method != nil {
		chosen = *method
	}

	// Convert DataTable to matrix and standardize using saved means and sds
	var rowNum, colNum int
	var data *mat.Dense
	var rowNames []string

	dt.AtomicDo(func(table *insyra.DataTable) {
		rowNum, colNum = table.Size()
		data = mat.NewDense(rowNum, colNum, nil)

		// Get row names
		rowNames = make([]string, rowNum)
		for i := 0; i < rowNum; i++ {
			rowNames[i] = table.GetRowNameByIndex(i)
		}

		for i := 0; i < rowNum; i++ {
			row := table.GetRow(i)
			for j := 0; j < colNum; j++ {
				value, ok := row.Get(j).(float64)
				if ok {
					// Standardize using saved means and sds
					if j < len(m.means) && j < len(m.sds) && m.sds[j] > 0 {
						data.Set(i, j, (value-m.means[j])/m.sds[j])
					} else {
						data.Set(i, j, value)
					}
				}
			}
		}
	})

	// Extract loadings matrix
	var loadings *mat.Dense
	m.Loadings.AtomicDo(func(table *insyra.DataTable) {
		lr, lc := table.Size()
		loadings = mat.NewDense(lr, lc, nil)
		for i := 0; i < lr; i++ {
			row := table.GetRow(i)
			for j := 0; j < lc; j++ {
				value, ok := row.Get(j).(float64)
				if ok {
					loadings.Set(i, j, value)
				}
			}
		}
	})

	// Extract uniquenesses
	var uniquenesses []float64
	if m.Uniquenesses != nil {
		m.Uniquenesses.AtomicDo(func(table *insyra.DataTable) {
			ur, _ := table.Size()
			uniquenesses = make([]float64, ur)
			for i := 0; i < ur; i++ {
				row := table.GetRow(i)
				value, ok := row.Get(0).(float64) // First column
				if ok {
					uniquenesses[i] = value
				}
			}
		})
	}

	// Extract phi if available
	var phiMat *mat.Dense
	if m.Phi != nil {
		m.Phi.AtomicDo(func(table *insyra.DataTable) {
			pr, pc := table.Size()
			if pr == 0 || pc == 0 {
				return
			}
			phiMat = mat.NewDense(pr, pc, nil)
			for i := 0; i < pr; i++ {
				row := table.GetRow(i)
				for j := 0; j < pc; j++ {
					if value, ok := row.Get(j).(float64); ok {
						phiMat.Set(i, j, value)
					}
				}
			}
		})
	}

	scores := computeFactorScores(data, loadings, phiMat, uniquenesses, chosen)

	// Generate factor column names
	_, numFactors := loadings.Dims()
	factorColNames := make([]string, numFactors)
	for i := 0; i < numFactors; i++ {
		factorColNames[i] = fmt.Sprintf("Factor_%d", i+1)
	}

	return matrixToDataTableWithNames(scores, "Scores", factorColNames, rowNames), nil
}

// ScreePlotData returns scree plot data (eigenvalues and cumulative proportion)
func ScreePlotData(dt insyra.IDataTable, standardize bool) (eigenDT insyra.IDataTable, cumDT insyra.IDataTable, err error) {
	if dt == nil {
		return nil, nil, errors.New("nil DataTable")
	}

	var rowNum, colNum int
	var data *mat.Dense

	dt.AtomicDo(func(table *insyra.DataTable) {
		rowNum, colNum = table.Size()
		data = mat.NewDense(rowNum, colNum, nil)
		for i := 0; i < rowNum; i++ {
			row := table.GetRow(i)
			for j := 0; j < colNum; j++ {
				value, ok := row.Get(j).(float64)
				if ok {
					data.Set(i, j, value)
				}
			}
		}
	})

	// Standardize if requested
	if standardize {
		for j := 0; j < colNum; j++ {
			col := mat.Col(nil, j, data)
			mean, std := stat.MeanStdDev(col, nil)
			if std == 0 {
				std = 1
			}
			for i := 0; i < rowNum; i++ {
				data.Set(i, j, (data.At(i, j)-mean)/std)
			}
		}
	}

	// Compute correlation or covariance matrix
	var corrMatrix *mat.SymDense
	if standardize {
		corrMatrix = mat.NewSymDense(colNum, nil)
		stat.CorrelationMatrix(corrMatrix, data, nil)
	} else {
		corrMatrix = mat.NewSymDense(colNum, nil)
		stat.CovarianceMatrix(corrMatrix, data, nil)
	}

	// Eigenvalue decomposition
	var eig mat.EigenSym
	if !eig.Factorize(corrMatrix, true) {
		return nil, nil, errors.New("eigenvalue decomposition failed")
	}

	eigenvalues := eig.Values(nil)

	// Sort in descending order
	sortedEigenvalues := make([]float64, len(eigenvalues))
	copy(sortedEigenvalues, eigenvalues)
	for i := 0; i < len(sortedEigenvalues)-1; i++ {
		for j := i + 1; j < len(sortedEigenvalues); j++ {
			if sortedEigenvalues[i] < sortedEigenvalues[j] {
				sortedEigenvalues[i], sortedEigenvalues[j] = sortedEigenvalues[j], sortedEigenvalues[i]
			}
		}
	}

	// Compute cumulative proportions
	totalVariance := 0.0
	for _, ev := range sortedEigenvalues {
		if ev > 0 {
			totalVariance += ev
		}
	}

	cumulativeProp := make([]float64, len(sortedEigenvalues))
	cumSum := 0.0
	for i := range sortedEigenvalues {
		if sortedEigenvalues[i] > 0 {
			cumSum += sortedEigenvalues[i]
		}
		if totalVariance > 0 {
			cumulativeProp[i] = cumSum / totalVariance
		}
	}

	// Generate factor names for rows
	factorNames := make([]string, len(sortedEigenvalues))
	for i := 0; i < len(sortedEigenvalues); i++ {
		factorNames[i] = fmt.Sprintf("Factor_%d", i+1)
	}

	eigenDT = vectorToDataTableWithNames(sortedEigenvalues, "Eigenvalue", "Eigenvalue", factorNames)
	cumDT = vectorToDataTableWithNames(cumulativeProp, "Cumulative", "Cumulative", factorNames)

	return eigenDT, cumDT, nil
}

func reflectFactorsForPositiveLoadings(loadings *mat.Dense) *mat.Dense {
	if loadings == nil {
		return nil
	}

	p, m := loadings.Dims()
	reflected := mat.NewDense(p, m, nil)
	reflected.Copy(loadings)

	for j := 0; j < m; j++ {
		// Calculate column sum to determine reflection
		// R's psych package typically reflects to maximize positive sum
		colSum := 0.0
		for i := 0; i < p; i++ {
			colSum += loadings.At(i, j)
		}

		// If column sum is negative, reflect the entire factor
		if colSum < 0 {
			for i := 0; i < p; i++ {
				reflected.Set(i, j, -loadings.At(i, j))
			}
		}
	}

	return reflected
}

// reflectRotationMatrix updates the rotation matrix to account for factor reflections
func reflectRotationMatrix(rotMatrix, reflectedLoadings, originalLoadings *mat.Dense) *mat.Dense {
	if rotMatrix == nil || reflectedLoadings == nil || originalLoadings == nil {
		return rotMatrix
	}

	_, m := reflectedLoadings.Dims()
	reflectedRot := mat.NewDense(m, m, nil)
	reflectedRot.Copy(rotMatrix)

	for j := 0; j < m; j++ {
		// Check if this factor was reflected
		reflected := false
		for i := 0; i < reflectedLoadings.RawMatrix().Rows; i++ {
			if math.Abs(reflectedLoadings.At(i, j)-(-originalLoadings.At(i, j))) < 1e-10 {
				reflected = true
				break
			}
		}

		if reflected {
			for k := 0; k < m; k++ {
				reflectedRot.Set(k, j, -rotMatrix.At(k, j))
			}
		}
	}

	return reflectedRot
}

// reflectPhiMatrix updates the phi matrix to account for factor reflections
func reflectPhiMatrix(phi, reflectedLoadings, originalLoadings *mat.Dense) *mat.Dense {
	if phi == nil || reflectedLoadings == nil || originalLoadings == nil {
		return phi
	}

	m, _ := phi.Dims()
	reflectedPhi := mat.NewDense(m, m, nil)
	reflectedPhi.Copy(phi)

	reflectionSigns := make([]float64, m)
	for j := 0; j < m; j++ {
		reflectionSigns[j] = 1.0
		// Check if this factor was reflected
		for i := 0; i < reflectedLoadings.RawMatrix().Rows; i++ {
			if math.Abs(reflectedLoadings.At(i, j)-(-originalLoadings.At(i, j))) < 1e-10 {
				reflectionSigns[j] = -1.0
				break
			}
		}
	}

	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			reflectedPhi.Set(i, j, phi.At(i, j)*reflectionSigns[i]*reflectionSigns[j])
		}
	}

	return reflectedPhi
}

// sortFactorsByExplainedVariance sorts factors by their explained variance following R's psych::fa logic.
// For orthogonal rotation: uses sum of squared loadings
// For oblique rotation: uses diag(Phi %*% t(loadings) %*% loadings)
func sortFactorsByExplainedVariance(loadings, rotMatrix, phi *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil {
		return nil, rotMatrix, phi
	}

	p, m := loadings.Dims()
	if m <= 1 {
		return loadings, rotMatrix, phi
	}

	// Calculate explained variance for each factor
	variances := make([]float64, m)
	if phi == nil {
		// Orthogonal rotation: diag(t(loadings) %*% loadings) = sum of squared loadings per factor
		for j := 0; j < m; j++ {
			sum := 0.0
			for i := 0; i < p; i++ {
				val := loadings.At(i, j)
				sum += val * val
			}
			variances[j] = sum
		}
	} else {
		// Oblique rotation: diag(Phi %*% t(loadings) %*% loadings)
		var loadingsT mat.Dense
		loadingsT.CloneFrom(loadings.T())

		var phiLoadingsT mat.Dense
		phiLoadingsT.Mul(phi, &loadingsT)

		var eigenMat mat.Dense
		eigenMat.Mul(&phiLoadingsT, loadings)

		// Extract diagonal elements
		for j := 0; j < m; j++ {
			variances[j] = eigenMat.At(j, j)
		}
	}

	// Sort factors by explained variance in descending order
	type factorPair struct {
		index    int
		variance float64
	}
	pairs := make([]factorPair, m)
	for j := 0; j < m; j++ {
		pairs[j] = factorPair{index: j, variance: variances[j]}
	}

	// Sort in descending order of explained variance
	for i := 0; i < len(pairs)-1; i++ {
		for k := i + 1; k < len(pairs); k++ {
			if pairs[i].variance < pairs[k].variance {
				pairs[i], pairs[k] = pairs[k], pairs[i]
			}
		}
	}

	// Check if already sorted
	isSorted := true
	for j := 0; j < m; j++ {
		if pairs[j].index != j {
			isSorted = false
			break
		}
	}
	if isSorted {
		return loadings, rotMatrix, phi
	}

	// Reorder loadings
	sortedLoadings := mat.NewDense(p, m, nil)
	for j := 0; j < m; j++ {
		oldIdx := pairs[j].index
		for i := 0; i < p; i++ {
			sortedLoadings.Set(i, j, loadings.At(i, oldIdx))
		}
	}

	// Reorder rotation matrix
	var sortedRotMatrix *mat.Dense
	if rotMatrix != nil {
		sortedRotMatrix = mat.NewDense(m, m, nil)
		for j := 0; j < m; j++ {
			oldIdx := pairs[j].index
			for k := 0; k < m; k++ {
				sortedRotMatrix.Set(k, j, rotMatrix.At(k, oldIdx))
			}
		}
	}

	// Reorder phi matrix
	var sortedPhi *mat.Dense
	if phi != nil {
		sortedPhi = mat.NewDense(m, m, nil)
		for i := 0; i < m; i++ {
			oldIdxI := pairs[i].index
			for j := 0; j < m; j++ {
				oldIdxJ := pairs[j].index
				sortedPhi.Set(i, j, phi.At(oldIdxI, oldIdxJ))
			}
		}
	}

	return sortedLoadings, sortedRotMatrix, sortedPhi
}

// normalizeLoadingsToUnitLength normalizes each factor's loadings to have unit length (sum of squares = 1)
// This matches the convention used by R's psych package for PAF
func normalizeLoadingsToUnitLength(loadings *mat.Dense) *mat.Dense {
	if loadings == nil {
		return nil
	}

	p, m := loadings.Dims()
	normalized := mat.NewDense(p, m, nil)

	for j := 0; j < m; j++ {
		// Calculate the sum of squares for this factor
		sumSquares := 0.0
		for i := 0; i < p; i++ {
			val := loadings.At(i, j)
			sumSquares += val * val
		}

		// Normalize if sum of squares is positive
		if sumSquares > 0 {
			scale := 1.0 / math.Sqrt(sumSquares)
			for i := 0; i < p; i++ {
				normalized.Set(i, j, loadings.At(i, j)*scale)
			}
		} else {
			// If all loadings are zero, keep as is
			for i := 0; i < p; i++ {
				normalized.Set(i, j, loadings.At(i, j))
			}
		}
	}

	return normalized
}
