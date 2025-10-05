// fa/fa.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// SMC computes the squared multiple correlations for a correlation matrix.
// This is a wrapper around Smc for compatibility.
func SMC(r *mat.Dense, isCorr bool) (*mat.VecDense, error) {
	if !isCorr {
		// If not correlation matrix, compute it first
		r = CorrelationMatrix(r)
	}
	return Smc(r), nil
}

// RotOpts represents rotation options
type RotOpts struct {
	Eps         float64
	MaxIter     int
	Alpha0      float64
	Gamma       float64
	PromaxPower int
	Restarts    int
}

// Rotate performs factor rotation on loadings.
// This is a wrapper around FaRotations for compatibility.
func Rotate(loadings *mat.Dense, method string, opts *RotOpts) (*mat.Dense, *mat.Dense, *mat.Dense, bool, error) {
	// loadings is p x nf (variables x factors)
	// Create correlation matrix (identity for now, as we don't have it)
	_, nf := loadings.Dims()
	r := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		r.Set(i, i, 1.0)
	}

	// Set default options if nil
	if opts == nil {
		opts = &RotOpts{
			Eps:         1e-5,
			MaxIter:     1000,
			Alpha0:      1.0,
			Gamma:       0.0,
			PromaxPower: 4,
			Restarts:    20,
		}
	}
	if opts.Restarts <= 0 {
		opts.Restarts = 1
	}

	// Call FaRotations
	res := FaRotations(loadings, r, method, opts.Gamma, opts.Restarts).(map[string]any)

	rotatedLoadings := res["loadings"].(*mat.Dense)
	rotMat := res["rotmat"].(*mat.Dense)

	var phiMat *mat.Dense
	if phi, ok := res["Phi"]; ok {
		if phiDense, ok := phi.(*mat.Dense); ok {
			phiMat = phiDense
		}
	}

	converged := true
	if conv, ok := res["convergence"]; ok {
		if convBool, ok := conv.(bool); ok {
			converged = convBool
		}
	}

	return rotatedLoadings, rotMat, phiMat, converged, nil
}
