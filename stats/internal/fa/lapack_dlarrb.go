// fa/lapack_dlarrb.go
//
// dlarrb — local-bisection refinement of selected eigenvalues of an
//          LDL^T factored tridiagonal, using twisted-factorization
//          Sturm counts (dlaneg).
// dlaneg — Sturm-sequence negcount via twisted factorization, with
//          NaN-tolerant block-loop safety. Used by dlarrb.
//
// Faithful translations of LAPACK 3.12.1 dlarrb.f and dlaneg.f.
package fa

import "math"

// dlaneg returns the count of eigenvalues of LDL^T - sigma I that are
// strictly less than zero, computed via the twisted factorization at
// twist index r.
//
//	d, lld : LDL^T factor (d length n, lld length n-1; lld[i]=l[i]^2*d[i])
//	sigma  : shift
//	pivmin : minimum pivot magnitude (signature parity; not used in
//	         the reference's IEEE-mode path, kept for API)
//	r      : twist index, 1 <= r <= n
func dlaneg(n int, d, lld []float64, sigma, pivmin float64, r int) int {
	const blklen = 128
	negcnt := 0

	// Upper part: L D L^T - sigma I = L+ D+ L+^T.
	t := -sigma
	for bj := 1; bj <= r-1; bj += blklen {
		neg1 := 0
		bsav := t
		jEnd := bj + blklen - 1
		if jEnd > r-1 {
			jEnd = r - 1
		}
		for j := bj; j <= jEnd; j++ {
			dPlus := d[j-1] + t
			if dPlus < 0 {
				neg1++
			}
			tmp := t / dPlus
			t = tmp*lld[j-1] - sigma
		}
		if math.IsNaN(t) {
			// Slower NaN-tolerant pass.
			neg1 = 0
			t = bsav
			for j := bj; j <= jEnd; j++ {
				dPlus := d[j-1] + t
				if dPlus < 0 {
					neg1++
				}
				tmp := t / dPlus
				if math.IsNaN(tmp) {
					tmp = 1
				}
				t = tmp*lld[j-1] - sigma
			}
		}
		negcnt += neg1
	}

	// Lower part: L D L^T - sigma I = U- D- U-^T.
	p := d[n-1] - sigma
	for bj := n - 1; bj >= r; bj -= blklen {
		neg2 := 0
		bsav := p
		jEnd := bj - blklen + 1
		if jEnd < r {
			jEnd = r
		}
		for j := bj; j >= jEnd; j-- {
			dMinus := lld[j-1] + p
			if dMinus < 0 {
				neg2++
			}
			tmp := p / dMinus
			p = tmp*d[j-1] - sigma
		}
		if math.IsNaN(p) {
			neg2 = 0
			p = bsav
			for j := bj; j >= jEnd; j-- {
				dMinus := lld[j-1] + p
				if dMinus < 0 {
					neg2++
				}
				tmp := p / dMinus
				if math.IsNaN(tmp) {
					tmp = 1
				}
				p = tmp*d[j-1] - sigma
			}
		}
		negcnt += neg2
	}

	// Twist index gamma: T was shifted by sigma initially.
	gamma := (t + sigma) + p
	if gamma < 0 {
		negcnt++
	}
	_ = pivmin
	return negcnt
}

