// fa/psych_fac.go
package fa

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

const (
	machineEpsilon         = 2.220446e-16
	eigenvalueMinThreshold = 100 * machineEpsilon
)

// FacOptions represents options for the Fac function
type FacOptions struct {
	NFactors      int
	NObs          float64
	Rotate        string
	Scores        string
	Residuals     bool
	SMC           interface{} // bool or []float64
	Covar         bool
	Missing       bool
	Impute        string
	MinErr        float64
	MaxIter       int
	Symmetric     bool
	Warnings      bool
	Fm            string
	Alpha         float64
	ObliqueScores bool
	NpObs         *mat.Dense
	Use           string
	Cor           string
	Correct       float64
	Weight        []float64
	NRotations    int
	Hyper         float64
	Smooth        bool

	// L-BFGS-B parameters used by ML / MINRES extraction. Defaults match
	// R's stats::optim: OptimFactr=1e7, OptimMaxIter=100. Caller can tighten
	// OptimFactr (down to 1) for machine-precision convergence on flat
	// objective surfaces (e.g. Heywood-prone problems where the default
	// terminates prematurely).
	OptimFactr   float64
	OptimMaxIter int
}

// FacResult represents the result of factor analysis
type FacResult struct {
	Values        []float64
	Rotation      string
	NObs          float64
	NpObs         *mat.Dense
	Communality   []float64
	Loadings      *mat.Dense
	Fit           float64
	Residual      *mat.Dense
	R             *mat.Dense
	Communalities []float64
	Uniquenesses  []float64
	EValues       []float64
	Model         *mat.Dense
	Fm            string
	RotMat        *mat.Dense
	Phi           *mat.Dense
	Structure     *mat.Dense
	Method        string
	Scores        *mat.Dense
	R2Scores      []float64
	Weights       *mat.Dense
	Factors       int
	Hyperplane    []float64
	Vaccounted    *mat.Dense
	ECV           []float64
}

