// fa/lapack_dlarrf.go
//
// dlarrf — given an LDL^T representation of a tridiagonal matrix and a
// cluster of close eigenvalues, find a new RRR L(+) D(+) L(+)^T such
// that at least one of its eigenvalues is relatively isolated.
//
// Faithful translation of LAPACK 3.12.1 dlarrf.f.
package fa

import "math"

// dlarrf:
//
//	n      : order of the matrix block
//	d, l   : LDL^T factors (d length n; l length n-1, unit subdiagonal)
//	ld     : l[i]*d[i] (length n-1)
//	clstrt : 1-based first index of the cluster in w[]
//	clend  : 1-based last index of the cluster
//	w      : eigenvalue approximations (1-based indices clstrt..clend valid)
//	wgap   : right-neighbor gaps in w (in/out, 1-based)
//	werr   : per-eigenvalue uncertainty semiwidths (1-based)
//	spdiam : Gerschgorin spectral-diameter estimate
//	clgapl : absolute gap on the left  end of the cluster
//	clgapr : absolute gap on the right end of the cluster
//	pivmin : minimum pivot magnitude
//	work   : workspace, length 2*n
//
// Outputs:
//
//	sigma  : the chosen shift
//	dplus  : D(+), length n
//	lplus  : L(+)'s subdiagonal, length n-1
//	info   : 0 success, 1 failure
func dlarrf(
	n int,
	d, l, ld []float64,
	clstrt, clend int,
	w, wgap, werr []float64,
	spdiam, clgapl, clgapr, pivmin float64,
	dplus, lplus, work []float64,
) (sigma float64, info int) {
	const (
		one        = 1.0
		two        = 2.0
		four       = 4.0
		quart      = 0.25
		maxgrowth1 = 8.0
		maxgrowth2 = 8.0
		ktrymax    = 1
		sLeft      = 1
		sRight     = 2
		eps        = 2.2204460492503131e-16
	)
	if n <= 0 {
		return 0, 0
	}

	fact := math.Pow(2, float64(ktrymax))
	shift := 0
	forcer := false

	// NOFAIL = .FALSE. (per reference quick-fix for bug 113).
	nofail := false

	// Average gap of the cluster (clend > clstrt assumed at call site;
	// guard against divide-by-zero).
	clwdth := math.Abs(w[clend-1]-w[clstrt-1]) + werr[clend-1] + werr[clstrt-1]
	denom := float64(clend - clstrt)
	avgap := 0.0
	if denom > 0 {
		avgap = clwdth / denom
	}
	mingap := math.Min(clgapl, clgapr)

	// Initial shifts to both ends of the cluster.
	lsigma := math.Min(w[clstrt-1], w[clend-1]) - werr[clstrt-1]
	rsigma := math.Max(w[clstrt-1], w[clend-1]) + werr[clend-1]

	// Small fudge to push the shift just outside the cluster.
	lsigma -= math.Abs(lsigma) * four * eps
	rsigma += math.Abs(rsigma) * four * eps

	// Upper bounds on how far we may back off.
	ldmax := quart*mingap + two*pivmin
	rdmax := quart*mingap + two*pivmin

	ldelta := math.Max(avgap, wgap[clstrt-1]) / fact
	rdelta := math.Max(avgap, wgap[clend-2]) / fact

	// Best representation tracking.
	sSafmin := lapackSafmin
	smlgrowth := one / sSafmin
	fail := float64(n-1) * mingap / (spdiam * eps)
	fail2 := float64(n-1) * mingap / (spdiam * math.Sqrt(eps))
	bestshift := lsigma

	ktry := 0
	growthbound := maxgrowth1 * spdiam

	var max1, max2 float64
	var sawnan1, sawnan2 bool
	var indx int

restart:
	for {
		sawnan1 = false
		sawnan2 = false
		// Don't back off too far.
		if ldelta > ldmax {
			ldelta = ldmax
		}
		if rdelta > rdmax {
			rdelta = rdmax
		}

		// --- Left end factorization ---
		s := -lsigma
		dplus[0] = d[0] + s
		if math.Abs(dplus[0]) < pivmin {
			dplus[0] = -pivmin
			sawnan1 = true
		}
		max1 = math.Abs(dplus[0])
		for i := 0; i < n-1; i++ {
			lplus[i] = ld[i] / dplus[i]
			s = s*lplus[i]*l[i] - lsigma
			dplus[i+1] = d[i+1] + s
			if math.Abs(dplus[i+1]) < pivmin {
				dplus[i+1] = -pivmin
				sawnan1 = true
			}
			if a := math.Abs(dplus[i+1]); a > max1 {
				max1 = a
			}
		}
		if math.IsNaN(max1) {
			sawnan1 = true
		}
		if forcer || (max1 <= growthbound && !sawnan1) {
			sigma = lsigma
			shift = sLeft
			break restart
		}

		// --- Right end factorization (into work[0..n-1] and work[n..2n-2]) ---
		s = -rsigma
		work[0] = d[0] + s
		if math.Abs(work[0]) < pivmin {
			work[0] = -pivmin
			sawnan2 = true
		}
		max2 = math.Abs(work[0])
		for i := 0; i < n-1; i++ {
			work[n+i] = ld[i] / work[i]
			s = s*work[n+i]*l[i] - rsigma
			work[i+1] = d[i+1] + s
			if math.Abs(work[i+1]) < pivmin {
				work[i+1] = -pivmin
				sawnan2 = true
			}
			if a := math.Abs(work[i+1]); a > max2 {
				max2 = a
			}
		}
		if math.IsNaN(max2) {
			sawnan2 = true
		}
		if forcer || (max2 <= growthbound && !sawnan2) {
			sigma = rsigma
			shift = sRight
			break restart
		}

		// Both shifts had too much growth — record the better one.
		if !(sawnan1 && sawnan2) {
			if !sawnan1 {
				indx = 1
				if max1 <= smlgrowth {
					smlgrowth = max1
					bestshift = lsigma
				}
			}
			if !sawnan2 {
				if sawnan1 || max2 <= max1 {
					indx = 2
				}
				if max2 <= smlgrowth {
					smlgrowth = max2
					bestshift = rsigma
				}
			}

			// Refined RRR test for isolated clusters.
			dorrr1 := clwdth < mingap/128.0 &&
				math.Min(max1, max2) < fail2 &&
				!sawnan1 && !sawnan2
			tryrrr1 := true
			if tryrrr1 && dorrr1 {
				if indx == 1 {
					tmp := math.Abs(dplus[n-1])
					znm2 := one
					prod := one
					oldp := one
					for i := n - 1; i >= 1; i-- {
						if prod <= eps {
							prod = ((dplus[i] * work[n+i]) / (dplus[i-1] * work[n+i-1])) * oldp
						} else {
							prod = prod * math.Abs(work[n+i-1])
						}
						oldp = prod
						znm2 += prod * prod
						v := math.Abs(dplus[i-1] * prod)
						if v > tmp {
							tmp = v
						}
					}
					rrr1 := tmp / (spdiam * math.Sqrt(znm2))
					if rrr1 <= maxgrowth2 {
						sigma = lsigma
						shift = sLeft
						break restart
					}
				} else if indx == 2 {
					tmp := math.Abs(work[n-1])
					znm2 := one
					prod := one
					oldp := one
					for i := n - 1; i >= 1; i-- {
						if prod <= eps {
							prod = ((work[i] * lplus[i]) / (work[i-1] * lplus[i-1])) * oldp
						} else {
							prod = prod * math.Abs(lplus[i-1])
						}
						oldp = prod
						znm2 += prod * prod
						v := math.Abs(work[i-1] * prod)
						if v > tmp {
							tmp = v
						}
					}
					rrr2 := tmp / (spdiam * math.Sqrt(znm2))
					if rrr2 <= maxgrowth2 {
						sigma = rsigma
						shift = sRight
						break restart
					}
				}
			}
		}

		// Refined RRR didn't help — either back off or give up.
		if ktry < ktrymax {
			lsigma = math.Max(lsigma-ldelta, lsigma-ldmax)
			rsigma = math.Min(rsigma+rdelta, rsigma+rdmax)
			ldelta = two * ldelta
			rdelta = two * rdelta
			ktry++
			continue restart
		}
		// Take best representation seen (forcer mode), if acceptable.
		if smlgrowth < fail || nofail {
			lsigma = bestshift
			rsigma = bestshift
			forcer = true
			continue restart
		}
		return 0, 1
	}

	// Convergence: if we accepted the right shift, copy work into dplus/lplus.
	if shift == sRight {
		copy(dplus[:n], work[:n])
		copy(lplus[:n-1], work[n:n+(n-1)])
	}
	_ = sSafmin
	return sigma, 0
}
