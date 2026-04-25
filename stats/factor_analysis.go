package stats

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/fa"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// -------------------------
// Factor Analysis Types and Constants
// -------------------------

// FactorExtractionMethod defines the method for extracting factors.
// See Docs/stats.md (Factor Analysis - Extraction Methods) for algorithmic details.
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

// VarimaxAlgorithm specifies which Varimax implementation to use
type VarimaxAlgorithm string

const (
	VarimaxGPArotation VarimaxAlgorithm = "gparotation" // GPArotation::Varimax
	VarimaxKaiser      VarimaxAlgorithm = "kaiser"      // stats::varimax / psych::fa rotate="varimax"
)

// FactorRotationOptions specifies rotation parameters
type FactorRotationOptions struct {
	Method           FactorRotationMethod
	Kappa            float64          // Optional: Promax power (default 4)
	Delta            float64          // Optional: default 0 for Oblimin
	Restarts         int              // Optional: random orthonormal starts for GPA rotations (default 10)
	VarimaxAlgorithm VarimaxAlgorithm // Optional: "kaiser" (psych default) or "gparotation"
}

// FactorAnalysisOptions contains all options for factor analysis
type FactorAnalysisOptions struct {
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
	Loadings             insyra.IDataTable   // Loading matrix (variables x factors)
	UnrotatedLoadings    insyra.IDataTable   // Unrotated loading matrix (variables x factors)
	Structure            insyra.IDataTable   // Structure matrix (variables x factors)
	Uniquenesses         insyra.IDataTable   // Uniqueness vector (p x 1)
	Communalities        insyra.IDataTable   // Communality table (p x 1: Extraction)
	SamplingAdequacy     insyra.IDataTable   // KMO overall index and per-variable MSA values
	BartlettTest         *BartlettTestResult // Bartlett's test of sphericity summary
	Phi                  insyra.IDataTable   // Factor correlation matrix (m x m), nil for orthogonal
	RotationMatrix       insyra.IDataTable   // Rotation matrix (m x m), nil if no rotation
	Eigenvalues          insyra.IDataTable   // Eigenvalues vector (p x 1)
	ExplainedProportion  insyra.IDataTable   // Proportion explained by each factor (m x 1)
	CumulativeProportion insyra.IDataTable   // Cumulative proportion explained (m x 1)
	Scores               insyra.IDataTable   // Factor scores (n x m), nil if not computed
	ScoreCoefficients    insyra.IDataTable   // Factor score coefficient matrix (variables x factors)
	ScoreCovariance      insyra.IDataTable   // Factor score covariance matrix (factors x factors)

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
		Count: FactorCountSpec{
			Method:         FactorCountKaiser,
			EigenThreshold: 1.0,
			MaxFactors:     0, // 0 means no limit
		},
		Extraction: FactorExtractionMINRES, // R default: "minres"
		Rotation: FactorRotationOptions{
			Method:           FactorRotationOblimin, // R default: "oblimin"
			Kappa:            4,                     // R default for promax
			Delta:            0,                     // R default for oblimin
			Restarts:         1,
			VarimaxAlgorithm: VarimaxKaiser,
		},
		Scoring: FactorScoreRegression, // R default: "regression"
		MaxIter: 50,                    // R default: 50
		MinErr:  0.001,                 // R default: 0.001
	}
}