// Fac performs factor analysis using various methods (mirrors psych::fac)
func Fac(r *mat.Dense, opts *FacOptions) (*FacResult, error) {
	if opts == nil {
		opts = &FacOptions{
			NFactors:      1,
			NObs:          -999,
			Rotate:        "oblimin",
			Scores:        "tenBerge",
			Residuals:     false,
			SMC:           true,
			Covar:         false,
			Missing:       false,
			Impute:        "median",
			MinErr:        0.001,
			MaxIter:       50,
			Symmetric:     true,
			Warnings:      true,
			Fm:            "minres",
			Alpha:         0.1,
			ObliqueScores: false,
			Use:           "pairwise",
			Cor:           "cor",
			Correct:       0.5,
			NRotations:    1,
			Hyper:         0.15,
			Smooth:        true,
		}
	}

	p, q := r.Dims()
	if p != q {
		return nil, fmt.Errorf("input matrix must be square")
	}

	// Normalize factor method
	fm := opts.Fm
	switch fm {
	case "mle", "MLE", "ML":
		fm = "ml"
	}

	validMethods := []string{"pa", "wls", "gls", "minres", "minchi", "uls", "ml", "ols", "old.min"}
	isValid := false
	for _, method := range validMethods {
		if fm == method {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("unsupported factor method: %s", fm)
	}

	// Handle matrix input
	rMat := mat.NewDense(p, p, nil)
	rMat.CloneFrom(r)

	// Handle SMC initialization
	var smcVec []float64
	if smcBool, ok := opts.SMC.(bool); ok {
		if smcBool {
			if opts.NFactors <= p {
				smcResult, smcDiagnostics := Smc(rMat, &SmcOptions{Covar: opts.Covar})
				if smcResult == nil {
					return nil, fmt.Errorf("failed to compute SMC communalities")
				}
				if errs, ok := smcDiagnostics["errors"].([]string); ok && len(errs) > 0 {
					return nil, fmt.Errorf("failed to compute SMC communalities: %v", errs)
				}
				smcVec = make([]float64, p)
				for i := 0; i < p; i++ {
					smcVec[i] = smcResult.AtVec(i)
				}
			} else {
				return nil, fmt.Errorf("number of factors (%d) cannot exceed number of variables (%d)", opts.NFactors, p)
			}
		} else {
			smcVec = make([]float64, p)
			for i := range smcVec {
				smcVec[i] = 1.0
			}
		}
	} else if smcSlice, ok := opts.SMC.([]float64); ok {
		smcVec = make([]float64, len(smcSlice))
		copy(smcVec, smcSlice)
	} else {
		smcVec = make([]float64, p)
		for i := range smcVec {
			smcVec[i] = 1.0
		}
	}

	// Set diagonal of r.mat to SMC values
	for i := 0; i < p; i++ {
		rMat.Set(i, i, smcVec[i])
	}

	// Perform factor extraction based on method
	var loadings *mat.Dense
	var communalities []float64
	var eValues []float64
	var err error

	switch fm {
	case "pa":
		loadings, communalities, eValues = principalAxisFactoring(r, rMat, opts.NFactors, opts.MinErr, opts.MaxIter, opts.Warnings)
	case "minres", "uls", "wls", "gls", "ols":
		loadings, communalities, eValues, err = minimumResidualFactoring(r, rMat, opts.NFactors, fm, opts.Covar, opts.MinErr, opts.MaxIter, opts.OptimFactr, opts.OptimMaxIter)
	case "ml":
		loadings, communalities, eValues, err = maximumLikelihoodFactoring(r, rMat, opts.NFactors, opts.Covar, opts.MinErr, opts.MaxIter, opts.OptimFactr, opts.OptimMaxIter)
	default:
		return nil, fmt.Errorf("unsupported factor method: %s", fm)
	}
	if err != nil {
		return nil, err
	}

	// Handle sign convention
	if opts.NFactors > 1 {
		signTot := make([]float64, opts.NFactors)
		for j := 0; j < opts.NFactors; j++ {
			sum := 0.0
			for i := 0; i < p; i++ {
				sum += loadings.At(i, j)
			}
			if sum < 0 {
				signTot[j] = -1
			} else {
				signTot[j] = 1
			}
		}
		for j := 0; j < opts.NFactors; j++ {
			for i := 0; i < p; i++ {
				loadings.Set(i, j, loadings.At(i, j)*signTot[j])
			}
		}
	} else {
		sum := 0.0
		for i := 0; i < p; i++ {
			sum += loadings.At(i, 0)
		}
		if sum < 0 {
			for i := 0; i < p; i++ {
				loadings.Set(i, 0, -loadings.At(i, 0))
			}
		}
	}

	// Set column names
	colNames := make([]string, opts.NFactors)
	switch fm {
	case "wls":
		for i := 0; i < opts.NFactors; i++ {
			colNames[i] = fmt.Sprintf("WLS%d", i+1)
		}
	case "pa":
		for i := 0; i < opts.NFactors; i++ {
			colNames[i] = fmt.Sprintf("PA%d", i+1)
		}
	case "gls":
		for i := 0; i < opts.NFactors; i++ {
			colNames[i] = fmt.Sprintf("GLS%d", i+1)
		}
	case "ml":
		for i := 0; i < opts.NFactors; i++ {
			colNames[i] = fmt.Sprintf("ML%d", i+1)
		}
	case "minres":
		for i := 0; i < opts.NFactors; i++ {
			colNames[i] = fmt.Sprintf("MR%d", i+1)
		}
	}

	// Replace zeros with small values
	for i := 0; i < p; i++ {
		for j := 0; j < opts.NFactors; j++ {
			if loadings.At(i, j) == 0 {
				loadings.Set(i, j, 1e-15)
			}
		}
	}

	// Compute model
	model := mat.NewDense(p, p, nil)
	model.Mul(loadings, loadings.T())

	// Initialize result
	result := &FacResult{
		Values:        make([]float64, p),
		Rotation:      opts.Rotate,
		NObs:          opts.NObs,
		NpObs:         opts.NpObs,
		Communality:   make([]float64, p),
		Loadings:      loadings,
		Fit:           0,
		Communalities: communalities,
		Uniquenesses:  make([]float64, p),
		EValues:       eValues,
		Model:         model,
		Fm:            fm,
		Method:        opts.Scores,
		Factors:       opts.NFactors,
		R:             r,
	}

	// Compute communality and uniquenesses
	for i := 0; i < p; i++ {
		result.Communality[i] = model.At(i, i)
		result.Uniquenesses[i] = r.At(i, i) - model.At(i, i)
	}

	// Check for ultra-Heywood case
	if !opts.Covar {
		for i := 0; i < p; i++ {
			if result.Communality[i] > 1.0 {
				insyra.LogWarning("fa", "Fac", "An ultra-Heywood case was detected. Examine the results carefully")
				break
			}
		}
	}

	return result, nil
}

// principalAxisFactoring implements Principal Axis Factoring
func principalAxisFactoring(r, rMat *mat.Dense, nfactors int, minErr float64, maxIter int, warnings bool) (*mat.Dense, []float64, []float64) {
	p, _ := r.Dims()
	comm := 0.0
	for i := 0; i < p; i++ {
		comm += rMat.At(i, i)
	}
	err := comm
	i := 1
	var values []float64
	var loadings *mat.Dense

	for err > minErr && i <= maxIter {
		// Eigen decomposition. R's eigen() returns symmetric eigenpairs in
		// descending order; Gonum's EigenSym returns ascending values.
		var vectors *mat.Dense
		var ok bool
		values, vectors, ok = symmetricEigenDescendingDsyevr(rMat)
		if !ok {
			break
		}

		// Extract loadings
		loadings = mat.NewDense(p, nfactors, nil)
		if nfactors > 1 {
			for j := 0; j < nfactors; j++ {
				eigenVal := values[j]
				if eigenVal < 0 {
					eigenVal = 0
				}
				sqrtEigenVal := math.Sqrt(eigenVal)
				for k := 0; k < p; k++ {
					loadings.Set(k, j, vectors.At(k, j)*sqrtEigenVal)
				}
			}
		} else {
			eigenVal := values[0]
			if eigenVal < 0 {
				eigenVal = 0
			}
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, 0, vectors.At(k, 0)*sqrtEigenVal)
			}
		}

		// Compute model
		model := mat.NewDense(p, p, nil)
		model.Mul(loadings, loadings.T())

		// Update communalities
		newComm := 0.0
		for k := 0; k < p; k++ {
			newVal := model.At(k, k)
			rMat.Set(k, k, newVal)
			newComm += newVal
		}

		err = math.Abs(comm - newComm)
		if math.IsNaN(err) {
			if warnings {
				insyra.LogWarning("fa", "principalAxisFactoring", "imaginary eigenvalue condition encountered in PAF; try again with lower communality estimates or SMC=FALSE")
			}
			break
		}
		comm = newComm
		i++
	}

	if i > maxIter && warnings {
		insyra.LogWarning("fa", "principalAxisFactoring", "maximum iteration exceeded")
	}

	if loadings == nil || values == nil {
		values, vectors, ok := symmetricEigenDescendingDsyevr(rMat)
		if !ok {
			return mat.NewDense(p, nfactors, nil), make([]float64, p), make([]float64, p)
		}
		loadings = mat.NewDense(p, nfactors, nil)
		for j := 0; j < nfactors; j++ {
			eigenVal := values[j]
			if eigenVal < 0 {
				eigenVal = 0
			}
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, j, vectors.At(k, j)*sqrtEigenVal)
			}
		}
	}

	// Extract eigenvalues
	eValues := make([]float64, p)
	for j := 0; j < p; j++ {
		eValues[j] = values[j]
	}

	// Extract communalities
	communalities := make([]float64, p)
	for j := 0; j < p; j++ {
		communalities[j] = rMat.At(j, j)
	}

	return loadings, communalities, eValues
}

