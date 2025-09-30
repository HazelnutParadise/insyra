package stats

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// -------------------------
// Factor Analysis Types and Constants
// -------------------------

// FactorExtractionMethod defines the method for extracting factors
type FactorExtractionMethod string

const (
	FactorExtractionPCA      FactorExtractionMethod = "pca"
	FactorExtractionPAF      FactorExtractionMethod = "paf"
	FactorExtractionML       FactorExtractionMethod = "ml"
	FactorExtractionBayesian FactorExtractionMethod = "bayesian"
)

// FactorRotationMethod defines the method for rotating factors
type FactorRotationMethod string

const (
	FactorRotationNone      FactorRotationMethod = "none"
	FactorRotationVarimax   FactorRotationMethod = "varimax"
	FactorRotationQuartimax FactorRotationMethod = "quartimax"
	FactorRotationEquamax   FactorRotationMethod = "equamax"
	FactorRotationPromax    FactorRotationMethod = "promax"
	FactorRotationOblimin   FactorRotationMethod = "oblimin"
)

// FactorScoreMethod defines the method for computing factor scores
type FactorScoreMethod string

const (
	FactorScoreRegression    FactorScoreMethod = "regression"
	FactorScoreBartlett      FactorScoreMethod = "bartlett"
	FactorScoreAndersonRubin FactorScoreMethod = "anderson-rubin"
)

// FactorCountMethod defines the method for determining number of factors
type FactorCountMethod string

const (
	FactorCountFixed            FactorCountMethod = "fixed"
	FactorCountKaiser           FactorCountMethod = "kaiser"
	FactorCountScree            FactorCountMethod = "scree"
	FactorCountParallelAnalysis FactorCountMethod = "parallel-analysis"
)

// -------------------------
// Options Structs
// -------------------------

// FactorCountSpec specifies how to determine the number of factors
type FactorCountSpec struct {
	Method               FactorCountMethod
	FixedK               int     // Optional: used when Method is CountFixed
	EigenThreshold       float64 // Optional: default 1.0 for CountKaiser
	MaxFactors           int     // Optional: 0 means no limit
	ParallelReplications int     // Optional: default 100 for CountParallelAnalysis
	ParallelPercentile   float64 // Optional: default 0.95 for CountParallelAnalysis
	EnableAutoScree      bool    // Optional: for CountScree
}

// FactorRotationOptions specifies rotation parameters
type FactorRotationOptions struct {
	Method       FactorRotationMethod
	Kappa        float64 // Optional: default p/2 for Equamax
	Delta        float64 // Optional: default 0 for Oblimin
	ForceOblique bool    // Optional
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
	Tol        float64 // Optional: default 1e-6
}

// -------------------------
// Result Structs
// -------------------------

