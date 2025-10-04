// fa/factor2cluster.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Factor2Cluster creates a cluster structure from factor loadings.
// Mirrors GPArotation::factor2cluster
func Factor2Cluster(loadings *mat.Dense) *mat.Dense {
	p, q := loadings.Dims()

	// For each variable, find the factor with maximum absolute loading
	clusters := make([]int, p)
	for i := 0; i < p; i++ {
		maxAbs := 0.0
		maxFactor := 0
		for j := 0; j < q; j++ {
			absVal := math.Abs(loadings.At(i, j))
			if absVal > maxAbs {
				maxAbs = absVal
				maxFactor = j
			}
		}
		clusters[i] = maxFactor
	}

	// Create cluster matrix
	clusterMat := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		clusterMat.Set(i, clusters[i], 1.0)
	}

	return clusterMat
}