// minimumResidualFactoring implements minimum residual factoring and related methods
func minimumResidualFactoring(r, rMat *mat.Dense, nfactors int, fm string, covar bool, minErr float64, maxIter int, optimFactr float64, optimMaxIter int) (*mat.Dense, []float64, []float64, error) {
	p, _ := r.Dims()

	// Get SMC values for initial communalities
	smcResult, smcDiagnostics := Smc(r, &SmcOptions{Covar: covar})
	if smcResult == nil {
		return nil, nil, nil, fmt.Errorf("failed to compute SMC communalities")
	}
	if errs, ok := smcDiagnostics["errors"].([]string); ok && len(errs) > 0 {
		return nil, nil, nil, fmt.Errorf("failed to compute SMC communalities: %v", errs)
	}
	smcVec := make([]float64, p)
	for i := 0; i < p; i++ {
		smcVec[i] = smcResult.AtVec(i)
	}

	// Set upper bound. R psych: upper <- max(S.smc, 1).
	upper := 1.0
	for _, v := range smcVec {
		if v > upper {
			upper = v
		}
	}

	// Initial uniquenesses. R's minres/uls family optimizes one uniqueness
	// parameter per variable, not one parameter per retained factor.
	start := make([]float64, p)
	for i := range start {
		start[i] = 1.0 - smcVec[i]
		if start[i] < 0.005 {
			start[i] = 0.005
		}
		if start[i] > 1.0 {
			start[i] = 1.0
		}
	}

	psi, err := optimizeBounded(
		start,
		0.005,
		upper,
		optimFactr,
		optimMaxIter,
		func(x []float64) float64 {
			return fitResiduals(x, r, nfactors, fm)
		},
		func(grad, x []float64) {
			faGrMinres(grad, x, r, nfactors)
		},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("minimum residual optimization failed: %w", err)
	}

	// Extract loadings
	loadings := faOutWLS(psi, r, nfactors)

	// Compute eigenvalues
	s := mat.NewDense(p, p, nil)
	s.CloneFrom(r)
	for i := 0; i < p; i++ {
		communality := 0.0
		for j := 0; j < nfactors; j++ {
			v := loadings.At(i, j)
			communality += v * v
		}
		s.Set(i, i, communality)
	}
	values, _, ok := symmetricEigenDescendingDsyevr(s)

	eValues := make([]float64, p)
	if ok {
		copy(eValues, values)
	}

	// Communalities = sum(loadings_i^2) per row — matches R's psych::fa,
	// which uses diag(L %*% t(L)) NOT 1 - psi. The two diverge at the
	// bounded-optim lower clamp.
	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		s := 0.0
		for j := 0; j < nfactors; j++ {
			v := loadings.At(i, j)
			s += v * v
		}
		communalities[i] = s
	}

	return loadings, communalities, eValues, nil
}

