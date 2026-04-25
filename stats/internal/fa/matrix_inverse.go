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
	var inv mat.Dense
	if err := inv.Inverse(x); err != nil {
		return nil, err
	}
	return &inv, nil
}
