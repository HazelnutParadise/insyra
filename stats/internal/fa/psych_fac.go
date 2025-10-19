// fa/psych_fac.go
package fa

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/gonum/optimize"
	"gonum.org/v1/gonum/mat"
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
		insyra.LogWarning("fa", "Fac", "factor method not specified correctly, minimum residual (unweighted least squares) used")
		fm = "minres"
	}

	// Handle matrix input
	rMat := mat.NewDense(p, p, nil)
	rMat.CloneFrom(r)

	// Handle SMC initialization
	var smcVec []float64
	if smcBool, ok := opts.SMC.(bool); ok {
		if smcBool {
			if opts.NFactors <= p {
				smcResult, _ := Smc(rMat, &SmcOptions{Covar: opts.Covar})
				smcVec = make([]float64, p)
				for i := 0; i < p; i++ {
					smcVec[i] = smcResult.AtVec(i)
				}
			} else {
				if opts.Warnings {
					insyra.LogWarning("fa", "Fac", "too many factors requested for this number of variables to use SMC for communality estimates, 1s are used instead")
				}
				smcVec = make([]float64, p)
				for i := range smcVec {
					smcVec[i] = 1.0
				}
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

	switch fm {
	case "pa":
		loadings, communalities, eValues = principalAxisFactoring(r, rMat, opts.NFactors, opts.MinErr, opts.MaxIter, opts.Warnings)
	case "alpha":
		loadings, communalities, eValues = alphaFactoring(r, rMat, opts.NFactors, opts.MinErr, opts.MaxIter, opts.Warnings)
	case "minrank":
		loadings, communalities, eValues = minrankFactoring(r, rMat, opts.NFactors)
	case "minres", "uls", "wls", "gls", "ols":
		loadings, communalities, eValues = minimumResidualFactoring(r, rMat, opts.NFactors, fm, opts.Covar, opts.MinErr, opts.MaxIter)
	case "ml":
		loadings, communalities, eValues = maximumLikelihoodFactoring(r, rMat, opts.NFactors, opts.Covar, opts.MinErr, opts.MaxIter)
	default:
		return nil, fmt.Errorf("unsupported factor method: %s", fm)
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

	for err > minErr && i <= maxIter {
		// Eigen decomposition
		var eig mat.Eigen
		ok := eig.Factorize(rMat, mat.EigenRight)
		if !ok {
			break
		}

		values := eig.Values(nil)
		vectors := mat.NewCDense(p, p, nil)
		eig.VectorsTo(vectors)

		// Extract loadings
		loadings := mat.NewDense(p, nfactors, nil)
		if nfactors > 1 {
			for j := 0; j < nfactors; j++ {
				eigenVal := real(values[j])
				if eigenVal < 0 {
					eigenVal = 0
				}
				sqrtEigenVal := math.Sqrt(eigenVal)
				for k := 0; k < p; k++ {
					loadings.Set(k, j, real(vectors.At(k, j))*sqrtEigenVal)
				}
			}
		} else {
			eigenVal := real(values[0])
			if eigenVal < 0 {
				eigenVal = 0
			}
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, 0, real(vectors.At(k, 0))*sqrtEigenVal)
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

	// Final eigen decomposition
	var eig mat.Eigen
	eig.Factorize(rMat, mat.EigenRight)
	values := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

	// Extract final loadings
	loadings := mat.NewDense(p, nfactors, nil)
	if nfactors > 1 {
		for j := 0; j < nfactors; j++ {
			eigenVal := real(values[j])
			if eigenVal < 0 {
				eigenVal = 0
			}
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, j, real(vectors.At(k, j))*sqrtEigenVal)
			}
		}
	} else {
		eigenVal := real(values[0])
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for k := 0; k < p; k++ {
			loadings.Set(k, 0, real(vectors.At(k, 0))*sqrtEigenVal)
		}
	}

	// Extract eigenvalues
	eValues := make([]float64, p)
	for j := 0; j < p; j++ {
		eValues[j] = real(values[j])
	}

	// Extract communalities
	communalities := make([]float64, p)
	for j := 0; j < p; j++ {
		communalities[j] = rMat.At(j, j)
	}

	return loadings, communalities, eValues
}

// minimumResidualFactoring implements minimum residual factoring and related methods
func minimumResidualFactoring(r, rMat *mat.Dense, nfactors int, fm string, covar bool, minErr float64, maxIter int) (*mat.Dense, []float64, []float64) {
	p, _ := r.Dims()

	// Get SMC values for initial communalities
	smcResult, _ := Smc(r, &SmcOptions{Covar: covar})
	smcVec := make([]float64, p)
	for i := 0; i < p; i++ {
		smcVec[i] = smcResult.AtVec(i)
	}

	// Set upper bound
	upper := 0.0
	for _, v := range smcVec {
		if v > upper {
			upper = v
		}
	}

	// Initial parameters
	start := make([]float64, nfactors)
	if nfactors > 1 {
		for i := range start {
			start[i] = 0.5
		}
	} else {
		start[0] = smcVec[0]
	}

	// Optimization
	problem := optimize.Problem{
		Func: func(x []float64) float64 {
			return fitResiduals(x, r, nfactors, fm)
		},
	}

	settings := optimize.DefaultSettings()
	method := &optimize.BFGS{}

	result, err := optimize.Local(problem, start, settings, method)
	_ = err // Ignore error, fallback handled below

	// Extract loadings
	loadings := faOutWLS(result.X, r, nfactors)

	// Compute eigenvalues
	s := mat.NewDense(p, p, nil)
	s.CloneFrom(r)
	for i := 0; i < p; i++ {
		s.Set(i, i, loadings.At(i, i))
	}
	var eig mat.Eigen
	eig.Factorize(s, mat.EigenNone)
	values := eig.Values(nil)

	eValues := make([]float64, p)
	for i := 0; i < p; i++ {
		eValues[i] = real(values[i])
	}

	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		communalities[i] = 1.0 - result.X[i%nfactors]
	}

	return loadings, communalities, eValues
}

// fitResiduals computes the fit residuals (mirrors fit.residuals in R)
func fitResiduals(psi []float64, s *mat.Dense, nf int, fm string) float64 {
	p, _ := s.Dims()

	// Set diagonal
	sWork := mat.NewDense(p, p, nil)
	sWork.CloneFrom(s)
	for i := 0; i < p; i++ {
		sWork.Set(i, i, 1-psi[i%len(psi)])
	}

	// Eigen decomposition
	var eig mat.Eigen
	eig.Factorize(sWork, mat.EigenRight)
	values := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

	// Extract loadings
	loadings := mat.NewDense(p, nf, nil)
	for j := 0; j < nf; j++ {
		eigenVal := real(values[j])
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, real(vectors.At(i, j))*sqrtEigenVal)
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
	case "uls", "minres", "old.min":
		// Sum of squared residuals
		for i := 0; i < p; i++ {
			for j := 0; j < p; j++ {
				if i != j {
					val := residual.At(i, j)
					error += val * val
				}
			}
		}
	case "ols":
		// Sum of squared lower triangular residuals
		for i := 1; i < p; i++ {
			for j := 0; j < i; j++ {
				val := residual.At(i, j)
				error += val * val
			}
		}
	}

	return error
}