// optimizeBounded minimizes fn(x) subject to lower <= x_i <= upper. The
// caller must supply a gradient routine. We dispatch to the L-BFGS-B
// implementation in lbfgsb.go with parameters set to mirror R's
// optim(method="L-BFGS-B", control=list(parscale=rep(0.01,n))) used by
// psych::fa for ML / minres factor extraction.
//
// factr controls the convergence tolerance (multiplier of machine epsilon);
// 1e7 matches R's default. maxIter caps the number of iterations; 100
// matches R's default.
func optimizeBounded(start []float64, lower, upper float64, factr float64, maxIter int,
	fn func([]float64) float64, grad func([]float64, []float64)) ([]float64, error) {
	if len(start) == 0 || upper <= lower || fn == nil || grad == nil {
		return nil, fmt.Errorf("invalid bounded optimization inputs")
	}
	if factr <= 0 {
		factr = 1e7
	}
	if maxIter <= 0 {
		maxIter = 100
	}
	n := len(start)
	lo := make([]float64, n)
	hi := make([]float64, n)
	parscale := make([]float64, n)
	for i := range start {
		lo[i] = lower
		hi[i] = upper
		parscale[i] = 0.01
	}
	gradWrap := func(g, x []float64) {
		for i := range g {
			g[i] = 0
		}
		grad(g, x)
	}
	res, err := lbfgsb(start, lo, hi, fn, gradWrap, lbfgsbParams{
		M: 5, Factr: factr, PgTol: 0, MaxIter: maxIter, Parscale: parscale,
	})
	if err != nil {
		return nil, fmt.Errorf("L-BFGS-B failed: %w", err)
	}
	return res.X, nil
}

func refineBoundedCoordinates(start []float64, lower, upper float64, maxIter int, fn func([]float64) float64) []float64 {
	if len(start) == 0 || fn == nil || upper <= lower {
		return nil
	}
	x := append([]float64(nil), start...)
	best := fn(x)
	iterations := maxIter
	if iterations < 100 {
		iterations = 100
	}
	for iter := 0; iter < iterations; iter++ {
		improved := false
		for idx := range x {
			current := x[idx]
			candidate, candidateValue := boundedCoordinateMinimize(x, idx, lower, upper, fn)
			if candidateValue+1e-12 < best {
				x[idx] = candidate
				best = candidateValue
				improved = true
			} else {
				x[idx] = current
			}
		}
		if !improved {
			break
		}
	}
	return x
}

