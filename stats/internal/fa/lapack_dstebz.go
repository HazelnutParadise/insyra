// fa/lapack_dstebz.go
//
// dstebz — compute selected eigenvalues of a symmetric tridiagonal
// matrix T by bisection (Sturm-sequence). Operates on T directly,
// not on an LDL^T factor (cf. dlarrd which works after a base RRR
// has been formed). Used by dsyevr in the bisection-fallback path.
//
// Faithful translation of LAPACK 3.12.1 dstebz.f.
package fa

import "math"

// dstebz returns selected eigenvalues of T.
//
// rang : 'A' (all), 'V' (interval (vl,vu]), 'I' (indices il..iu)
// order: 'B' (group by block, sort within), 'E' (sort entire matrix)
// abstol: absolute tolerance (use ulp*||T|| if <= 0)
// d, e : tridiag (length n, n-1)
// work : workspace, length 4n
// iwork: workspace, length 3n
//
// Outputs:
//
//	m, w[1..m]    : eigenvalues
//	nsplit, isplit: block partition
//	iblock[1..m]  : block index for each eigenvalue
func dstebz(
	rang, order byte,
	n int,
	vl, vu float64,
	il, iu int,
	abstol float64,
	d, e []float64,
	w []float64,
	iblock []int,
	isplit []int,
	work []float64,
	iwork []int,
) (m, nsplit, info int) {
	const (
		fudge  = 2.1
		relfac = 2.0
		eps    = 2.2204460492503131e-16
		safmin = lapackSafmin
	)
	var irange int
	switch rang {
	case 'A', 'a':
		irange = 1
	case 'V', 'v':
		irange = 2
	case 'I', 'i':
		irange = 3
	default:
		return 0, 0, -1
	}
	var iorder int
	switch order {
	case 'B', 'b':
		iorder = 2
	case 'E', 'e':
		iorder = 1
	default:
		return 0, 0, -2
	}
	if n < 0 {
		return 0, 0, -3
	}
	if irange == 2 && vl >= vu {
		return 0, 0, -5
	}
	if irange == 3 && (il < 1 || il > max(1, n)) {
		return 0, 0, -6
	}
	if irange == 3 && (iu < min(n, il) || iu > n) {
		return 0, 0, -7
	}

	ncnvrg := false
	toofew := false
	if n == 0 {
		return 0, 0, 0
	}
	if irange == 3 && il == 1 && iu == n {
		irange = 1
	}

	rtoli := eps * relfac
	nb := 0

	if n == 1 {
		nsplit = 1
		isplit[0] = 1
		if irange == 2 && (vl >= d[0] || vu < d[0]) {
			return 0, 1, 0
		}
		w[0] = d[0]
		iblock[0] = 1
		return 1, 1, 0
	}

	// Compute splitting points and squared off-diagonals into work[0..n-2].
	nsplit = 1
	work[n-1] = 0
	pivmin := 1.0
	for j := 2; j <= n; j++ {
		tmp1 := e[j-2] * e[j-2]
		if math.Abs(d[j-1]*d[j-2])*eps*eps+safmin > tmp1 {
			isplit[nsplit-1] = j - 1
			nsplit++
			work[j-2] = 0
		} else {
			work[j-2] = tmp1
			if tmp1 > pivmin {
				pivmin = tmp1
			}
		}
	}
	isplit[nsplit-1] = n
	pivmin *= safmin

	// Compute interval and ATOLI.
	var atoli, wl, wu, wlu, wul float64
	var nwl, nwu int
	if irange == 3 {
		gu := d[0]
		gl := d[0]
		tmp1 := 0.0
		for j := 1; j <= n-1; j++ {
			tmp2 := math.Sqrt(work[j-1])
			if v := d[j-1] + tmp1 + tmp2; v > gu {
				gu = v
			}
			if v := d[j-1] - tmp1 - tmp2; v < gl {
				gl = v
			}
			tmp1 = tmp2
		}
		if v := d[n-1] + tmp1; v > gu {
			gu = v
		}
		if v := d[n-1] - tmp1; v < gl {
			gl = v
		}
		tnorm := math.Max(math.Abs(gl), math.Abs(gu))
		gl = gl - fudge*tnorm*eps*float64(n) - fudge*2*pivmin
		gu = gu + fudge*tnorm*eps*float64(n) + fudge*pivmin

		if abstol <= 0 {
			atoli = eps * tnorm
		} else {
			atoli = abstol
		}

		// Set up dlaebz IJOB=3 inputs (mmax=2, minp=2).
		// nval = iwork[4..5] = [il-1, iu]; nab = iwork[0..3];
		// ab = work[n..n+3]; c = work[n+4..n+5].
		ab := []float64{gl, gl, gu, gu}
		cBuf := []float64{gl, gu}
		nab := []int{-1, -1, n + 1, n + 1}
		nval := []int{il - 1, iu}
		innerWork := make([]float64, 2)
		innerIwork := make([]int, 2)

		_, iinfo := dlaebz(3, int((math.Log(tnorm+pivmin)-math.Log(pivmin))/math.Log(2))+2,
			n, 2, 2, nb, atoli, rtoli, pivmin,
			d, e, work, // squared off-diagonals
			nval, ab, cBuf, nab, innerWork, innerIwork)
		if iinfo != 0 {
			return 0, nsplit, iinfo
		}

		if nval[1] == iu {
			wl = ab[0]
			wlu = ab[2]
			nwl = nab[0]
			wu = ab[3]
			wul = ab[1]
			nwu = nab[3]
		} else {
			wl = ab[1]
			wlu = ab[3]
			nwl = nab[1]
			wu = ab[2]
			wul = ab[0]
			nwu = nab[2]
		}
		if nwl < 0 || nwl >= n || nwu < 1 || nwu > n {
			return 0, nsplit, 4
		}
	} else {
		tnorm := math.Max(math.Abs(d[0])+math.Abs(e[0]),
			math.Abs(d[n-1])+math.Abs(e[n-2]))
		for j := 2; j <= n-1; j++ {
			if v := math.Abs(d[j-1]) + math.Abs(e[j-2]) + math.Abs(e[j-1]); v > tnorm {
				tnorm = v
			}
		}
		if abstol <= 0 {
			atoli = eps * tnorm
		} else {
			atoli = abstol
		}
		if irange == 2 {
			wl = vl
			wu = vu
		}
	}

	// Loop over blocks; reset NWL/NWU for re-accumulation.
	m = 0
	iend := 0
	nwl = 0
	nwu = 0
	idumma := []int{0}

	for jb := 1; jb <= nsplit; jb++ {
		ioff := iend
		ibegin := ioff + 1
		iend = isplit[jb-1]
		in := iend - ioff

		if in == 1 {
			if irange == 1 || wl >= d[ibegin-1]-pivmin {
				nwl++
			}
			if irange == 1 || wu >= d[ibegin-1]-pivmin {
				nwu++
			}
			if irange == 1 || (wl < d[ibegin-1]-pivmin && wu >= d[ibegin-1]-pivmin) {
				m++
				w[m-1] = d[ibegin-1]
				iblock[m-1] = jb
			}
			continue
		}

		// General case.
		gu := d[ibegin-1]
		gl := d[ibegin-1]
		tmp1 := 0.0
		for j := ibegin; j <= iend-1; j++ {
			tmp2 := math.Abs(e[j-1])
			if v := d[j-1] + tmp1 + tmp2; v > gu {
				gu = v
			}
			if v := d[j-1] - tmp1 - tmp2; v < gl {
				gl = v
			}
			tmp1 = tmp2
		}
		if v := d[iend-1] + tmp1; v > gu {
			gu = v
		}
		if v := d[iend-1] - tmp1; v < gl {
			gl = v
		}
		bnorm := math.Max(math.Abs(gl), math.Abs(gu))
		gl = gl - fudge*bnorm*eps*float64(in) - fudge*pivmin
		gu = gu + fudge*bnorm*eps*float64(in) + fudge*pivmin

		var blockAtoli float64
		if abstol <= 0 {
			blockAtoli = eps * math.Max(math.Abs(gl), math.Abs(gu))
		} else {
			blockAtoli = abstol
		}

		if irange > 1 {
			if gu < wl {
				nwl += in
				nwu += in
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

		// Per-block dlaebz IJOB=1 then 2 (mmax=in, minp=1).
		ab := make([]float64, 2*in)
		nab := make([]int, 2*in)
		cBuf := make([]float64, in)
		innerWork := make([]float64, in)
		innerIwork := make([]int, in)
		ab[0] = gl
		ab[in] = gu
		_, iinfo := dlaebz(1, 0, in, in, 1, nb, blockAtoli, rtoli, pivmin,
			d[ibegin-1:], e[ibegin-1:], work[ibegin-1:],
			idumma, ab, cBuf, nab, innerWork, innerIwork)
		if iinfo != 0 {
			return m, nsplit, iinfo
		}
		nwl += nab[0]
		nwu += nab[in]
		iwoff := m - nab[0]

		itmax := int((math.Log(gu-gl+pivmin)-math.Log(pivmin))/math.Log(2)) + 2
		iout, iinfo2 := dlaebz(2, itmax, in, in, 1, nb, blockAtoli, rtoli, pivmin,
			d[ibegin-1:], e[ibegin-1:], work[ibegin-1:],
			idumma, ab, cBuf, nab, innerWork, innerIwork)
		if iinfo2 != 0 {
			return m, nsplit, iinfo2
		}

		// Copy eigenvalues into W and IBLOCK.
		for j := 1; j <= iout; j++ {
			tmp1 := 0.5 * (ab[j-1] + ab[in+j-1])
			var ib int
			if j > iout-iinfo2 {
				ncnvrg = true
				ib = -jb
			} else {
				ib = jb
			}
			for je := nab[j-1] + 1 + iwoff; je <= nab[in+j-1]+iwoff; je++ {
				if je >= 1 && je <= len(w) {
					w[je-1] = tmp1
					iblock[je-1] = ib
				}
			}
		}
		im := 0
		for j := 1; j <= iout; j++ {
			im += nab[in+j-1] - nab[j-1]
		}
		m += im
	}

	// Discard logic for INDRNG.
	if irange == 3 {
		idiscl := il - 1 - nwl
		idiscu := nwu - iu
		if idiscl > 0 || idiscu > 0 {
			im := 0
			for je := 1; je <= m; je++ {
				if w[je-1] <= wlu && idiscl > 0 {
					idiscl--
				} else if w[je-1] >= wul && idiscu > 0 {
					idiscu--
				} else {
					im++
					w[im-1] = w[je-1]
					iblock[im-1] = iblock[je-1]
				}
			}
			m = im
		}
		if idiscl > 0 || idiscu > 0 {
			// Bad-arithmetic recovery: kill smallest/largest by iblock=0.
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
						if iblock[je-1] != 0 && (w[je-1] > wkill || iw == 0) {
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
					iblock[im-1] = iblock[je-1]
				}
			}
			m = im
		}
		if idiscl < 0 || idiscu < 0 {
			toofew = true
		}
	}

	// Sort if ORDER='E' and nsplit>1.
	if iorder == 1 && nsplit > 1 {
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
				itmp1 := iblock[ie-1]
				w[ie-1] = w[je-1]
				iblock[ie-1] = iblock[je-1]
				w[je-1] = tmp1
				iblock[je-1] = itmp1
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
	return m, nsplit, info
}