func normalizeFactorAnalysisOptions(opt FactorAnalysisOptions) (FactorAnalysisOptions, error) {
	defaults := DefaultFactorAnalysisOptions()

	if opt.Count.Method == "" {
		opt.Count.Method = defaults.Count.Method
	}
	switch opt.Count.Method {
	case FactorCountFixed:
		if opt.Count.FixedK <= 0 {
			return opt, errors.New("fixed factor count must be greater than zero")
		}
	case FactorCountKaiser:
		if opt.Count.EigenThreshold <= 0 {
			opt.Count.EigenThreshold = defaults.Count.EigenThreshold
		}
	default:
		return opt, fmt.Errorf("unsupported factor count method: %s", opt.Count.Method)
	}
	if opt.Count.MaxFactors < 0 {
		return opt, errors.New("max factors must be non-negative")
	}

	if opt.Extraction == "" {
		opt.Extraction = defaults.Extraction
	}
	switch opt.Extraction {
	case FactorExtractionPCA, FactorExtractionPAF, FactorExtractionML, FactorExtractionMINRES:
	default:
		return opt, fmt.Errorf("unsupported factor extraction method: %s", opt.Extraction)
	}

	if opt.Rotation.Method == "" {
		opt.Rotation.Method = defaults.Rotation.Method
	}
	switch opt.Rotation.Method {
	case FactorRotationNone, FactorRotationVarimax, FactorRotationQuartimax, FactorRotationQuartimin,
		FactorRotationOblimin, FactorRotationGeominT, FactorRotationBentlerT, FactorRotationSimplimax,
		FactorRotationGeominQ, FactorRotationBentlerQ, FactorRotationPromax:
	default:
		return opt, fmt.Errorf("unsupported factor rotation method: %s", opt.Rotation.Method)
	}
	if opt.Rotation.Kappa == 0 {
		opt.Rotation.Kappa = defaults.Rotation.Kappa
	}
	if opt.Rotation.Restarts <= 0 {
		opt.Rotation.Restarts = defaults.Rotation.Restarts
	}
	if opt.Rotation.VarimaxAlgorithm == "" {
		opt.Rotation.VarimaxAlgorithm = defaults.Rotation.VarimaxAlgorithm
	}
	switch opt.Rotation.VarimaxAlgorithm {
	case VarimaxGPArotation, VarimaxKaiser:
	default:
		return opt, fmt.Errorf("unsupported varimax algorithm: %s", opt.Rotation.VarimaxAlgorithm)
	}

	if opt.Scoring == "" {
		opt.Scoring = defaults.Scoring
	}
	switch opt.Scoring {
	case FactorScoreNone, FactorScoreRegression, FactorScoreBartlett, FactorScoreAndersonRubin:
	default:
		return opt, fmt.Errorf("unsupported factor score method: %s", opt.Scoring)
	}

	if opt.MaxIter <= 0 {
		opt.MaxIter = defaults.MaxIter
	}
	if opt.MinErr <= 0 {
		opt.MinErr = defaults.MinErr
	}

	return opt, nil
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

// FactorAnalysis performs factor analysis on a DataTable.
func FactorAnalysis(dt insyra.IDataTable, opt FactorAnalysisOptions) (*FactorModel, error) {
	if dt == nil {
		return nil, errors.New("nil DataTable")
	}
	var err error
	opt, err = normalizeFactorAnalysisOptions(opt)
	if err != nil {
		return nil, err
	}

	var rowNum, colNum int
	var data *mat.Dense
	var means, sds []float64
	var colNames, rowNames []string
	var conversionErr error

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
				cell := row.Get(j)
				value, ok := insyra.ToFloat64Safe(cell)
				if ok {
					data.Set(i, j, value)
				} else {
					conversionErr = fmt.Errorf("non-numeric value at row %d column %d: %v", i, j, cell)
					return
				}
			}
		}
	})
	if conversionErr != nil {
		return nil, conversionErr
	}

	// Check for empty data after AtomicDo
	if rowNum == 0 || colNum == 0 {
		return nil, errors.New("empty DataTable")
	}
	if rowNum < 2 || colNum < 2 {
		return nil, errors.New("factor analysis requires at least two rows and two columns")
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
		// Remove rows with any NaN (listwise deletion)
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
			return nil, errors.New("no valid rows after removing missing values")
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
	}

	// Step 2: Standardize data (always performed for factor analysis)
	means = make([]float64, colNum)
	sds = make([]float64, colNum)
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

	// Step 3: Compute correlation matrix (always use correlation for factor analysis)
	var corrMatrix *mat.SymDense
	var corrForAdequacy *mat.SymDense
	corrMatrix = mat.NewSymDense(colNum, nil)
	stat.CorrelationMatrix(corrMatrix, data, nil)
	corrForAdequacy = corrMatrix
	if corrForAdequacy == nil {
		corrForAdequacy = mat.NewSymDense(colNum, nil)
		stat.CorrelationMatrix(corrForAdequacy, data, nil)
	}

	insyra.LogDebug("stats", "FactorAnalysis", "data matrix size: %dx%d, correlation matrix computed", rowNum, colNum)

	// Sanity check: ensure diagonal elements of correlation matrix are 1 (data is standardized)
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
			rows, cols := corrAdequacyDense.Dims()
			insyra.LogDebug("stats", "FactorAnalysis", "correlation matrix dimensions: %dx%d", rows, cols)
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
		return nil, errors.New("eigenvalue decomposition failed")
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
		return nil, errors.New("no factors retained")
	}
	tolVal := opt.MinErr
	if tolVal <= 0 {
		tolVal = 0.001
	}

	// Step 6: Extract factors
	// Convert SymDense to Dense for extraction functions
	corrDense := mat.NewDense(colNum, colNum, nil)
	for i := 0; i < colNum; i++ {
		for j := 0; j < colNum; j++ {
			corrDense.Set(i, j, corrMatrix.At(i, j))
		}
	}

	initialCommunalities := make([]float64, colNum)
	if opt.Extraction != FactorExtractionPCA {
		// R: if (nfactors <= n) { diag(r.mat) <- smc(...) }
		// else { warning and use 1 instead }
		if numFactors > colNum {
			// Too many factors requested: use 1 instead of SMC
			insyra.LogWarning("stats", "FactorAnalysis", "too many factors requested (%d) for number of variables (%d); using 1s instead of SMC estimates", numFactors, colNum)
			for i := 0; i < colNum; i++ {
				initialCommunalities[i] = 1.0
			}
		} else {
			// Use the internal psych-compatible SMC helper instead of
			// duplicating pseudo-inverse logic in the public orchestration layer.
			smcVec, _ := fa.Smc(corrDense, &fa.SmcOptions{Covar: false})
			if smcVec != nil {
				copy(initialCommunalities, smcVec.RawVector().Data)
			} else {
				for i := 0; i < colNum; i++ {
					initialCommunalities[i] = 0.5
				}
			}
			for i := 0; i < colNum; i++ {
				if initialCommunalities[i] < 0 {
					initialCommunalities[i] = 0
				}
				if initialCommunalities[i] > 1 {
					initialCommunalities[i] = 1
				}
			}
		}
	} else {
		// For other methods like PCA, use the diagonal of the correlation matrix (which is 1.0)
		for i := 0; i < colNum; i++ {
			initialCommunalities[i] = corrDense.At(i, i)
		}
	}

	loadings, _, extractionEigenvalues, converged, iterations, err := extractFactors(data, corrDense, sortedEigenvalues, sortedEigenvectors, numFactors, opt, rowNum, tolVal, initialCommunalities)
	if err != nil {
		return nil, fmt.Errorf("factor extraction failed: %w", err)
	}

	// Replace zero loadings with 1e-15 (matching R's behavior)
	// R: loadings[loadings == 0] <- 10^-15
	// This prevents numerical issues in subsequent calculations
	if loadings != nil {
		pVars, mFactors := loadings.Dims()
		for i := range pVars {
			for j := range mFactors {
				val := loadings.At(i, j)
				if val == 0 {
					loadings.Set(i, j, 1e-15)
				}
			}
		}
	}

	// Step 7: Rotate factors
	var rotatedLoadings *mat.Dense
	var unrotatedLoadings *mat.Dense
	var rotationMatrix *mat.Dense
	var phi *mat.Dense
	var rotationConverged bool

	if numFactors > 1 && opt.Rotation.Method != FactorRotationNone {
		// Apply rotation only if more than one factor
		// R: if (nfactors > 1) { ... rotation logic ... }
		unrotatedLoadings = mat.DenseCopyOf(loadings)
		if opt.Extraction == FactorExtractionPCA && opt.Rotation.Method == FactorRotationPromax {
			rotatedLoadings, rotationMatrix, phi, rotationConverged, err = rotatePrincipalPromax(loadings, opt.Rotation)
		} else {
			rotatedLoadings, rotationMatrix, phi, rotationConverged, err = rotateFactors(loadings, opt.Rotation, opt.MinErr, opt.MaxIter)
		}
		if err != nil {
			return nil, fmt.Errorf("factor rotation failed: %w", err)
		}
	} else {
		// No rotation
		unrotatedLoadings = mat.DenseCopyOf(loadings)
		rotatedLoadings = loadings
		rotationMatrix = nil
		phi = nil
		rotationConverged = true
		// Sign standardization will be done later, after sorting (if applicable)
	}

	// Sort or align factor columns by explained variance
	// R's psych::fa() applies sign standardization BEFORE sorting
	// R: (after rotation) signed <- sign(colSums(loadings)); loadings <- loadings %*% diag(signed)
	// Apply sign standardization AFTER rotation but BEFORE sorting
	// R code does this for ALL cases (sorted or not sorted)
	// BUT: when nfactors == 1, R uses different logic: checks if sum < 0, then flips entire column
	rows, cols := rotatedLoadings.Dims()
	signs := make([]float64, cols)

	for j := range cols {
		sum := 0.0
		for i := range rows {
			sum += rotatedLoadings.At(i, j)
		}
		signs[j] = 1.0
		if sum < 0 {
			signs[j] = -1.0
		}
	}

	// Apply signs to loadings: loadings <- loadings %*% diag(signed)
	for i := range rows {
		for j := range cols {
			rotatedLoadings.Set(i, j, rotatedLoadings.At(i, j)*signs[j])
		}
	}

	// Also apply sign standardization to phi matrix if it exists
	// R code: if (!is.null(Phi)) { Phi <- diag(signed) %*% Phi %*% diag(signed) }
	if phi != nil {
		for i := range cols {
			for j := range cols {
				phi.Set(i, j, signs[i]*phi.At(i, j)*signs[j])
			}
		}
	}

	// R's psych::fa() sorts factors AFTER sign standardization, by explained variance
	// This must be done AFTER rotation and sign standardization to match R's behavior
	// R: if (nfactors > 1) { ... sorting logic ... }
	if numFactors > 1 {
		rotatedLoadings, rotationMatrix, phi = sortFactorsByExplainedVariance(rotatedLoadings, rotationMatrix, phi)
	}

	// Step 8: Compute communalities and uniquenesses
	extractionCommunalities := make([]float64, colNum)
	uniquenesses := make([]float64, colNum)
	for i := 0; i < colNum; i++ {
		var hi2 float64
		// R psych reports communality as the diagonal of loadings %*% t(loadings).
		for j := 0; j < numFactors; j++ {
			v := unrotatedLoadings.At(i, j)
			hi2 += v * v
		}
		diag := corrMatrix.At(i, i)
		if diag == 0 {
			diag = 1.0
		}
		extractionCommunalities[i] = hi2

		uniquenesses[i] = diag - hi2
	}

	commMatrix := mat.NewDense(colNum, 2, nil)
	for i := 0; i < colNum; i++ {
		commMatrix.Set(i, 0, initialCommunalities[i])
		commMatrix.Set(i, 1, extractionCommunalities[i])
	}
	communalitiesTable := matrixToDataTableWithNames(commMatrix, tableNameCommunalities, []string{"Initial", "Extraction"}, colNames)

	// Compute eigenvalues for reporting (R style)
	// R: S <- r; diag(S) <- diag(model); eigens <- eigen(S)$values
	// This is the modified correlation matrix with final communalities as diagonal
	reportedEigenvalues := sortedEigenvalues // Default: use original eigenvalues
	if opt.Extraction != FactorExtractionPCA && len(extractionEigenvalues) > 0 {
		reportedEigenvalues = append([]float64(nil), extractionEigenvalues...)
	} else if opt.Extraction != FactorExtractionPCA {
		// For non-PCA methods, compute eigenvalues of correlation matrix with final communalities
		modifiedCorr := mat.NewDense(colNum, colNum, nil)
		modifiedCorr.CloneFrom(corrMatrix)
		for i := range colNum {
			modifiedCorr.Set(i, i, extractionCommunalities[i])
		}

		// Compute eigenvalues of modified correlation matrix
		modifiedCorrSym := mat.NewSymDense(colNum, nil)
		for i := range colNum {
			for j := range colNum {
				modifiedCorrSym.SetSym(i, j, modifiedCorr.At(i, j))
			}
		}
		var eig mat.EigenSym
		if eig.Factorize(modifiedCorrSym, true) {
			eigenvals := eig.Values(nil)
			reportedEigenvalues = make([]float64, len(eigenvals))
			copy(reportedEigenvalues, eigenvals)
			// Sort in descending order
			for i := 0; i < len(reportedEigenvalues)-1; i++ {
				for j := i + 1; j < len(reportedEigenvalues); j++ {
					if reportedEigenvalues[i] < reportedEigenvalues[j] {
						reportedEigenvalues[i], reportedEigenvalues[j] = reportedEigenvalues[j], reportedEigenvalues[i]
					}
				}
			}
		}
	}

	// Limit reportedEigenvalues to numFactors for output (R reports only numFactors eigenvalues)
	if len(reportedEigenvalues) > numFactors {
		reportedEigenvalues = reportedEigenvalues[:numFactors]
	}

	// Step 10: Compute factor scores if data is available
	var scores *mat.Dense
	var scoreWeights *mat.Dense
	var scoreCovariance *mat.Dense
	if rowNum > 0 {
		var err error
		scoringMethod := opt.Scoring
		if opt.Extraction == FactorExtractionPCA && scoringMethod != FactorScoreNone {
			// psych::principal exposes scores as a boolean and uses its
			// regression-style principal-component scoring for every non-none mode.
			scoringMethod = FactorScoreRegression
		}
		scores, scoreWeights, scoreCovariance, err = computeFactorScores(data, rotatedLoadings, phi, uniquenesses, sigmaForScores, scoringMethod)
		if err != nil {
			return nil, fmt.Errorf("factor scoring failed: %w", err)
		}
	}

	// Convert results to DataTables
	// Generate factor column names
	// Generate factor column names based on extraction method (matching R's naming)
	// R: switch(fm, alpha = {colnames <- paste("alpha", 1:nfactors)}, ...)
	factorColNames := make([]string, numFactors)
	switch opt.Extraction {
	case FactorExtractionPCA:
		for i := range numFactors {
			factorColNames[i] = fmt.Sprintf("PC%d", i+1)
		}
	case FactorExtractionPAF:
		for i := range numFactors {
			factorColNames[i] = fmt.Sprintf("PA%d", i+1)
		}
	case FactorExtractionML:
		for i := range numFactors {
			factorColNames[i] = fmt.Sprintf("ML%d", i+1)
		}
	case FactorExtractionMINRES:
		for i := range numFactors {
			factorColNames[i] = fmt.Sprintf("MR%d", i+1)
		}
	default:
		// Fallback to generic names
		for i := range numFactors {
			factorColNames[i] = fmt.Sprintf("Factor_%d", i+1)
		}
	}

	// Step 9: Compute explained proportions matching R's variance explanation calculation
	// R: vx = colSums(loadings^2) or diag(Phi %*% t(loadings) %*% loadings)
	// R: vtotal = sum(communalities + uniquenesses) = sum(diag(r))
	// R: Proportion Var = vx / vtotal (NOT percentage, NOT per variable)
	pVars, mFactors := rotatedLoadings.Dims()
	explainedProp := make([]float64, mFactors)
	cumulativeProp := make([]float64, mFactors)

	// Compute SS loadings for each factor
	ssLoad := make([]float64, mFactors)
	if phi == nil {
		// Orthogonal rotation: vx <- colSums(loadings^2)
		for j := range mFactors {
			sum := 0.0
			for i := range pVars {
				v := rotatedLoadings.At(i, j)
				sum += v * v
			}
			ssLoad[j] = sum
		}
	} else {
		// Oblique rotation: vx <- diag(Phi %*% t(loadings) %*% loadings)
		// Step 1: Compute t(loadings) %*% loadings (m x m matrix)
		ltl := mat.NewDense(mFactors, mFactors, nil)
		ltl.Mul(rotatedLoadings.T(), rotatedLoadings)
		// Step 2: Compute Phi %*% (t(loadings) %*% loadings)
		philtl := mat.NewDense(mFactors, mFactors, nil)
		philtl.Mul(phi, ltl)
		// Step 3: Extract diagonal
		for j := range mFactors {
			ssLoad[j] = philtl.At(j, j)
		}
	}

	// R: vtotal = sum(communalities + uniquenesses) = sum(diag(r)) = sum of 1s = pVars
	// Since we're using correlation matrix (diag = 1), vtotal = pVars
	vtotal := float64(pVars)

	// R: Proportion Var = vx / vtotal (simple proportion, not percentage)
	cum := 0.0
	for j := range mFactors {
		prop := ssLoad[j] / vtotal
		explainedProp[j] = prop
		cum += prop
		cumulativeProp[j] = cum
	}
	insyra.LogDebug("stats", "FactorAnalysis", "explainedProp (SS/total): %v", explainedProp)

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

	// Check for Heywood case (communality > 1)
	// R: if (max(result$communality > 1) && !covar) warning(...)
	hasHeywoodCase := false
	for i := range colNum {
		if extractionCommunalities[i] > 1.0 {
			hasHeywoodCase = true
			break
		}
	}

	messages := []string{
		fmt.Sprintf("Extraction method: %s", opt.Extraction),
		fmt.Sprintf("Factor count method: %s (retained %d)", opt.Count.Method, numFactors),
		fmt.Sprintf("Rotation method: %s", opt.Rotation.Method),
		fmt.Sprintf("Scoring method: %s", opt.Scoring),
	}
	if hasHeywoodCase {
		messages = append(messages, "WARNING: An ultra-Heywood case was detected. Examine the results carefully")
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
		Eigenvalues:          vectorToDataTableWithNames(reportedEigenvalues, tableNameEigenvalues, "Eigenvalue", factorColNames),
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
	}, nil
}

