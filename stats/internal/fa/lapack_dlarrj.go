// fa/lapack_dlarrj.go
//
// dlarrj — limited bisection refinement of selected eigenvalues of a
// symmetric tridiagonal matrix (the unfactored form). Used by dstemr.
// Faithful translation of LAPACK 3.12.1 dlarrj.f.
package fa

import "math"

// dlarrj refines eigenvalue intervals via Sturm-sequence bisection on
// the tridiagonal T (with squared off-diagonals e2[]). Differs from
// dlarrb in that the Sturm count is computed on T directly (not on a
// twisted factorization), so no L/D inputs are needed.
//
// Inputs/IO:
//
//	d, e2     : tridiagonal diagonal and squared off-diagonals
//	ifirst,
//	ilast     : 1-based eigenvalue index range
//	rtol      : relative tolerance
//	offset    : array index offset; w[i-offset] is the i-th eigenvalue
//	w         : eigenvalue approximations (in/out)
//	werr      : uncertainty radii (in/out)
//	work      : workspace, length 2*n
//	iwork     : workspace, length 2*n
//	pivmin    : minimum pivot
//	spdiam    : spectral diameter
func dlarrj(
	n int,
	d, e2 []float64,
	ifirst, ilast int,
	rtol float64,
	offset int,
	w, werr []float64,
	work []float64,
	iwork []int,
	pivmin, spdiam float64,
) (info int) {
	if n <= 0 {
		return 0
	}
	maxitr := int((math.Log(spdiam+pivmin)-math.Log(pivmin))/math.Log(2)) + 2

	i1 := ifirst
	i2 := ilast
	nint := 0
	prev := 0

	for i := i1; i <= i2; i++ {
		k := 2 * i
		ii := i - offset
		left := w[ii-1] - werr[ii-1]
		mid := w[ii-1]
		right := w[ii-1] + werr[ii-1]
		width := right - mid
		tmp := math.Max(math.Abs(left), math.Abs(right))

		if width < rtol*tmp {
			iwork[k-2] = -1
			if i == i1 && i < i2 {
				i1 = i + 1
			}
			if prev >= i1 && i <= i2 {
				iwork[2*prev-2] = i + 1
			}
		} else {
			prev = i
			// Expand left until cnt(left) <= i-1.
			fac := 1.0
			for {
				cnt := 0
				s := left
				dplus := d[0] - s
				if dplus < 0 {
					cnt++
				}
				for j := 2; j <= n; j++ {
					dplus = d[j-1] - s - e2[j-2]/dplus
					if dplus < 0 {
						cnt++
					}
				}
				if cnt <= i-1 {
					break
				}
				left -= werr[ii-1] * fac
				fac *= 2
			}
			// Expand right until cnt(right) >= i.
			fac = 1.0
			var cnt int
			for {
				cnt = 0
				s := right
				dplus := d[0] - s
				if dplus < 0 {
					cnt++
				}
				for j := 2; j <= n; j++ {
					dplus = d[j-1] - s - e2[j-2]/dplus
					if dplus < 0 {
						cnt++
					}
				}
				if cnt >= i {
					break
				}
				right += werr[ii-1] * fac
				fac *= 2
			}
			nint++
			iwork[k-2] = i + 1
			iwork[k-1] = cnt
		}
		work[k-2] = left
		work[k-1] = right
	}

	savi1 := i1
	iter := 0
	for nint > 0 {
		prev = i1 - 1
		i := i1
		olnint := nint
		for ip := 1; ip <= olnint; ip++ {
			k := 2 * i
			next := iwork[k-2]
			left := work[k-2]
			right := work[k-1]
			mid := 0.5 * (left + right)
			width := right - mid
			tmp := math.Max(math.Abs(left), math.Abs(right))
			if width < rtol*tmp || iter == maxitr {
				nint--
				iwork[k-2] = 0
				if i1 == i {
					i1 = next
				} else if prev >= i1 {
					iwork[2*prev-2] = next
				}
				i = next
				continue
			}
			prev = i

			// One bisection step.
			cnt := 0
			s := mid
			dplus := d[0] - s
			if dplus < 0 {
				cnt++
			}
			for j := 2; j <= n; j++ {
				dplus = d[j-1] - s - e2[j-2]/dplus
				if dplus < 0 {
					cnt++
				}
			}
			if cnt <= i-1 {
				work[k-2] = mid
			} else {
				work[k-1] = mid
			}
			i = next
		}
		iter++
		if iter > maxitr {
			break
		}
	}

	// Finalize converged intervals.
	for i := savi1; i <= ilast; i++ {
		k := 2 * i
		ii := i - offset
		if iwork[k-2] == 0 {
			w[ii-1] = 0.5 * (work[k-2] + work[k-1])
			werr[ii-1] = work[k-1] - w[ii-1]
		}
	}
	return 0
}
