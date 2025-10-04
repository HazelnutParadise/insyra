// fa/smc.go
package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// SMC computes squared multiple correlations for each variable.
// If R is square, it is treated as a correlation/covariance matrix (converted to correlation).
// If R is data (n x p, n!=p), it will compute correlation (or covariance if covar=true) first.
// It mirrors the core steps of psych::smc:
//
//	R.inv <- Pinv(R)
//	smc <- 1 - 1/diag(R.inv)
//
// and clamps to [0,1], with the covar=TRUE multiplier by variances.
func SMC(R *mat.Dense, covar bool) (*mat.VecDense, error) {
	r, c := R.Dims()
	if c == 0 {
		return mat.NewVecDense(0, nil), nil
	}
	var Corr mat.Dense
	var vari *mat.VecDense

	if r != c {
		// R is data: compute covariance or correlation, then cov2cor.
		var S mat.Dense
		if covar {
			cov(&S, R) // unbiased sample covariance
			vari = diagVec(&S)
			cov2cor(&Corr, &S) // scale to correlation
		} else {
			cor(&Corr, R) // Pearson correlation
			// build a mock "variances" = 1 for later covar scaling (no-op)
			vari = mat.NewVecDense(Corr.RawMatrix().Rows, nil)
			for i := 0; i < vari.Len(); i++ {
				vari.SetVec(i, 1)
			}
		}
	} else {
		// R is a square matrix: treat as covariance or correlation; convert to correlation.
		// Store original variances if covar==true, to multiply back later.
		if covar {
			vari = diagVec(R)
		}
		cov2cor(&Corr, R)
		// If no covar scaling requested, set vari to 1s for convenience.
		if vari == nil {
			vari = mat.NewVecDense(Corr.RawMatrix().Rows, nil)
			for i := 0; i < vari.Len(); i++ {
				vari.SetVec(i, 1)
			}
		}
	}

	// Handle any NaN/Inf entries conservatively:
	// replace with 0 on diagonal if needed, and try to fill NAs on off-diagonals with 0.
	cleanCorr(&Corr)

	// Pseudoinverse
	pinv := Pinv(&Corr, math.Sqrt(math.SmallestNonzeroFloat64)) // ~ R default tol
	if pinv == nil {
		return nil, errors.New("pinv failed")
	}

	// smc = 1 - 1/diag(R.inv)
	n := Corr.RawMatrix().Rows
	out := mat.NewVecDense(n, nil)
	for i := 0; i < n; i++ {
		rii := pinv.At(i, i)
		if rii == 0 || math.IsNaN(rii) || math.IsInf(rii, 0) {
			out.SetVec(i, 0)
		} else {
			out.SetVec(i, 1.0-1.0/rii)
		}
	}

	// Clamp to [0,1] like psych::smc does (and fix any NA to 1 then clamp)
	for i := 0; i < n; i++ {
		v := out.AtVec(i)
		if math.IsNaN(v) {
			v = 1
		}
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		out.SetVec(i, v)
	}

	// If covar==TRUE, scale by original variances
	if covar {
		for i := 0; i < n; i++ {
			out.SetVec(i, out.AtVec(i)*vari.AtVec(i))
		}
	}
	return out, nil
}

// ----- helpers -----

