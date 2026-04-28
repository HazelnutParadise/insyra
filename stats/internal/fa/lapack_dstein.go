// fa/lapack_dstein.go
//
// dstein — compute eigenvectors of a symmetric tridiagonal matrix by
// inverse iteration starting from given eigenvalues. Used by dsyevr's
// bisection-fallback path (paired with dstebz which produces the
// eigenvalues themselves).
//
// Faithful translation of LAPACK 3.12.1 dstein.f.
package fa

import "math"

// idamax returns the 1-based index of the element with largest |x[k]|.
func idamax(n int, x []float64, incx int) int {
	if n < 1 || incx <= 0 {
		return 0
	}
	if n == 1 {
		return 1
	}
	imax := 1
	dmax := math.Abs(x[0])
	if incx == 1 {
		for i := 2; i <= n; i++ {
			if v := math.Abs(x[i-1]); v > dmax {
				imax = i
				dmax = v
			}
		}
	} else {
		ix := 1 + incx
		for i := 2; i <= n; i++ {
			if v := math.Abs(x[ix-1]); v > dmax {
				imax = i
				dmax = v
			}
			ix += incx
		}
	}
	return imax
}

// dstein computes m eigenvectors of T (diag d, off-diag e) for the
// eigenvalues w[0..m-1] (1-based blocks via iblock/isplit, ascending
// within each block). Eigenvectors written as columns of z (ldz×m,
// column-major).
//
//	work : workspace, length 5*n
//	iwork: workspace, length n
//	ifail: length m, holds indices of failed eigenvalues (output)
func dstein(
	n int,
	d, e []float64,
	m int,
	w []float64,
	iblock, isplit []int,
	z []float64,
	ldz int,
	work []float64,
	iwork []int,
	ifail []int,
) (info int) {
	const (
		maxits = 5
		extra  = 2
		eps    = 2.2204460492503131e-16
	)
	for i := 0; i < m; i++ {
		ifail[i] = 0
	}
	if n < 0 {
		return -1
	}
	if m < 0 || m > n {
		return -4
	}
	if ldz < max(1, n) {
		return -9
	}
	for j := 2; j <= m; j++ {
		if iblock[j-1] < iblock[j-2] {
			return -6
		}
		if iblock[j-1] == iblock[j-2] && w[j-1] < w[j-2] {
			return -5
		}
	}
	if n == 0 || m == 0 {
		return 0
	}
	if n == 1 {
		z[0] = 1
		return 0
	}

	// Initialize seed for dlarnv.
	iseed := []int{1, 1, 1, 1}

	// Workspace partition (0-based offsets into work):
	//   indrv1: random vector x      length n
	//   indrv2: U super-diag (b)     length n
	//   indrv3: L sub-diag (c)       length n
	//   indrv4: D diagonal (a)       length n
	//   indrv5: D[k-2] second super  length n
	const indrv1 = 0
	indrv2 := n
	indrv3 := 2 * n
	indrv4 := 3 * n
	indrv5 := 4 * n

	j1 := 1
	for nblk := 1; nblk <= iblock[m-1]; nblk++ {
		var b1 int
		if nblk == 1 {
			b1 = 1
		} else {
			b1 = isplit[nblk-2] + 1
		}
		bn := isplit[nblk-1]
		blksiz := bn - b1 + 1

		var onenrm, ortol, dtpcrt float64
		if blksiz > 1 {
			onenrm = math.Abs(d[b1-1]) + math.Abs(e[b1-1])
			if v := math.Abs(d[bn-1]) + math.Abs(e[bn-2]); v > onenrm {
				onenrm = v
			}
			for i := b1 + 1; i <= bn-1; i++ {
				if v := math.Abs(d[i-1]) + math.Abs(e[i-2]) + math.Abs(e[i-1]); v > onenrm {
					onenrm = v
				}
			}
			ortol = 1e-3 * onenrm
			dtpcrt = math.Sqrt(0.1 / float64(blksiz))
		}

		jblk := 0
		gpind := j1
		var xjm float64

		for j := j1; j <= m; j++ {
			if iblock[j-1] != nblk {
				j1 = j
				goto nextBlock
			}
			jblk++
			xj := w[j-1]

			if blksiz == 1 {
				work[indrv1] = 1
			} else {
				if jblk > 1 {
					eps1 := math.Abs(eps * xj)
					pertol := 10 * eps1
					sep := xj - xjm
					if sep < pertol {
						xj = xjm + pertol
					}
				}

				its := 0
				nrmchk := 0

				// Random starting vector.
				dlarnv(2, blksiz, iseed, work[indrv1:])
				// Copy T (a, b, c).
				copy(work[indrv4:indrv4+blksiz], d[b1-1:b1-1+blksiz])
				copy(work[indrv2+1:indrv2+1+blksiz-1], e[b1-1:b1-1+blksiz-1])
				copy(work[indrv3:indrv3+blksiz-1], e[b1-1:b1-1+blksiz-1])

				// LU with pivoting.
				tol := 0.0
				inWS := iwork[:blksiz]
				dlagtf(blksiz, work[indrv4:indrv4+blksiz], xj,
					work[indrv2+1:indrv2+blksiz], work[indrv3:indrv3+blksiz-1],
					work[indrv5:indrv5+blksiz-2], tol, inWS)

				// Inverse iteration loop.
				for {
					its++
					if its > maxits {
						info++
						ifail[info-1] = j
						break
					}

					// Normalize / scale Pb.
					jmax := idamax(blksiz, work[indrv1:], 1)
					scl := float64(blksiz) * onenrm * math.Max(eps, math.Abs(work[indrv4+blksiz-1])) /
						math.Abs(work[indrv1+jmax-1])
					dscal(blksiz, scl, work[indrv1:], 1)

					// Solve LU = Pb.
					tolPtr := tol
					dlagts(-1, blksiz,
						work[indrv4:indrv4+blksiz],
						work[indrv2+1:indrv2+blksiz],
						work[indrv3:indrv3+blksiz-1],
						work[indrv5:indrv5+blksiz-2],
						inWS,
						work[indrv1:indrv1+blksiz],
						&tolPtr)
					tol = tolPtr

					// Reorthogonalize via modified Gram-Schmidt for close evals.
					if jblk != 1 {
						if math.Abs(xj-xjm) > ortol {
							gpind = j
						}
						if gpind != j {
							for i := gpind; i <= j-1; i++ {
								ztr := -ddot(blksiz, work[indrv1:], 1, z[(i-1)*ldz+b1-1:], 1)
								daxpy(blksiz, ztr, z[(i-1)*ldz+b1-1:], 1, work[indrv1:], 1)
							}
						}
					}

					jmax = idamax(blksiz, work[indrv1:], 1)
					nrm := math.Abs(work[indrv1+jmax-1])

					if nrm < dtpcrt {
						continue
					}
					nrmchk++
					if nrmchk < extra+1 {
						continue
					}
					break
				}

				// Normalize accepted iterate.
				scl := 1.0 / dnrm2(blksiz, work[indrv1:], 1)
				jmax := idamax(blksiz, work[indrv1:], 1)
				if work[indrv1+jmax-1] < 0 {
					scl = -scl
				}
				dscal(blksiz, scl, work[indrv1:], 1)
			}

			// Zero column j of z, then place iterate at z[b1..bn, j].
			base := (j - 1) * ldz
			for i := 0; i < n; i++ {
				z[base+i] = 0
			}
			for i := 1; i <= blksiz; i++ {
				z[base+b1+i-2] = work[indrv1+i-1]
			}

			xjm = xj
		}
	nextBlock:
	}
	return info
}
