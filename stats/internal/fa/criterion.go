// fa/criterion.go
package fa

import (
	"fmt"
	"strings"

	"gonum.org/v1/gonum/mat"
)

// Criterion evaluates a rotation criterion at a loading matrix, exactly as
// the gradient projection search evaluates it: the same vgQ function, and
// for Varimax the same Kaiser row normalisation the search applies. It exists
// so a solution can be judged by the quantity the rotation minimises rather
// than by which local minimum a reference implementation happened to reach —
// two loading matrices are equivalent answers when their criterion values
// agree, and the lower one is the better answer when they do not.
//
// method is a FaRotations method name. gamma is oblimin's, delta is geomin's;
// both are ignored by the others.
func Criterion(method string, L *mat.Dense, gamma, delta float64) (float64, error) {
	if L == nil {
		return 0, fmt.Errorf("criterion: nil loadings")
	}
	rows, cols := L.Dims()
	if rows == 0 || cols == 0 {
		return 0, fmt.Errorf("criterion: empty loadings")
	}
	switch strings.ToLower(method) {
	case "varimax":
		// Varimax runs with normalize = true (see Varimax), so its
		// criterion is evaluated on Kaiser-normalised rows.
		weights := NormalizingWeight(L, true)
		Lw := mat.DenseCopyOf(L)
		for i := 0; i < rows; i++ {
			w := weights.AtVec(i)
			if w == 0 {
				continue
			}
			for j := 0; j < cols; j++ {
				Lw.Set(i, j, Lw.At(i, j)/w)
			}
		}
		_, f, _ := vgQVarimax(Lw)
		return f, nil
	case "quartimax":
		_, f, _ := vgQQuartimax(L)
		return f, nil
	case "geomint", "geominq", "geomin":
		if delta <= 0 {
			delta = 0.01
		}
		_, f, _ := vgQGeomin(L, delta)
		return f, nil
	case "bentlert", "bentlerq", "bentler":
		_, f, _, err := vgQBentler(L)
		return f, err
	case "quartimin":
		_, f, _ := vgQQuartimin(L)
		return f, nil
	case "oblimin":
		_, f, err := vgQOblimin(L, gamma)
		return f, err
	case "simplimax":
		_, f, _ := vgQSimplimax(L, rows)
		return f, nil
	}
	return 0, fmt.Errorf("criterion: %q is not a gradient projection criterion", method)
}
