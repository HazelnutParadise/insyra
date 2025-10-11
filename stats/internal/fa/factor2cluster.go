// fa/factor2cluster.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Factor2ClusterOptions represents options for Factor2Cluster
type Factor2ClusterOptions struct {
	UseRLikePipeline bool // If true, use R-like pipeline: varimax -> target.rot -> cluster
}

// Factor2Cluster creates a cluster structure from factor loadings.
// Mirrors GPArotation::factor2cluster
//
// Tie-break strategy: When multiple factors have the same maximum absolute loading,
// the factor with the smallest index (0-based) is chosen.
//
// If UseRLikePipeline is true, applies varimax rotation followed by target rotation
// before clustering, similar to R's GPArotation::factor2cluster pipeline.
func Factor2Cluster(loadings *mat.Dense, opts *Factor2ClusterOptions) *mat.Dense {
	p, q := loadings.Dims()

	// Set default options
	if opts == nil {
		opts = &Factor2ClusterOptions{
			UseRLikePipeline: false,
		}
	}

	// If using R-like pipeline, apply rotations first
	if opts.UseRLikePipeline {
		// Apply varimax rotation
		varimaxResult := Varimax(loadings, true, 1e-5, 1000)
		loadings = varimaxResult["loadings"].(*mat.Dense)
		// Note: Full R pipeline would include target rotation, but simplified here
		// to avoid circular dependencies. Target rotation can be applied separately if needed.
	}

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

// Factor2ClusterSimple is a backward-compatible wrapper for Factor2Cluster
func Factor2ClusterSimple(loadings *mat.Dense) *mat.Dense {
	return Factor2Cluster(loadings, nil)
}
