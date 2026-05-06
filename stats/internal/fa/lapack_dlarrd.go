// fa/lapack_dlarrd.go
//
// dlarrd — compute the eigenvalues of a symmetric tridiagonal matrix
// to suitable accuracy. Auxiliary code called by dstemr (and dstebz).
//
// Faithful translation of LAPACK 3.12.1 reference Fortran (dlarrd.f).
package fa

import "math"

// dlarrd computes selected eigenvalues of a symmetric tridiagonal
// matrix T to suitable accuracy by bisection.
//
// Inputs:
//
//	rang    : 'A' (all), 'V' (interval (vl,vu]), 'I' (indices il..iu)
//	order   : 'B' (group by block), 'E' (sort entire matrix)
//	n       : matrix order
//	vl, vu  : value range (used only when rang=='V')
//	il, iu  : index range (used only when rang=='I')
//	gers    : Gerschgorin intervals, length 2n; (gers[2i], gers[2i+1]) for i=0..n-1
//	reltol  : minimum relative interval width
//	d       : diagonal, length n
//	e       : off-diagonal, length n-1
//	e2      : squared off-diagonal, length n-1
//	pivmin  : minimum pivot magnitude
//	nsplit  : number of diagonal blocks
//	isplit  : splitting indices, length nsplit
//
// Outputs:
//
//	m       : number of eigenvalues found
//	w       : eigenvalue approximations (length n; first m valid)
//	werr    : error bounds (length n; first m valid)
//	wl, wu  : interval enclosing all wanted eigenvalues
//	iblock  : block index for each eigenvalue (length n)
//	indexw  : within-block index for each eigenvalue (length n)
//	work    : workspace, length 4n
//	iwork   : workspace, length 3n
//	info    : 0 ok; >0 various non-fatal issues; <0 illegal arg
func dlarrd(
	rang, order byte,
	n int,
	vl, vu float64,
	il, iu int,
	gers []float64,
	reltol float64,
	d, e, e2 []float64,
	pivmin float64,
	nsplit int,
	isplit []int,
	w, werr []float64,
	iblock, indexw []int,
	work []float64,
	iwork []int,
) (m int, wl, wu float64, info int) {

	const (
		fudge   = 2.0
		allRng  = 1
		valRng  = 2
		indRng  = 3
		safmin  = lapackSafmin
		eps     = 2.2204460492503131e-16
		uflow   = lapackSafmin
	)

	m = 0
	if n <= 0 {
		return 0, 0, 0, 0
	}

	// Decode RANGE.
	var irange int
	switch rang {
	case 'A', 'a':
		irange = allRng
	case 'V', 'v':
		irange = valRng
	case 'I', 'i':
		irange = indRng
	default:
		return 0, 0, 0, -1
	}

	// Validate ORDER.
	if !(order == 'B' || order == 'b' || order == 'E' || order == 'e') {
		return 0, 0, 0, -2
	}
	if irange == valRng && vl >= vu {
		return 0, 0, 0, -5
	}
	if irange == indRng && (il < 1 || il > max(1, n)) {
		return 0, 0, 0, -6
	}
	if irange == indRng && (iu < min(n, il) || iu > n) {
		return 0, 0, 0, -7
	}

	ncnvrg := false
	toofew := false

	// Simplification: 'I' with full range becomes 'A'.
	if irange == indRng && il == 1 && iu == n {
		irange = allRng
	}

	// Special case n=1.
	if n == 1 {
		take := false
		switch irange {
		case allRng:
			take = true
		case valRng:
			take = d[0] > vl && d[0] <= vu
		case indRng:
			take = il == 1 && iu == 1
		}
		if take {
			m = 1
			w[0] = d[0]
			werr[0] = 0
			iblock[0] = 1
			indexw[0] = 1
		}
		return m, wl, wu, 0
	}

	// NB: minimum vector length for vector bisection. Our dlaebz works
	// fine with the scalar path; force serial.
	nb := 0

	// Find global spectral radius from Gerschgorin intervals.
	gl := d[0]
	gu := d[0]
	for i := 1; i <= n; i++ {
		if gers[2*(i-1)] < gl {
			gl = gers[2*(i-1)]
		}
		if gers[2*(i-1)+1] > gu {
			gu = gers[2*(i-1)+1]
		}
	}
	tnorm := math.Max(math.Abs(gl), math.Abs(gu))
	gl = gl - fudge*tnorm*eps*float64(n) - fudge*2*pivmin
	gu = gu + fudge*tnorm*eps*float64(n) + fudge*2*pivmin

	rtoli := reltol
	atoli := fudge*2*uflow + fudge*2*pivmin

	// IDUMMA: nval workspace for IJOB=1 calls (size 1, never read).
	idumma := []int{0}

	if irange == indRng {
		// RANGE='I': bisection-invert N(w) to get the (il-1, iu] index
		// interval, with mmax=2, minp=2.
		itmax := int((math.Log(tnorm+pivmin)-math.Log(pivmin))/math.Log(2)) + 2

		// Set up ab (mmax=2, layout: ab[(jp-1)*2 + (ji-1)]) and nab.
		ab := make([]float64, 4)
		nab := make([]int, 4)
		cBuf := []float64{gl, gu}
		nval := []int{il - 1, iu}

		// AB(1,1)=GL, AB(2,1)=GL, AB(1,2)=GU, AB(2,2)=GU
		ab[0] = gl
		ab[1] = gl
		ab[2] = gu
		ab[3] = gu
		// NAB(1,1)=-1, NAB(2,1)=-1, NAB(1,2)=N+1, NAB(2,2)=N+1
		nab[0] = -1
		nab[1] = -1
		nab[2] = n + 1
		nab[3] = n + 1

		// Inner workspace: dlaebz needs work (mmax) and iwork (mmax),
		// length 2 each.
		innerWork := make([]float64, 2)
		innerIwork := make([]int, 2)

		_, iinfo := dlaebz(3, itmax, n, 2, 2, nb, atoli, rtoli, pivmin,
			d, e, e2, nval, ab, cBuf, nab, innerWork, innerIwork)
		if iinfo != 0 {
			return m, wl, wu, iinfo
		}
		// Output intervals may not be ordered by ascending negcount.
		var nwl, nwu int
		var wlu, wul float64
		if nval[1] == iu {
			wl = ab[0]   // WORK(N+1) = AB(1,1)
			wlu = ab[2]  // WORK(N+3) = AB(1,2)
			nwl = nab[0] // IWORK(1) = NAB(1,1)
			wu = ab[3]   // WORK(N+4) = AB(2,2)
			wul = ab[1]  // WORK(N+2) = AB(2,1)
			nwu = nab[3] // IWORK(4) = NAB(2,2)
		} else {
			wl = ab[1]   // AB(2,1)
			wlu = ab[3]  // AB(2,2)
			nwl = nab[1] // NAB(2,1)
			wu = ab[2]   // AB(1,2)
			wul = ab[0]  // AB(1,1)
			nwu = nab[2] // NAB(1,2)
		}
		if nwl < 0 || nwl >= n || nwu < 1 || nwu > n {
			return m, wl, wu, 4
		}
		_ = nwl
		_ = nwu
		// IJOB=3 output (wlu/wul) is preserved for the discard tests.
		// NWL/NWU will be RECOMPUTED by the block loop below — the
		// reference resets these to 0 and re-accumulates per-block.
		wluKeep := wlu
		wulKeep := wul

		// Now perform the per-block eigenvalue refinement.
		mFound, nwlBlk, nwuBlk, infoOut := dlarrdBlockLoop(
			n, irange, nb, atoli, rtoli, pivmin, tnorm,
			d, e, e2, gers, isplit, nsplit,
			wl, wu,
			w, werr, iblock, indexw,
			work, iwork,
			&ncnvrg)
		if infoOut != 0 {
			return mFound, wl, wu, infoOut
		}
		m = mFound

		// Apply discard logic for INDRNG using block-loop-accumulated NWL/NWU.
		idiscl := il - 1 - nwlBlk
		idiscu := nwuBlk - iu

		if idiscl > 0 {
			im := 0
			for je := 1; je <= m; je++ {
				if w[je-1] <= wluKeep && idiscl > 0 {
					idiscl--
				} else {
					im++
					w[im-1] = w[je-1]
					werr[im-1] = werr[je-1]
					indexw[im-1] = indexw[je-1]
					iblock[im-1] = iblock[je-1]
				}
			}
			m = im
		}
		if idiscu > 0 {
			im := m + 1
			for je := m; je >= 1; je-- {
				if w[je-1] >= wulKeep && idiscu > 0 {
					idiscu--
				} else {
					im--
					w[im-1] = w[je-1]
					werr[im-1] = werr[je-1]
					indexw[im-1] = indexw[je-1]
					iblock[im-1] = iblock[je-1]
				}
			}
			jee := 0
			for je := im; je <= m; je++ {
				jee++
				w[jee-1] = w[je-1]
				werr[jee-1] = werr[je-1]
				indexw[jee-1] = indexw[je-1]
				iblock[jee-1] = iblock[je-1]
			}
			m = m - im + 1
		}

		if idiscl > 0 || idiscu > 0 {
			// Bad-arithmetic recovery: kill smallest idiscl / largest
			// idiscu eigenvalues by setting iblock=0, then compact.
			if idiscl > 0 {
				wkill := wu
				for jdisc := 1; jdisc <= idiscl; jdisc++ {
					iw := 0
					for je := 1; je <= m; je++ {
						if iblock[je-1] != 0 && (w[je-1] < wkill || iw == 0) {
							iw = je
							wkill = w[je-1]
						}
					}
					if iw != 0 {
						iblock[iw-1] = 0
					}
				}
			}
			if idiscu > 0 {
				wkill := wl
				for jdisc := 1; jdisc <= idiscu; jdisc++ {
					iw := 0
					for je := 1; je <= m; je++ {
						if iblock[je-1] != 0 && (w[je-1] >= wkill || iw == 0) {
							iw = je
							wkill = w[je-1]
						}
					}
					if iw != 0 {
						iblock[iw-1] = 0
					}
				}
			}
			im := 0
			for je := 1; je <= m; je++ {
				if iblock[je-1] != 0 {
					im++
					w[im-1] = w[je-1]
					werr[im-1] = werr[je-1]
					indexw[im-1] = indexw[je-1]
					iblock[im-1] = iblock[je-1]
				}
			}
			m = im
		}
		if idiscl < 0 || idiscu < 0 {
			toofew = true
		}
		_ = idumma
	} else {
		// RANGE = 'A' or 'V'.
		if irange == valRng {
			wl = vl
			wu = vu
		} else {
			wl = gl
			wu = gu
		}

		mFound, _, _, infoOut := dlarrdBlockLoop(
			n, irange, nb, atoli, rtoli, pivmin, tnorm,
			d, e, e2, gers, isplit, nsplit,
			wl, wu,
			w, werr, iblock, indexw,
			work, iwork,
			&ncnvrg)
		if infoOut != 0 {
			return mFound, wl, wu, infoOut
		}
		m = mFound
	}

	if (irange == allRng && m != n) || (irange == indRng && m != iu-il+1) {
		toofew = true
	}

	// ORDER='E' and nsplit>1: sort eigenvalues from smallest to largest.
	if (order == 'E' || order == 'e') && nsplit > 1 {
		for je := 1; je <= m-1; je++ {
			ie := 0
			tmp1 := w[je-1]
			for j := je + 1; j <= m; j++ {
				if w[j-1] < tmp1 {
					ie = j
					tmp1 = w[j-1]
				}
			}
			if ie != 0 {
				tmp2 := werr[ie-1]
				itmp1 := iblock[ie-1]
				itmp2 := indexw[ie-1]
				w[ie-1] = w[je-1]
				werr[ie-1] = werr[je-1]
				iblock[ie-1] = iblock[je-1]
				indexw[ie-1] = indexw[je-1]
				w[je-1] = tmp1
				werr[je-1] = tmp2
				iblock[je-1] = itmp1
				indexw[je-1] = itmp2
			}
		}
	}

	info = 0
	if ncnvrg {
		info++
	}
	if toofew {
		info += 2
	}
	_ = safmin
	return m, wl, wu, info
}