// FactorAnalysisResult contains the output of factor analysis
type FactorAnalysisResult struct {
	Loadings             insyra.IDataTable // Loading matrix (variables × factors)
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

// FactorModel holds the factor analysis model
type FactorModel struct {
	Result FactorAnalysisResult

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

// DefaultFactorAnalysisOptions returns default options for factor analysis
func DefaultFactorAnalysisOptions() FactorAnalysisOptions {
	return FactorAnalysisOptions{
		Preprocess: FactorPreprocessOptions{
			Standardize: true,
			Missing:     "listwise",
		},
		Count: FactorCountSpec{
			Method:               FactorCountKaiser,
			EigenThreshold:       1.0,
			MaxFactors:           0, // 0 means no limit
			ParallelReplications: 100,
			ParallelPercentile:   0.95,
			EnableAutoScree:      false,
		},
		Extraction: FactorExtractionPCA,
		Rotation: FactorRotationOptions{
			Method:       FactorRotationVarimax,
			Kappa:        0, // Will be set to p/2 if needed
			Delta:        0,
			ForceOblique: false,
		},
		Scoring: FactorScoreRegression,
		MaxIter: 100,
		Tol:     1e-6,
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
	if opt.Tol <= 0 {
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
	loadings, converged, iterations, err := extractFactors(data, corrDense, sortedEigenvalues, sortedEigenvectors, numFactors, opt)
	if err != nil {
		insyra.LogWarning("stats", "FactorAnalysis", "factor extraction failed: %v", err)
		return nil
	}

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

	// Step 8: Compute communalities and uniquenesses
	communalities := make([]float64, colNum)
	uniquenesses := make([]float64, colNum)
	for i := 0; i < colNum; i++ {
		comm := 0.0
		for j := 0; j < numFactors; j++ {
			comm += rotatedLoadings.At(i, j) * rotatedLoadings.At(i, j)
		}
		communalities[i] = comm
		diag := corrMatrix.At(i, i)
		if diag == 0 {
			diag = 1.0
		}
		uniq := diag - comm
		if uniq < 0 {
			uniq = 0
		}
		uniquenesses[i] = uniq
		if uniquenesses[i] < 1e-9 {
			uniquenesses[i] = 1e-9
		}
	}

	// Step 9: Compute explained proportions
	totalVariance := 0.0
	for _, ev := range sortedEigenvalues {
		if ev > 0 {
			totalVariance += ev
		}
	}
	explainedProp := make([]float64, numFactors)
	cumulativeProp := make([]float64, numFactors)
	cumSum := 0.0
	for i := 0; i < numFactors; i++ {
		if i < len(sortedEigenvalues) {
			explainedProp[i] = sortedEigenvalues[i] / totalVariance
		} else {
			explainedProp[i] = 0
		}
		cumSum += explainedProp[i]
		cumulativeProp[i] = cumSum
	}

	// Step 10: Compute factor scores if data is available
	var scores *mat.Dense
	if rowNum > 0 {
		scores = computeFactorScores(data, rotatedLoadings, uniquenesses, opt.Scoring)
	}

	// Convert results to DataTables
	// Generate factor column names
	factorColNames := make([]string, numFactors)
	for i := 0; i < numFactors; i++ {
		factorColNames[i] = fmt.Sprintf("Factor_%d", i+1)
	}

	messages := []string{
		fmt.Sprintf("Extraction method: %s", opt.Extraction),
		fmt.Sprintf("Factor count method: %s (retained %d)", opt.Count.Method, numFactors),
		fmt.Sprintf("Rotation method: %s", opt.Rotation.Method),
		fmt.Sprintf("Scoring method: %s", opt.Scoring),
	}
	if opt.Count.Method == FactorCountParallelAnalysis {
		rep := opt.Count.ParallelReplications
		if rep <= 0 {
			rep = 100
		}
		pct := opt.Count.ParallelPercentile
		if pct <= 0 || pct >= 1 {
			pct = 0.95
		}
		messages = append(messages, fmt.Sprintf("Parallel analysis: %d replications at %.2f percentile", rep, pct))
	}
	if iterations > 0 {
		messages = append(messages, fmt.Sprintf("Extraction iterations: %d (tol %.2g)", iterations, opt.Tol))
	}
	if !converged && (opt.Extraction == FactorExtractionPAF || opt.Extraction == FactorExtractionML || opt.Extraction == FactorExtractionBayesian) {
		messages = append(messages, "Warning: extraction did not converge within limits")
	}
	if phi != nil {
		messages = append(messages, "Oblique rotation applied; factor correlation matrix provided")
	}
	messages = append(messages, "Factor analysis completed")

	result := FactorAnalysisResult{
		Loadings:             matrixToDataTableWithNames(rotatedLoadings, "Factor Loadings", factorColNames, colNames),
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
		Result:      result,
		scoreMethod: opt.Scoring,
		extraction:  opt.Extraction,
		rotation:    opt.Rotation.Method,
		means:       means,
		sds:         sds,
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

	case FactorCountScree:
		return applyFactorLimits(screeCount(eigenvalues, spec.EnableAutoScree), spec.MaxFactors, maxPossible)

	case FactorCountParallelAnalysis:
		count := parallelAnalysisCount(eigenvalues, spec, maxPossible, sampleSize)
		if count == 0 {
			count = countByThreshold(eigenvalues, 1.0)
		}
		return applyFactorLimits(count, spec.MaxFactors, maxPossible)

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

func screeCount(eigenvalues []float64, auto bool) int {
	if len(eigenvalues) == 0 {
		return 1
	}
	if len(eigenvalues) < 3 {
		return 1
	}
	if auto {
		bestIdx := 0
		bestScore := math.Inf(1)
		for i := 1; i < len(eigenvalues)-1; i++ {
			left := eigenvalues[i-1] - eigenvalues[i]
			right := eigenvalues[i] - eigenvalues[i+1]
			score := math.Abs(right - left)
			if score < bestScore {
				bestScore = score
				bestIdx = i
			}
		}
		return bestIdx + 1
	}

	maxDiff := 0.0
	elbowIdx := 0
	for i := 0; i < len(eigenvalues)-1; i++ {
		diff := eigenvalues[i] - eigenvalues[i+1]
		if diff > maxDiff {
			maxDiff = diff
			elbowIdx = i
		}
	}
	return elbowIdx + 1
}

func parallelAnalysisCount(eigenvalues []float64, spec FactorCountSpec, variables int, sampleSize int) int {
	if variables == 0 || sampleSize <= 1 {
		return 0
	}
	reps := spec.ParallelReplications
	if reps <= 0 {
		reps = 100
	}
	percentile := spec.ParallelPercentile
	if percentile <= 0 || percentile >= 1 {
		percentile = 0.95
	}

	randSrc := rand.New(rand.NewSource(42))
	simEigen := make([][]float64, reps)

	for r := 0; r < reps; r++ {
		simData := mat.NewDense(sampleSize, variables, nil)
		for i := 0; i < sampleSize; i++ {
			for j := 0; j < variables; j++ {
				simData.Set(i, j, randSrc.NormFloat64())
			}
		}

		corr := mat.NewSymDense(variables, nil)
		stat.CorrelationMatrix(corr, simData, nil)

		var eig mat.EigenSym
		if !eig.Factorize(corr, true) {
			continue
		}

		vals := eig.Values(nil)
		sort.Sort(sort.Reverse(sort.Float64Slice(vals)))
		simEigen[r] = vals
	}

	thresholds := make([]float64, variables)
	for j := 0; j < variables; j++ {
		samples := make([]float64, 0, reps)
		for r := 0; r < reps; r++ {
			if len(simEigen[r]) == 0 {
				continue
			}
			samples = append(samples, simEigen[r][j])
		}
		if len(samples) == 0 {
			thresholds[j] = 1.0
			continue
		}
		sort.Float64s(samples)
		idx := int(math.Ceil(percentile*float64(len(samples)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		thresholds[j] = samples[idx]
	}

	limit := min(len(eigenvalues), variables)
	count := 0
	for j := 0; j < limit; j++ {
		if eigenvalues[j] > thresholds[j] {
			count++
		}
	}
	return count
}

// extractFactors extracts factors using the specified method
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions) (*mat.Dense, bool, int, error) {
	switch opt.Extraction {
	case FactorExtractionPCA:
		return extractPCA(eigenvalues, eigenvectors, numFactors)

	case FactorExtractionPAF:
		return extractPAF(corrMatrix, numFactors, opt.MaxIter, opt.Tol)

	case FactorExtractionML:
		return extractML(corrMatrix, numFactors, opt.MaxIter, opt.Tol)

	case FactorExtractionBayesian:
		return extractBayesian(data, corrMatrix, numFactors, opt)

	default:
		// Default to PCA
		return extractPCA(eigenvalues, eigenvectors, numFactors)
	}
}

// extractPCA extracts factors using Principal Component Analysis
func extractPCA(eigenvalues []float64, eigenvectors *mat.Dense, numFactors int) (*mat.Dense, bool, int, error) {
	p := eigenvectors.RawMatrix().Rows
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

	return loadings, true, 0, nil
}

// extractPAF extracts factors using Principal Axis Factoring
func extractPAF(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64) (*mat.Dense, bool, int, error) {
	p := corrMatrix.RawMatrix().Rows

	// Initialize communalities with squared multiple correlations
	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		// Simple initialization: use correlation with other variables
		sum := 0.0
		for j := 0; j < p; j++ {
			if i != j {
				val := corrMatrix.At(i, j)
				sum += val * val
			}
		}
		communalities[i] = sum / float64(p-1)
		if communalities[i] > 1.0 {
			communalities[i] = 1.0
		}
	}

	var loadings *mat.Dense
	converged := false

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

		// Eigenvalue decomposition
		var eig mat.EigenSym
		symReduced := mat.NewSymDense(p, nil)
		for i := 0; i < p; i++ {
			for j := i; j < p; j++ {
				val := (reducedCorr.At(i, j) + reducedCorr.At(j, i)) / 2
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

		// Update communalities
		newCommunalities := make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < numFactors; j++ {
				sum += loadings.At(i, j) * loadings.At(i, j)
			}
			newCommunalities[i] = sum
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

		if maxChange < tol {
			converged = true
			return loadings, converged, iter + 1, nil
		}
	}

	return loadings, converged, maxIter, nil
}

// extractML extracts factors using Maximum Likelihood estimation
func extractML(corrMatrix *mat.Dense, numFactors int, maxIter int, tol float64) (*mat.Dense, bool, int, error) {
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

	initial, _, _, err := extractPAF(corrMatrix, numFactors, min(maxIter, 50), math.Max(tol, 1e-6))
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

	loadings := mat.Dense{}
	loadings.CloneFrom(initial)

	psMin := 1e-6
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
			sigma.Set(i, i, sigma.At(i, i)+psi[i])
		}

		var invSigma mat.Dense
		if err := invSigma.Inverse(&sigma); err != nil {
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
			m.Set(i, i, m.At(i, i)+1.0)
		}

		invSqrt, err := inverseSqrtDense(&m)
		if err != nil {
			return nil, false, iter, fmt.Errorf("ml: inverse sqrt failed: %w", err)
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

		if maxChange < tol {
			converged = true
			return &loadings, converged, iter + 1, nil
		}
	}

	return &loadings, converged, maxIter, nil
}

// extractBayesian extracts factors using a simple Bayesian shrinkage approach
func extractBayesian(data, corrMatrix *mat.Dense, numFactors int, opt FactorAnalysisOptions) (*mat.Dense, bool, int, error) {
	p, _ := corrMatrix.Dims()
	if numFactors <= 0 || numFactors > p {
		return nil, false, 0, fmt.Errorf("invalid number of factors: %d", numFactors)
	}

	rows := 0
	if data != nil {
		rows, _ = data.Dims()
	}

	alpha := 0.1
	if rows > 0 {
		alpha = float64(numFactors) / float64(rows+numFactors)
		if alpha < 0.05 {
			alpha = 0.05
		}
		if alpha > 0.3 {
			alpha = 0.3
		}
	}

	shrinked := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			val := (1 - alpha) * corrMatrix.At(i, j)
			if i == j {
				val += alpha
			}
			shrinked.Set(i, j, val)
		}
	}

	loadings, converged, iters, err := extractML(shrinked, numFactors, opt.MaxIter, opt.Tol)
	if err != nil {
		return extractML(corrMatrix, numFactors, opt.MaxIter, opt.Tol)
	}

	return loadings, converged, iters, nil
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
	p, _ := loadings.Dims()

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

	case FactorRotationEquamax:
		gamma := float64(p) / (2.0 * float64(m))
		if opt.Kappa != 0 {
			gamma = opt.Kappa
		}
		rotated, rotMatrix, err := rotateOrthomax(loadings, gamma, rotMaxIter, rotTol)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "equamax rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, nil

	case FactorRotationPromax:
		kappa := opt.Kappa
		if kappa == 0 {
			kappa = 4
		}
		rotated, rotMatrix, phi, err := rotatePromax(loadings, kappa, rotMaxIter, rotTol, opt.ForceOblique)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "promax rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, phi

	case FactorRotationOblimin:
		delta := opt.Delta
		rotated, rotMatrix, phi, err := rotateOblimin(loadings, delta, rotMaxIter, rotTol, opt.ForceOblique)
		if err != nil {
			insyra.LogWarning("stats", "rotateFactors", "oblimin rotation failed: %v", err)
			return loadings, nil, nil
		}
		return rotated, rotMatrix, phi

	default:
		return loadings, nil, nil
	}
}

func rotateOrthomax(loadings *mat.Dense, gamma float64, maxIter int, tol float64) (*mat.Dense, *mat.Dense, error) {
	p, m := loadings.Dims()
	rotation := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		rotation.Set(i, i, 1)
	}

	rotated := mat.NewDense(p, m, nil)
	rotated.Copy(loadings)

	prevObj := orthomaxObjective(rotated, gamma)

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
			rowScale := gamma * rowNorms[i] / float64(p)
			for j := 0; j < m; j++ {
				val := rotated.At(i, j)
				term.Set(i, j, 4*val*(val*val-rowScale))
			}
		}

		var grad mat.Dense
		grad.Mul(loadings.T(), term)

		var gradT mat.Dense
		gradT.CloneFrom(grad.T())

		var skew mat.Dense
		skew.Sub(&grad, &gradT)
		skew.Scale(0.5, &skew)

		norm := frobeniusNormDense(&skew)
		if norm < tol {
			return rotated, rotation, nil
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
			trialRotated.Mul(loadings, &q)

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

	return rotated, rotation, nil
}

func rotatePromax(loadings *mat.Dense, kappa float64, maxIter int, tol float64, forceOblique bool) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	orthoLoadings, orthoRot, err := rotateOrthomax(loadings, 1.0, maxIter, tol)
	if err != nil {
		return nil, nil, nil, err
	}

	p, m := orthoLoadings.Dims()
	target := mat.NewDense(p, m, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < m; j++ {
			val := orthoLoadings.At(i, j)
			sign := 1.0
			if val < 0 {
				sign = -1.0
			}
			target.Set(i, j, sign*math.Pow(math.Abs(val), kappa))
		}
	}

	var ft mat.Dense
	ft.CloneFrom(orthoLoadings.T())

	var ftf mat.Dense
	ftf.Mul(&ft, orthoLoadings)

	var ftfInv mat.Dense
	if err := ftfInv.Inverse(&ftf); err != nil {
		ftfRegularized := mat.Dense{}
		ftfRegularized.CloneFrom(&ftf)
		for i := 0; i < m; i++ {
			ftfRegularized.Set(i, i, ftfRegularized.At(i, i)+1e-6)
		}
		if err := ftfInv.Inverse(&ftfRegularized); err != nil {
			return nil, nil, nil, fmt.Errorf("promax: unable to invert matrix: %w", err)
		}
	}

	var ftTarget mat.Dense
	ftTarget.Mul(&ft, target)

	var trans mat.Dense
	trans.Mul(&ftfInv, &ftTarget)

	if !forceOblique {
		var svd mat.SVD
		if svd.Factorize(&trans, mat.SVDThin) {
			var u, vt mat.Dense
			svd.UTo(&u)
			svd.VTo(&vt)
			var ortho mat.Dense
			ortho.Mul(&u, &vt)
			var finalRot mat.Dense
			finalRot.Mul(orthoRot, &ortho)
			var rotated mat.Dense
			rotated.Mul(loadings, &finalRot)
			return &rotated, &finalRot, nil, nil
		}
	}

	var rotated mat.Dense
	rotated.Mul(orthoLoadings, &trans)

	var combined mat.Dense
	combined.Mul(orthoRot, &trans)

	var transInv mat.Dense
	if err := invertWithFallback(&transInv, &trans); err != nil {
		return &rotated, &combined, nil, fmt.Errorf("promax: transformation not invertible: %w", err)
	}

	var transInvT mat.Dense
	transInvT.CloneFrom(transInv.T())

	var phi mat.Dense
	phi.Mul(&transInv, &transInvT)

	return &rotated, &combined, &phi, nil
}

func rotateOblimin(loadings *mat.Dense, delta float64, maxIter int, tol float64, forceOblique bool) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
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

	if !forceOblique {
		var svd mat.SVD
		if svd.Factorize(trans, mat.SVDThin) {
			var u, vt mat.Dense
			svd.UTo(&u)
			svd.VTo(&vt)
			var ortho mat.Dense
			ortho.Mul(&u, &vt)
			var rotatedOrtho mat.Dense
			rotatedOrtho.Mul(loadings, &ortho)
			return &rotatedOrtho, &ortho, nil, nil
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

	return rotated, trans, &phi, nil
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
func computeFactorScores(data, loadings *mat.Dense, uniquenesses []float64, method FactorScoreMethod) *mat.Dense {
	n, p := data.Dims()
	_, m := loadings.Dims()

	scores := mat.NewDense(n, m, nil)

	switch method {
	case FactorScoreRegression:
		// Regression method: Scores = X * Loadings * (Loadings' * Loadings)^-1
		var loadingsTrans mat.Dense
		loadingsTrans.CloneFrom(loadings.T())

		var prod mat.Dense
		prod.Mul(&loadingsTrans, loadings)

		var inv mat.Dense
		if err := inv.Inverse(&prod); err != nil {
			// If inversion fails, use pseudo-inverse or fallback
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
			if uniquenesses[i] > 0 {
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
		if err := inv.Inverse(&temp2); err != nil {
			// Fallback to regression method
			return computeFactorScores(data, loadings, uniquenesses, FactorScoreRegression)
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
		if err := inv.Inverse(&prod); err != nil {
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
		return computeFactorScores(data, loadings, uniquenesses, FactorScoreRegression)
	}

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
	m.Result.Loadings.AtomicDo(func(table *insyra.DataTable) {
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
	if m.Result.Uniquenesses != nil {
		m.Result.Uniquenesses.AtomicDo(func(table *insyra.DataTable) {
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

	scores := computeFactorScores(data, loadings, uniquenesses, chosen)

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
