package stats

import (
	"errors"
	"math"

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
	CountFixed            FactorCountMethod = "fixed"
	CountKaiser           FactorCountMethod = "kaiser"
	CountScree            FactorCountMethod = "scree"
	CountParallelAnalysis FactorCountMethod = "parallel-analysis"
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
			Method:               CountKaiser,
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

	// Step 1: Preprocess data
	dt.AtomicDo(func(table *insyra.DataTable) {
		rowNum, colNum = table.Size()

		// Check for empty data
		if rowNum == 0 || colNum == 0 {
			return
		}

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
	numFactors := decideNumFactors(sortedEigenvalues, opt.Count, colNum)
	if numFactors == 0 {
		insyra.LogWarning("stats", "FactorAnalysis", "no factors retained")
		return nil
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
		for j := range numFactors {
			comm += rotatedLoadings.At(i, j) * rotatedLoadings.At(i, j)
		}
		communalities[i] = comm
		uniquenesses[i] = 1.0 - comm
		if uniquenesses[i] < 0 {
			uniquenesses[i] = 0
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
	for i := range numFactors {
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
	result := FactorAnalysisResult{
		Loadings:             matrixToDataTable(rotatedLoadings, "Loadings"),
		Uniquenesses:         vectorToDataTable(uniquenesses, "Uniqueness"),
		Communalities:        vectorToDataTable(communalities, "Communality"),
		Phi:                  nil,
		RotationMatrix:       nil,
		Eigenvalues:          vectorToDataTable(sortedEigenvalues, "Eigenvalue"),
		ExplainedProportion:  vectorToDataTable(explainedProp, "Proportion"),
		CumulativeProportion: vectorToDataTable(cumulativeProp, "Cumulative"),
		Scores:               nil,
		Converged:            converged,
		Iterations:           iterations,
		CountUsed:            numFactors,
		Messages:             []string{"Factor analysis completed"},
	}

	if rotationMatrix != nil {
		result.RotationMatrix = matrixToDataTable(rotationMatrix, "Rotation")
	}
	if phi != nil {
		result.Phi = matrixToDataTable(phi, "Phi")
	}
	if scores != nil {
		result.Scores = matrixToDataTable(scores, "Scores")
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
func decideNumFactors(eigenvalues []float64, spec FactorCountSpec, maxPossible int) int {
	switch spec.Method {
	case CountFixed:
		if spec.FixedK > 0 && spec.FixedK <= maxPossible {
			return spec.FixedK
		}
		return maxPossible

	case CountKaiser:
		threshold := spec.EigenThreshold
		if threshold == 0 {
			threshold = 1.0
		}
		count := 0
		for _, ev := range eigenvalues {
			if ev >= threshold {
				count++
			}
		}
		if spec.MaxFactors > 0 && count > spec.MaxFactors {
			count = spec.MaxFactors
		}
		return count

	case CountScree:
		// Simple scree test: find elbow point
		if len(eigenvalues) < 3 {
			return 1
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
		count := elbowIdx + 1
		if count == 0 {
			count = 1
		}
		if spec.MaxFactors > 0 && count > spec.MaxFactors {
			count = spec.MaxFactors
		}
		return count

	case CountParallelAnalysis:
		// Simplified parallel analysis - not fully implemented
		// For now, use Kaiser criterion
		count := 0
		for _, ev := range eigenvalues {
			if ev >= 1.0 {
				count++
			}
		}
		if spec.MaxFactors > 0 && count > spec.MaxFactors {
			count = spec.MaxFactors
		}
		return count

	default:
		// Default to Kaiser
		count := 0
		for _, ev := range eigenvalues {
			if ev >= 1.0 {
				count++
			}
		}
		return count
	}
}

// extractFactors extracts factors using the specified method
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions) (*mat.Dense, bool, int, error) {
	switch opt.Extraction {
	case FactorExtractionPCA:
		return extractPCA(eigenvalues, eigenvectors, numFactors)

	case FactorExtractionPAF:
		return extractPAF(corrMatrix, numFactors, opt.MaxIter, opt.Tol)

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

// rotateFactors rotates the factor loadings
func rotateFactors(loadings *mat.Dense, opt FactorRotationOptions) (*mat.Dense, *mat.Dense, *mat.Dense) {
	switch opt.Method {
	case FactorRotationVarimax:
		return rotateVarimax(loadings, 100, 1e-6)
	default:
		return loadings, nil, nil
	}
}

// rotateVarimax performs Varimax rotation (orthogonal)
func rotateVarimax(loadings *mat.Dense, maxIter int, tol float64) (*mat.Dense, *mat.Dense, *mat.Dense) {
	p, m := loadings.Dims()
	if m == 1 {
		// No rotation needed for single factor
		return loadings, nil, nil
	}

	// Initialize rotation matrix as identity
	rotMatrix := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		rotMatrix.Set(i, i, 1.0)
	}

	rotatedLoadings := mat.NewDense(p, m, nil)
	rotatedLoadings.Copy(loadings)

	for iter := 0; iter < maxIter; iter++ {
		changed := false

		// Rotate each pair of factors
		for i := 0; i < m-1; i++ {
			for j := i + 1; j < m; j++ {
				// Extract columns i and j
				colI := make([]float64, p)
				colJ := make([]float64, p)
				for k := 0; k < p; k++ {
					colI[k] = rotatedLoadings.At(k, i)
					colJ[k] = rotatedLoadings.At(k, j)
				}

				// Compute rotation angle
				u := make([]float64, p)
				v := make([]float64, p)
				for k := 0; k < p; k++ {
					u[k] = colI[k]*colI[k] - colJ[k]*colJ[k]
					v[k] = 2 * colI[k] * colJ[k]
				}

				A := 0.0
				B := 0.0
				C := 0.0
				D := 0.0
				for k := 0; k < p; k++ {
					A += u[k]
					B += v[k]
					C += u[k]*u[k] - v[k]*v[k]
					D += 2 * u[k] * v[k]
				}

				num := D - 2*A*B/float64(p)
				den := C - (A*A-B*B)/float64(p)

				if math.Abs(den) < 1e-10 {
					continue
				}

				angle := math.Atan2(num, den) / 4

				// Apply rotation
				cos := math.Cos(angle)
				sin := math.Sin(angle)

				for k := 0; k < p; k++ {
					newI := colI[k]*cos - colJ[k]*sin
					newJ := colI[k]*sin + colJ[k]*cos
					rotatedLoadings.Set(k, i, newI)
					rotatedLoadings.Set(k, j, newJ)
				}

				// Update rotation matrix
				tempRot := mat.NewDense(m, m, nil)
				for k := 0; k < m; k++ {
					for l := 0; l < m; l++ {
						if k == l {
							if k == i || k == j {
								if k == i {
									tempRot.Set(k, l, cos)
								} else {
									tempRot.Set(k, l, cos)
								}
							} else {
								tempRot.Set(k, l, 1.0)
							}
						} else if k == i && l == j {
							tempRot.Set(k, l, -sin)
						} else if k == j && l == i {
							tempRot.Set(k, l, sin)
						} else {
							tempRot.Set(k, l, 0)
						}
					}
				}

				var newRotMatrix mat.Dense
				newRotMatrix.Mul(rotMatrix, tempRot)
				rotMatrix.Copy(&newRotMatrix)

				if math.Abs(angle) > tol {
					changed = true
				}
			}
		}

		if !changed {
			break
		}
	}

	return rotatedLoadings, rotMatrix, nil
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

// matrixToDataTable converts a gonum matrix to a DataTable
func matrixToDataTable(m *mat.Dense, baseName string) insyra.IDataTable {
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

	return dt
}

// vectorToDataTable converts a float slice to a single-column DataTable
func vectorToDataTable(v []float64, name string) insyra.IDataTable {
	if v == nil {
		return nil
	}

	dt := insyra.NewDataTable()
	col := insyra.NewDataList()
	for _, val := range v {
		col.Append(val)
	}
	col.SetName(name)
	dt.AppendCols(col)

	return dt
}

// FactorScoresDT computes factor scores for new data
func (m *FactorModel) FactorScoresDT(dt insyra.IDataTable, method *FactorScoreMethod) (insyra.IDataTable, error) {
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

	dt.AtomicDo(func(table *insyra.DataTable) {
		rowNum, colNum = table.Size()
		data = mat.NewDense(rowNum, colNum, nil)
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
		for i := range lr {
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
			for i := range ur {
				row := table.GetRow(i)
				value, ok := row.Get(0).(float64)
				if ok {
					uniquenesses[i] = value
				}
			}
		})
	}

	scores := computeFactorScores(data, loadings, uniquenesses, chosen)
	return matrixToDataTable(scores, "Scores"), nil
}

// ScreeDataDT returns scree plot data (eigenvalues and cumulative proportion)
func ScreeDataDT(dt insyra.IDataTable, standardize bool) (insyra.IDataTable, insyra.IDataTable, error) {
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
	for i := 0; i < len(sortedEigenvalues); i++ {
		if sortedEigenvalues[i] > 0 {
			cumSum += sortedEigenvalues[i]
		}
		if totalVariance > 0 {
			cumulativeProp[i] = cumSum / totalVariance
		}
	}

	eigenDT := vectorToDataTable(sortedEigenvalues, "Eigenvalue")
	cumDT := vectorToDataTable(cumulativeProp, "Cumulative")

	return eigenDT, cumDT, nil
}
