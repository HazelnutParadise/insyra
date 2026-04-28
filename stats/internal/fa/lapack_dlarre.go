// fa/lapack_dlarre.go
//
// dlarre — given a tridiagonal matrix T, set up base RRRs (one per
// split block) and compute initial eigenvalue approximations to be
// refined later by dlarrv.
//
// Faithful translation of LAPACK 3.12.1 dlarre.f. dqds calls into
// gonum's Dlasq2 (already a faithful Go port of the reference).
package fa

import (
	"math"

	"gonum.org/v1/gonum/lapack/gonum"
)

// dlarre top-level signature.
//
// Inputs:
//
//	rang        : 'A', 'V', 'I'
//	n           : matrix order
//	vl, vu      : value range (in/out — set on output for ALLRNG)
//	il, iu      : index range
//	d, e, e2    : diagonal, off-diagonal, off-diagonal² (modified in place;
//	              e/e2 zeroed at split points; d/e overwritten with LDL^T
//	              factors per block)
//	rtol1, rtol2: bisection tolerances (only when bisection is used)
//	spltol      : split tolerance (passed to dlarra)
//	work        : workspace, length 6*n
//	iwork       : workspace, length 5*n
//
// Outputs:
//
//	nsplit, isplit              : block partition (length n)
//	m, w, werr, wgap            : eigenvalue count + per-eigenvalue data
//	iblock, indexw              : block index / within-block index
//	gers                        : Gerschgorin intervals (length 2n)
//	pivmin                      : minimum pivot used
//	info                        : 0 ok; nonzero per reference
//
// Returns updated (vl, vu, m, nsplit, pivmin, info).
func dlarre(
	rang byte,
	n int,
	vl, vu float64,
	il, iu int,
	d, e, e2 []float64,
	rtol1, rtol2, spltol float64,
	isplit []int,
	w, werr, wgap []float64,
	iblock, indexw []int,
	gers []float64,
	work []float64,
	iwork []int,
) (vlOut, vuOut float64, m, nsplit int, pivmin float64, info int) {

	const (
		fac       = 0.5
		hndrd     = 100.0
		pert      = 8.0
		maxgrowth = 64.0
		fudge     = 2.0
		maxtry    = 6
		allRng    = 1
		indRng    = 2
		valRng    = 3
		eps       = 2.2204460492503131e-16
		safmin    = lapackSafmin
	)

	vlOut, vuOut = vl, vu
	if n <= 0 {
		return vlOut, vuOut, 0, 0, 0, 0
	}

	var irange int
	switch rang {
	case 'A', 'a':
		irange = allRng
	case 'V', 'v':
		irange = valRng
	case 'I', 'i':
		irange = indRng
	}

	rtl := math.Sqrt(eps)
	bsrtol := math.Sqrt(eps)

	// 1x1 quick return.
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
			wgap[0] = 0
			iblock[0] = 1
			indexw[0] = 1
			gers[0] = d[0]
			gers[1] = d[0]
		}
		e[0] = 0
		return vlOut, vuOut, m, 0, safmin, 0
	}

	// Init WERR/WGAP/Gerschgorin and find spectrum bounds.
	gl := d[0]
	gu := d[0]
	eold := 0.0
	emax := 0.0
	e[n-1] = 0
	for i := 1; i <= n; i++ {
		werr[i-1] = 0
		wgap[i-1] = 0
		var eabs float64
		if i <= n-1 {
			eabs = math.Abs(e[i-1])
		} else {
			eabs = 0
		}
		if eabs >= emax {
			emax = eabs
		}
		tmp1 := eabs + eold
		gers[2*(i-1)] = d[i-1] - tmp1
		if gers[2*(i-1)] < gl {
			gl = gers[2*(i-1)]
		}
		gers[2*(i-1)+1] = d[i-1] + tmp1
		if gers[2*(i-1)+1] > gu {
			gu = gers[2*(i-1)+1]
		}
		eold = eabs
	}
	pivmin = safmin * math.Max(1, emax*emax)
	spdiam := gu - gl

	// Splitting points.
	nsplit, _ = dlarra(n, d, e, e2, spltol, spdiam, isplit)

	// Initialize USEDQD; ALLRNG uses dqds by default.
	forceB := false
	usedqd := irange == allRng && !forceB

	var mm int
	if irange == allRng && !forceB {
		vlOut = gl
		vuOut = gu
	} else {
		// Crude approximations + (vl, vu) interval via dlarrd.
		var iinfo int
		var vlNew, vuNew float64
		mm, vlNew, vuNew, iinfo = dlarrd(rang, 'B', n,
			vlOut, vuOut, il, iu, gers, bsrtol, d, e, e2, pivmin, nsplit, isplit,
			w, werr, iblock, indexw, work, iwork)
		if iinfo != 0 {
			return vlOut, vuOut, m, nsplit, pivmin, -1
		}
		vlOut, vuOut = vlNew, vuNew
		// Zero out trailing entries.
		for i := mm + 1; i <= n; i++ {
			w[i-1] = 0
			werr[i-1] = 0
			iblock[i-1] = 0
			indexw[i-1] = 0
		}
	}

	// Loop over unreduced blocks.
	ibegin := 1
	wbegin := 1
	for jblk := 1; jblk <= nsplit; jblk++ {
		iend := isplit[jblk-1]
		in := iend - ibegin + 1

		if in == 1 {
			// 1x1 block.
			take := false
			switch {
			case irange == allRng:
				take = true
			case irange == valRng:
				take = d[ibegin-1] > vlOut && d[ibegin-1] <= vuOut
			case irange == indRng:
				take = iblock[wbegin-1] == jblk
			}
			if take {
				m++
				w[m-1] = d[ibegin-1]
				werr[m-1] = 0
				wgap[m-1] = 0
				iblock[m-1] = jblk
				indexw[m-1] = 1
				wbegin++
			}
			e[iend-1] = 0
			ibegin = iend + 1
			continue
		}

		e[iend-1] = 0
		// Local Gerschgorin for the block.
		gl = d[ibegin-1]
		gu = d[ibegin-1]
		for i := ibegin; i <= iend; i++ {
			if gers[2*(i-1)] < gl {
				gl = gers[2*(i-1)]
			}
			if gers[2*(i-1)+1] > gu {
				gu = gers[2*(i-1)+1]
			}
		}
		spdiam = gu - gl

		var mb, indl, indu, wend int
		if !(irange == allRng && !forceB) {
			// Count eigenvalues in this block.
			mb = 0
			for i := wbegin; i <= mm; i++ {
				if iblock[i-1] == jblk {
					mb++
				} else {
					break
				}
			}
			if mb == 0 {
				e[iend-1] = 0
				ibegin = iend + 1
				continue
			}
			usedqd = mb > int(fac*float64(in)) && !forceB
			wend = wbegin + mb - 1
			sigma0 := 0.0
			for i := wbegin; i <= wend-1; i++ {
				gap := w[i] - werr[i] - (w[i-1] + werr[i-1])
				if gap < 0 {
					gap = 0
				}
				wgap[i-1] = gap
			}
			gapEnd := vuOut - sigma0 - (w[wend-1] + werr[wend-1])
			if gapEnd < 0 {
				gapEnd = 0
			}
			wgap[wend-1] = gapEnd
			indl = indexw[wbegin-1]
			indu = indexw[wend-1]
		}

		var isleft, isrght float64
		if (irange == allRng && !forceB) || usedqd {
			// dqds path: extremal eigenvalues via dlarrk.
			tmp, tmp1, iinfo := dlarrk(in, 1, gl, gu, d[ibegin-1:], e2[ibegin-1:], pivmin, rtl)
			if iinfo != 0 {
				return vlOut, vuOut, m, nsplit, pivmin, -1
			}
			isleft = math.Max(gl, tmp-tmp1-hndrd*eps*math.Abs(tmp-tmp1))
			tmp, tmp1, iinfo = dlarrk(in, in, gl, gu, d[ibegin-1:], e2[ibegin-1:], pivmin, rtl)
			if iinfo != 0 {
				return vlOut, vuOut, m, nsplit, pivmin, -1
			}
			isrght = math.Min(gu, tmp+tmp1+hndrd*eps*math.Abs(tmp+tmp1))
			spdiam = isrght - isleft
		} else {
			isleft = math.Max(gl, w[wbegin-1]-werr[wbegin-1]-hndrd*eps*math.Abs(w[wbegin-1]-werr[wbegin-1]))
			isrght = math.Min(gu, w[wend-1]+werr[wend-1]+hndrd*eps*math.Abs(w[wend-1]+werr[wend-1]))
		}

		// Decide shift end and dqds vs bisection.
		var s1, s2 float64
		var cnt1, cnt2 int
		if irange == allRng && !forceB {
			usedqd = true
			indl = 1
			indu = in
			mb = in
			wend = wbegin + mb - 1
			s1 = isleft + 0.25*spdiam
			s2 = isrght - 0.25*spdiam
		} else {
			if usedqd {
				s1 = isleft + 0.25*spdiam
				s2 = isrght - 0.25*spdiam
			} else {
				tmp := math.Min(isrght, vuOut) - math.Max(isleft, vlOut)
				s1 = math.Max(isleft, vlOut) + 0.25*tmp
				s2 = math.Min(isrght, vuOut) - 0.25*tmp
			}
		}

		if mb > 1 {
			_, cnt1, cnt2, _ = dlarrc('T', in, s1, s2, d[ibegin-1:], e[ibegin-1:], pivmin)
		}

		var sigma, sgndef float64
		switch {
		case mb == 1:
			sigma = gl
			sgndef = 1
		case cnt1-indl >= indu-cnt2:
			switch {
			case irange == allRng && !forceB:
				sigma = math.Max(isleft, gl)
			case usedqd:
				sigma = isleft
			default:
				sigma = math.Max(isleft, vlOut)
			}
			sgndef = 1
		default:
			switch {
			case irange == allRng && !forceB:
				sigma = math.Min(isrght, gu)
			case usedqd:
				sigma = isrght
			default:
				sigma = math.Min(isrght, vuOut)
			}
			sgndef = -1
		}

		// Initial increment TAU.
		var tau float64
		if usedqd {
			tau = spdiam*eps*float64(n) + 2*pivmin
			tau = math.Max(tau, 2*eps*math.Abs(sigma))
		} else if mb > 1 {
			clwdth := w[wend-1] + werr[wend-1] - w[wbegin-1] - werr[wbegin-1]
			avgap := math.Abs(clwdth / float64(wend-wbegin))
			if sgndef == 1 {
				tau = 0.5 * math.Max(wgap[wbegin-1], avgap)
				tau = math.Max(tau, werr[wbegin-1])
			} else {
				tau = 0.5 * math.Max(wgap[wend-2], avgap)
				tau = math.Max(tau, werr[wend-1])
			}
		} else {
			tau = werr[wbegin-1]
		}

		// MAXTRY iterations of factorization with not-too-much element growth.
		representationFound := false
		for idum := 1; idum <= maxtry; idum++ {
			dpivot := d[ibegin-1] - sigma
			work[0] = dpivot
			dmax := math.Abs(work[0])
			j := ibegin
			for i := 1; i <= in-1; i++ {
				work[2*in+i-1] = 1.0 / work[i-1]
				tmp := e[j-1] * work[2*in+i-1]
				work[in+i-1] = tmp
				dpivot = (d[j] - sigma) - tmp*e[j-1]
				work[i] = dpivot
				if math.Abs(dpivot) > dmax {
					dmax = math.Abs(dpivot)
				}
				j++
			}
			norep := dmax > maxgrowth*spdiam
			if usedqd && !norep {
				for i := 0; i < in; i++ {
					if sgndef*work[i] < 0 {
						norep = true
						break
					}
				}
			}
			if norep {
				if idum == maxtry-1 {
					if sgndef == 1 {
						sigma = gl - fudge*spdiam*eps*float64(n) - fudge*2*pivmin
					} else {
						sigma = gu + fudge*spdiam*eps*float64(n) + fudge*2*pivmin
					}
				} else {
					sigma = sigma - sgndef*tau
					tau = 2 * tau
				}
			} else {
				representationFound = true
				break
			}
		}
		if !representationFound {
			return vlOut, vuOut, m, nsplit, pivmin, 2
		}

		// Store shift and the new D, L into D[ibegin..iend], E[ibegin..iend-1].
		e[iend-1] = sigma
		copy(d[ibegin-1:ibegin-1+in], work[:in])
		copy(e[ibegin-1:ibegin-1+in-1], work[in:in+in-1])

		if mb > 1 {
			// Perturb each entry by a small random relative amount.
			iseed := []int{1, 1, 1, 1}
			dlarnv(2, 2*in-1, iseed, work)
			for i := 1; i <= in-1; i++ {
				d[ibegin+i-2] = d[ibegin+i-2] * (1 + eps*pert*work[i-1])
				e[ibegin+i-2] = e[ibegin+i-2] * (1 + eps*pert*work[in+i-1])
			}
			d[iend-1] = d[iend-1] * (1 + eps*4*work[in-1])
		}

		// Compute eigenvalues either by bisection or dqds.
		if !usedqd {
			// Shift approximations to representation; refine via dlarrb.
			for j := wbegin; j <= wend; j++ {
				w[j-1] -= sigma
				werr[j-1] += math.Abs(w[j-1]) * eps
			}
			// LLD = D * E^2 used by dlaneg via dlarrb.
			for i := ibegin; i <= iend-1; i++ {
				work[i-1] = d[i-1] * e[i-1] * e[i-1]
			}
			iinfo := dlarrb(in, d[ibegin-1:], work[ibegin-1:],
				indl, indu, rtol1, rtol2, indl-1,
				w[wbegin-1:], wgap[wbegin-1:], werr[wbegin-1:],
				work[2*n:], iwork, pivmin, spdiam, in)
			if iinfo != 0 {
				return vlOut, vuOut, m, nsplit, pivmin, -4
			}
			lastGap := (vuOut - sigma) - (w[wend-1] + werr[wend-1])
			if lastGap < 0 {
				lastGap = 0
			}
			wgap[wend-1] = lastGap
			for i := indl; i <= indu; i++ {
				m++
				iblock[m-1] = jblk
				indexw[m-1] = i
			}
		} else {
			// dqds via gonum.
			rtol := math.Log(float64(in)) * 4 * eps
			j := ibegin
			for i := 1; i <= in-1; i++ {
				work[2*i-2] = math.Abs(d[j-1])
				work[2*i-1] = e[j-1] * e[j-1] * work[2*i-2]
				j++
			}
			work[2*in-2] = math.Abs(d[iend-1])
			work[2*in-1] = 0
			var impl gonum.Implementation
			iinfo := impl.Dlasq2(in, work)
			if iinfo != 0 {
				return vlOut, vuOut, m, nsplit, pivmin, -5
			}
			for i := 0; i < in; i++ {
				if work[i] < 0 {
					return vlOut, vuOut, m, nsplit, pivmin, -6
				}
			}
			if sgndef > 0 {
				for i := indl; i <= indu; i++ {
					m++
					w[m-1] = work[in-i]
					iblock[m-1] = jblk
					indexw[m-1] = i
				}
			} else {
				for i := indl; i <= indu; i++ {
					m++
					w[m-1] = -work[i-1]
					iblock[m-1] = jblk
					indexw[m-1] = i
				}
			}
			for i := m - mb + 1; i <= m; i++ {
				werr[i-1] = rtol * math.Abs(w[i-1])
			}
			for i := m - mb + 1; i <= m-1; i++ {
				gap := w[i] - werr[i] - (w[i-1] + werr[i-1])
				if gap < 0 {
					gap = 0
				}
				wgap[i-1] = gap
			}
			lastGap := (vuOut - sigma) - (w[m-1] + werr[m-1])
			if lastGap < 0 {
				lastGap = 0
			}
			wgap[m-1] = lastGap
		}
		ibegin = iend + 1
		wbegin = wend + 1
	}

	return vlOut, vuOut, m, nsplit, pivmin, 0
}
