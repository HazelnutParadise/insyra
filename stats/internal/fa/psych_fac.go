// fa/psych_fac.go
package fa

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	statslinalg "github.com/HazelnutParadise/insyra/stats/internal/linalg"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
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

	validMethods := []string{"pa", "alpha", "minrank", "wls", "gls", "minres", "minchi", "uls", "ml", "ols", "old.min"}
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
	case "alpha":
		loadings, communalities, eValues = alphaFactoring(r, rMat, opts.NFactors, opts.MinErr, opts.MaxIter, opts.Warnings)
	case "minrank":
		loadings, communalities, eValues = minrankFactoring(r, rMat, opts.NFactors)
	case "minres", "uls", "wls", "gls", "ols":
		loadings, communalities, eValues, err = minimumResidualFactoring(r, rMat, opts.NFactors, fm, opts.Covar, opts.MinErr, opts.MaxIter)
	case "ml":
		loadings, communalities, eValues, err = maximumLikelihoodFactoring(r, rMat, opts.NFactors, opts.Covar, opts.MinErr, opts.MaxIter)
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
	case "alpha":
		for i := 0; i < opts.NFactors; i++ {
			colNames[i] = fmt.Sprintf("alpha%d", i+1)
		}
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
		values, vectors, ok = statslinalg.SymmetricEigenDescending(rMat)
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
		values, vectors, ok := statslinalg.SymmetricEigenDescending(rMat)
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
func minimumResidualFactoring(r, rMat *mat.Dense, nfactors int, fm string, covar bool, minErr float64, maxIter int) (*mat.Dense, []float64, []float64, error) {
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
		maxIter,
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
	values, _, ok := statslinalg.SymmetricEigenDescending(s)

	eValues := make([]float64, p)
	if ok {
		copy(eValues, values)
	}

	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		communality := 1.0 - psi[i]
		if communality < 0 {
			communality = 0
		}
		if communality > 1 {
			communality = 1
		}
		communalities[i] = communality
	}

	return loadings, communalities, eValues, nil
}

func optimizeBounded(start []float64, lower, upper float64, maxIter int, fn func([]float64) float64, grad func([]float64, []float64)) ([]float64, error) {
	if len(start) == 0 || upper <= lower || fn == nil {
		return nil, fmt.Errorf("invalid bounded optimization inputs")
	}
	if grad != nil {
		return optimizeBoundedProjected(start, lower, upper, maxIter, fn, grad)
	}
	y0 := make([]float64, len(start))
	for i, v := range start {
		y0[i] = boundedToUnbounded(v, lower, upper)
	}

	x := make([]float64, len(start))
	problem := optimize.Problem{
		Func: func(y []float64) float64 {
			unboundedToBounded(x, y, lower, upper)
			return fn(x)
		},
	}
	if grad != nil {
		gx := make([]float64, len(start))
		problem.Grad = func(gy, y []float64) {
			unboundedToBounded(x, y, lower, upper)
			for i := range gx {
				gx[i] = 0
			}
			grad(gx, x)
			for i := range gy {
				gy[i] = gx[i] * boundedDerivative(y[i], lower, upper)
			}
		}
	}

	majorIterations := maxIter
	if majorIterations < 100 {
		majorIterations = 100
	}
	settings := &optimize.Settings{MajorIterations: majorIterations}
	method := optimize.Method(&optimize.NelderMead{})
	if grad != nil {
		method = &optimize.LBFGS{
			GradStopThreshold: math.NaN(),
			Store:             15,
		}
	}

	result, err := optimize.Minimize(problem, y0, settings, method)
	if err != nil {
		return nil, err
	}
	if result == nil || result.X == nil {
		return nil, fmt.Errorf("optimizer returned no solution")
	}

	out := make([]float64, len(start))
	unboundedToBounded(out, result.X, lower, upper)
	return out, nil
}

