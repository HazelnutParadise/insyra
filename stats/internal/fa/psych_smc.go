// fa/psych_smc.go
package fa

import (
	"math"

	statslinalg "github.com/HazelnutParadise/insyra/stats/internal/linalg"
	"gonum.org/v1/gonum/mat"
)

// SmcOptions represents options for Smc function
type SmcOptions struct {
	Covar    bool    // If true, use covariance matrix instead of correlation
	Pairwise bool    // If true, use pairwise correlations for NA handling
	Tol      float64 // Tolerance for matrix inversion
}

// Smc computes the squared multiple correlations (SMC) for a correlation matrix.
// Mirrors psych::smc with complete NA handling and covar/pairwise options
func Smc(r *mat.Dense, opts *SmcOptions) (*mat.VecDense, map[string]interface{}) {
	p, q := r.Dims()

	// Set default options
	if opts == nil {
		opts = &SmcOptions{
			Covar:    false,
			Pairwise: false,
			Tol:      1e-8,
		}
	}

	diagnostics := map[string]interface{}{
		"wasImputed":       false,
		"imputationMethod": "none",
		"errors":           []string{},
	}

	// If not square, assume it's data matrix, compute correlation/covariance
	if p != q {
		if opts.Pairwise {
			if opts.Covar {
				r = statslinalg.CovarianceDensePairwise(r)
			} else {
				r = statslinalg.CorrelationDensePairwise(r)
			}
			diagnostics["wasImputed"] = true
			diagnostics["imputationMethod"] = "pairwise"
		} else {
			// Standard correlation/covariance computation
			if opts.Covar {
				r = statslinalg.CovarianceDense(r)
			} else {
				r = statslinalg.CorrelationDense(r)
			}
		}
		p, _ = r.Dims()
	}

	// Handle covar parameter - convert covariance to correlation if needed
	var vari []float64
	var corrMatrix *mat.Dense

	if opts.Covar {
		// If input is covariance matrix, convert to correlation
		vari = make([]float64, p)
		for i := 0; i < p; i++ {
			vari[i] = r.At(i, i)
		}
		corrMatrix = mat.NewDense(p, p, nil)
		corrMatrix.CloneFrom(r)
		for i := 0; i < p; i++ {
			for j := 0; j < p; j++ {
				corrMatrix.Set(i, j, r.At(i, j)/math.Sqrt(vari[i]*vari[j]))
			}
		}
	} else {
		// Input is already correlation matrix
		corrMatrix = mat.NewDense(p, p, nil)
		corrMatrix.CloneFrom(r)
		vari = make([]float64, p)
		for i := 0; i < p; i++ {
			vari[i] = 1.0 // For correlation matrix, diagonal is 1
		}
	}

	// Check for NA values in correlation matrix
	hasNA := false
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if math.IsNaN(corrMatrix.At(i, j)) {
				hasNA = true
				break
			}
		}
		if hasNA {
			break
		}
	}

	var tempR *mat.Dense
	var wcl []int
	var maxr []float64

	if hasNA {
		// Handle missing values - find variables with most NAs
		tempR = mat.NewDense(p, p, nil)
		tempR.CloneFrom(corrMatrix)
		vr := make([]float64, p)
		for i := 0; i < p; i++ {
			vr[i] = tempR.At(i, i)
		}

		// Set diagonal to 0 temporarily to find max correlations
		for i := 0; i < p; i++ {
			tempR.Set(i, i, 0)
		}

		maxr = make([]float64, p)
		for j := 0; j < p; j++ {
			maxAbs := 0.0
			for i := 0; i < p; i++ {
				if i != j && !math.IsNaN(tempR.At(i, j)) {
					absVal := math.Abs(tempR.At(i, j))
					if absVal > maxAbs {
						maxAbs = absVal
					}
				}
			}
			maxr[j] = maxAbs
		}

		// Restore diagonal
		for i := 0; i < p; i++ {
			tempR.Set(i, i, vr[i])
		}

		// Find variables to remove (similar to R's approach)
		bad := true
		for bad {
			// Count NAs for each variable
			naCounts := make([]int, p)
			for i := 0; i < p; i++ {
				for j := 0; j < p; j++ {
					if math.IsNaN(tempR.At(i, j)) {
						naCounts[i]++
					}
				}
			}

			// Find variable with most NAs
			maxCount := 0
			maxIdx := -1
			for i := 0; i < p; i++ {
				if naCounts[i] > maxCount {
					maxCount = naCounts[i]
					maxIdx = i
				}
			}

			if maxIdx >= 0 {
				wcl = append(wcl, maxIdx)
				// Remove this variable from tempR
				newSize := p - len(wcl)
				newTempR := mat.NewDense(newSize, newSize, nil)
				rowIdx := 0
				for i := 0; i < p; i++ {
					if !contains(wcl, i) {
						colIdx := 0
						for j := 0; j < p; j++ {
							if !contains(wcl, j) {
								newTempR.Set(rowIdx, colIdx, tempR.At(i, j))
								colIdx++
							}
						}
						rowIdx++
					}
				}
				tempR = newTempR

				// Check if still has NAs
				stillHasNA := false
				for i := 0; i < newSize; i++ {
					for j := 0; j < newSize; j++ {
						if math.IsNaN(tempR.At(i, j)) {
							stillHasNA = true
							break
						}
					}
					if stillHasNA {
						break
					}
				}
				bad = stillHasNA
			} else {
				bad = false
			}
		}

		diagnostics["wasImputed"] = true
		diagnostics["imputationMethod"] = "remove_variables_with_most_NAs"
	}

	// Compute pseudoinverse and SMC
	var RInv *mat.Dense
	var err error
	if tempR != nil {
		rows, _ := tempR.Dims()
		if rows > 0 {
			RInv, err = Pinv(tempR, opts.Tol)
		}
	} else {
		RInv, err = Pinv(corrMatrix, opts.Tol)
	}

	if err != nil || RInv == nil {
		diagnostics["errors"] = append(diagnostics["errors"].([]string), "pseudoinverse computation failed")
		return nil, diagnostics
	}

	// Compute SMC = 1 - 1/diag(R.inv)
	var smc []float64
	if tempR != nil {
		rows, _ := tempR.Dims()
		if rows > 0 {
			RInvRows, _ := RInv.Dims()
			smc = make([]float64, RInvRows)
			for i := 0; i < len(smc); i++ {
				diagVal := RInv.At(i, i)
				if diagVal > 0 {
					smc[i] = 1.0 - 1.0/diagVal
				} else {
					diagnostics["errors"] = append(diagnostics["errors"].([]string), "non-positive inverse diagonal in SMC computation")
					smc[i] = 1.0
				}
			}
		}
	} else {
		RInvRows, _ := RInv.Dims()
		smc = make([]float64, RInvRows)
		for i := 0; i < len(smc); i++ {
			diagVal := RInv.At(i, i)
			if diagVal > 0 {
				smc[i] = 1.0 - 1.0/diagVal
			} else {
				diagnostics["errors"] = append(diagnostics["errors"].([]string), "non-positive inverse diagonal in SMC computation")
				smc[i] = 1.0
			}
		}
	}

	// Handle NA case
	var smcNA []float64
	if tempR != nil {
		rows, _ := tempR.Dims()
		if rows > 0 {
			RNaInv, err := Pinv(tempR, opts.Tol)
			if err == nil && RNaInv != nil {
				RNaInvRows, _ := RNaInv.Dims()
				smcNA = make([]float64, RNaInvRows)
				for i := 0; i < len(smcNA); i++ {
					diagVal := RNaInv.At(i, i)
					if diagVal > 0 {
						smcNA[i] = 1.0 - 1.0/diagVal
					} else {
						diagnostics["errors"] = append(diagnostics["errors"].([]string), "non-positive inverse diagonal in SMC NA computation")
						smcNA[i] = 1.0
					}
				}
			} else {
				diagnostics["errors"] = append(diagnostics["errors"].([]string), "pseudoinverse computation failed for SMC NA reconstruction")
				smcNA = make([]float64, len(smc))
				for i := range smcNA {
					smcNA[i] = 1.0
				}
			}
		}
	} else {
		smcNA = make([]float64, len(smc))
		copy(smcNA, smc)
	}

	// Apply bounds and handle special cases
	for i := range smc {
		if math.IsNaN(smc[i]) {
			diagnostics["errors"] = append(diagnostics["errors"].([]string), "NaN SMC value")
			smc[i] = 1.0
		}
		if smc[i] > 1.0 {
			smc[i] = 1.0
		}
		if smc[i] < 0.0 {
			smc[i] = 0.0
		}
	}

	// Fill smcAll
	smcAll := make([]float64, p)
	for i := range smcAll {
		if !contains(wcl, i) {
			smcAll[i] = smc[i-len(wcl)]
		} else {
			smcAll[i] = smcNA[i-len(wcl)]
		}
	}

	// Handle remaining NAs
	for i := range smcAll {
		if math.IsNaN(smcAll[i]) && maxr != nil && i < len(maxr) {
			smcAll[i] = maxr[i]
		}
	}

	// Apply covariance scaling if needed
	if opts.Covar {
		for i := range smcAll {
			smcAll[i] *= vari[i]
		}
	}

	// Create result vector
	result := mat.NewVecDense(p, smcAll)
	return result, diagnostics
}

// Helper function to check if slice contains value
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
