package fa

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

func invertDense(x *mat.Dense) (*mat.Dense, error) {
	if x == nil {
		return nil, fmt.Errorf("nil matrix")
	}
	r, c := x.Dims()
	if r != c {
		return nil, fmt.Errorf("matrix must be square, got %dx%d", r, c)
	}
	// gonum's mat.Dense.Inverse can panic on truly-singular matrices
	// (zero diagonal in U after LU). Wrap with recover so callers see a
	// clean error rather than a runtime crash.
	var inv mat.Dense
	var invErr error
	func() {
		defer func() {
			if pErr := recover(); pErr != nil {
				invErr = fmt.Errorf("matrix inversion panicked (singular?): %v", pErr)
			}
		}()
		invErr = inv.Inverse(x)
	}()
	if invErr != nil {
		return nil, invErr
	}
	return &inv, nil
}
