package stats

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/fa"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
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
	Delta            float64          // Optional: Oblimin gamma (default 0)
	GeominEpsilon    float64          // Optional: Geomin delta (default 0.01)
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

	// OptimFactr controls the L-BFGS-B convergence tolerance used by ML
	// and MINRES factor extraction: the optimizer terminates when the
	// relative function change drops below OptimFactr * machine epsilon.
	// Defaults to 1e7 (≈2.2e-9 absolute), matching R psych::fa /
	// stats::optim's "moderate accuracy" default. Use 1 for machine
	// precision (≈2.2e-16) when you want the true stationary point on
	// near-Heywood / flat objective surfaces; the default terminates
	// prematurely on those problems and settles at a non-stationary
	// boundary point. Use 1e12 for low precision (faster).
	OptimFactr float64

	// OptimMaxIter caps L-BFGS-B iterations for ML / MINRES extraction.
	// Defaults to 100 (matching R stats::optim default). Increase for
	// ill-conditioned problems where the optimizer would otherwise hit
	// the iteration cap before converging.
	OptimMaxIter int
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
		Scoring:      FactorScoreRegression, // R default: "regression"
		MaxIter:      50,                    // R default: 50
		MinErr:       0.001,                 // R default: 0.001
		OptimFactr:   1e7,                   // R default: stats::optim factr
		OptimMaxIter: 100,                   // R default: stats::optim maxit
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
	} else if opt.Rotation.Kappa < 0 {
		return opt, fmt.Errorf("rotation Kappa (Promax power) must be positive, got %g", opt.Rotation.Kappa)
	}
	if opt.Rotation.GeominEpsilon < 0 {
		return opt, fmt.Errorf("rotation GeominEpsilon must be non-negative, got %g", opt.Rotation.GeominEpsilon)
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
	if opt.OptimFactr <= 0 {
		opt.OptimFactr = defaults.OptimFactr
	}
	if opt.OptimMaxIter <= 0 {
		opt.OptimMaxIter = defaults.OptimMaxIter
	}

	return opt, nil
}