func optimizeBoundedProjected(start []float64, lower, upper float64, maxIter int, fn func([]float64) float64, grad func([]float64, []float64)) ([]float64, error) {
	x := append([]float64(nil), start...)
	projectIntoBounds(x, lower, upper)
	fx := fn(x)
	if math.IsNaN(fx) || math.IsInf(fx, 0) {
		return nil, fmt.Errorf("invalid initial objective value: %g", fx)
	}

	g := make([]float64, len(x))
	direction := make([]float64, len(x))
	trial := make([]float64, len(x))
	const parscale = 0.01
	const c1 = 1e-4
	const projectedStepTol = 1e-4
	iterations := maxIter
	if iterations < 1000 {
		iterations = 1000
	}

	for iter := 0; iter < iterations; iter++ {
		for i := range g {
			g[i] = 0
		}
		grad(g, x)

		projectedNorm := 0.0
		directionalDerivative := 0.0
		for i := range x {
			if x[i] <= lower+1e-12 && g[i] >= 0 {
				direction[i] = 0
				continue
			}
			if x[i] >= upper-1e-12 && g[i] <= 0 {
				direction[i] = 0
				continue
			}
			direction[i] = -g[i] * parscale * parscale
			if x[i]+direction[i] < lower {
				direction[i] = lower - x[i]
			}
			if x[i]+direction[i] > upper {
				direction[i] = upper - x[i]
			}
			projectedNorm += direction[i] * direction[i]
			directionalDerivative += g[i] * direction[i]
		}

		smallProjectedStep := math.Sqrt(projectedNorm) < projectedStepTol

		step := 1.0
		accepted := false
		for ls := 0; ls < 40; ls++ {
			for i := range x {
				trial[i] = x[i] + step*direction[i]
			}
			projectIntoBounds(trial, lower, upper)
			ft := fn(trial)
			if !math.IsNaN(ft) && !math.IsInf(ft, 0) && ft <= fx+c1*step*directionalDerivative {
				copy(x, trial)
				fx = ft
				accepted = true
				break
			}
			step *= 0.5
		}
		if !accepted {
			return nil, fmt.Errorf("bounded optimizer line search failed to find a feasible descent step")
		}
		if smallProjectedStep {
			return x, nil
		}
	}

	return nil, fmt.Errorf("bounded optimizer failed to converge in %d iterations", iterations)
}

func projectIntoBounds(x []float64, lower, upper float64) {
	for i := range x {
		if x[i] < lower {
			x[i] = lower
		}
		if x[i] > upper {
			x[i] = upper
		}
	}
}

func boundedToUnbounded(x, lower, upper float64) float64 {
	const eps = 1e-4
	if x <= lower {
		x = lower + eps*(upper-lower)
	}
	if x >= upper {
		x = upper - eps*(upper-lower)
	}
	r := (x - lower) / (upper - lower)
	if r < eps {
		r = eps
	}
	if r > 1-eps {
		r = 1 - eps
	}
	return math.Log(r / (1 - r))
}

func unboundedToBounded(dst, src []float64, lower, upper float64) {
	for i, y := range src {
		if y >= 0 {
			e := math.Exp(-y)
			dst[i] = lower + (upper-lower)/(1+e)
		} else {
			e := math.Exp(y)
			dst[i] = lower + (upper-lower)*e/(1+e)
		}
	}
}

