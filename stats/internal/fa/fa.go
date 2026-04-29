// fa/fa.go
package fa

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// RotOpts represents rotation options
type RotOpts struct {
	Eps           float64
	MaxIter       int
	Alpha0        float64
	Gamma         float64 // Oblimin gamma
	GeominEpsilon float64 // Geomin delta (R psych default 0.01)
	PromaxPower   int
	Restarts      int
}

// Rotate performs factor rotation on loadings.
// This is a wrapper around FaRotations for compatibility.
func Rotate(loadings *mat.Dense, method string, opts *RotOpts) (*mat.Dense, *mat.Dense, *mat.Dense, bool, error) {
	// loadings is p x nf (variables x factors). The rotation helpers operate
	// on the factor-space identity when no factor correlation has been induced yet.
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
	res := FaRotations(loadings, r, method, opts.Gamma, opts.Restarts, opts.PromaxPower, opts.GeominEpsilon).(map[string]any)
	if errMsg, ok := res["error"].(string); ok && errMsg != "" {
		return nil, nil, nil, false, fmt.Errorf("rotation failed: %s", errMsg)
	}

	rotatedLoadings, ok := res["loadings"].(*mat.Dense)
	if !ok || rotatedLoadings == nil {
		return nil, nil, nil, false, fmt.Errorf("rotation did not return loadings")
	}
	rotMat, ok := res["rotmat"].(*mat.Dense)
	if !ok || rotMat == nil {
		return nil, nil, nil, false, fmt.Errorf("rotation did not return rotation matrix")
	}

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

