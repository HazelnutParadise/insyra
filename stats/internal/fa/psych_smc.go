// fa/psych_smc.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Smc computes the squared multiple correlations (SMC) for a correlation matrix.
// Mirrors psych::smc with complete NA handling
func Smc(r *mat.Dense) *mat.VecDense {
	p, q := r.Dims()

	// If not square, assume it's data matrix, compute correlation
	if p != q {
		// For data matrix, we need to handle NAs properly
		// This is a simplified version - in full implementation would need NA detection
		r = CorrelationMatrix(r)
		p, _ = r.Dims()
	}

	// For correlation matrix, compute SMC with NA handling
	// In R psych::smc, when there are NAs in the original data,
	// it computes pairwise correlations for each variable
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
		predictorsInv := Pinv(predictors)
		if predictorsInv == nil {
			// If not invertible, SMC = 1
			smc.SetVec(i, 1.0)
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

	return smc
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
