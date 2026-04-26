// fa/factor2cluster.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Factor2Cluster creates a cluster structure from factor loadings.
// Mirrors GPArotation::factor2cluster: assigns each variable to the factor with
// the highest |loading| if that exceeds cut (R default 0.3), else no cluster.
func Factor2Cluster(loadings *mat.Dense) *mat.Dense {
	const cut = 0.3
	p, q := loadings.Dims()
	clusterMat := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		maxAbs := 0.0
		maxFactor := -1
		for j := 0; j < q; j++ {
			absVal := math.Abs(loadings.At(i, j))
			if absVal > maxAbs {
				maxAbs = absVal
				maxFactor = j
			}
		}
		if maxFactor >= 0 && maxAbs > cut {
			clusterMat.Set(i, maxFactor, 1.0)
		}
	}
	return clusterMat
}