func boundedCoordinateMinimize(x []float64, idx int, lower, upper float64, fn func([]float64) float64) (float64, float64) {
	const gr = 0.6180339887498949
	orig := x[idx]
	bestX := orig
	bestValue := fn(x)

	eval := func(v float64) float64 {
		x[idx] = v
		return fn(x)
	}

	lowerValue := eval(lower)
	if lowerValue < bestValue {
		bestValue = lowerValue
		bestX = lower
	}
	upperValue := eval(upper)
	if upperValue < bestValue {
		bestValue = upperValue
		bestX = upper
	}

	a, b := lower, upper
	c := b - gr*(b-a)
	d := a + gr*(b-a)
	fc := eval(c)
	fd := eval(d)
	for iter := 0; iter < 80 && math.Abs(b-a) > 1e-12; iter++ {
		if fc < fd {
			b = d
			d = c
			fd = fc
			c = b - gr*(b-a)
			fc = eval(c)
		} else {
			a = c
			c = d
			fc = fd
			d = a + gr*(b-a)
			fd = eval(d)
		}
	}
	mid := 0.5 * (a + b)
	midValue := eval(mid)
	if midValue < bestValue {
		bestValue = midValue
		bestX = mid
	}
	x[idx] = orig
	return bestX, bestValue
}

// fitResiduals computes the fit residuals (mirrors fit.residuals in R)
func fitResiduals(psi []float64, s *mat.Dense, nf int, fm string) float64 {
	p, _ := s.Dims()

	// Set diagonal
	sWork := mat.NewDense(p, p, nil)
	sWork.CloneFrom(s)
	for i := 0; i < p; i++ {
		sWork.Set(i, i, 1-psi[i])
	}

	// Eigen decomposition
	values, vectors, ok := symmetricEigenDescendingDsyevr(sWork)
	if !ok {
		return math.Inf(1)
	}

	// Extract loadings
	loadings := mat.NewDense(p, nf, nil)
	for j := 0; j < nf; j++ {
		eigenVal := values[j]
		if eigenVal < machineEpsilon {
			eigenVal = eigenvalueMinThreshold
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, vectors.At(i, j)*sqrtEigenVal)
		}
	}

	// Compute model
	model := mat.NewDense(p, p, nil)
	model.Mul(loadings, loadings.T())

	// Compute residuals
	residual := mat.NewDense(p, p, nil)
	residual.Sub(s, model)

	// Apply method-specific residual computation
	var error float64
	switch fm {
	case "uls":
		// R: residual <- (S - model)^2
		for i := 0; i < p; i++ {
			for j := 0; j < p; j++ {
				val := residual.At(i, j)
				error += val * val
			}
		}
	case "ols", "minres", "old.min":
		// R: residual <- residual[lower.tri(residual)]^2
		// lower.tri() returns elements in COLUMN-MAJOR order:
		// (1,0), (2,0), (3,0), ..., (p-1, 0), (2,1), (3,1), ...
		// Match this accumulation order so the floating-point sum is bit-
		// identical to R's; row-major would round differently and that 1-ulp
		// difference cascades through L-BFGS-B convergence checks.
		for j := 0; j < p-1; j++ {
			for i := j + 1; i < p; i++ {
				val := residual.At(i, j)
				error += val * val
			}
		}
	}

	return error
}

// faGrMinres computes psych::fac's FAgr.minres gradient.
func faGrMinres(grad []float64, psi []float64, s *mat.Dense, nf int) {
	p, _ := s.Dims()
	sWork := mat.NewDense(p, p, nil)
	sWork.CloneFrom(s)
	for i := 0; i < p; i++ {
		sWork.Set(i, i, sWork.At(i, i)-psi[i])
	}

	values, vectors, ok := symmetricEigenDescendingDsyevr(sWork)
	if !ok {
		for i := range grad {
			grad[i] = 0
		}
		return
	}

	loadings := mat.NewDense(p, nf, nil)
	for j := 0; j < nf; j++ {
		eigenVal := values[j]
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, vectors.At(i, j)*sqrtEigenVal)
		}
	}

	model := mat.NewDense(p, p, nil)
	model.Mul(loadings, loadings.T())
	for i := 0; i < p; i++ {
		grad[i] = model.At(i, i) + psi[i] - s.At(i, i)
	}
}

