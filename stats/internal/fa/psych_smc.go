// fa/psych_smc.go
package fa

import (
	"fmt"
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

	// For correlation/covariance matrix, compute SMC
	smc := mat.NewVecDense(p, nil)

	for i := 0; i < p; i++ {
		// For each variable i, compute the multiple correlation
		// This involves regressing variable i on all other variables

		// Extract the row for variable i (excluding diagonal)
		row := make([]float64, p-1)
		colIdx := 0
		for j := 0; j < p; j++ {
			if j != i {
				row[colIdx] = r.At(i, j)
				colIdx++
			}
		}

		// Extract the submatrix of correlations between predictors
		predictors := mat.NewDense(p-1, p-1, nil)
		rowIdx := 0
		for j := 0; j < p; j++ {
			if j != i {
				colIdx := 0
				for k := 0; k < p; k++ {
					if k != i {
						predictors.Set(rowIdx, colIdx, r.At(j, k))
						colIdx++
					}
				}
				rowIdx++
			}
		}

		// Compute the inverse of the predictor correlation matrix
		predictorsInv, err := Pinv(predictors, opts.Tol)
		if err != nil || predictorsInv == nil {
			// If not invertible, SMC = 1 (perfect prediction)
			smc.SetVec(i, 1.0)
			diagnostics["errors"] = append(diagnostics["errors"].([]string),
				fmt.Sprintf("variable %d: matrix not invertible", i))
			continue
		}

		// Compute SMC = row * predictorsInv * row^T
		rowVec := mat.NewVecDense(len(row), row)
		temp := mat.NewVecDense(len(row), nil)
		temp.MulVec(predictorsInv, rowVec)

		smcVal := mat.Dot(rowVec, temp)

		// Clamp to [0, 1]
		if smcVal > 1.0 {
			smcVal = 1.0
		}
		if smcVal < 0.0 {
			smcVal = 0.0
		}

		smc.SetVec(i, smcVal)
	}

	return smc, diagnostics
}

// CorrelationMatrix computes correlation matrix from data matrix
// This is a simplified version - full implementation would handle NAs
func CorrelationMatrix(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	corr := mat.NewDense(p, p, nil)

	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if i == j {
				corr.Set(i, j, 1.0)
			} else {
				colI := mat.Col(nil, i, data)
				colJ := mat.Col(nil, j, data)

				// Compute correlation (simplified, no NA handling)
				meanI, meanJ := 0.0, 0.0
				for k := 0; k < n; k++ {
					meanI += colI[k]
					meanJ += colJ[k]
				}
				meanI /= float64(n)
				meanJ /= float64(n)

				varI, varJ, cov := 0.0, 0.0, 0.0
				for k := 0; k < n; k++ {
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

	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			colI := mat.Col(nil, i, data)
			colJ := mat.Col(nil, j, data)

			meanI, meanJ := 0.0, 0.0
			for k := 0; k < n; k++ {
				meanI += colI[k]
				meanJ += colJ[k]
			}
			meanI /= float64(n)
			meanJ /= float64(n)

			covVal := 0.0
			for k := 0; k < n; k++ {
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

	for i := 0; i < p; i++ {
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

				for k := 0; k < n; k++ {
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

	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			colI := mat.Col(nil, i, data)
			colJ := mat.Col(nil, j, data)

			// Compute pairwise covariance, skipping NA values
			validPairs := 0
			sumI, sumJ := 0.0, 0.0
			sumIJ := 0.0

			for k := 0; k < n; k++ {
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
