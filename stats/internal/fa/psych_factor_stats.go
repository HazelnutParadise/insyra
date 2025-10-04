// fa/psych_factor_stats.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// FactorStats is an alias for FaStats.
// Mirrors psych::factor.stats
func FactorStats(r *mat.Dense, f *mat.Dense, phi *mat.Dense, nObs float64, npObs *mat.VecDense, alpha float64, fm string, smooth, coarse bool) interface{} {
	// Call FaStats - assuming it's implemented elsewhere
	// For now, placeholder
	return nil
}