func boundedDerivative(y, lower, upper float64) float64 {
	var s float64
	if y >= 0 {
		e := math.Exp(-y)
		s = 1 / (1 + e)
	} else {
		e := math.Exp(y)
		s = e / (1 + e)
	}
	return (upper - lower) * s * (1 - s)
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
	values, vectors, ok := statslinalg.SymmetricEigenDescending(sWork)
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
		for i := 1; i < p; i++ {
			for j := 0; j < i; j++ {
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

	values, vectors, ok := statslinalg.SymmetricEigenDescending(sWork)
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
	values, vectors, ok := statslinalg.SymmetricEigenDescending(sWork)
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
func maximumLikelihoodFactoring(r, rMat *mat.Dense, nfactors int, covar bool, minErr float64, maxIter int) (*mat.Dense, []float64, []float64, error) {
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
		maxIter,
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
	psi = refineBoundedCoordinates(psi, 0.005, upper, maxIter, func(x []float64) float64 {
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
	values, _, ok := statslinalg.SymmetricEigenDescending(s)

	eValues := make([]float64, p)
	if ok {
		copy(eValues, values)
	}

	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		communalities[i] = 1.0 - psi[i]
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
	values, _, ok := statslinalg.SymmetricEigenDescending(temp)
	if !ok {
		return math.Inf(1)
	}

	// Extract eigenvalues after nf
	sum := 0.0
	for i := nf; i < p; i++ {
		eigenVal := values[i]
		sum += math.Log(eigenVal) - eigenVal
	}
	sum += -float64(nf) + float64(p)

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
	values, vectors, ok := statslinalg.SymmetricEigenDescending(temp)
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
	values, vectors, ok := statslinalg.SymmetricEigenDescending(temp)
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

// alphaFactoring implements alpha factoring (mirrors alpha method in R psych)
func alphaFactoring(r, rMat *mat.Dense, nfactors int, minErr float64, maxIter int, warnings bool) (*mat.Dense, []float64, []float64) {
	p, _ := r.Dims()

	// Get eigenvalues for initial communalities
	eValues, vectors, ok := statslinalg.SymmetricEigenDescending(r)
	if !ok {
		return mat.NewDense(p, nfactors, nil), make([]float64, p), make([]float64, p)
	}

	// Initialize communalities
	h2 := make([]float64, p)
	for i := 0; i < p; i++ {
		h2[i] = rMat.At(i, i)
	}

	// Iterative process
	i := 1
	for i <= maxIter {
		// Update r.mat
		rMatWork := mat.NewDense(p, p, nil)
		rMatWork.CloneFrom(r)
		for j := 0; j < p; j++ {
			rMatWork.Set(j, j, h2[j])
		}

		// Eigen decomposition
		values, vectorsWork, ok := statslinalg.SymmetricEigenDescending(rMatWork)
		if !ok {
			break
		}

		// Extract loadings
		loadings := mat.NewDense(p, nfactors, nil)
		if nfactors > 1 {
			for j := 0; j < nfactors; j++ {
				eigenVal := values[j]
				sqrtEigenVal := math.Sqrt(eigenVal)
				for k := 0; k < p; k++ {
					loadings.Set(k, j, vectorsWork.At(k, j)*sqrtEigenVal)
				}
			}
		} else {
			eigenVal := values[0]
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, 0, vectorsWork.At(k, 0)*sqrtEigenVal)
			}
		}

		// Compute model
		model := mat.NewDense(p, p, nil)
		model.Mul(loadings, loadings.T())

		// Update communalities
		newH2 := make([]float64, p)
		for j := 0; j < p; j++ {
			newH2[j] = h2[j] * model.At(j, j)
		}

		// Check convergence
		err := 0.0
		for j := 0; j < p; j++ {
			diff := h2[j] - newH2[j]
			err += diff * diff
		}
		err = math.Sqrt(err)

		if err < minErr {
			break
		}

		h2 = newH2
		i++

		if i > maxIter && warnings {
			insyra.LogWarning("fa", "alphaFactoring", "maximum iteration exceeded")
		}
	}

	// Final loadings with sqrt(H2) scaling
	loadings := mat.NewDense(p, nfactors, nil)
	if nfactors > 1 {
		for j := 0; j < nfactors; j++ {
			eigenVal := eValues[j]
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, j, vectors.At(k, j)*sqrtEigenVal*math.Sqrt(h2[k]))
			}
		}
	} else {
		eigenVal := eValues[0]
		sqrtEigenVal := math.Sqrt(eigenVal)
		for k := 0; k < p; k++ {
			loadings.Set(k, 0, vectors.At(k, 0)*sqrtEigenVal*math.Sqrt(h2[k]))
		}
	}

	// Extract eigenvalues
	finalEValues := make([]float64, p)
	for j := 0; j < p; j++ {
		finalEValues[j] = eValues[j]
	}

	// Communalities
	communalities := make([]float64, p)
	copy(communalities, h2)

	return loadings, communalities, finalEValues
}

// minrankFactoring implements minimum rank factoring (mirrors minrank method in R psych)
func minrankFactoring(r, rMat *mat.Dense, nfactors int) (*mat.Dense, []float64, []float64) {
	p, _ := r.Dims()

	// For simplicity, implement a basic version using eigenvalue decomposition
	// In R psych, this uses glb.algebraic for more sophisticated minimum rank approximation

	// Get communality estimate
	comGlb := 0.0
	for i := 0; i < p; i++ {
		comGlb += rMat.At(i, i)
	}
	comGlb /= float64(p)

	// Set diagonal to 1 - communality estimate
	rWork := mat.NewDense(p, p, nil)
	rWork.CloneFrom(r)
	for i := 0; i < p; i++ {
		rWork.Set(i, i, 1.0-comGlb)
	}

	// Eigen decomposition
	values, vectors, ok := statslinalg.SymmetricEigenDescending(rWork)
	if !ok {
		return mat.NewDense(p, nfactors, nil), make([]float64, p), make([]float64, p)
	}

	// Extract loadings
	loadings := mat.NewDense(p, nfactors, nil)
	for j := 0; j < nfactors; j++ {
		eigenVal := values[j]
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, vectors.At(i, j)*sqrtEigenVal)
		}
	}

	// Extract eigenvalues from original matrix
	eValues, _, ok := statslinalg.SymmetricEigenDescending(r)

	finalEValues := make([]float64, p)
	if ok {
		copy(finalEValues, eValues)
	}

	// Communalities
	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		communalities[i] = comGlb
	}

	return loadings, communalities, finalEValues
}
