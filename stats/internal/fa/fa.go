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
	smc, _ := Smc(r, nil) // Use default options
	return smc, nil
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
	for i := range nf {
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

// RotateWithDiagnostics performs factor rotation on loadings with diagnostics.
// Returns rotated loadings, rotation matrix, Phi matrix, convergence status, and diagnostics map.
func RotateWithDiagnostics(loadings *mat.Dense, method string, opts *RotOpts) (*mat.Dense, *mat.Dense, *mat.Dense, bool, map[string]interface{}, error) {
	// loadings is p x nf (variables x factors)
	// Create correlation matrix (identity for now, as we don't have it)
	_, nf := loadings.Dims()
	r := mat.NewDense(nf, nf, nil)
	for i := range nf {
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

	// Build diagnostics map
	diagnostics := map[string]interface{}{
		"method":    method,
		"converged": converged,
		"restarts":  opts.Restarts,
	}

	if f, ok := res["f"].(float64); ok {
		diagnostics["objective"] = f
	}

	if iterations, ok := res["iterations"].(int); ok {
		diagnostics["iterations"] = iterations
	}

	return rotatedLoadings, rotMat, phiMat, converged, diagnostics, nil
}
