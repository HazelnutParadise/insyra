// fa/factor2cluster.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Factor2Cluster creates a cluster structure from factor loadings.
// Mirrors GPArotation::factor2cluster: assigns each variable to the factor with
// the highest |loading| if that exceeds cut (R default 0.3), else no cluster.
//
// Preserves the sign of the dominant loading (R uses sign(L[i, j*])): a
// variable that loads -0.85 on factor j is assigned -1, not +1, in the
// cluster matrix. The previous implementation always wrote +1.
func Factor2Cluster(loadings *mat.Dense) *mat.Dense {
	const cut = 0.3
	p, q := loadings.Dims()
	clusterMat := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		maxAbs := 0.0
		maxFactor := -1
		var maxSign float64 = 1.0
		for j := 0; j < q; j++ {
			val := loadings.At(i, j)
			absVal := math.Abs(val)
			if absVal > maxAbs {
				maxAbs = absVal
				maxFactor = j
				if val < 0 {
					maxSign = -1.0
				} else {
					maxSign = 1.0
				}
			}
		}
		if maxFactor >= 0 && maxAbs > cut {
			clusterMat.Set(i, maxFactor, maxSign)
		}
	}
	return clusterMat
}