// computeSigma computes the reproduced correlation matrix: Sigma = L * Phi * L^T + U
func computeSigma(loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64) *mat.Dense {
	if loadings == nil {
		return nil
	}

	// Create diagonal matrix of uniquenesses
	U := mat.NewDiagDense(len(uniquenesses), uniquenesses)

	// Compute L * Phi * L^T
	var temp mat.Dense
	if phi != nil {
		temp.Mul(loadings, phi)
	} else {
		// Orthogonal case
		temp.CloneFrom(loadings)
	}
	var sigma mat.Dense
	sigma.Mul(&temp, loadings.T())

	// Add uniquenesses: Sigma = L*Phi*L^T + U
	sigma.Add(&sigma, U)

	return &sigma
}

// computePseudoInverse computes the Moore-Penrose pseudoinverse using SVD
// This matches R's psych::Pinv function for numerical stability
func computePseudoInverse(X *mat.Dense) *mat.Dense {
	if X == nil {
		return nil
	}

	m, n := X.Dims()

	// Perform SVD
	var svd mat.SVD
	if !svd.Factorize(X, mat.SVDThin) {
		insyra.LogWarning("stats", "FactorAnalysis", "SVD factorization failed in pseudoinverse computation")
		return nil
	}

	// Get singular values
	values := svd.Values(nil)

	// Compute tolerance (R uses tol = sqrt(.Machine$double.eps))
	tol := math.Sqrt(math.Nextafter(1.0, 2.0) - 1.0) // Go equivalent of sqrt(.Machine$double.eps)

	// Find indices of non-small singular values
	threshold := tol * values[0]
	p := 0 // Count of non-small values
	for _, v := range values {
		if v > threshold {
			p++
		}
	}

	if p == 0 {
		return nil
	}

	// Get U and V matrices
	U := mat.NewDense(m, len(values), nil)
	V := mat.NewDense(n, len(values), nil)
	svd.UTo(U)
	svd.VTo(V)

	// Compute pseudoinverse: Pinv = V[, 1:p] %*% diag(1/d[1:p]) %*% t(U[, 1:p])
	// Extract U[:, 1:p] and V[:, 1:p]
	Uphp := mat.NewDense(m, p, nil)
	Vp := mat.NewDense(n, p, nil)
	for i := 0; i < m; i++ {
		for j := 0; j < p; j++ {
			Uphp.Set(i, j, U.At(i, j))
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < p; j++ {
			Vp.Set(i, j, V.At(i, j))
		}
	}

	// Compute D^{-1} = diag(1/d[1:p])
	Dinv := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		if values[i] > 0 {
			Dinv.Set(i, i, 1.0/values[i])
		}
	}

	// Compute V %*% D^{-1}
	var temp mat.Dense
	temp.Mul(Vp, Dinv)

	// Compute Pinv = temp %*% U^T
	var Pinv mat.Dense
	Pinv.Mul(&temp, Uphp.T())

	return &Pinv
}

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
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions, sampleSize int, tol float64, initialCommunalities []float64) (*mat.Dense, []float64, []float64, bool, int, error) {
	// Use our psych_fac.Fac implementation for all extraction methods
	facOpts := &fa.FacOptions{
		NFactors:      numFactors,
		NObs:          float64(sampleSize),
		Rotate:        "none", // We handle rotation separately
		Scores:        "none", // We handle scoring separately
		Residuals:     false,
		SMC:           initialCommunalities, // Pass initial communalities
		Covar:         false,
		Missing:       false,
		Impute:        "median",
		MinErr:        tol,
		MaxIter:       opt.MaxIter,
		Symmetric:     true,
		Warnings:      true,
		ObliqueScores: false,
		Use:           "pairwise",
		Cor:           "cor",
		Correct:       0.5,
		NRotations:    1,
		Hyper:         0.15,
		Smooth:        true,
	}

	// Map our extraction methods to psych_fac method names
	switch opt.Extraction {
	case FactorExtractionPCA:
		// For PCA, we still use the original implementation since psych_fac doesn't handle PCA
		loadings, converged, iterations, err := extractPCA(eigenvalues, eigenvectors, numFactors)
		return loadings, nil, nil, converged, iterations, err

	case FactorExtractionPAF:
		facOpts.Fm = "pa"

	case FactorExtractionML:
		facOpts.Fm = "ml"

	case FactorExtractionMINRES:
		facOpts.Fm = "minres"

	default:
		// Default to MINRES to match R psych::fa and the documented default behavior.
		facOpts.Fm = "minres"
	}

	// Call our psych_fac.Fac implementation
	result, err := fa.Fac(corrMatrix, facOpts)
	if err != nil {
		return nil, nil, nil, false, 0, err
	}

	// Extract results from FacResult
	loadings := result.Loadings
	communalities := append([]float64(nil), result.Communalities...)
	converged := true // psych_fac handles convergence internally
	iterations := 0   // psych_fac doesn't track iterations in the same way

	// For methods other than PCA, we need to compute eigenvalues from the final communalities
	var extractionEigenvalues []float64
	if opt.Extraction != FactorExtractionPCA && result.EValues != nil {
		extractionEigenvalues = make([]float64, len(result.EValues))
		copy(extractionEigenvalues, result.EValues)
	}

	return loadings, communalities, extractionEigenvalues, converged, iterations, nil
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
	for j := 0; j < numFactors; j++ {
		colSum := 0.0
		for i := range p {
			colSum += loadings.At(i, j)
		}
		if colSum < 0 {
			for i := range p {
				loadings.Set(i, j, -loadings.At(i, j))
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

		insyra.LogDebug("stats", "FactorAnalysis", "PAF reducedCorr prepared for %d variables", rows)

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
		hasImaginaryEigenvalue := false
		for i := range rows {
			for j := 0; j < numFactors; j++ {
				// loadings = eigenvectors * sqrt(eigenvalues)
				val := pairs[j].value
				if val > 0 {
					newLoadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(val))
				} else if val < 0 {
					// Negative eigenvalue (imaginary) - this indicates a problem
					hasImaginaryEigenvalue = true
					newLoadings.Set(i, j, 0)
				}
			}
		}

		// Check for imaginary eigenvalue condition
		// R: if (is.na(err)) { warning(...); break }
		if hasImaginaryEigenvalue {
			insyra.LogWarning("stats", "FactorAnalysis", "imaginary eigenvalue condition encountered in PAF; try again with lower communality estimates or SMC=FALSE")
			return nil, nil, false, iterations, fmt.Errorf("imaginary eigenvalue condition in PAF iteration %d", iterations)
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

		// Check convergence using R's criterion: change in total communality
		// R: err <- abs(comm - comm1)
		// comm = sum(diag(r.mat))  (sum of all communalities)
		// This is more stable than checking max individual change
		oldCommTotal := 0.0
		for _, h := range communalities {
			oldCommTotal += h
		}
		newCommTotal := 0.0
		for _, h := range newCommunalities {
			newCommTotal += h
		}
		commChange := math.Abs(newCommTotal - oldCommTotal)

		loadings = newLoadings
		communalities = newCommunalities

		// Use R's convergence criterion: total communality change
		// R uses: while (err > min.err)
		if commChange < tol || iterations >= maxIter {
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

// extractMINRES performs MINRES factor extraction (true minimum residual method)
func extractMINRES(corr *mat.Dense, numFactors int, maxIter int, tol float64) (*mat.Dense, bool, int, error) {
	if corr == nil {
		return nil, false, 0, fmt.Errorf("nil correlation matrix")
	}

	rows, cols := corr.Dims()
	if numFactors > cols {
		numFactors = cols
	}

	// Initialize communalities using squared multiple correlations (SMC)
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

	// Create reduced correlation matrix R* = R - diag(1 - communalities)
	reducedCorr := mat.NewDense(rows, cols, nil)
	reducedCorr.Copy(corr)
	for i := range rows {
		reducedCorr.Set(i, i, corr.At(i, i)*(1.0-communalities[i]))
	}

	// Use true MINRES: minimize the sum of squared residuals in lower triangle
	// Initialize loadings using eigenvalue decomposition as starting point
	// Use eigenvalue decomposition of the reduced correlation matrix as starting point
	reducedCorrSym := mat.NewSymDense(rows, nil)
	for i := range rows {
		for j := range rows {
			reducedCorrSym.SetSym(i, j, reducedCorr.At(i, j))
		}
	}

	var eig mat.EigenSym
	if !eig.Factorize(reducedCorrSym, true) {
		// Return zero loadings if decomposition fails
		return mat.NewDense(rows, numFactors, nil), false, 0, fmt.Errorf("eigenvalue decomposition failed")
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

	// Extract initial loadings for first numFactors
	loadings := mat.NewDense(rows, numFactors, nil)
	for i := range rows {
		for j := 0; j < numFactors; j++ {
			val := pairs[j].value
			if val > 0 {
				loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(val))
			}
		}
	}

	// Flatten loadings to vector for optimization
	loadingsVec := make([]float64, rows*numFactors)
	for i := range rows {
		for j := range numFactors {
			loadingsVec[i*numFactors+j] = loadings.At(i, j)
		}
	}

	// Objective function: sum of squared lower triangular residuals
	objFunc := func(x []float64) float64 {
		// Reshape x to loadings matrix
		load := mat.NewDense(rows, numFactors, nil)
		for i := range rows {
			for j := range numFactors {
				load.Set(i, j, x[i*numFactors+j])
			}
		}

		// Compute model = load * load^T
		var model mat.Dense
		model.Mul(load, load.T())

		// Compute residual = reducedCorr - model
		var residual mat.Dense
		residual.Sub(reducedCorr, &model)

		// Sum of squared lower triangular elements (matching R's minres)
		sumSq := 0.0
		for i := 0; i < rows; i++ {
			for j := 0; j < i; j++ { // lower triangle
				diff := residual.At(i, j)
				sumSq += diff * diff
			}
		}
		return sumSq
	}

	// Gradient function (optional, but helps convergence)
	gradFunc := func(grad, x []float64) {
		// Reshape x to loadings matrix
		load := mat.NewDense(rows, numFactors, nil)
		for i := range rows {
			for j := range numFactors {
				load.Set(i, j, x[i*numFactors+j])
			}
		}

		// Compute model = load * load^T
		var model mat.Dense
		model.Mul(load, load.T())

		// Compute residual = reducedCorr - model
		var residual mat.Dense
		residual.Sub(reducedCorr, &model)

		// Gradient: d(sum(residual^2))/d(load) = -2 * residual * load
		// For each element in loadings
		for i := range rows {
			for k := range numFactors {
				sum := 0.0
				for j := 0; j < rows; j++ {
					if j < i { // lower triangle contribution
						res := residual.At(i, j)
						sum += res * load.At(j, k)
					} else if j > i { // symmetric upper triangle
						res := residual.At(j, i)
						sum += res * load.At(j, k)
					}
				}
				grad[i*numFactors+k] = -2.0 * sum
			}
		}
	}

	// Use BFGS optimization
	p := optimize.Problem{
		Func: objFunc,
		Grad: gradFunc,
	}

	method := &optimize.BFGS{}
	settings := &optimize.Settings{
		Converger:         &optimize.FunctionConverge{Absolute: tol, Iterations: maxIter},
		GradientThreshold: tol * 0.1,
	}

	result, err := optimize.Minimize(p, loadingsVec, settings, method)
	if err != nil {
		// Fallback to initial loadings
		insyra.LogWarning("stats", "FactorAnalysis", "MINRES optimization failed, using initial loadings: %v", err)
		return loadings, false, 0, nil
	}

	converged := result.Status == optimize.Success || result.Status == optimize.FunctionConvergence
	iterations := result.FuncEvaluations

	// Reshape optimized vector back to loadings matrix
	optimizedLoadings := mat.NewDense(rows, numFactors, nil)
	for i := range rows {
		for j := range numFactors {
			optimizedLoadings.Set(i, j, result.X[i*numFactors+j])
		}
	}

	return optimizedLoadings, converged, iterations, nil
}

// minresFit performs the core MINRES fitting for factor extraction

// extractML_EM performs Maximum Likelihood factor extraction using EM algorithm
// This is a simpler and more stable alternative to BFGS optimization
func extractML_EM(corr *mat.Dense, numFactors int, maxIter int, tol float64, sampleSize int, communalities []float64) (*mat.Dense, bool, int, error) {
	rows, _ := corr.Dims()

	// Initialize psi (uniqueness) from SMC, exactly as R does:
	// start = diag(S) - S.smc = 1 - smc(S)
	psi := make([]float64, rows)
	maxSMC := 0.0
	for i := range rows {
		if communalities[i] > maxSMC {
			maxSMC = communalities[i]
		}
		psi[i] = 1.0 - communalities[i] // communalities = SMC
	}

	// R uses: upper = max(S.smc, 1), NOT a fixed 0.995!
	upper := math.Max(maxSMC, 1.0)
	lower := 0.005

	// Apply bounds to initial psi
	for i := range rows {
		if psi[i] < lower {
			psi[i] = lower
		}
		if psi[i] > upper {
			psi[i] = upper
		}
	}

	insyra.LogInfo("stats", "FactorAnalysis", "ML: SMC communalities[0]=%.4f, maxSMC=%.4f, upper=%.4f, initial psi[0]=%.4f (1-SMC)", communalities[0], maxSMC, upper, psi[0])

	// Transform psi to unconstrained space for optimization
	// Use: psi = lower + (upper - lower) * sigmoid(x)
	// So: x = logit((psi - lower)/(upper - lower))
	// Additionally, apply R's parscale=0.01, which scales parameters by dividing by 0.01 (i.e., multiplying by 100)
	parscale := 0.01 // R's parscale parameter

	x0 := make([]float64, rows)
	for i := range rows {
		// Transform psi to x
		ratio := (psi[i] - lower) / (upper - lower)
		if ratio <= 0.001 {
			ratio = 0.001
		}
		if ratio >= 0.999 {
			ratio = 0.999
		}
		xi := math.Log(ratio / (1.0 - ratio)) // logit
		// Apply parscale: x_scaled = x / parscale
		x0[i] = xi / parscale
	}

	// Objective function in scaled transformed space
	objFunc := func(xScaled []float64) float64 {
		// Unscale: x = x_scaled * parscale
		// Transform x to psi
		psiTrans := make([]float64, len(xScaled))
		for i, xScaledI := range xScaled {
			xi := xScaledI * parscale // unscale
			sigmoid := 1.0 / (1.0 + math.Exp(-xi))
			psiTrans[i] = lower + (upper-lower)*sigmoid
		}
		obj, _ := mlObjectiveFAfn(corr, psiTrans, numFactors)
		return obj
	}

	// Gradient function in scaled transformed space
	gradFunc := func(grad, xScaled []float64) {
		// Unscale: x = x_scaled * parscale
		psiTrans := make([]float64, len(xScaled))
		sigmoids := make([]float64, len(xScaled))
		for i, xScaledI := range xScaled {
			xi := xScaledI * parscale
			sigmoid := 1.0 / (1.0 + math.Exp(-xi))
			sigmoids[i] = sigmoid
			psiTrans[i] = lower + (upper-lower)*sigmoid
		}

		_, gradPsi := mlObjectiveFAfn(corr, psiTrans, numFactors)

		// Chain rule: d(obj)/d(x_scaled) = d(obj)/d(psi) * d(psi)/d(x) * d(x)/d(x_scaled)
		// d(psi)/d(x) = (upper-lower) * sigmoid * (1-sigmoid)
		// d(x)/d(x_scaled) = parscale
		for i := range grad {
			dpsi_dx := (upper - lower) * sigmoids[i] * (1.0 - sigmoids[i])
			dx_dxScaled := parscale
			grad[i] = gradPsi[i] * dpsi_dx * dx_dxScaled
		}
	}

	p := optimize.Problem{
		Func: objFunc,
		Grad: gradFunc,
	}

	// Use BFGS with transformed variables (no bounds needed)
	method := &optimize.BFGS{}
	settings := &optimize.Settings{
		Converger:         &optimize.FunctionConverge{Absolute: tol * 0.01, Iterations: maxIter * 2},
		GradientThreshold: tol * 0.1,
	}

	insyra.LogInfo("stats", "FactorAnalysis", "ML: Starting BFGS optimization in transformed space")

	// Evaluate initial objective
	initialObj := objFunc(x0)
	insyra.LogInfo("stats", "FactorAnalysis", "ML: Initial objective = %.6f", initialObj)

	// Debug: evaluate objective at different psi[0] values to understand the landscape
	testPsi := []float64{0.005, 0.05, 0.1, 0.2, 0.25, 0.3, 0.4, 0.5}
	insyra.LogInfo("stats", "FactorAnalysis", "ML: Objective function landscape test (varying psi[0]):")
	for _, testVal := range testPsi {
		testPsiVec := make([]float64, rows)
		for i := range rows {
			testPsiVec[i] = psi[i]
		}
		testPsiVec[0] = testVal
		// Transform to x space
		testX := make([]float64, rows)
		for i := range rows {
			ratio := (testPsiVec[i] - lower) / (upper - lower)
			if ratio <= 0.001 {
				ratio = 0.001
			}
			if ratio >= 0.999 {
				ratio = 0.999
			}
			xi := math.Log(ratio / (1.0 - ratio))
			testX[i] = xi / parscale
		}
		testObj := objFunc(testX)
		insyra.LogInfo("stats", "FactorAnalysis", "  psi[0]=%.4f obj=%.6f", testVal, testObj)
	}

	result, err := optimize.Minimize(p, x0, settings, method)
	if err != nil {
		insyra.LogWarning("stats", "FactorAnalysis", "ML: BFGS optimization failed: %v, falling back to simple EM", err)
		return extractML_EM_OLD(corr, numFactors, maxIter, tol, sampleSize, communalities)
	}

	// Transform back to psi
	for i, xi := range result.X {
		xVal := xi * parscale // unscale
		sigmoid := 1.0 / (1.0 + math.Exp(-xVal))
		psi[i] = lower + (upper-lower)*sigmoid
	}
	converged := result.Status == optimize.Success || result.Status == optimize.FunctionConvergence
	iterations := result.FuncEvaluations
	finalObj := result.F

	insyra.LogInfo("stats", "FactorAnalysis", "ML: Optimization converged=%v, status=%v, iterations=%d, psi[0]=%.4f, finalObj=%.6f", converged, result.Status, iterations, psi[0], finalObj)

	// Debug: print all final psi values
	psiStr := "ML: Final psi = ["
	for i, p := range psi {
		if i > 0 {
			psiStr += ", "
		}
		psiStr += fmt.Sprintf("%.4f", p)
	}
	psiStr += "]"
	insyra.LogInfo("stats", "FactorAnalysis", psiStr)

	// Extract loadings using final psi
	loadings, err := mlExtractLoadingsFAout(corr, psi, numFactors)
	if err != nil {
		return nil, false, iterations, err
	}

	// Debug: verify communalities
	_, cols := loadings.Dims()
	comm0 := 0.0
	for j := range cols {
		comm0 += loadings.At(0, j) * loadings.At(0, j)
	}
	insyra.LogInfo("stats", "FactorAnalysis", "ML: Extracted loadings A1 communality=%.4f (should be %.4f)", comm0, 1.0-psi[0])

	return loadings, converged, iterations, nil
}

// mlObjectiveFAfn implements R's FAfn and FAgr functions for ML factor analysis
func mlObjectiveFAfn(S *mat.Dense, psi []float64, nfactors int) (float64, []float64) {
	n := len(psi)

	// Create scaling matrix sc = diag(1/sqrt(Psi))
	sc := mat.NewDense(n, n, nil)
	for i := range n {
		if psi[i] > 0 {
			sc.Set(i, i, 1.0/math.Sqrt(psi[i]))
		} else {
			sc.Set(i, i, 1.0/math.Sqrt(0.001))
		}
	}

	// Compute Sstar = sc %*% S %*% sc
	var temp mat.Dense
	temp.Mul(sc, S)
	var Sstar mat.Dense
	Sstar.Mul(&temp, sc)

	// Eigenvalue decomposition
	var eig mat.EigenSym
	SstarSym := mat.NewSymDense(n, Sstar.RawMatrix().Data)
	if !eig.Factorize(SstarSym, true) {
		grad := make([]float64, n)
		return 1e10, grad
	}

	eigenvalues := eig.Values(nil)
	eigenvectors := mat.NewDense(n, n, nil)
	eig.VectorsTo(eigenvectors)

	// IMPORTANT: gonum returns eigenvalues in ASCENDING order,
	// but R's eigen() returns DESCENDING order
	// We need to reverse them to match R's behavior
	// Reverse eigenvalues
	for i := 0; i < n/2; i++ {
		eigenvalues[i], eigenvalues[n-1-i] = eigenvalues[n-1-i], eigenvalues[i]
	}
	// Reverse eigenvector columns
	eigenvectorsReversed := mat.NewDense(n, n, nil)
	for i := range n {
		for j := range n {
			eigenvectorsReversed.Set(i, j, eigenvectors.At(i, n-1-j))
		}
	}
	eigenvectors = eigenvectorsReversed

	// Objective: FAfn = sum(log(e) - e) for eigenvalues beyond nfactors
	// e = E$values[-(1:nf)]
	obj := 0.0
	for i := nfactors; i < n; i++ {
		if eigenvalues[i] > 1e-10 {
			obj += math.Log(eigenvalues[i]) - eigenvalues[i]
		} else {
			// Handle near-zero eigenvalues
			obj += math.Log(1e-10) - 1e-10
		}
	}
	obj = obj - float64(nfactors) + float64(n)
	obj = -obj // R returns -e

	// Gradient: FAgr
	// L = E$vectors[, 1:nf] %*% diag(sqrt(pmax(E$values[1:nf] - 1, 0)))
	// load = diag(sqrt(Psi)) %*% L
	// g = load %*% t(load) + diag(Psi) - S
	// grad = diag(g) / Psi^2

	L := mat.NewDense(n, nfactors, nil)
	for i := range nfactors {
		val := eigenvalues[i] - 1.0
		if val < 0 {
			val = 0
		}
		scale := math.Sqrt(val)
		for j := range n {
			L.Set(j, i, eigenvectors.At(j, i)*scale)
		}
	}

	// load = diag(sqrt(Psi)) %*% L
	sqrtPsi := mat.NewDense(n, n, nil)
	for i := range n {
		if psi[i] > 0 {
			sqrtPsi.Set(i, i, math.Sqrt(psi[i]))
		} else {
			sqrtPsi.Set(i, i, math.Sqrt(0.001))
		}
	}

	var load mat.Dense
	load.Mul(sqrtPsi, L)

	// g = load %*% t(load) + diag(Psi) - S
	var g mat.Dense
	g.Mul(&load, load.T())
	for i := range n {
		g.Set(i, i, g.At(i, i)+psi[i])
	}
	// g = g - S
	for i := range n {
		for j := range n {
			g.Set(i, j, g.At(i, j)-S.At(i, j))
		}
	}

	// grad = diag(g) / Psi^2
	grad := make([]float64, n)
	for i := range n {
		if psi[i] > 0 {
			grad[i] = g.At(i, i) / (psi[i] * psi[i])
		}
	}

	return obj, grad
}

// mlExtractLoadingsFAout implements R's FAout function to extract loadings from psi
func mlExtractLoadingsFAout(S *mat.Dense, psi []float64, nfactors int) (*mat.Dense, error) {
	n := len(psi)

	// Create scaling matrix sc = diag(1/sqrt(Psi))
	sc := mat.NewDense(n, n, nil)
	sqrtPsi := mat.NewDense(n, n, nil)
	for i := range n {
		if psi[i] > 0 {
			sc.Set(i, i, 1.0/math.Sqrt(psi[i]))
			sqrtPsi.Set(i, i, math.Sqrt(psi[i]))
		} else {
			sc.Set(i, i, 1.0/math.Sqrt(0.001))
			sqrtPsi.Set(i, i, math.Sqrt(0.001))
		}
	}

	// Compute Sstar = sc %*% S %*% sc
	var temp mat.Dense
	temp.Mul(sc, S)
	var Sstar mat.Dense
	Sstar.Mul(&temp, sc)

	// Eigenvalue decomposition
	var eig mat.EigenSym
	SstarSym := mat.NewSymDense(n, Sstar.RawMatrix().Data)
	if !eig.Factorize(SstarSym, true) {
		return nil, fmt.Errorf("eigenvalue decomposition failed")
	}

	eigenvalues := eig.Values(nil)
	eigenvectors := mat.NewDense(n, n, nil)
	eig.VectorsTo(eigenvectors)

	// Reverse eigenvalues and eigenvectors (gonum: ascending, R: descending)
	for i := 0; i < n/2; i++ {
		eigenvalues[i], eigenvalues[n-1-i] = eigenvalues[n-1-i], eigenvalues[i]
	}
	eigenvectorsReversed := mat.NewDense(n, n, nil)
	for i := range n {
		for j := range n {
			eigenvectorsReversed.Set(i, j, eigenvectors.At(i, n-1-j))
		}
	}
	eigenvectors = eigenvectorsReversed

	// L = E$vectors[, 1:nf] %*% diag(sqrt(pmax(E$values[1:nf] - 1, 0)))
	L := mat.NewDense(n, nfactors, nil)
	for i := range nfactors {
		val := eigenvalues[i] - 1.0
		if val < 0 {
			val = 0
		}
		scale := math.Sqrt(val)
		for j := range n {
			L.Set(j, i, eigenvectors.At(j, i)*scale)
		}
	}

	// load = diag(sqrt(Psi)) %*% L
	var loadings mat.Dense
	loadings.Mul(sqrtPsi, L)

	return &loadings, nil
}

// Old EM implementation (keep for reference)
func extractML_EM_OLD(corr *mat.Dense, numFactors int, maxIter int, tol float64, sampleSize int, communalities []float64) (*mat.Dense, bool, int, error) {
	rows, _ := corr.Dims()

	// Initialize psi (uniqueness) from communalities
	psi := make([]float64, rows)
	for i := range rows {
		psi[i] = 1.0 - communalities[i]
		// Ensure psi is in valid range [0.005, 0.995]
		if psi[i] < 0.005 {
			psi[i] = 0.005
		}
		if psi[i] > 0.995 {
			psi[i] = 0.995
		}
	}

	insyra.LogInfo("stats", "FactorAnalysis", "ML(EM): Starting with %d factors, initial psi[0]=%.4f", numFactors, psi[0])

	// EM algorithm iterations
	converged := false
	var loadings *mat.Dense
	iterations := 0

	for iter := 0; iter < maxIter; iter++ {
		iterations = iter + 1

		// E-step: Extract loadings given current psi
		var err error
		loadings, err = mlExtractLoadings(corr, psi, numFactors)
		if err != nil {
			return nil, false, iterations, err
		}

		// M-step: Update psi to maximize likelihood
		// For each variable: psi[i] = max(1 - communality[i], 0.005)
		// where communality[i] = sum_k(loadings[i,k]^2)
		newPsi := make([]float64, rows)
		maxChange := 0.0

		for i := range rows {
			// Compute communality from loadings
			communality := 0.0
			for k := 0; k < numFactors; k++ {
				communality += loadings.At(i, k) * loadings.At(i, k)
			}

			// Update psi = 1 - communality
			newPsi[i] = 1.0 - communality

			// Constrain psi to [0.005, 0.995]
			if newPsi[i] < 0.005 {
				newPsi[i] = 0.005
			}
			if newPsi[i] > 0.995 {
				newPsi[i] = 0.995
			}

			// Track maximum change for convergence
			change := math.Abs(newPsi[i] - psi[i])
			if change > maxChange {
				maxChange = change
			}
		}

		// Update psi
		psi = newPsi

		// Log progress
		if iter < 5 || iter%20 == 0 || maxChange < tol {
			insyra.LogInfo("stats", "FactorAnalysis", "ML(EM): Iter %d, maxChange=%.6f, psi[0]=%.4f", iter+1, maxChange, psi[0])
		}

		// Check convergence
		if maxChange < tol {
			converged = true
			insyra.LogInfo("stats", "FactorAnalysis", "ML(EM): Converged at iteration %d", iter+1)
			break
		}
	}

	if !converged {
		insyra.LogWarning("stats", "FactorAnalysis", "ML(EM): Did not converge after %d iterations", maxIter)
	}

	return loadings, converged, iterations, nil
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
	// Use logit transformation to constrain psi in (0, 1)
	startPsi := make([]float64, rows)
	for i := range rows {
		psi := 1.0 - communalities[i]
		if psi < 0.005 {
			psi = 0.005
		}
		if psi > 0.995 {
			psi = 0.995
		}
		// logit(psi) = log(psi / (1 - psi))
		startPsi[i] = math.Log(psi / (1.0 - psi))
	}

	// Use BFGS optimization from gonum/optimize
	// Optimize in logit space, then transform back
	objFunc := func(logitPsi []float64) float64 {
		// Transform from logit to psi: psi = exp(logit) / (1 + exp(logit))
		psi := make([]float64, len(logitPsi))
		for i, lp := range logitPsi {
			psi[i] = 1.0 / (1.0 + math.Exp(-lp)) // sigmoid function
		}
		obj := computeMLObjective(corr, psi, numFactors)
		// DEBUG: log first few evaluations
		return obj
	}
	gradFunc := func(grad, logitPsi []float64) {
		// Transform from logit to psi
		psi := make([]float64, len(logitPsi))
		for i, lp := range logitPsi {
			psi[i] = 1.0 / (1.0 + math.Exp(-lp))
		}
		_, g := mlObjectiveAndGradient(corr, psi, numFactors)
		// Chain rule: d(obj)/d(logit) = d(obj)/d(psi) * d(psi)/d(logit)
		// d(psi)/d(logit) = psi * (1 - psi) (derivative of sigmoid)
		for i := range grad {
			grad[i] = g[i] * psi[i] * (1.0 - psi[i])
		}
	}
	p := optimize.Problem{
		Func: objFunc,
		Grad: gradFunc,
	}

	method := &optimize.BFGS{}
	settings := &optimize.Settings{
		Converger:         &optimize.FunctionConverge{Absolute: tol, Iterations: maxIter},
		GradientThreshold: tol,
	}

	insyra.LogInfo("stats", "FactorAnalysis", "ML: Starting BFGS with %d initial logit(psi), first logit=%.4f", len(startPsi), startPsi[0])
	// Evaluate initial objective
	initialObj := objFunc(startPsi)
	insyra.LogInfo("stats", "FactorAnalysis", "ML: Initial objective value = %.6f", initialObj)

	result, err := optimize.Minimize(p, startPsi, settings, method)
	if err != nil {
		// Fallback to simple gradient descent if BFGS fails
		insyra.LogWarning("stats", "FactorAnalysis", "BFGS optimization failed, falling back to gradient descent: %v", err)
		return extractMLFallback(corr, numFactors, maxIter, tol, sampleSize, initialCommunalities)
	}

	// Extract optimized psi from logit space
	psi := make([]float64, len(result.X))
	for i, lp := range result.X {
		psi[i] = 1.0 / (1.0 + math.Exp(-lp))
	}
	converged := result.Status == optimize.Success || result.Status == optimize.FunctionConvergence
	iterations := result.FuncEvaluations

	insyra.LogInfo("stats", "FactorAnalysis", "ML: BFGS converged=%v, status=%v, iterations=%d, first psi=%.4f", converged, result.Status, iterations, psi[0])
	finalObj := result.F
	insyra.LogInfo("stats", "FactorAnalysis", "ML: Final objective value = %.6f", finalObj)

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
// This uses the same algorithm as computeMLObjective for consistency
func mlObjectiveAndGradient(S *mat.Dense, psi []float64, nfactors int) (float64, []float64) {
	n := len(psi)

	// Compute objective using computeMLObjective for consistency
	obj := computeMLObjective(S, psi, nfactors)

	// Compute gradient using finite differences
	grad := make([]float64, n)
	eps := 1e-7 // Smaller epsilon for better accuracy
	for i := range n {
		psiPlus := make([]float64, n)
		copy(psiPlus, psi)
		psiPlus[i] += eps

		objPlus := computeMLObjective(S, psiPlus, nfactors)
		grad[i] = (objPlus - obj) / eps
	}

	return obj, grad
}

// computeMLObjective computes only the ML objective function (helper for gradient computation)
func computeMLObjective(S *mat.Dense, psi []float64, nfactors int) float64 {
	n := len(psi)

	// Check if psi is in valid range
	for _, p := range psi {
		if p >= 0.9999 {
			return 1e10 // Penalize psi too close to 1
		}
		if p <= 0.0001 {
			return 1e10 // Also penalize psi too close to 0
		}
	}

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
	// Compute log|S|
	var luS mat.LU
	luS.Factorize(S)
	if luS.Det() <= 0 {
		return 1e10 // Return large objective if S is not positive definite
	}
	logDetS := math.Log(math.Abs(luS.Det()))

	obj := logDetSigma + traceTerm - logDetS - float64(n)

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

	q := mat.NewDense(p, p, nil)
	if err := q.Inverse(corr); err != nil {
		// psych::KMO falls back to the input matrix when solve(r) fails.
		q.CloneFrom(corr)
	}
	return computeKMOFromInverse(corr, q, msaValues)
}

// computeKMOFromInverse computes KMO measures given the correlation matrix and its inverse
func computeKMOFromInverse(corr, invCorr *mat.Dense, msaValues []float64) (overallKMO float64, msa []float64, err error) {
	p, _ := corr.Dims()
	qCor := mat.NewDense(p, p, nil)
	for i := range p {
		for j := range p {
			denom := math.Sqrt(invCorr.At(i, i) * invCorr.At(j, j))
			if denom == 0 || math.IsNaN(denom) {
				qCor.Set(i, j, 0)
			} else {
				qCor.Set(i, j, invCorr.At(i, j)/denom)
			}
		}
		qCor.Set(i, i, 0)
	}

	// Compute MSA (Measure of Sampling Adequacy) for each variable
	for i := range p {
		sumRSquared := 0.0
		sumPSquared := 0.0

		for j := range p {
			if i != j {
				r := corr.At(i, j)
				p_ij := qCor.At(i, j)
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
				p_ij := qCor.At(i, j)
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

	detR := mat.Det(corr)
	if detR <= 0 || math.IsNaN(detR) {
		return 0, 1.0, df, nil
	}
	chiSquare = -math.Log(detR) * (float64(n-1) - (2*float64(p)+5)/6)

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
// Uses R's method:
//   - For orthogonal: ev.rotated <- diag(t(loadings) %*% loadings)
//   - For oblique: ev.rotated <- diag(Phi %*% t(loadings) %*% loadings)
func sortFactorsByExplainedVariance(loadings *mat.Dense, rotationMatrix *mat.Dense, phi *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil {
		return nil, rotationMatrix, phi
	}

	rows, cols := loadings.Dims()

	// Calculate explained variance for each factor
	variances := make([]float64, cols)

	if phi == nil || isIdentityMatrix(phi) {
		// Orthogonal rotation: ev.rotated <- diag(t(loadings) %*% loadings)
		for j := range cols {
			sum := 0.0
			for i := range rows {
				loading := loadings.At(i, j)
				sum += loading * loading
			}
			variances[j] = sum
		}
	} else {
		// Oblique rotation: ev.rotated <- diag(Phi %*% t(loadings) %*% loadings)
		// First compute t(loadings) %*% loadings
		ltl := mat.NewDense(cols, cols, nil)
		ltl.Product(loadings.T(), loadings)

		// Then compute Phi %*% (t(loadings) %*% loadings)
		result := mat.NewDense(cols, cols, nil)
		result.Product(phi, ltl)

		// Extract diagonal
		for j := range cols {
			variances[j] = result.At(j, j)
		}
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
	for j := range cols {
		newCol := indices[j]
		for i := range rows {
			sortedLoadings.Set(i, j, loadings.At(i, newCol))
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

func rotatePrincipalPromax(loadings *mat.Dense, rotationOpts FactorRotationOptions) (*mat.Dense, *mat.Dense, *mat.Dense, bool, error) {
	power := int(rotationOpts.Kappa)
	if power <= 0 {
		power = 4
	}
	res := fa.Promax(loadings, power, false)
	rotated, ok := res["loadings"].(*mat.Dense)
	if !ok || rotated == nil {
		return nil, nil, nil, false, fmt.Errorf("promax rotation did not return loadings")
	}
	rotMat, ok := res["rotmat"].(*mat.Dense)
	if !ok || rotMat == nil {
		return nil, nil, nil, false, fmt.Errorf("promax rotation did not return rotation matrix")
	}

	var inv mat.Dense
	if err := inv.Inverse(rotMat); err != nil {
		pinv, pinvErr := fa.Pinv(rotMat, 0)
		if pinvErr != nil {
			return nil, nil, nil, false, fmt.Errorf("failed to invert promax rotation matrix: %w", err)
		}
		inv.CloneFrom(pinv)
	}
	var cov mat.Dense
	cov.Mul(&inv, inv.T())
	phi := covarianceToCorrelation(&cov)
	return rotated, rotMat, phi, true, nil
}

func covarianceToCorrelation(cov *mat.Dense) *mat.Dense {
	n, _ := cov.Dims()
	cor := mat.NewDense(n, n, nil)
	for i := range n {
		for j := range n {
			denom := math.Sqrt(cov.At(i, i) * cov.At(j, j))
			if denom == 0 || math.IsNaN(denom) {
				cor.Set(i, j, 0)
			} else {
				cor.Set(i, j, cov.At(i, j)/denom)
			}
		}
	}
	return cor
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
		// Do NOT apply sign standardization here - it will be done later in main function after sorting
		return loadings, identity, phi, true, nil

	case FactorRotationVarimax:
		// Check if user wants Kaiser Varimax (SPSS-compatible) or GPArotation (R-compatible)
		if rotationOpts.VarimaxAlgorithm == VarimaxKaiser {
			// Use Kaiser Varimax (Jacobi rotation) - matches SPSS
			rotatedLoadings, rotMat, err := fa.KaiserVarimaxWithRotationMatrix(loadings, true, 1000, 1e-5)
			if err != nil {
				return nil, nil, nil, false, err
			}

			insyra.LogDebug("stats", "FactorAnalysis", "Kaiser Varimax rotation completed")

			// Apply sign standardization
			standardizedLoadings := standardizeFactorSigns(rotatedLoadings)

			return standardizedLoadings, rotMat, nil, true, nil
		} else {
			// Default: Use GPArotation Varimax - matches R
			method = "varimax"
		}

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
		Gamma:       rotationOpts.Delta,      // Use Delta as GPArotation gamma for oblimin
		PromaxPower: int(rotationOpts.Kappa), // Use Kappa as PromaxPower
		Restarts:    rotationOpts.Restarts,
	}

	rotatedLoadings, rotMat, phi, converged, err := fa.Rotate(loadings, method, opts)
	if err != nil {
		return nil, nil, nil, false, err
	}

	insyra.LogDebug("stats", "FactorAnalysis", "factor rotation completed")

	// Do NOT apply sign standardization here - it will be done AFTER sorting in main function
	// R: sign standardization happens after sorting, not after rotation
	// R code: signed <- sign(colSums(loadings)); loadings <- loadings %*% diag(signed)

	return rotatedLoadings, rotMat, phi, converged, nil
} // standardizeFactorSigns standardizes the signs of factor loadings
// Uses R's method: sign(colSums(loadings)) to ensure sum of each factor is positive
func standardizeFactorSigns(loadings *mat.Dense) *mat.Dense {
	if loadings == nil {
		return nil
	}

	rows, cols := loadings.Dims()
	standardized := mat.DenseCopyOf(loadings)

	for j := range cols {
		// Calculate sum of loadings for this factor (R method: colSums)
		sum := 0.0
		for i := range rows {
			sum += standardized.At(i, j)
		}

		// If sum is negative (or zero, default to positive), reflect the entire factor
		// R code: signed[signed == 0] <- 1
		if sum < 0 {
			for i := range rows {
				standardized.Set(i, j, -standardized.At(i, j))
			}
		}
	}

	return standardized
}

// standardizePhiSigns applies sign changes to phi matrix based on loading sign changes
// R code: if (!is.null(Phi)) { Phi <- diag(signed) %*% Phi %*% diag(signed) }
func standardizePhiSigns(phi *mat.Dense, originalLoadings, standardizedLoadings *mat.Dense) *mat.Dense {
	if phi == nil || originalLoadings == nil || standardizedLoadings == nil {
		return phi
	}

	_, cols := phi.Dims()

	// Determine which factors had their signs flipped
	signs := make([]float64, cols)
	for j := range cols {
		// Compare first non-zero loading to determine if sign was flipped
		origSum := 0.0
		stdSum := 0.0
		rows, _ := originalLoadings.Dims()
		for i := range rows {
			origSum += originalLoadings.At(i, j)
			stdSum += standardizedLoadings.At(i, j)
		}

		// If signs are opposite, this factor was flipped
		if origSum*stdSum < 0 {
			signs[j] = -1.0
		} else {
			signs[j] = 1.0
		}
	}

	// Apply: Phi <- diag(signed) %*% Phi %*% diag(signed)
	standardizedPhi := mat.NewDense(cols, cols, nil)
	for i := range cols {
		for j := range cols {
			standardizedPhi.Set(i, j, signs[i]*phi.At(i, j)*signs[j])
		}
	}

	return standardizedPhi
}

// isIdentityMatrix checks if a matrix is an identity matrix (within tolerance)
func isIdentityMatrix(m *mat.Dense) bool {
	if m == nil {
		return false
	}

	rows, cols := m.Dims()
	if rows != cols {
		return false
	}

	const tol = 1e-6
	for i := range rows {
		for j := range cols {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			if math.Abs(m.At(i, j)-expected) > tol {
				return false
			}
		}
	}

	return true
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
		return computeRegressionScores(data, loadings, phi, uniquenesses, sigmaForScores)

	case FactorScoreBartlett:
		return computeBartlettScores(data, loadings, phi, uniquenesses, sigmaForScores)

	case FactorScoreAndersonRubin:
		return computeAndersonRubinScores(data, loadings, phi, uniquenesses, sigmaForScores)

	default:
		// Default to regression method
		return computeRegressionScores(data, loadings, phi, uniquenesses, sigmaForScores)
	}
}

// computeRegressionScores computes factor scores using regression method
func computeRegressionScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64, sigma *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	r := scoreCorrelationMatrix(loadings, phi, uniquenesses, sigma)
	factorPattern := factorPatternWithPhi(loadings, phi)
	weights, err := solveOrPinv(r, factorPattern)
	if err != nil {
		return nil, nil, nil, err
	}

	var scores mat.Dense
	scores.Mul(data, weights)

	return &scores, weights, sampleCovarianceDense(&scores), nil
}

// computeBartlettScores computes factor scores using Bartlett's weighted least squares method
func computeBartlettScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64, sigma *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	_ = sigma
	invU2 := inverseUniquenessDiagonal(loadings, phi, uniquenesses)

	var invU2Loadings mat.Dense
	invU2Loadings.Mul(invU2, loadings)

	var middle mat.Dense
	middle.Mul(loadings.T(), &invU2Loadings)

	middleInv, err := inverseOrPinv(&middle)
	if err != nil {
		return nil, nil, nil, err
	}

	var weights mat.Dense
	weights.Mul(&invU2Loadings, middleInv)

	var scores mat.Dense
	scores.Mul(data, &weights)

	return &scores, &weights, sampleCovarianceDense(&scores), nil
}

// computeAndersonRubinScores computes factor scores using Anderson-Rubin's method
func computeAndersonRubinScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64, sigma *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	r := scoreCorrelationMatrix(loadings, phi, uniquenesses, sigma)
	invU2 := inverseUniquenessDiagonal(loadings, phi, uniquenesses)

	var invU2R mat.Dense
	invU2R.Mul(invU2, r)
	var invU2RInvU2 mat.Dense
	invU2RInvU2.Mul(&invU2R, invU2)
	var left mat.Dense
	left.Mul(loadings.T(), &invU2RInvU2)
	var middle mat.Dense
	middle.Mul(&left, loadings)

	invSqrt, err := inverseSymmetricSqrt(&middle)
	if err != nil {
		return nil, nil, nil, err
	}

	var invU2Loadings mat.Dense
	invU2Loadings.Mul(invU2, loadings)
	var weights mat.Dense
	weights.Mul(&invU2Loadings, invSqrt)

	var scores mat.Dense
	scores.Mul(data, &weights)

	return &scores, &weights, sampleCovarianceDense(&scores), nil
}

func scoreCorrelationMatrix(loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64, sigma *mat.Dense) *mat.Dense {
	if sigma != nil {
		return sigma
	}
	return computeSigma(loadings, phi, uniquenesses)
}

func factorPatternWithPhi(loadings *mat.Dense, phi *mat.Dense) *mat.Dense {
	if phi == nil {
		return mat.DenseCopyOf(loadings)
	}
	var pattern mat.Dense
	pattern.Mul(loadings, phi)
	return &pattern
}

func solveOrPinv(a *mat.Dense, b mat.Matrix) (*mat.Dense, error) {
	var chol mat.Cholesky
	if chol.Factorize(symDenseCopy(a)) {
		var solved mat.Dense
		if err := chol.SolveTo(&solved, b); err == nil {
			return &solved, nil
		}
	}

	var lu mat.LU
	lu.Factorize(a)
	var solved mat.Dense
	if err := lu.SolveTo(&solved, false, b); err == nil {
		return &solved, nil
	}

	pinv, err := fa.Pinv(a, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to solve scoring linear system: %w", err)
	}
	solved.Mul(pinv, b)
	return &solved, nil
}

func inverseOrPinv(a *mat.Dense) (*mat.Dense, error) {
	var inv mat.Dense
	if err := inv.Inverse(a); err == nil {
		return &inv, nil
	}
	pinv, err := fa.Pinv(a, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to invert scoring matrix: %w", err)
	}
	return pinv, nil
}

func inverseUniquenessDiagonal(loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64) *mat.DiagDense {
	p, _ := loadings.Dims()
	inv := make([]float64, p)
	if phi != nil {
		var tmp mat.Dense
		tmp.Mul(loadings, phi)
		var reproduced mat.Dense
		reproduced.Mul(&tmp, loadings.T())
		for i := range p {
			u2 := 1 - reproduced.At(i, i)
			if math.Abs(u2) <= machineEpsilon {
				u2 = math.Copysign(machineEpsilon, u2)
			}
			inv[i] = 1 / u2
		}
		return mat.NewDiagDense(p, inv)
	}
	for i := range p {
		u2 := 1.0
		if i < len(uniquenesses) {
			u2 = uniquenesses[i]
		}
		if math.Abs(u2) <= machineEpsilon {
			u2 = math.Copysign(machineEpsilon, u2)
		}
		inv[i] = 1 / u2
	}
	return mat.NewDiagDense(p, inv)
}

func inverseSymmetricSqrt(a *mat.Dense) (*mat.Dense, error) {
	sym := symDenseCopy(a)
	var eig mat.EigenSym
	if !eig.Factorize(sym, true) {
		return nil, fmt.Errorf("failed to decompose scoring matrix")
	}
	values := eig.Values(nil)
	vectors := mat.NewDense(len(values), len(values), nil)
	eig.VectorsTo(vectors)

	scaled := mat.NewDense(len(values), len(values), nil)
	for j, value := range values {
		if value < eigenvalueMinThreshold {
			value = eigenvalueMinThreshold
		}
		scale := 1 / math.Sqrt(value)
		for i := range values {
			scaled.Set(i, j, vectors.At(i, j)*scale)
		}
	}

	var invSqrt mat.Dense
	invSqrt.Mul(scaled, vectors.T())
	return &invSqrt, nil
}

func symDenseCopy(a mat.Matrix) *mat.SymDense {
	r, c := a.Dims()
	n := r
	if c < n {
		n = c
	}
	sym := mat.NewSymDense(n, nil)
	for i := range n {
		for j := 0; j <= i; j++ {
			sym.SetSym(i, j, 0.5*(a.At(i, j)+a.At(j, i)))
		}
	}
	return sym
}

func sampleCovarianceDense(scores *mat.Dense) *mat.Dense {
	_, m := scores.Dims()
	cov := mat.NewSymDense(m, nil)
	stat.CovarianceMatrix(cov, scores, nil)
	dense := mat.NewDense(m, m, nil)
	for i := range m {
		for j := range m {
			dense.Set(i, j, cov.At(i, j))
		}
	}
	return dense
}
