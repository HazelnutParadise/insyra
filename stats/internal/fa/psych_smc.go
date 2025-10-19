// fa/psych_smc.go
package fa

import (
	"math"

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
			// For pairwise, we need to handle NAs - simplified for now
			if opts.Covar {
				r = CovarianceMatrixPairwise(r)
			} else {
				r = CorrelationMatrixPairwise(r)
			}
			diagnostics["wasImputed"] = true
			diagnostics["imputationMethod"] = "pairwise"
		} else {
			// Standard correlation/covariance computation
			if opts.Covar {
				r = CovarianceMatrix(r)
			} else {
				r = CorrelationMatrix(r)
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
		// Fallback: set SMC to 1
		smc := mat.NewVecDense(p, nil)
		for i := range p {
			smc.SetVec(i, 1.0)
		}
		diagnostics["errors"] = append(diagnostics["errors"].([]string), "pseudoinverse computation failed")
		return smc, diagnostics
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
					smc[i] = 1.0 // Fallback
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
				smc[i] = 1.0 // Fallback
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
						smcNA[i] = 1.0
					}
				}
			} else {
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

// CorrelationMatrix computes correlation matrix from data matrix
// This is a simplified version - full implementation would handle NAs
func CorrelationMatrix(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	corr := mat.NewDense(p, p, nil)

	for i := range p {
		for j := range p {
			if i == j {
				corr.Set(i, j, 1.0)
			} else {
				colI := mat.Col(nil, i, data)
				colJ := mat.Col(nil, j, data)

				// Compute correlation (simplified, no NA handling)
				meanI, meanJ := 0.0, 0.0
				for k := range n {
					meanI += colI[k]
					meanJ += colJ[k]
				}
				meanI /= float64(n)
				meanJ /= float64(n)

				varI, varJ, cov := 0.0, 0.0, 0.0
				for k := range n {
					devI := colI[k] - meanI
					devJ := colJ[k] - meanJ
					varI += devI * devI
					varJ += devJ * devJ
					cov += devI * devJ
				}

				if varI > 0 && varJ > 0 {
					corr.Set(i, j, cov/math.Sqrt(varI*varJ))
				} else {
					corr.Set(i, j, 0.0)
				}
			}
		}
	}

	return corr
}

// CovarianceMatrix computes covariance matrix from data matrix
func CovarianceMatrix(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	cov := mat.NewDense(p, p, nil)

	for i := range p {
		for j := range p {
			colI := mat.Col(nil, i, data)
			colJ := mat.Col(nil, j, data)

			meanI, meanJ := 0.0, 0.0
			for k := range n {
				meanI += colI[k]
				meanJ += colJ[k]
			}
			meanI /= float64(n)
			meanJ /= float64(n)

			covVal := 0.0
			for k := range n {
				covVal += (colI[k] - meanI) * (colJ[k] - meanJ)
			}
			covVal /= float64(n - 1) // Sample covariance

			cov.Set(i, j, covVal)
		}
	}

	return cov
}

// CorrelationMatrixPairwise computes correlation matrix using pairwise complete observations
func CorrelationMatrixPairwise(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	corr := mat.NewDense(p, p, nil)

	for i := range p {
		for j := 0; j < p; j++ {
			if i == j {
				corr.Set(i, j, 1.0)
			} else {
				colI := mat.Col(nil, i, data)
				colJ := mat.Col(nil, j, data)

				// Compute pairwise correlation, skipping NA values
				validPairs := 0
				sumI, sumJ := 0.0, 0.0
				sumI2, sumJ2 := 0.0, 0.0
				sumIJ := 0.0

				for k := range n {
					valI, valJ := colI[k], colJ[k]
					if !math.IsNaN(valI) && !math.IsNaN(valJ) {
						validPairs++
						sumI += valI
						sumJ += valJ
						sumI2 += valI * valI
						sumJ2 += valJ * valJ
						sumIJ += valI * valJ
					}
				}

				if validPairs > 1 {
					meanI := sumI / float64(validPairs)
					meanJ := sumJ / float64(validPairs)

					varI := (sumI2 - float64(validPairs)*meanI*meanI) / float64(validPairs-1)
					varJ := (sumJ2 - float64(validPairs)*meanJ*meanJ) / float64(validPairs-1)
					cov := (sumIJ - float64(validPairs)*meanI*meanJ) / float64(validPairs-1)

					if varI > 0 && varJ > 0 {
						corr.Set(i, j, cov/math.Sqrt(varI*varJ))
					} else {
						corr.Set(i, j, 0.0)
					}
				} else {
					corr.Set(i, j, 0.0)
				}
			}
		}
	}

	return corr
}

// CovarianceMatrixPairwise computes covariance matrix using pairwise complete observations
func CovarianceMatrixPairwise(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	cov := mat.NewDense(p, p, nil)

	for i := range p {
		for j := range p {
			colI := mat.Col(nil, i, data)
			colJ := mat.Col(nil, j, data)

			// Compute pairwise covariance, skipping NA values
			validPairs := 0
			sumI, sumJ := 0.0, 0.0
			sumIJ := 0.0

			for k := range n {
				valI, valJ := colI[k], colJ[k]
				if !math.IsNaN(valI) && !math.IsNaN(valJ) {
					validPairs++
					sumI += valI
					sumJ += valJ
					sumIJ += valI * valJ
				}
			}

			if validPairs > 1 {
				meanI := sumI / float64(validPairs)
				meanJ := sumJ / float64(validPairs)
				covVal := (sumIJ - float64(validPairs)*meanI*meanJ) / float64(validPairs-1)
				cov.Set(i, j, covVal)
			} else {
				cov.Set(i, j, 0.0)
			}
		}
	}

	return cov
}
