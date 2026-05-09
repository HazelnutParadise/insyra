// fa/lapack_dlaebz.go
//
// dlaebz — workhorse bisection / interval-refinement routine for
// symmetric tridiagonal eigenvalue computations. Used by dstebz (and
// indirectly by dsyevr's bisection-fallback path).
//
// Faithful translation of LAPACK 3.12.1 reference Fortran (dlaebz.f).
// We keep the column-major (mmax, 2) layout for ab and nab — the
// caller is responsible for allocating those as length-(2*mmax) flat
// slices addressed as ab[(jp-1)*mmax + (ji-1)].
package fa

import "math"

// dlaebz performs interval-refinement / inversion of N(w) for a
// symmetric tridiagonal matrix.
//
// ijob:
//
//	1 — compute NAB for the initial intervals.
//	2 — bisection iteration to find eigenvalues of T.
//	3 — bisection to invert N(w) (find a point with N(w)=NVAL).
//
// Layout: ab and nab are flat slices of length 2*mmax addressed as
//
//	ab[(jp-1)*mmax + (ji-1)]   (Fortran AB(ji, jp))
//
// nval, c, work, iwork are length mmax.
//
// Returns mout and info per the reference.
func dlaebz(
	ijob, nitmax, n, mmax, minp, nbmin int,
	abstol, reltol, pivmin float64,
	d, e, e2 []float64,
	nval []int,
	ab []float64,
	c []float64,
	nab []int,
	work []float64,
	iwork []int,
) (mout, info int) {
	if ijob < 1 || ijob > 3 {
		return 0, -1
	}

	// Helpers to address (mmax,2) column-major slices.
	abIdx := func(ji, jp int) int { return (jp-1)*mmax + (ji - 1) }
	nabIdx := func(ji, jp int) int { return (jp-1)*mmax + (ji - 1) }

	// IJOB == 1: compute NAB and MOUT for initial intervals.
	if ijob == 1 {
		mout = 0
		for ji := 1; ji <= minp; ji++ {
			for jp := 1; jp <= 2; jp++ {
				tmp1 := d[0] - ab[abIdx(ji, jp)]
				if math.Abs(tmp1) < pivmin {
					tmp1 = -pivmin
				}
				cnt := 0
				if tmp1 <= 0 {
					cnt = 1
				}
				for j := 2; j <= n; j++ {
					tmp1 = d[j-1] - e2[j-2]/tmp1 - ab[abIdx(ji, jp)]
					if math.Abs(tmp1) < pivmin {
						tmp1 = -pivmin
					}
					if tmp1 <= 0 {
						cnt++
					}
				}
				nab[nabIdx(ji, jp)] = cnt
			}
			mout += nab[nabIdx(ji, 2)] - nab[nabIdx(ji, 1)]
		}
		return mout, 0
	}

	// Initialize for the iteration loop.
	// KF and KL: intervals 1..KF-1 have converged; KF..KL still being refined.
	kf := 1
	kl := minp

	// IJOB == 2: initialize C to midpoints. IJOB == 3: caller-supplied.
	if ijob == 2 {
		for ji := 1; ji <= minp; ji++ {
			c[ji-1] = 0.5 * (ab[abIdx(ji, 1)] + ab[abIdx(ji, 2)])
		}
	}

	// Iteration loop.
	for jit := 1; jit <= nitmax; jit++ {
		var klnew int
		if (kl-kf+1) >= nbmin && nbmin > 0 {
			// Parallel (vectorisable) version of the loop.
			for ji := kf; ji <= kl; ji++ {
				// Compute N(c) for interval ji.
				wji := d[0] - c[ji-1]
				icnt := 0
				if wji <= pivmin {
					icnt = 1
					if wji > -pivmin {
						wji = -pivmin
					}
				}
				for j := 2; j <= n; j++ {
					wji = d[j-1] - e2[j-2]/wji - c[ji-1]
					if wji <= pivmin {
						icnt++
						if wji > -pivmin {
							wji = -pivmin
						}
					}
				}
				work[ji-1] = wji
				iwork[ji-1] = icnt
			}

			if ijob <= 2 {
				// IJOB == 2: choose all sub-intervals containing eigenvalues.
				klnew = kl
				for ji := kf; ji <= kl; ji++ {
					// Insure that N(w) is monotone.
					iv := iwork[ji-1]
					if iv < nab[nabIdx(ji, 1)] {
						iv = nab[nabIdx(ji, 1)]
					}
					if iv > nab[nabIdx(ji, 2)] {
						iv = nab[nabIdx(ji, 2)]
					}
					iwork[ji-1] = iv

					switch {
					case iv == nab[nabIdx(ji, 2)]:
						// No eigenvalue in upper half — keep lower.
						ab[abIdx(ji, 2)] = c[ji-1]
					case iv == nab[nabIdx(ji, 1)]:
						// No eigenvalue in lower half — keep upper.
						ab[abIdx(ji, 1)] = c[ji-1]
					default:
						klnew++
						if klnew <= mmax {
							ab[abIdx(klnew, 2)] = ab[abIdx(ji, 2)]
							nab[nabIdx(klnew, 2)] = nab[nabIdx(ji, 2)]
							ab[abIdx(klnew, 1)] = c[ji-1]
							nab[nabIdx(klnew, 1)] = iv
							ab[abIdx(ji, 2)] = c[ji-1]
							nab[nabIdx(ji, 2)] = iv
						} else {
							info = mmax + 1
						}
					}
				}
				if info != 0 {
					return mout, info
				}
				kl = klnew
			} else {
				// IJOB == 3: binary search keeping N(w)=NVAL.
				for ji := kf; ji <= kl; ji++ {
					if iwork[ji-1] <= nval[ji-1] {
						ab[abIdx(ji, 1)] = c[ji-1]
						nab[nabIdx(ji, 1)] = iwork[ji-1]
					}
					if iwork[ji-1] >= nval[ji-1] {
						ab[abIdx(ji, 2)] = c[ji-1]
						nab[nabIdx(ji, 2)] = iwork[ji-1]
					}
				}
			}
		} else {
			// Serial version of the loop.
			klnew = kl
			for ji := kf; ji <= kl; ji++ {
				tmp1 := c[ji-1]
				tmp2 := d[0] - tmp1
				itmp1 := 0
				if tmp2 <= pivmin {
					itmp1 = 1
					if tmp2 > -pivmin {
						tmp2 = -pivmin
					}
				}
				for j := 2; j <= n; j++ {
					tmp2 = d[j-1] - e2[j-2]/tmp2 - tmp1
					if tmp2 <= pivmin {
						itmp1++
						if tmp2 > -pivmin {
							tmp2 = -pivmin
						}
					}
				}

				if ijob <= 2 {
					// Insure monotone.
					if itmp1 < nab[nabIdx(ji, 1)] {
						itmp1 = nab[nabIdx(ji, 1)]
					}
					if itmp1 > nab[nabIdx(ji, 2)] {
						itmp1 = nab[nabIdx(ji, 2)]
					}
					switch {
					case itmp1 == nab[nabIdx(ji, 2)]:
						ab[abIdx(ji, 2)] = tmp1
					case itmp1 == nab[nabIdx(ji, 1)]:
						ab[abIdx(ji, 1)] = tmp1
					case klnew < mmax:
						klnew++
						ab[abIdx(klnew, 2)] = ab[abIdx(ji, 2)]
						nab[nabIdx(klnew, 2)] = nab[nabIdx(ji, 2)]
						ab[abIdx(klnew, 1)] = tmp1
						nab[nabIdx(klnew, 1)] = itmp1
						ab[abIdx(ji, 2)] = tmp1
						nab[nabIdx(ji, 2)] = itmp1
					default:
						return mout, mmax + 1
					}
				} else {
					if itmp1 <= nval[ji-1] {
						ab[abIdx(ji, 1)] = tmp1
						nab[nabIdx(ji, 1)] = itmp1
					}
					if itmp1 >= nval[ji-1] {
						ab[abIdx(ji, 2)] = tmp1
						nab[nabIdx(ji, 2)] = itmp1
					}
				}
			}
			kl = klnew
		}

		// Check for convergence — compress converged intervals to the
		// front, increment kf past them.
		kfnew := kf
		for ji := kf; ji <= kl; ji++ {
			tmp1 := math.Abs(ab[abIdx(ji, 2)] - ab[abIdx(ji, 1)])
			tmp2 := math.Max(math.Abs(ab[abIdx(ji, 2)]), math.Abs(ab[abIdx(ji, 1)]))
			thr := math.Max(abstol, math.Max(pivmin, reltol*tmp2))
			if tmp1 < thr || nab[nabIdx(ji, 1)] >= nab[nabIdx(ji, 2)] {
				if ji > kfnew {
					t1 := ab[abIdx(ji, 1)]
					t2 := ab[abIdx(ji, 2)]
					it1 := nab[nabIdx(ji, 1)]
					it2 := nab[nabIdx(ji, 2)]
					ab[abIdx(ji, 1)] = ab[abIdx(kfnew, 1)]
					ab[abIdx(ji, 2)] = ab[abIdx(kfnew, 2)]
					nab[nabIdx(ji, 1)] = nab[nabIdx(kfnew, 1)]
					nab[nabIdx(ji, 2)] = nab[nabIdx(kfnew, 2)]
					ab[abIdx(kfnew, 1)] = t1
					ab[abIdx(kfnew, 2)] = t2
					nab[nabIdx(kfnew, 1)] = it1
					nab[nabIdx(kfnew, 2)] = it2
					if ijob == 3 {
						itmp := nval[ji-1]
						nval[ji-1] = nval[kfnew-1]
						nval[kfnew-1] = itmp
					}
				}
				kfnew++
			}
		}
		kf = kfnew

		// Choose midpoints for the still-active intervals.
		for ji := kf; ji <= kl; ji++ {
			c[ji-1] = 0.5 * (ab[abIdx(ji, 1)] + ab[abIdx(ji, 2)])
		}

		if kf > kl {
			break
		}
		_ = jit // (silence "declared but not used" if early-exit pattern changes)
	}

	// Converged.
	if kl+1-kf > 0 {
		info = kl + 1 - kf
	}
	mout = kl
	return mout, info
}