// dlarrb performs local bisection refinement of eigenvalues of an
// LDL^T-factored tridiagonal in indices ifirst..ilast.
//
// Inputs/IO:
//
//	d, lld   : factors (length n, n-1)
//	w        : eigenvalue approximations (1-based offset access via offset)
//	werr     : uncertainty radii (in/out)
//	wgap     : gaps to right neighbour (in/out)
//	ifirst,
//	ilast    : 1-based eigenvalue index range
//	rtol1,
//	rtol2    : relative tolerances
//	offset   : array index offset; w[i-offset] is the i-th eigenvalue
//	work     : workspace, length 2*n
//	iwork    : workspace, length 2*n
//	pivmin   : minimum pivot
//	spdiam   : spectral diameter
//	twist    : twist index for dlaneg (0 = use n)
func dlarrb(
	n int,
	d, lld []float64,
	ifirst, ilast int,
	rtol1, rtol2 float64,
	offset int,
	w, wgap, werr []float64,
	work []float64,
	iwork []int,
	pivmin, spdiam float64,
	twist int,
) (info int) {
	if n <= 0 {
		return 0
	}
	maxitr := int((math.Log(spdiam+pivmin)-math.Log(pivmin))/math.Log(2)) + 2
	mnwdth := 2 * pivmin

	r := twist
	if r < 1 || r > n {
		r = n
	}

	// Initialize unconverged interval list: WORK(2*I-1)..WORK(2*I) holds
	// (left, right) for eigenvalue i. IWORK(2*I-1) is the next-i in the
	// linked list of unconverged intervals (-1 or 0 for converged).
	// IWORK(2*I) holds the negcount at the right endpoint.
	i1 := ifirst
	nint := 0
	prev := 0
	rgap := wgap[i1-offset-1]

	for i := ifirst; i <= ilast; i++ {
		k := 2 * i
		ii := i - offset
		left := w[ii-1] - werr[ii-1]
		right := w[ii-1] + werr[ii-1]
		lgap := rgap
		rgap = wgap[ii-1]
		gap := math.Min(lgap, rgap)

		// Expand left until negcnt(left) <= i-1.
		back := werr[ii-1]
		for {
			negcnt := dlaneg(n, d, lld, left, pivmin, r)
			if negcnt <= i-1 {
				break
			}
			left -= back
			back *= 2
		}
		// Expand right until negcnt(right) >= i.
		back = werr[ii-1]
		for {
			negcnt := dlaneg(n, d, lld, right, pivmin, r)
			if negcnt >= i {
				break
			}
			right += back
			back *= 2
		}

		width := 0.5 * math.Abs(left-right)
		tmp := math.Max(math.Abs(left), math.Abs(right))
		cvrgd := math.Max(rtol1*gap, rtol2*tmp)
		if width <= cvrgd || width <= mnwdth {
			iwork[k-2] = -1
			if i == i1 && i < ilast {
				i1 = i + 1
			}
			if prev >= i1 && i <= ilast {
				iwork[2*prev-2] = i + 1
			}
		} else {
			prev = i
			nint++
			iwork[k-2] = i + 1
			iwork[k-1] = dlaneg(n, d, lld, right, pivmin, r)
		}
		work[k-2] = left
		work[k-1] = right
	}

	// Iterative bisection of unconverged intervals.
	iter := 0
	for nint > 0 {
		prev = i1 - 1
		i := i1
		olnint := nint
		for ip := 1; ip <= olnint; ip++ {
			k := 2 * i
			ii := i - offset
			rgapLocal := wgap[ii-1]
			lgap := rgapLocal
			if ii > 1 {
				lgap = wgap[ii-2]
			}
			gap := math.Min(lgap, rgapLocal)
			next := iwork[k-2]
			left := work[k-2]
			right := work[k-1]
			mid := 0.5 * (left + right)

			width := right - mid
			tmp := math.Max(math.Abs(left), math.Abs(right))
			cvrgd := math.Max(rtol1*gap, rtol2*tmp)
			if width <= cvrgd || width <= mnwdth || iter == maxitr {
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

			negcnt := dlaneg(n, d, lld, mid, pivmin, r)
			if negcnt <= i-1 {
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

	// Finalize: store midpoints into w and half-widths into werr.
	for i := ifirst; i <= ilast; i++ {
		k := 2 * i
		ii := i - offset
		if iwork[k-2] == 0 {
			w[ii-1] = 0.5 * (work[k-2] + work[k-1])
			werr[ii-1] = work[k-1] - w[ii-1]
		}
	}
	// Recompute gaps.
	for i := ifirst + 1; i <= ilast; i++ {
		ii := i - offset
		gap := w[ii-1] - werr[ii-1] - w[ii-2] - werr[ii-2]
		if gap < 0 {
			gap = 0
		}
		wgap[ii-2] = gap
	}
	return 0
}
