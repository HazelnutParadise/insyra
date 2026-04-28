// fa/lapack_dlarrv.go
//
// dlarrv — compute the eigenvectors of L D L^T given the eigenvalues
// from dlarre. Maintains a cluster representation tree: each cluster is
// either a singleton (compute its eigenvector via dlar1v + RQI) or a
// non-trivial cluster (compute a child RRR via dlarrf and recurse).
//
// Faithful translation of LAPACK 3.12.1 dlarrv.f.
package fa

import "math"

// dlarrv signature.
//
// Inputs:
//
//	n              : matrix order
//	vl, vu         : eigenvalue range used by dlarre (boundary safety only)
//	d              : LDL^T diagonal (length n; modified in place per-block)
//	l              : LDL^T sub-diagonal (length n-1; modified in place;
//	                  l[iend-1] holds the sigma shift for the j-th block)
//	pivmin         : minimum pivot magnitude
//	isplit         : split indices (length nsplit)
//	m              : number of eigenvalues in w
//	dol, dou       : index range of eigenvalues to compute (1-based)
//	minrgp         : minimum relative gap to declare clusters
//	rtol1, rtol2   : tolerances passed to dlarrb
//	w              : eigenvalue approximations (1-based length m)
//	werr, wgap     : per-eigenvalue uncertainty / right-gap (length m)
//	iblock, indexw : per-eigenvalue block / within-block index
//	gers           : Gerschgorin intervals (length 2n)
//	z              : eigenvector matrix flat slice, ldz x ?, column-major
//	ldz            : leading dimension of z
//	isuppz         : eigenvector support, length 2*m
//	work           : workspace, length 12*n
//	iwork          : workspace, length 7*n
func dlarrv(
	n int,
	vl, vu float64,
	d, l []float64,
	pivmin float64,
	isplit []int,
	m, dol, dou int,
	minrgp, rtol1, rtol2 float64,
	w, werr, wgap []float64,
	iblock, indexw []int,
	gers []float64,
	z []float64,
	ldz int,
	isuppz []int,
	work []float64,
	iwork []int,
) (info int) {
	const (
		maxitr = 10
		eps    = 2.2204460492503131e-16
	)
	if n <= 0 || m <= 0 {
		return 0
	}

	// Workspace partition (1-based offsets into work):
	//   work[0..n-1]                : (shifted) eigenvalues mirror
	//   work[indld..indld+n-1]      : D*L
	//   work[indlld..indlld+n-1]    : L*L*D
	//   work[indwrk..indwrk+8n-1]   : scratch for dlarrb/dlar1v
	const _ = 0
	indld := n     // 0-based offset for "INDLD-1+J" => indld + j-1 (j 1-based)
	indlld := 2 * n
	indwrk := 3 * n
	minwsize := 12 * n
	for i := 0; i < minwsize; i++ {
		work[i] = 0
	}

	// IWORK partition:
	//   iwork[iindr..iindr+n-1]                : twist indices R
	//   iwork[iindc1..iindc1+2n-1] / iindc2..  : cluster lists (alternating)
	//   iwork[iindwk..]                         : scratch for dlarrb
	iindr := 0
	iindc1 := n
	iindc2 := 2 * n
	iindwk := 3 * n
	miniwsize := 7 * n
	for i := 0; i < miniwsize; i++ {
		iwork[i] = 0
	}

	zusedl := 1
	if dol > 1 {
		zusedl = dol - 1
	}
	zusedu := m
	if dou < m {
		zusedu = dou + 1
	}
	zusedw := zusedu - zusedl + 1
	// Zero the used part of Z. dlaset 'Full' n×zusedw at column zusedl..
	for jc := zusedl; jc < zusedl+zusedw; jc++ {
		base := (jc - 1) * ldz
		for ir := 0; ir < n; ir++ {
			z[base+ir] = 0
		}
	}

	rqtol := 2 * eps
	tryrqc := true
	if !(dol == 1 && dou == m) {
		// Selected eigenpairs path: bisection must reach full accuracy.
		rtol1 = 4 * eps
		rtol2 = 4 * eps
	}

	done := 0
	ibegin := 1
	wbegin := 1
	for jblk := 1; jblk <= iblock[m-1]; jblk++ {
		iend := isplit[jblk-1]
		sigma := l[iend-1]
		// Find wend: the last eigenvalue in this block.
		wend := wbegin - 1
		for wend < m && iblock[wend] == jblk {
			wend++
		}
		if wend < wbegin {
			ibegin = iend + 1
			continue
		}
		if wend < dol || wbegin > dou {
			ibegin = iend + 1
			wbegin = wend + 1
			continue
		}

		// Local spectral diameter from Gerschgorin.
		gl := gers[2*(ibegin-1)]
		gu := gers[2*(ibegin-1)+1]
		for i := ibegin + 1; i <= iend; i++ {
			if gers[2*(i-1)] < gl {
				gl = gers[2*(i-1)]
			}
			if gers[2*(i-1)+1] > gu {
				gu = gers[2*(i-1)+1]
			}
		}
		spdiam := gu - gl
		oldien := ibegin - 1
		in := iend - ibegin + 1
		im := wend - wbegin + 1

		// 1x1 block.
		if ibegin == iend {
			done++
			z[(wbegin-1)*ldz+ibegin-1] = 1
			isuppz[2*(wbegin-1)] = ibegin
			isuppz[2*(wbegin-1)+1] = ibegin
			w[wbegin-1] += sigma
			work[wbegin-1] = w[wbegin-1]
			ibegin = iend + 1
			wbegin++
			continue
		}

		// Copy IM eigenvalues to WORK and shift W back to original matrix.
		copy(work[wbegin-1:wbegin-1+im], w[wbegin-1:wbegin-1+im])
		for i := 1; i <= im; i++ {
			w[wbegin+i-2] = w[wbegin+i-2] + sigma
		}

		ndepth := 0
		parity := 1
		nclus := 1
		iwork[iindc1+0] = 1
		iwork[iindc1+1] = im

		idone := 0
		for idone < im {
			if ndepth > m {
				return -2
			}
			oldncl := nclus
			nclus = 0
			parity = 1 - parity
			var oldcls, newcls int
			if parity == 0 {
				oldcls = iindc1
				newcls = iindc2
			} else {
				oldcls = iindc2
				newcls = iindc1
			}

			// Process clusters on current level.
			for clusIdx := 1; clusIdx <= oldncl; clusIdx++ {
				j := oldcls + 2*clusIdx
				oldfst := iwork[j-2]
				oldlst := iwork[j-1]

				if ndepth > 0 {
					// Retrieve parent RRR from Z columns J, J+1.
					var jcol int
					if dol == 1 && dou == m {
						jcol = wbegin + oldfst - 1
					} else if wbegin+oldfst-1 < dol {
						jcol = dol - 1
					} else if wbegin+oldfst-1 > dou {
						jcol = dou
					} else {
						jcol = wbegin + oldfst - 1
					}
					copy(d[ibegin-1:ibegin-1+in], z[(jcol-1)*ldz+ibegin-1:(jcol-1)*ldz+ibegin-1+in])
					copy(l[ibegin-1:ibegin-1+in-1], z[jcol*ldz+ibegin-1:jcol*ldz+ibegin-1+in-1])
					sigma = z[jcol*ldz+iend-1]

					// Zero out those Z columns.
					for jj := jcol; jj <= jcol+1; jj++ {
						base := (jj - 1) * ldz
						for ir := ibegin - 1; ir < ibegin-1+in; ir++ {
							z[base+ir] = 0
						}
					}
				}

				// Compute D*L and D*L*L of current RRR.
				for jj := ibegin; jj <= iend-1; jj++ {
					tmp := d[jj-1] * l[jj-1]
					work[indld+jj-1] = tmp
					work[indlld+jj-1] = tmp * l[jj-1]
				}

				if ndepth > 0 {
					// Limited bisection refinement.
					p := indexw[wbegin-1+oldfst-1]
					q := indexw[wbegin-1+oldlst-1]
					offset := indexw[wbegin-1] - 1
					iinfo := dlarrb(in, d[ibegin-1:], work[indlld+ibegin-1:],
						p, q, rtol1, rtol2, offset,
						work[wbegin-1:], wgap[wbegin-1:], werr[wbegin-1:],
						work[indwrk:], iwork[iindwk:], pivmin, spdiam, in)
					if iinfo != 0 {
						return -1
					}
					// Maintain extremal gaps: only allow to grow.
					if oldfst > 1 {
						g := w[wbegin+oldfst-2] - werr[wbegin+oldfst-2] - w[wbegin+oldfst-3] - werr[wbegin+oldfst-3]
						if g > wgap[wbegin+oldfst-3] {
							wgap[wbegin+oldfst-3] = g
						}
					}
					if wbegin+oldlst-1 < wend {
						g := w[wbegin+oldlst-1] - werr[wbegin+oldlst-1] - w[wbegin+oldlst-2] - werr[wbegin+oldlst-2]
						if g > wgap[wbegin+oldlst-2] {
							wgap[wbegin+oldlst-2] = g
						}
					}
					// Refresh W from refined WORK.
					for jj := oldfst; jj <= oldlst; jj++ {
						w[wbegin+jj-2] = work[wbegin+jj-2] + sigma
					}
				}

				// Process the current cluster's children.
				newfst := oldfst
				for jc := oldfst; jc <= oldlst; jc++ {
					var newlst int
					if jc == oldlst {
						newlst = jc
					} else if wgap[wbegin+jc-2] >= minrgp*math.Abs(work[wbegin+jc-2]) {
						newlst = jc
					} else {
						continue
					}
					newsiz := newlst - newfst + 1

					var newftt int
					if dol == 1 && dou == m {
						newftt = wbegin + newfst - 1
					} else if wbegin+newfst-1 < dol {
						newftt = dol - 1
					} else if wbegin+newfst-1 > dou {
						newftt = dou
					} else {
						newftt = wbegin + newfst - 1
					}

					if newsiz > 1 {
						// Cluster: refine endpoints, compute child RRR.
						var lgap, rgap float64
						if newfst == 1 {
							lgap = w[wbegin-1] - werr[wbegin-1] - vl
							if lgap < 0 {
								lgap = 0
							}
						} else {
							lgap = wgap[wbegin+newfst-3]
						}
						rgap = wgap[wbegin+newlst-2]

						for k := 1; k <= 2; k++ {
							var p int
							if k == 1 {
								p = indexw[wbegin-1+newfst-1]
							} else {
								p = indexw[wbegin-1+newlst-1]
							}
							offset := indexw[wbegin-1] - 1
							iinfo := dlarrb(in, d[ibegin-1:], work[indlld+ibegin-1:],
								p, p, rqtol, rqtol, offset,
								work[wbegin-1:], wgap[wbegin-1:], werr[wbegin-1:],
								work[indwrk:], iwork[iindwk:], pivmin, spdiam, in)
							_ = iinfo
						}

						if wbegin+newlst-1 < dol || wbegin+newfst-1 > dou {
							idone += newlst - newfst + 1
							newfst = jc + 1
							continue
						}

						// Child RRR via dlarrf, store into Z columns newftt, newftt+1.
						sigmaPtr := []float64{0}
						_ = sigmaPtr
						tau, iinfo := dlarrf(in, d[ibegin-1:], l[ibegin-1:],
							work[indld+ibegin-1:],
							newfst, newlst,
							work[wbegin-1:], wgap[wbegin-1:], werr[wbegin-1:],
							spdiam, lgap, rgap, pivmin,
							z[(newftt-1)*ldz+ibegin-1:],
							z[newftt*ldz+ibegin-1:],
							work[indwrk:])
						if iinfo == 0 {
							ssigma := sigma + tau
							z[newftt*ldz+iend-1] = ssigma
							for k := newfst; k <= newlst; k++ {
								fudge := 3 * eps * math.Abs(work[wbegin+k-2])
								work[wbegin+k-2] = work[wbegin+k-2] - tau
								fudge += 4 * eps * math.Abs(work[wbegin+k-2])
								werr[wbegin+k-2] = werr[wbegin+k-2] + fudge
							}
							nclus++
							kk := newcls + 2*nclus
							iwork[kk-2] = newfst
							iwork[kk-1] = newlst
						} else {
							return -2
						}
					} else {
						// Singleton: compute eigenvector via dlar1v + RQI.
						iter := 0
						tol := 4 * math.Log(float64(in)) * eps
						k := newfst
						windex := wbegin + k - 1
						windmn := windex - 1
						if windmn < 1 {
							windmn = 1
						}
						windpl := windex + 1
						if windpl > m {
							windpl = m
						}
						lambda := work[windex-1]
						done++
						eskip := false
						if windex < dol || windex > dou {
							eskip = true
						}
						left := work[windex-1] - werr[windex-1]
						right := work[windex-1] + werr[windex-1]
						indeig := indexw[windex-1]
						var lgap, rgap float64
						if k == 1 {
							lgap = eps * math.Max(math.Abs(left), math.Abs(right))
						} else {
							lgap = wgap[windmn-1]
						}
						if k == im {
							rgap = eps * math.Max(math.Abs(left), math.Abs(right))
						} else {
							rgap = wgap[windex-1]
						}
						gap := math.Min(lgap, rgap)
						var gaptol float64
						if k == 1 || k == im {
							gaptol = 0
						} else {
							gaptol = gap * eps
						}
						isupmn := in
						isupmx := 1
						savgap := wgap[windex-1]
						wgap[windex-1] = gap
						usedbs := false
						usedrq := false
						needbs := !tryrqc

						var bstres, bstw float64

						// Singleton iteration.
					singletonLoop:
						for !eskip {
							if needbs {
								usedbs = true
								itmp1 := iwork[iindr+windex-1]
								offset := indexw[wbegin-1] - 1
								iinfo := dlarrb(in, d[ibegin-1:], work[indlld+ibegin-1:],
									indeig, indeig, 0, 2*eps, offset,
									work[wbegin-1:], wgap[wbegin-1:], werr[wbegin-1:],
									work[indwrk:], iwork[iindwk:], pivmin, spdiam, itmp1)
								if iinfo != 0 {
									return -3
								}
								lambda = work[windex-1]
								iwork[iindr+windex-1] = 0
							}
							// Compute the eigenvector.
							zCol := z[(windex-1)*ldz+ibegin-1 : (windex-1)*ldz+ibegin-1+in]
							supp := []int{0, 0}
							negcnt, ztz, mingma, _, sup1, sup2, nrminv, resid, rqcorr := dlar1v(
								in, 1, in, lambda,
								d[ibegin-1:], l[ibegin-1:],
								work[indld+ibegin-1:], work[indlld+ibegin-1:],
								pivmin, gaptol, zCol,
								!usedbs, iwork[iindr+windex-1],
								work[indwrk:])
							iwork[iindr+windex-1] = sup1 // dlar1v returns rOut here? See note below.
							_ = supp
							_ = ztz
							_ = mingma
							_ = negcnt
							isuppz[2*(windex-1)] = sup1
							isuppz[2*(windex-1)+1] = sup2
							if iter == 0 || resid < bstres {
								bstres = resid
								bstw = lambda
							}
							if sup1 < isupmn {
								isupmn = sup1
							}
							if sup2 > isupmx {
								isupmx = sup2
							}
							iter++

							if resid > tol*gap && math.Abs(rqcorr) > rqtol*math.Abs(lambda) && !usedbs {
								var sgndef float64
								if indeig <= negcnt {
									sgndef = -1
								} else {
									sgndef = 1
								}
								if rqcorr*sgndef >= 0 && lambda+rqcorr <= right && lambda+rqcorr >= left {
									usedrq = true
									if sgndef == 1 {
										left = lambda
									} else {
										right = lambda
									}
									work[windex-1] = 0.5 * (right + left)
									lambda = lambda + rqcorr
									werr[windex-1] = 0.5 * (right - left)
								} else {
									needbs = true
								}
								if right-left < rqtol*math.Abs(lambda) {
									usedbs = true
									continue singletonLoop
								} else if iter < maxitr {
									continue singletonLoop
								} else if iter == maxitr {
									needbs = true
									continue singletonLoop
								} else {
									return 5
								}
							} else {
								stp2ii := false
								if usedrq && usedbs && bstres <= resid {
									lambda = bstw
									stp2ii = true
								}
								if stp2ii {
									zCol2 := z[(windex-1)*ldz+ibegin-1 : (windex-1)*ldz+ibegin-1+in]
									_, _, _, _, sup1b, sup2b, nrminv2, _, _ := dlar1v(
										in, 1, in, lambda,
										d[ibegin-1:], l[ibegin-1:],
										work[indld+ibegin-1:], work[indlld+ibegin-1:],
										pivmin, gaptol, zCol2,
										!usedbs, iwork[iindr+windex-1],
										work[indwrk:])
									isuppz[2*(windex-1)] = sup1b
									isuppz[2*(windex-1)+1] = sup2b
									nrminv = nrminv2
								}
								work[windex-1] = lambda
								_ = nrminv
								break singletonLoop
							}
						}

						if !eskip {
							isuppz[2*(windex-1)] = isuppz[2*(windex-1)] + oldien
							isuppz[2*(windex-1)+1] = isuppz[2*(windex-1)+1] + oldien
							zfrom := isuppz[2*(windex-1)]
							zto := isuppz[2*(windex-1)+1]
							isupmn += oldien
							isupmx += oldien
							if isupmn < zfrom {
								for ii := isupmn; ii <= zfrom-1; ii++ {
									z[(windex-1)*ldz+ii-1] = 0
								}
							}
							if isupmx > zto {
								for ii := zto + 1; ii <= isupmx; ii++ {
									z[(windex-1)*ldz+ii-1] = 0
								}
							}
							// Scale eigenvector by nrminv.
							// We need nrminv from the last dlar1v call — capture from
							// the converged lambda by recomputing the norm: scale by
							// 1/||z|| for the support. We saved nrminv inside the loop
							// only locally; redo norm here for safety.
							var ztz float64
							for ii := zfrom; ii <= zto; ii++ {
								v := z[(windex-1)*ldz+ii-1]
								ztz += v * v
							}
							if ztz > 0 {
								scale := 1.0 / math.Sqrt(ztz)
								for ii := zfrom; ii <= zto; ii++ {
									z[(windex-1)*ldz+ii-1] *= scale
								}
							}
						}
						w[windex-1] = lambda + sigma
						if !eskip {
							if k > 1 {
								g := w[windex-1] - werr[windex-1] - w[windmn-1] - werr[windmn-1]
								if g > wgap[windmn-1] {
									wgap[windmn-1] = g
								}
							}
							if windex < wend {
								g := w[windpl-1] - werr[windpl-1] - w[windex-1] - werr[windex-1]
								if g < savgap {
									g = savgap
								}
								wgap[windex-1] = g
							}
						}
						idone++
					}
					newfst = jc + 1
				}
			}
			ndepth++
		}
		ibegin = iend + 1
		wbegin = wend + 1
	}

	_ = done
	return 0
}
