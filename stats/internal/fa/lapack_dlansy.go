// fa/lapack_dlansy.go
//
// dlansy — symmetric matrix norm. Faithful translation of LAPACK
// 3.12.1 dlansy.f. Supports norm = 'M' (max abs), '1'/'O'/'I' (1-norm
// or inf-norm; equal for symmetric matrices), 'F'/'E' (Frobenius).
package fa

import "math"

// dlansy computes the requested norm of a symmetric matrix A stored
// in column-major flat layout (lda × ?).  uplo: 'U' or 'L'.  work
// must have length at least n when norm is '1'/'O'/'I'.
func dlansy(norm, uplo byte, n int, a []float64, lda int, work []float64) float64 {
	if n == 0 {
		return 0
	}
	upper := uplo == 'U' || uplo == 'u'

	switch norm {
	case 'M', 'm':
		// max(|A(i,j)|).
		var value float64
		if upper {
			for j := 1; j <= n; j++ {
				for i := 1; i <= j; i++ {
					sum := math.Abs(a[(j-1)*lda+i-1])
					if value < sum || math.IsNaN(sum) {
						value = sum
					}
				}
			}
		} else {
			for j := 1; j <= n; j++ {
				for i := j; i <= n; i++ {
					sum := math.Abs(a[(j-1)*lda+i-1])
					if value < sum || math.IsNaN(sum) {
						value = sum
					}
				}
			}
		}
		return value

	case 'I', 'i', 'O', 'o', '1':
		// 1-norm = inf-norm for symmetric.
		var value float64
		if upper {
			// LAPACK reference dlansy.f initializes work to zero before
			// the main loop. The upper branch's `work[i-1] += absa`
			// reads-modifies-writes, so a dirty buffer (e.g. reused from
			// another LAPACK call) would corrupt the row sum.
			for i := 0; i < n; i++ {
				work[i] = 0
			}
			for j := 1; j <= n; j++ {
				sum := 0.0
				for i := 1; i <= j-1; i++ {
					absa := math.Abs(a[(j-1)*lda+i-1])
					sum += absa
					work[i-1] += absa
				}
				work[j-1] = sum + math.Abs(a[(j-1)*lda+j-1])
			}
			for i := 1; i <= n; i++ {
				if value < work[i-1] || math.IsNaN(work[i-1]) {
					value = work[i-1]
				}
			}
		} else {
			for i := 0; i < n; i++ {
				work[i] = 0
			}
			for j := 1; j <= n; j++ {
				sum := work[j-1] + math.Abs(a[(j-1)*lda+j-1])
				for i := j + 1; i <= n; i++ {
					absa := math.Abs(a[(j-1)*lda+i-1])
					sum += absa
					work[i-1] += absa
				}
				if value < sum || math.IsNaN(sum) {
					value = sum
				}
			}
		}
		return value

	case 'F', 'f', 'E', 'e':
		// Frobenius norm via dlassq, accumulating off-diagonals × 2 + diag.
		scale := 0.0
		sum := 1.0
		if upper {
			for j := 2; j <= n; j++ {
				scale, sum = dlassq(j-1, a[(j-1)*lda:], 1, scale, sum)
			}
		} else {
			for j := 1; j <= n-1; j++ {
				scale, sum = dlassq(n-j, a[(j-1)*lda+j:], 1, scale, sum)
			}
		}
		sum *= 2
		scale, sum = dlassq(n, a, lda+1, scale, sum)
		return scale * math.Sqrt(sum)
	}
	return 0
}