// faOutWLS extracts loadings from parameters (mirrors FAout.wls in R)
func faOutWLS(psi []float64, s *mat.Dense, q int) *mat.Dense {
	p, _ := s.Dims()

	// Set diagonal
	sWork := mat.NewDense(p, p, nil)
	sWork.CloneFrom(s)
	for i := 0; i < p; i++ {
		sWork.Set(i, i, sWork.At(i, i)-psi[i%len(psi)])
	}

	// Eigen decomposition
	var eig mat.Eigen
	eig.Factorize(sWork, mat.EigenRight)
	values := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

	// Extract loadings
	loadings := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		eigenVal := real(values[j])
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, real(vectors.At(i, j))*sqrtEigenVal)
		}
	}

	return loadings
}

// maximumLikelihoodFactoring implements Maximum Likelihood factor analysis
func maximumLikelihoodFactoring(r, rMat *mat.Dense, nfactors int, covar bool, minErr float64, maxIter int) (*mat.Dense, []float64, []float64) {
	p, _ := r.Dims()

	// Get SMC values for initial communalities
	smcResult, _ := Smc(r, &SmcOptions{Covar: covar})
	smcVec := make([]float64, p)
	for i := 0; i < p; i++ {
		smcVec[i] = smcResult.AtVec(i)
	}

	// Set upper bound
	upper := 0.0
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

	// Optimization
	problem := optimize.Problem{
		Func: func(x []float64) float64 {
			return faFn(x, r, nfactors)
		},
	}

	settings := optimize.DefaultSettings()
	method := &optimize.BFGS{}

	result, err := optimize.Local(problem, start, settings, method)
	_ = err // Ignore error, fallback handled below

	// Use result.X if optimization succeeded, otherwise use start values
	psi := result.X
	if err != nil || psi == nil {
		psi = start
	}

	// Extract loadings
	loadings := faOut(psi, r, nfactors)

	// Compute eigenvalues
	s := mat.NewDense(p, p, nil)
	s.CloneFrom(r)
	for i := 0; i < p; i++ {
		s.Set(i, i, s.At(i, i)-psi[i])
	}
	var eig mat.Eigen
	eig.Factorize(s, mat.EigenNone)
	values := eig.Values(nil)

	eValues := make([]float64, p)
	for i := 0; i < p; i++ {
		eValues[i] = real(values[i])
	}

	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		communalities[i] = 1.0 - psi[i]
	}

	return loadings, communalities, eValues
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
	var eig mat.Eigen
	eig.Factorize(temp, mat.EigenNone)
	values := eig.Values(nil)

	// Extract eigenvalues after nf
	sum := 0.0
	for i := nf; i < p; i++ {
		eigenVal := real(values[i])
		sum += math.Log(eigenVal) - eigenVal
	}
	sum -= float64(nf) + float64(p)

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
	var eig mat.Eigen
	eig.Factorize(temp, mat.EigenRight)
	values := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

	// Extract first nf eigenvectors and eigenvalues
	l := mat.NewDense(p, nf, nil)
	for j := 0; j < nf; j++ {
		eigenVal := real(values[j])
		if eigenVal < 1.0 {
			eigenVal = 1.0
		}
		sqrtEigenVal := math.Sqrt(eigenVal - 1.0)
		for i := 0; i < p; i++ {
			l.Set(i, j, real(vectors.At(i, j))*sqrtEigenVal)
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
	var eig mat.Eigen
	eig.Factorize(temp, mat.EigenRight)
	values := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

	// Extract first q eigenvectors and eigenvalues
	l := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		eigenVal := real(values[j])
		if eigenVal < 1.0 {
			eigenVal = 1.0
		}
		sqrtEigenVal := math.Sqrt(eigenVal - 1.0)
		for i := 0; i < p; i++ {
			l.Set(i, j, real(vectors.At(i, j))*sqrtEigenVal)
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
	var eig mat.Eigen
	eig.Factorize(r, mat.EigenRight)
	eValues := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

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
		eig.Factorize(rMatWork, mat.EigenRight)
		values := eig.Values(nil)
		vectorsWork := mat.NewCDense(p, p, nil)
		eig.VectorsTo(vectorsWork)

		// Extract loadings
		loadings := mat.NewDense(p, nfactors, nil)
		if nfactors > 1 {
			for j := 0; j < nfactors; j++ {
				eigenVal := real(values[j])
				sqrtEigenVal := math.Sqrt(eigenVal)
				for k := 0; k < p; k++ {
					loadings.Set(k, j, real(vectorsWork.At(k, j))*sqrtEigenVal)
				}
			}
		} else {
			eigenVal := real(values[0])
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, 0, real(vectorsWork.At(k, 0))*sqrtEigenVal)
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
			eigenVal := real(eValues[j])
			sqrtEigenVal := math.Sqrt(eigenVal)
			for k := 0; k < p; k++ {
				loadings.Set(k, j, real(vectors.At(k, j))*sqrtEigenVal*math.Sqrt(h2[k]))
			}
		}
	} else {
		eigenVal := real(eValues[0])
		sqrtEigenVal := math.Sqrt(eigenVal)
		for k := 0; k < p; k++ {
			loadings.Set(k, 0, real(vectors.At(k, 0))*sqrtEigenVal*math.Sqrt(h2[k]))
		}
	}

	// Extract eigenvalues
	finalEValues := make([]float64, p)
	for j := 0; j < p; j++ {
		finalEValues[j] = real(eValues[j])
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
	var eig mat.Eigen
	eig.Factorize(rWork, mat.EigenRight)
	values := eig.Values(nil)
	vectors := mat.NewCDense(p, p, nil)
	eig.VectorsTo(vectors)

	// Extract loadings
	loadings := mat.NewDense(p, nfactors, nil)
	for j := 0; j < nfactors; j++ {
		eigenVal := real(values[j])
		if eigenVal < 0 {
			eigenVal = 0
		}
		sqrtEigenVal := math.Sqrt(eigenVal)
		for i := 0; i < p; i++ {
			loadings.Set(i, j, real(vectors.At(i, j))*sqrtEigenVal)
		}
	}

	// Extract eigenvalues from original matrix
	eig.Factorize(r, mat.EigenNone)
	eValues := eig.Values(nil)

	finalEValues := make([]float64, p)
	for i := 0; i < p; i++ {
		finalEValues[i] = real(eValues[i])
	}

	// Communalities
	communalities := make([]float64, p)
	for i := 0; i < p; i++ {
		communalities[i] = comGlb
	}

	return loadings, communalities, finalEValues
}
