// fa/factor2cluster.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Factor2ClusterOptions represents options for Factor2Cluster
type Factor2ClusterOptions struct {
	Cut              float64 // Loading threshold below which variables are not assigned to any cluster
	UseRLikePipeline bool    // If true, use R-like pipeline: varimax -> target.rot -> cluster
}

// Factor2Cluster creates a cluster structure from factor loadings.
// Mirrors GPArotation::factor2cluster exactly
//
// For each variable, assign it to the factor with the highest absolute loading
// if that loading exceeds the cut threshold. Otherwise, assign to no cluster (0).
func Factor2Cluster(loadings *mat.Dense, opts *Factor2ClusterOptions) *mat.Dense {
	p, q := loadings.Dims()

	// Set default options
	if opts == nil {
		opts = &Factor2ClusterOptions{
			Cut:              0.3, // Default cut value from R
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
				maxFactor = j + 1 // R uses 1-based indexing
			}
		}
		// Only assign if max loading exceeds cut threshold
		if maxAbs <= opts.Cut {
			clusters[i] = 0 // No cluster assignment
		} else {
			clusters[i] = maxFactor
		}
	}

	// Create cluster matrix
	clusterMat := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		if clusters[i] > 0 {
			clusterMat.Set(i, clusters[i]-1, 1.0) // Convert back to 0-based indexing
		}
	}

	return clusterMat
}

// Factor2ClusterSimple is a backward-compatible wrapper for Factor2Cluster
func Factor2ClusterSimple(loadings *mat.Dense) *mat.Dense {
	return Factor2Cluster(loadings, nil)
}