// cor computes Pearson correlation of columns (pairwise complete by ignoring NaNs).
func cor(dst *mat.Dense, X *mat.Dense) {
	n, p := X.Dims()
	dst.ReuseAs(p, p)
	means := make([]float64, p)
	counts := make([]int, p)

	// Column means (ignoring NaN)
	for j := 0; j < p; j++ {
		sum := 0.0
		cnt := 0
		for i := 0; i < n; i++ {
			v := X.At(i, j)
			if !math.IsNaN(v) {
				sum += v
				cnt++
			}
		}
		if cnt > 0 {
			means[j] = sum / float64(cnt)
			counts[j] = cnt
		} else {
			means[j] = math.NaN()
			counts[j] = 0
		}
	}

	// Covariance normalized to correlation
	for j := 0; j < p; j++ {
		for k := j; k < p; k++ {
			num := 0.0
			cj, ck := 0, 0
			for i := 0; i < n; i++ {
				xj := X.At(i, j)
				xk := X.At(i, k)
				if !math.IsNaN(xj) && !math.IsNaN(xk) {
					num += (xj - means[j]) * (xk - means[k])
					cj++
					ck++
				}
			}
			denj := 0.0
			denk := 0.0
			for i := 0; i < n; i++ {
				xj := X.At(i, j)
				if !math.IsNaN(xj) {
					denj += (xj - means[j]) * (xj - means[j])
				}
				xk := X.At(i, k)
				if !math.IsNaN(xk) {
					denk += (xk - means[k]) * (xk - means[k])
				}
			}
			v := math.NaN()
			if denj > 0 && denk > 0 {
				v = num / math.Sqrt(denj*denk)
			}
			dst.Set(j, k, v)
			dst.Set(k, j, v)
		}
	}
	// set diag to 1 (if NaN, set to 1 to keep PSD attempts like R's cov2cor path)
	for j := 0; j < p; j++ {
		dst.Set(j, j, 1)
	}
}

// cov computes unbiased sample covariance of columns (ignoring NaNs pairwise).
func cov(dst *mat.Dense, X *mat.Dense) {
	n, p := X.Dims()
	dst.ReuseAs(p, p)
	means := make([]float64, p)
	ns := make([]int, p)
	// means
	for j := 0; j < p; j++ {
		sum := 0.0
		cnt := 0
		for i := 0; i < n; i++ {
			v := X.At(i, j)
			if !math.IsNaN(v) {
				sum += v
				cnt++
			}
		}
		if cnt > 0 {
			means[j] = sum / float64(cnt)
			ns[j] = cnt
		} else {
			means[j] = math.NaN()
			ns[j] = 0
		}
	}
	// cov_ij
	for j := 0; j < p; j++ {
		for k := j; k < p; k++ {
			sum := 0.0
			cnt := 0
			for i := 0; i < n; i++ {
				xj := X.At(i, j)
				xk := X.At(i, k)
				if !math.IsNaN(xj) && !math.IsNaN(xk) {
					sum += (xj - means[j]) * (xk - means[k])
					cnt++
				}
			}
			v := math.NaN()
			if cnt > 1 {
				v = sum / float64(cnt-1)
			}
			dst.Set(j, k, v)
			dst.Set(k, j, v)
		}
	}
}

// cov2cor: C -> R = D^{-1/2} C D^{-1/2}, where D = diag(C).
func cov2cor(dst *mat.Dense, C mat.Matrix) {
	n, _ := C.Dims()
	dst.ReuseAs(n, n)
	// extract variances
	d := make([]float64, n)
	for i := 0; i < n; i++ {
		d[i] = C.At(i, i)
		if d[i] <= 0 || math.IsNaN(d[i]) || math.IsInf(d[i], 0) {
			d[i] = 1 // avoid blow-ups; aligns with psych::smc path that sanitizes later
		}
	}
	// scale
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			dst.Set(i, j, C.At(i, j)/math.Sqrt(d[i]*d[j]))
		}
	}
	// force diag = 1
	for i := 0; i < n; i++ {
		dst.Set(i, i, 1)
	}
}

// diagVec extracts the diagonal as a vector.
func diagVec(M mat.Matrix) *mat.VecDense {
	r, _ := M.Dims()
	v := mat.NewVecDense(r, nil)
	for i := 0; i < r; i++ {
		v.SetVec(i, M.At(i, i))
	}
	return v
}

// cleanCorr replaces NaN/Inf on off-diagonals by 0 and fixes diagonals to 1.
func cleanCorr(R *mat.Dense) {
	n, _ := R.Dims()
	maxAbs := 0.0
	// find a max |r| for fallback
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			v := R.At(i, j)
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				ab := math.Abs(v)
				if ab > maxAbs {
					maxAbs = ab
				}
			}
		}
	}
	// replace bad entries
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			v := R.At(i, j)
			if i == j {
				R.Set(i, j, 1)
				continue
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				// conservative fallback similar in spirit to psych replacement by max correlation
				if maxAbs == 0 {
					R.Set(i, j, 0)
				} else {
					// keep the sign as 0 here (unknown), safer is 0.
					R.Set(i, j, 0)
				}
			}
		}
	}
}