// Internal constants aligned with R's psych::fa and GPArotation package
const (
	// Correlation matrix diagonal checks
	corrDiagTolerance    = 1e-6 // Tolerance for diagonal deviation from 1.0
	corrDiagLogThreshold = 1e-8 // Threshold for logging diagonal deviations

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

	// Check for missing / non-finite values. Treat ±Inf as missing too —
	// without this, an Inf in the data would propagate into the correlation
	// matrix and produce silently-NaN output downstream.
	hasNonFinite := false
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			v := data.At(i, j)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				hasNonFinite = true
				break
			}
		}
		if hasNonFinite {
			break
		}
	}

	if hasNonFinite {
		// Remove rows with any NaN/Inf (listwise deletion)
		validRows := make([]int, 0, rowNum)
		for i := 0; i < rowNum; i++ {
			valid := true
			for j := 0; j < colNum; j++ {
				v := data.At(i, j)
				if math.IsNaN(v) || math.IsInf(v, 0) {
					valid = false
					break
				}
			}
			if valid {
				validRows = append(validRows, i)
			}
		}
		if len(validRows) < 2 {
			return nil, fmt.Errorf("factor analysis requires at least two complete rows; %d remained after listwise deletion", len(validRows))
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
		if std == 0 {
			return nil, fmt.Errorf("factor analysis undefined for zero-variance column %d", j)
		}
		means[j] = mean
		sds[j] = std
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
			return nil, fmt.Errorf("failed to compute KMO/MSA: %w", kmoErr)
		} else {
			rows, cols := corrAdequacyDense.Dims()
			insyra.LogDebug("stats", "FactorAnalysis", "correlation matrix dimensions: %dx%d", rows, cols)
			samplingAdequacyTable = kmoToDataTable(overallKMO, msaValues, colNames)
		}

		if chi, pval, df, bartErr := computeBartlettFromCorrelation(corrAdequacyDense, rowNum); bartErr != nil {
			return nil, fmt.Errorf("failed to compute Bartlett's test: %w", bartErr)
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

	// Step 4: Eigenvalue decomposition (descending order, matching R's eigen())
	sortedEigenvalues, sortedEigenvectors, ok := fa.SymmetricEigenDescendingDsyevr(corrMatrix)
	if !ok {
		return nil, errors.New("eigenvalue decomposition failed")
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
			return nil, fmt.Errorf("number of factors (%d) cannot exceed number of variables (%d)", numFactors, colNum)
		} else {
			// Use the internal psych-compatible SMC helper instead of
			// duplicating pseudo-inverse logic in the public orchestration layer.
			smcVec, smcDiagnostics := fa.Smc(corrDense, &fa.SmcOptions{Covar: false})
			if smcVec == nil || smcVec.Len() != colNum {
				return nil, fmt.Errorf("failed to compute SMC communalities")
			}
			if errs, ok := smcDiagnostics["errors"].([]string); ok && len(errs) > 0 {
				return nil, fmt.Errorf("failed to compute SMC communalities: %s", strings.Join(errs, "; "))
			}
			copy(initialCommunalities, smcVec.RawVector().Data)
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
	if loadings == nil {
		// Catastrophic extraction failure (e.g., PAF eigendecomposition
		// returned ok=false). The internal layer doesn't always surface
		// this as an error, so guard explicitly to avoid a downstream nil
		// pointer panic in mat.DenseCopyOf / sign-fix.
		return nil, fmt.Errorf("factor extraction returned nil loadings (degenerate input?)")
	}
	// Sanity check loadings dimensions — extraction should produce p × numFactors.
	// If extraction returned wrong dimensions (rare bug in internal path),
	// catch here rather than letting it cause a confusing panic deeper in
	// the pipeline.
	if pL, mL := loadings.Dims(); pL != colNum || mL != numFactors {
		return nil, fmt.Errorf("factor extraction produced wrong dimensions: got %dx%d, want %dx%d",
			pL, mL, colNum, numFactors)
	}

	// Replace zero loadings with 1e-15 (matching R's behavior).
	{
		pVars, mFactors := loadings.Dims()
		for i := range pVars {
			for j := range mFactors {
				if loadings.At(i, j) == 0 {
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
		unrotatedLoadings = mat.DenseCopyOf(loadings)
		rotatedLoadings = loadings
		rotationMatrix = nil
		phi = nil
		rotationConverged = true
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

	// R psych::fa does NOT apply the sign flip to rot.mat. It only flips
	// the reported loadings and Phi. The internal T is left as-is so the
	// reported `fit$rot.mat` is whatever the rotation routine emitted.

	// R's psych::fa() sorts factors AFTER sign standardization, by explained variance
	// This must be done AFTER rotation and sign standardization to match R's behavior
	// R: if (nfactors > 1) { ... sorting logic ... }
	if numFactors > 1 {
		isPCA := opt.Extraction == FactorExtractionPCA
		rotatedLoadings, rotationMatrix, phi = sortFactorsByExplainedVariance(rotatedLoadings, rotationMatrix, phi, isPCA)

		// R's reference for `unrotated_loadings` is fit0$loadings from a separate
		// fa(rotation="none") call, which itself runs R's SS-loadings sort. Apply
		// the same sort independently to our captured unrotated copy so both
		// match R's column ordering. The unrotated SS-sort permutation can
		// differ from the rotated one when rotation is not identity.
		unrotatedLoadings, _, _ = sortFactorsByExplainedVariance(unrotatedLoadings, nil, nil, isPCA)
	}

	// Step 8: Compute communalities and uniquenesses from the *unrotated*
	// loadings. The reported "communalities" field corresponds to psych::fa's
	// fit$communality (singular) = diag(model) = sum(L²), per
	// crosslang_baseline.R line 301 — NOT fit$communalities (plural) which
	// would be 1 - psi_optim. Per psych::fac.R lines 670/684:
	//   result$communality   <- diag(model)
	//   result$uniquenesses  <- diag(r - model)
	// The rotation block does not refresh `model`, so even for oblique
	// rotations the reported communality / uniqueness use unrotated L.
	extractionCommunalities := make([]float64, colNum)
	uniquenesses := make([]float64, colNum)
	for i := 0; i < colNum; i++ {
		var hi2 float64
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
		if vals, _, ok := fa.SymmetricEigenDescendingDsyevr(modifiedCorr); ok {
			reportedEigenvalues = vals
		}
	}

	// Limit/pad reportedEigenvalues to exactly numFactors for output. R always
	// reports an nFactors-length vector; if extraction returned fewer values
	// (rare: small p, degenerate eigendecomposition), pad with zeros so
	// downstream consumers see a consistent shape.
	switch {
	case len(reportedEigenvalues) > numFactors:
		reportedEigenvalues = reportedEigenvalues[:numFactors]
	case len(reportedEigenvalues) < numFactors:
		padded := make([]float64, numFactors)
		copy(padded, reportedEigenvalues)
		reportedEigenvalues = padded
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
			// We mirror that for R-baseline parity, but warn so callers who
			// explicitly asked for Bartlett/Anderson-Rubin know it was silently
			// downgraded — they likely want non-PCA extraction (PAF/ML/MINRES)
			// to get true AR/Bartlett scoring.
			if scoringMethod != FactorScoreRegression {
				insyra.LogWarning("stats", "FactorAnalysis",
					"%s scoring is not supported for PCA extraction (psych::principal uses regression-style scoring); silently using FactorScoreRegression. For true %s scoring, use PAF/ML/MINRES extraction.",
					scoringMethod, scoringMethod)
			}
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
		return nil, fmt.Errorf("unsupported extraction method: %s", opt.Extraction)
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

	return &FactorModel{FactorAnalysisResult: result}, nil
}

// computeSigma computes the reproduced correlation matrix: Sigma = L * Phi * L^T + U
func computeSigma(loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64) *mat.Dense {
	if loadings == nil {
		return nil
	}
	p, _ := loadings.Dims()
	if len(uniquenesses) != p {
		// Defensive: callers should always supply p uniquenesses, but
		// silently truncating/zero-padding could mask upstream bugs.
		return nil
	}

	// Floor any Heywood-case negative uniquenesses to 0 before forming
	// Sigma so downstream Cholesky / Eigen calls don't blow up on a
	// non-PSD matrix. This mirrors R psych's display-time clamp.
	uClamped := make([]float64, p)
	for i, u := range uniquenesses {
		if u < 0 {
			uClamped[i] = 0
		} else {
			uClamped[i] = u
		}
	}
	U := mat.NewDiagDense(p, uClamped)

	// Compute L * Phi * L^T
	var temp mat.Dense
	if phi != nil {
		temp.Mul(loadings, phi)
	} else {
		temp.CloneFrom(loadings)
	}
	var sigma mat.Dense
	sigma.Mul(&temp, loadings.T())

	// Add uniquenesses: Sigma = L·Phi·L^T + U
	sigma.Add(&sigma, U)

	return &sigma
}

// decideNumFactors determines the number of factors to extract
func decideNumFactors(eigenvalues []float64, spec FactorCountSpec, maxPossible int, sampleSize int) int {
	_ = sampleSize // reserved for future parallel-analysis-style methods
	switch spec.Method {
	case FactorCountFixed:
		if spec.FixedK > 0 && spec.FixedK <= maxPossible {
			return spec.FixedK
		}
		if spec.FixedK > maxPossible {
			insyra.LogWarning("stats", "FactorAnalysis",
				"FixedK=%d exceeds variable count %d; capping to %d",
				spec.FixedK, maxPossible, maxPossible)
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

// extractFactors wraps the internal extraction functions. The `data`
// parameter is unused (extraction operates on the correlation matrix);
// it is retained in the signature for symmetry with future extensions
// that may need raw observations (e.g. weighted least squares).
func extractFactors(data, corrMatrix *mat.Dense, eigenvalues []float64, eigenvectors *mat.Dense, numFactors int, opt FactorAnalysisOptions, sampleSize int, tol float64, initialCommunalities []float64) (*mat.Dense, []float64, []float64, bool, int, error) {
	_ = data
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
		OptimFactr:    opt.OptimFactr,
		OptimMaxIter:  opt.OptimMaxIter,
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
		return nil, nil, nil, false, 0, fmt.Errorf("unsupported extraction method: %s", opt.Extraction)
	}

	// Call our psych_fac.Fac implementation
	result, err := fa.Fac(corrMatrix, facOpts)
	if err != nil {
		return nil, nil, nil, false, 0, err
	}

	// Extract results from FacResult
	loadings := result.Loadings
	communalities := append([]float64(nil), result.Communalities...)
	// L-BFGS-B converged flag from the optimizer (ML/MINRES); PCA/PAF treat
	// convergence as their own loop's exit condition (see psych_fac.go: PAF
	// always returns Converged=true after its iteration).
	converged := result.Converged
	iterations := result.Iterations

	// For methods other than PCA, we need to compute eigenvalues from the final communalities
	var extractionEigenvalues []float64
	if opt.Extraction != FactorExtractionPCA && result.EValues != nil {
		extractionEigenvalues = make([]float64, len(result.EValues))
		copy(extractionEigenvalues, result.EValues)
	}

	return loadings, communalities, extractionEigenvalues, converged, iterations, nil
}

// computePCALoadings constructs factor loadings for PCA given eigenvalues/vectors.
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

// computeKMOMeasures computes Kaiser-Meyer-Olkin measure and individual MSA values with improved numerical stability
func computeKMOMeasures(corr *mat.Dense) (overallKMO float64, msaValues []float64, err error) {
	if corr == nil {
		return 0, nil, fmt.Errorf("nil correlation matrix")
	}

	p, _ := corr.Dims()
	msaValues = make([]float64, p)

	// Invert correlation matrix; gonum may panic on truly singular matrices,
	// so wrap with recover and surface a clean error instead of crashing.
	q := mat.NewDense(p, p, nil)
	var invErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				invErr = fmt.Errorf("inversion panicked (correlation matrix is singular): %v", r)
			}
		}()
		invErr = q.Inverse(corr)
	}()
	if invErr != nil {
		return 0, nil, fmt.Errorf("failed to invert correlation matrix for KMO/MSA: %w", invErr)
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

// kmoToDataTable converts KMO results to DataTable. The output is
// (p+1) × 1 with rows = per-variable MSA followed by the overall KMO.
// Row labels are taken from colNames (variable names) plus a final
// "Overall" row so consumers can tell which row is which.
func kmoToDataTable(overallKMO float64, msaValues []float64, colNames []string) *insyra.DataTable {
	msaList := insyra.NewDataList().SetName("MSA")

	for i := range msaValues {
		msaList.Append(msaValues[i])
	}
	msaList.Append(overallKMO)

	insyra.LogDebug("stats", "FactorAnalysis", "KMO values: MSA=%v, overall=%.6f", msaValues, overallKMO)

	dt := insyra.NewDataTable(msaList)
	// Build row labels: one per variable, plus "Overall" for the trailing
	// aggregate value. Without this the consumer can't tell which row is
	// which variable, nor that the last row is the global KMO.
	rowLabels := make([]string, 0, len(msaValues)+1)
	for i := range msaValues {
		if i < len(colNames) {
			rowLabels = append(rowLabels, colNames[i])
		} else {
			rowLabels = append(rowLabels, fmt.Sprintf("Var%d", i+1))
		}
	}
	rowLabels = append(rowLabels, "Overall")
	dt.SetRowNames(rowLabels)
	dt.SetName(tableNameSamplingAdequacy)
	return dt
}

// computeBartlettFromCorrelation computes Bartlett's test of sphericity with improved numerical stability
func computeBartlettFromCorrelation(corr *mat.Dense, n int) (chiSquare float64, pValue float64, df int, err error) {
	if corr == nil {
		return 0, 0, 0, fmt.Errorf("nil correlation matrix")
	}

	p, _ := corr.Dims()
	df = p * (p - 1) / 2

	if n <= 1 {
		return 0, 0, df, fmt.Errorf("Bartlett's test requires sample size > 1, got %d", n)
	}

	detR := mat.Det(corr)
	if math.IsNaN(detR) {
		return 0, 0, df, fmt.Errorf("correlation matrix determinant is NaN")
	}
	// True correlation matrices are PSD with det >= 0; tiny negatives are
	// numerical noise on near-singular matrices, treat as 0 (singular).
	if detR <= 0 {
		return 0, 0, df, fmt.Errorf("correlation matrix is singular (det=%g); Bartlett's test undefined", detR)
	}
	chiSquare = -math.Log(detR) * (float64(n-1) - (2*float64(p)+5)/6)
	pValue = chiSquaredPValue(chiSquare, float64(df))

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

	dt := insyra.NewDataTable(dataLists...)
	if tableName != "" {
		dt.SetName(tableName)
	}
	// Set row names if provided. SetRowNames uses each entry up to the
	// number of rows; pad/truncate to match.
	if len(rowNames) > 0 {
		rn := rowNames
		if len(rn) > rows {
			rn = rn[:rows]
		}
		dt.SetRowNames(rn)
	}
	return dt
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
	if tableName != "" {
		dt.SetName(tableName)
	}

	// Set row names if provided
	if len(rowNames) > 0 && len(rowNames) >= len(vector) {
		dt.SetRowNames(rowNames[:len(vector)])
	}

	return dt
}

// sortFactorsByExplainedVariance sorts factors by explained variance in
// descending order. R's psych::principal (PCA) uses diag(t(L) %*% L)
// regardless of rotation; psych::fa for oblique rotations uses
// diag(Phi %*% t(L) %*% L). The flag selects which convention.
func sortFactorsByExplainedVariance(loadings *mat.Dense, rotationMatrix *mat.Dense, phi *mat.Dense, isPCA bool) (*mat.Dense, *mat.Dense, *mat.Dense) {
	if loadings == nil {
		return nil, rotationMatrix, phi
	}

	rows, cols := loadings.Dims()
	variances := make([]float64, cols)

	if phi == nil || isPCA {
		// Either orthogonal (no Phi) or PCA: ev <- diag(t(L) %*% L).
		for j := range cols {
			sum := 0.0
			for i := range rows {
				loading := loadings.At(i, j)
				sum += loading * loading
			}
			variances[j] = sum
		}
	} else {
		// psych::fa oblique: ev <- diag(Phi %*% t(L) %*% L).
		ltl := mat.NewDense(cols, cols, nil)
		ltl.Product(loadings.T(), loadings)
		result := mat.NewDense(cols, cols, nil)
		result.Product(phi, ltl)
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

	// R psych::fa does NOT permute rot.mat when sorting factors; it just
	// permutes loadings, Phi, and structure. Pass rotationMatrix through
	// unchanged to match psych's reported `fit$rot.mat`.
	return sortedLoadings, rotationMatrix, sortedPhi
}

func rotatePrincipalPromax(loadings *mat.Dense, rotationOpts FactorRotationOptions) (*mat.Dense, *mat.Dense, *mat.Dense, bool, error) {
	power := int(rotationOpts.Kappa)
	if power <= 0 {
		power = 4
	}
	res := fa.Promax(loadings, power, false)
	if errMsg, ok := res["error"].(string); ok && errMsg != "" {
		return nil, nil, nil, false, fmt.Errorf("promax rotation failed: %s", errMsg)
	}
	rotated, ok := res["loadings"].(*mat.Dense)
	if !ok || rotated == nil {
		return nil, nil, nil, false, fmt.Errorf("promax rotation did not return loadings")
	}
	rotMat, ok := res["rotmat"].(*mat.Dense)
	if !ok || rotMat == nil {
		return nil, nil, nil, false, fmt.Errorf("promax rotation did not return rotation matrix")
	}
	phi, ok := res["Phi"].(*mat.Dense)
	if !ok || phi == nil {
		return nil, nil, nil, false, fmt.Errorf("promax rotation did not return Phi")
	}
	return rotated, rotMat, phi, true, nil
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
		return nil, nil, nil, false, fmt.Errorf("unsupported rotation method: %s", rotationOpts.Method)
	}

	// Use fa.Rotate function
	opts := &fa.RotOpts{
		Eps:           minErr,                  // Use MinErr from function parameter
		MaxIter:       maxIter,                 // Use MaxIter from function parameter
		Gamma:         rotationOpts.Delta,      // Use Delta as GPArotation gamma for oblimin
		GeominEpsilon: rotationOpts.GeominEpsilon,
		PromaxPower:   int(rotationOpts.Kappa), // Use Kappa as PromaxPower
		Restarts:      rotationOpts.Restarts,
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

// computeFactorScores computes factor scores using the specified method
func computeFactorScores(data *mat.Dense, loadings *mat.Dense, phi *mat.Dense, uniquenesses []float64, sigmaForScores *mat.Dense, method FactorScoreMethod) (*mat.Dense, *mat.Dense, *mat.Dense, error) {
	if data == nil || loadings == nil {
		return nil, nil, nil, fmt.Errorf("nil input matrices")
	}

	switch method {
	case FactorScoreNone:
		// R psych returns NULL when scores=FALSE; mirror that with nil so
		// the public API doesn't fill `Scores` with a meaningless zero
		// matrix the user didn't ask for.
		return nil, nil, nil, nil

	case FactorScoreRegression:
		return computeRegressionScores(data, loadings, phi, uniquenesses, sigmaForScores)

	case FactorScoreBartlett:
		return computeBartlettScores(data, loadings, phi, uniquenesses, sigmaForScores)

	case FactorScoreAndersonRubin:
		return computeAndersonRubinScores(data, loadings, phi, uniquenesses, sigmaForScores)

	default:
		return nil, nil, nil, fmt.Errorf("unsupported factor score method: %s", method)
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

// computeBartlettScores computes factor scores using Bartlett's weighted least squares method.
// R psych: weights = inv.U2 %*% L %*% solve(t(L) %*% inv.U2 %*% L). Bartlett uses pattern L
// (not S = L·Φ); ScoreCovariance is reported as the sample covariance of
// the produced scores, matching the R baseline script's stats::cov(scores).
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

// computeAndersonRubinScores computes factor scores using Anderson-Rubin's method.
// Empirically the R baseline cache rejects S=L·Φ — psych::factor.scores either
// uses L (factor pattern) directly or `S` is bound differently than the agent
// trace claimed. Use L throughout to match the baseline outputs.
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
	// Mirror R psych::factor.scores: try solve(R, S); if singular, fall back to
	// ginv(R) %*% S. gonum mat.LU.SolveTo can panic on a structurally
	// singular A (zero diagonal in U), so wrap in recover and route to Pinv.
	var solved *mat.Dense
	func() {
		defer func() {
			if r := recover(); r != nil {
				solved = nil
			}
		}()
		var lu mat.LU
		lu.Factorize(a)
		var s mat.Dense
		if err := lu.SolveTo(&s, false, b); err == nil {
			solved = &s
		}
	}()
	if solved != nil {
		return solved, nil
	}
	pinv, err := fa.Pinv(a, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to solve scoring linear system: %w", err)
	}
	var out mat.Dense
	out.Mul(pinv, b)
	return &out, nil
}

func inverseOrPinv(a *mat.Dense) (*mat.Dense, error) {
	// gonum's mat.Dense.Inverse can panic on truly-singular matrices.
	// Recover and fall back to SVD pseudoinverse.
	var ok bool
	var inv mat.Dense
	func() {
		defer func() {
			if r := recover(); r != nil {
				ok = false
			}
		}()
		if err := inv.Inverse(a); err == nil {
			ok = true
		}
	}()
	if ok {
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
	// Fall back to uniqueness=1 for missing entries silently — but only as a
	// last resort. A length mismatch usually means the caller threaded the
	// wrong array; log a warning so it doesn't go unnoticed.
	if len(uniquenesses) < p {
		insyra.LogWarning("stats", "inverseUniquenessDiagonal",
			"uniquenesses length %d < variables %d; padding with 1.0",
			len(uniquenesses), p)
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
	// Use the R-bit-perfect Dsyevr port: gonum's mat.EigenSym uses dsyev
	// (QL/QR) which can drift from R's dsyevr (MRRR) at ULP level on
	// ill-conditioned scoring matrices, propagating into Anderson-Rubin
	// weights via the V·Λ^{-1/2}·V^T sum.
	values, vectors, ok := fa.SymmetricEigenDescendingDsyevr(a)
	if !ok {
		return nil, fmt.Errorf("failed to decompose scoring matrix")
	}

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