// faOutWLS extracts loadings from parameters (mirrors FAout.wls in R)
func faOutWLS(psi []float64, s *mat.Dense, q int) *mat.Dense {
	p, _ := s.Dims()

	// Set diagonal
	sWork := mat.NewDense(p, p, nil)
	sWork.CloneFrom(s)
	for i := 0; i < p; i++ {
		sWork.Set(i, i, sWork.At(i, i)-psi[i])
	}

	// Eigen decomposition
	values, vectors, ok := symmetricEigenDescendingDsyevr(sWork)
	if !ok {
		return mat.NewDense(p, q, nil)
	}

	// Extract loadings
	loadings := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		eigenVal := values[j]
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, vectors.At(i, j)*sqrtEigenVal)
		}
	}

	return loadings
}

// maximumLikelihoodFactoring implements Maximum Likelihood factor analysis
func maximumLikelihoodFactoring(r, rMat *mat.Dense, nfactors int, covar bool, minErr float64, maxIter int, optimFactr float64, optimMaxIter int) (*mat.Dense, []float64, []float64, error) {
	p, _ := r.Dims()

	// Get SMC values for initial communalities
	smcResult, smcDiagnostics := Smc(r, &SmcOptions{Covar: covar})
	if smcResult == nil {
		return nil, nil, nil, fmt.Errorf("failed to compute SMC communalities")
	}
	if errs, ok := smcDiagnostics["errors"].([]string); ok && len(errs) > 0 {
		return nil, nil, nil, fmt.Errorf("failed to compute SMC communalities: %v", errs)
	}
	smcVec := make([]float64, p)
	for i := 0; i < p; i++ {
		smcVec[i] = smcResult.AtVec(i)
	}

	// Set upper bound. R psych: upper <- max(S.smc, 1).
	upper := 1.0
	for _, v := range smcVec {
		if v > upper {
			upper = v
		}
	}

	// Initial parameters
	start := make([]float64, p)
	for i := 0; i < p; i++ {
		start[i] = r.At(i, i) - smcVec[i]
		if start[i] < 0.005 {
			start[i] = 0.005
		}
		if start[i] > upper {
			start[i] = upper
		}
	}

	psi, err := optimizeBounded(
		start,
		0.005,
		upper,
		optimFactr,
		optimMaxIter,
		func(x []float64) float64 {
			return faFn(x, r, nfactors)
		},
		func(grad, x []float64) {
			faGr(grad, x, r, nfactors)
		},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("maximum likelihood optimization failed: %w", err)
	}
	psi = refineBoundedCoordinates(psi, 0.005, upper, optimMaxIter, func(x []float64) float64 {
		return faFn(x, r, nfactors)
	})

	// Extract loadings
	loadings := faOut(psi, r, nfactors)

	// Compute eigenvalues
	s := mat.NewDense(p, p, nil)
	s.CloneFrom(r)
	for i := 0; i < p; i++ {
		communality := 0.0
		for j := 0; j < nfactors; j++ {
			v := loadings.At(i, j)
			communality += v * v
		}
		s.Set(i, i, communality)
	}
	values, _, ok := symmetricEigenDescendingDsyevr(s)

	eValues := make([]float64, p)
	if ok {
		copy(eValues, values)
	}

	// Communalities = sum(loadings_i^2) — matches R's psych::fa, which uses
	// diag(Lambda %*% t(Lambda)) NOT 1 - psi. The two diverge when psi hits
	// the bounded-optim lower clamp (0.005), so the bug only shows up on
	// near-singular correlation matrices but is the root cause of ~10% of
	// our R-parity failures (e.g. generated_near_collinear / ml).
	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		s := 0.0
		for j := 0; j < nfactors; j++ {
			v := loadings.At(i, j)
			s += v * v
		}
		communalities[i] = s
	}

	return loadings, communalities, eValues, nil
}