// dlarrdBlockLoop is the per-block eigenvalue computation factored out
// of dlarrd's main routine (the JBLK loop). Returns the cumulative m
// and any non-zero info from dlaebz calls.
func dlarrdBlockLoop(
	n int,
	irange int,
	nb int,
	atoli, rtoli, pivmin, tnorm float64,
	d, e, e2 []float64,
	gers []float64,
	isplit []int,
	nsplit int,
	wl, wu float64,
	w, werr []float64,
	iblock, indexw []int,
	work []float64,
	iwork []int,
	ncnvrg *bool,
) (m, nwl, nwu, info int) {
	const fudge = 2.0
	const eps = 2.2204460492503131e-16

	m = 0
	nwl = 0
	nwu = 0
	iend := 0

	idumma := []int{0}
	_ = idumma

	for jblk := 1; jblk <= nsplit; jblk++ {
		ioff := iend
		ibegin := ioff + 1
		iend = isplit[jblk-1]
		in := iend - ioff

		if in == 1 {
			// 1x1 block.
			if wl >= d[ibegin-1]-pivmin {
				nwl++
			}
			if wu >= d[ibegin-1]-pivmin {
				nwu++
			}
			take := false
			if irange == 1 /*allRng*/ {
				take = true
			} else if wl < d[ibegin-1]-pivmin && wu >= d[ibegin-1]-pivmin {
				take = true
			}
			if take {
				m++
				w[m-1] = d[ibegin-1]
				werr[m-1] = 0
				iblock[m-1] = jblk
				indexw[m-1] = 1
			}
			continue
		}

		// General block of size in >= 2.
		gl := d[ibegin-1]
		gu := d[ibegin-1]
		for j := ibegin; j <= iend; j++ {
			if gers[2*(j-1)] < gl {
				gl = gers[2*(j-1)]
			}
			if gers[2*(j-1)+1] > gu {
				gu = gers[2*(j-1)+1]
			}
		}
		gl = gl - fudge*tnorm*eps*float64(in) - fudge*pivmin
		gu = gu + fudge*tnorm*eps*float64(in) + fudge*pivmin

		if irange > 1 {
			if gu < wl {
				continue
			}
			if gl < wl {
				gl = wl
			}
			if gu > wu {
				gu = wu
			}
			if gl >= gu {
				continue
			}
		}

		// Allocate per-block workspace for dlaebz with mmax=in.
		// AB layout: 2*in flat slots. NAB layout: 2*in ints.
		// C: in floats. Inner work: in floats / in ints.
		ab := make([]float64, 2*in)
		nab := make([]int, 2*in)
		cBuf := make([]float64, in)
		innerWork := make([]float64, in)
		innerIwork := make([]int, in)

		// First call: IJOB=1, mmax=in, minp=1.
		ab[0] = gl    // AB(1,1)
		ab[in] = gu   // AB(1,2)  → flat index in*1 + 0 = in
		_, iinfo := dlaebz(1, 0, in, in, 1, nb, atoli, rtoli, pivmin,
			d[ibegin-1:], e[ibegin-1:], e2[ibegin-1:],
			idumma, ab, cBuf, nab, innerWork, innerIwork)
		if iinfo != 0 {
			return m, nwl, nwu, iinfo
		}
		// Accumulate per-block negcounts at GL/GU into NWL/NWU.
		nwl += nab[0]    // NAB(1,1)
		nwu += nab[in]   // NAB(1,2)
		iwoff := m - nab[0]

		// Second call: IJOB=2, mmax=in, minp=1.
		itmax := int((math.Log(gu-gl+pivmin)-math.Log(pivmin))/math.Log(2)) + 2
		iout, iinfo2 := dlaebz(2, itmax, in, in, 1, nb, atoli, rtoli, pivmin,
			d[ibegin-1:], e[ibegin-1:], e2[ibegin-1:],
			idumma, ab, cBuf, nab, innerWork, innerIwork)
		if iinfo2 != 0 {
			return m, nwl, nwu, iinfo2
		}

		// Copy eigenvalues into W, WERR, INDEXW, IBLOCK.
		// In the per-block dlaebz output:
		//   ab[(0)*in + j-1] = AB(j,1)
		//   ab[(1)*in + j-1] = AB(j,2)
		//   nab[(0)*in + j-1] = NAB(j,1)
		//   nab[(1)*in + j-1] = NAB(j,2)
		for j := 1; j <= iout; j++ {
			tmp1 := 0.5 * (ab[j-1] + ab[in+j-1])
			tmp2 := 0.5 * math.Abs(ab[j-1]-ab[in+j-1])
			var ib int
			if j > iout-iinfo2 {
				*ncnvrg = true
				ib = -jblk
			} else {
				ib = jblk
			}
			for je := nab[j-1] + 1 + iwoff; je <= nab[in+j-1]+iwoff; je++ {
				if je >= 1 && je <= len(w) {
					w[je-1] = tmp1
					werr[je-1] = tmp2
					indexw[je-1] = je - iwoff
					iblock[je-1] = ib
				}
			}
		}

		// Final iout's IM: the dlaebz return value `mout` for IJOB=2
		// is the number of intervals (kl), so the count of eigenvalues
		// computed in this block is the same iout count above (each
		// interval contributes nab[in+j-1] - nab[j-1] = 1 by IJOB=2's
		// guarantee that intervals only contain a single eigenvalue
		// upon convergence).
		// However, the reference accumulates `IM` as the IJOB=1 mout
		// (= total eigenvalues in the block region), so we follow that.
		// Re-derive im from (sum over j: nab[in+j-1] - nab[j-1]):
		im := 0
		for j := 1; j <= iout; j++ {
			im += nab[in+j-1] - nab[j-1]
		}
		m += im
	}
	return m, nwl, nwu, 0
}