// faFn computes the maximum likelihood objective function (mirrors FAfn in R)
func faFn(psi []float64, s *mat.Dense, nf int) float64 {
	p, _ := s.Dims()

	// Create scaling matrix
	sc := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		sc.Set(i, i, 1.0/math.Sqrt(psi[i]))
	}

	// Compute Sstar = sc %*% S %*% sc
	sStar := mat.NewDense(p, p, nil)
	sStar.Mul(sc, s)
	temp := mat.NewDense(p, p, nil)
	temp.Mul(sStar, sc)

	// Eigen decomposition
	values, _, ok := symmetricEigenDescendingDsyevr(temp)
	if !ok {
		return math.Inf(1)
	}

	// Extract eigenvalues after nf
	// R: sum = sum + log(eigenVal) - eigenVal (left-fold). Replicate exact
	// accumulation order — `x += a - b` would compile as x + (a-b) which
	// differs at ULP level from ((x + a) - b).
	sum := 0.0
	for i := nf; i < p; i++ {
		eigenVal := values[i]
		sum += math.Log(eigenVal)
		sum -= eigenVal
	}
	sum -= float64(nf)
	sum += float64(p)

	return -sum
}

// faGr computes the gradient for maximum likelihood (mirrors FAgr in R)
func faGr(grad []float64, psi []float64, s *mat.Dense, nf int) []float64 {
	p, _ := s.Dims()

	// Create scaling matrix
	sc := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		sc.Set(i, i, 1.0/math.Sqrt(psi[i]))
	}

	// Compute Sstar = sc %*% S %*% sc
	sStar := mat.NewDense(p, p, nil)
	sStar.Mul(sc, s)
	temp := mat.NewDense(p, p, nil)
	temp.Mul(sStar, sc)

	// Eigen decomposition
	values, vectors, ok := symmetricEigenDescendingDsyevr(temp)
	if !ok {
		for i := range grad {
			grad[i] = 0
		}
		return grad
	}

	// Extract first nf eigenvectors and eigenvalues
	l := mat.NewDense(p, nf, nil)
	for j := 0; j < nf; j++ {
		eigenVal := values[j]
		if eigenVal < 1.0 {
			eigenVal = 1.0
		}
		sqrtEigenVal := math.Sqrt(eigenVal - 1.0)
		for i := 0; i < p; i++ {
			l.Set(i, j, vectors.At(i, j)*sqrtEigenVal)
		}
	}

	// Compute load = diag(sqrt(Psi)) %*% L
	load := mat.NewDense(p, nf, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < nf; j++ {
			load.Set(i, j, math.Sqrt(psi[i])*l.At(i, j))
		}
	}

	// Compute model = load %*% t(load) + diag(Psi)
	model := mat.NewDense(p, p, nil)
	model.Mul(load, load.T())
	for i := 0; i < p; i++ {
		model.Set(i, i, model.At(i, i)+psi[i])
	}

	// Compute g = model - S
	g := mat.NewDense(p, p, nil)
	g.Sub(model, s)

	// Compute gradient: diag(g)/Psi^2
	for i := 0; i < p; i++ {
		grad[i] = g.At(i, i) / (psi[i] * psi[i])
	}

	return grad
}

// faOut extracts loadings from parameters for ML (mirrors FAout in R)
func faOut(psi []float64, s *mat.Dense, q int) *mat.Dense {
	p, _ := s.Dims()

	// Create scaling matrix
	sc := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		sc.Set(i, i, 1.0/math.Sqrt(psi[i]))
	}

	// Compute Sstar = sc %*% S %*% sc
	sStar := mat.NewDense(p, p, nil)
	sStar.Mul(sc, s)
	temp := mat.NewDense(p, p, nil)
	temp.Mul(sStar, sc)

	// Eigen decomposition
	values, vectors, ok := symmetricEigenDescendingDsyevr(temp)
	if !ok {
		return mat.NewDense(p, q, nil)
	}

	// Extract first q eigenvectors and eigenvalues
	l := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		eigenVal := values[j]
		if eigenVal < 1.0 {
			eigenVal = 1.0
		}
		sqrtEigenVal := math.Sqrt(eigenVal - 1.0)
		for i := 0; i < p; i++ {
			l.Set(i, j, vectors.At(i, j)*sqrtEigenVal)
		}
	}

	// Compute load = diag(sqrt(Psi)) %*% L
	loadings := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			loadings.Set(i, j, math.Sqrt(psi[i])*l.At(i, j))
		}
	}

	return loadings
}

