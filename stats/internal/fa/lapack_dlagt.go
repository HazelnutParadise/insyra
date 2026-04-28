// fa/lapack_dlagt.go
//
// dlagtf — LU factorization of an n-by-n tridiagonal matrix T - λI
//          (with row interchanges via diagonal pivoting).
// dlagts — solve the system (T - λI) x = scaled y for y given the
//          factor from dlagtf. Used by dstein for inverse-iteration
//          eigenvector refinement.
//
// Faithful translations of LAPACK 3.12.1 dlagtf.f and dlagts.f.
package fa

import "math"

// dlagtf factors T - λI = P L U where T has subdiagonal c, diagonal a,
// superdiagonal b. Inputs are modified in place:
//
//	a (length n)   : on output, diagonal of U
//	b (length n-1) : on output, superdiagonal of U
//	c (length n-1) : on output, subdiagonal multipliers (L) or pivoted
//	d (length n-2) : on output, second superdiagonal of U (if pivoting)
//	in (length n)  : on output, in[k]=1 if rows k and k+1 were swapped;
//	                 in[n-1] holds the smallest pivoting index found,
//	                 or 0 if all pivots safely > tol.
func dlagtf(n int, a []float64, lambda float64, b, c, d []float64, tol float64, in []int) (info int) {
	if n < 0 {
		return -1
	}
	if n == 0 {
		return 0
	}
	a[0] -= lambda
	in[n-1] = 0
	if n == 1 {
		if a[0] == 0 {
			in[0] = 1
		}
		return 0
	}
	const eps = 2.2204460492503131e-16
	tl := math.Max(tol, eps)
	scale1 := math.Abs(a[0]) + math.Abs(b[0])
	for k := 1; k <= n-1; k++ {
		a[k] -= lambda
		scale2 := math.Abs(c[k-1]) + math.Abs(a[k])
		if k < n-1 {
			scale2 += math.Abs(b[k])
		}
		var piv1 float64
		if a[k-1] != 0 {
			piv1 = math.Abs(a[k-1]) / scale1
		}
		var piv2 float64
		if c[k-1] == 0 {
			in[k-1] = 0
			scale1 = scale2
			if k < n-1 {
				d[k-1] = 0
			}
		} else {
			piv2 = math.Abs(c[k-1]) / scale2
			if piv2 <= piv1 {
				in[k-1] = 0
				scale1 = scale2
				c[k-1] = c[k-1] / a[k-1]
				a[k] = a[k] - c[k-1]*b[k-1]
				if k < n-1 {
					d[k-1] = 0
				}
			} else {
				in[k-1] = 1
				mult := a[k-1] / c[k-1]
				a[k-1] = c[k-1]
				temp := a[k]
				a[k] = b[k-1] - mult*temp
				if k < n-1 {
					d[k-1] = b[k]
					b[k] = -mult * d[k-1]
				}
				b[k-1] = temp
				c[k-1] = mult
			}
		}
		if math.Max(piv1, piv2) <= tl && in[n-1] == 0 {
			in[n-1] = k
		}
	}
	if math.Abs(a[n-1]) <= scale1*tl && in[n-1] == 0 {
		in[n-1] = n
	}
	return 0
}

// dlagts solves the tridiagonal system (T - λI) x = y given the dlagtf
// factorization in (a, b, c, d, in). job:
//
//	 1 → solve (T - λI) x = y
//	-1 → solve (T - λI) x = y, perturb tiny pivots
//	 2 → solve (T - λI)^T x = y
//	-2 → solve (T - λI)^T x = y, perturb tiny pivots
//
// y is overwritten with x. tolPtr is in/out: if job<0 and *tolPtr<=0,
// dlagts computes a safe tolerance and writes it back.
//
// Returns info==k>0 if a tiny pivot at index k caused breakdown
// (only when job > 0). Returns 0 on success.
func dlagts(job, n int, a, b, c, d []float64, in []int, y []float64, tolPtr *float64) (info int) {
	if job == 0 || job > 2 || job < -2 {
		return -1
	}
	if n < 0 {
		return -2
	}
	if n == 0 {
		return 0
	}
	const (
		eps   = 2.2204460492503131e-16
		sfmin = lapackSafmin
	)
	bignum := 1.0 / sfmin
	tol := *tolPtr
	if job < 0 && tol <= 0 {
		tol = math.Abs(a[0])
		if n > 1 {
			if v := math.Abs(a[1]); v > tol {
				tol = v
			}
			if v := math.Abs(b[0]); v > tol {
				tol = v
			}
		}
		for k := 3; k <= n; k++ {
			if v := math.Abs(a[k-1]); v > tol {
				tol = v
			}
			if v := math.Abs(b[k-2]); v > tol {
				tol = v
			}
			if v := math.Abs(d[k-3]); v > tol {
				tol = v
			}
		}
		tol *= eps
		if tol == 0 {
			tol = eps
		}
		*tolPtr = tol
	}

	if job == 1 || job == -1 {
		// L y' = y forward.
		for k := 2; k <= n; k++ {
			if in[k-2] == 0 {
				y[k-1] = y[k-1] - c[k-2]*y[k-2]
			} else {
				temp := y[k-2]
				y[k-2] = y[k-1]
				y[k-1] = temp - c[k-2]*y[k-1]
			}
		}
		if job == 1 {
			for k := n; k >= 1; k-- {
				var temp float64
				if k <= n-2 {
					temp = y[k-1] - b[k-1]*y[k] - d[k-1]*y[k+1]
				} else if k == n-1 {
					temp = y[k-1] - b[k-1]*y[k]
				} else {
					temp = y[k-1]
				}
				ak := a[k-1]
				absak := math.Abs(ak)
				if absak < 1 {
					if absak < sfmin {
						if absak == 0 || math.Abs(temp)*sfmin > absak {
							return k
						}
						temp *= bignum
						ak *= bignum
					} else if math.Abs(temp) > absak*bignum {
						return k
					}
				}
				y[k-1] = temp / ak
			}
		} else {
			for k := n; k >= 1; k-- {
				var temp float64
				if k <= n-2 {
					temp = y[k-1] - b[k-1]*y[k] - d[k-1]*y[k+1]
				} else if k == n-1 {
					temp = y[k-1] - b[k-1]*y[k]
				} else {
					temp = y[k-1]
				}
				ak := a[k-1]
				pert := math.Copysign(tol, ak)
				for {
					absak := math.Abs(ak)
					if absak < 1 {
						if absak < sfmin {
							if absak == 0 || math.Abs(temp)*sfmin > absak {
								ak += pert
								pert *= 2
								continue
							}
							temp *= bignum
							ak *= bignum
						} else if math.Abs(temp) > absak*bignum {
							ak += pert
							pert *= 2
							continue
						}
					}
					break
				}
				y[k-1] = temp / ak
			}
		}
	} else {
		// JOB = ±2.
		if job == 2 {
			for k := 1; k <= n; k++ {
				var temp float64
				if k >= 3 {
					temp = y[k-1] - b[k-2]*y[k-2] - d[k-3]*y[k-3]
				} else if k == 2 {
					temp = y[k-1] - b[k-2]*y[k-2]
				} else {
					temp = y[k-1]
				}
				ak := a[k-1]
				absak := math.Abs(ak)
				if absak < 1 {
					if absak < sfmin {
						if absak == 0 || math.Abs(temp)*sfmin > absak {
							return k
						}
						temp *= bignum
						ak *= bignum
					} else if math.Abs(temp) > absak*bignum {
						return k
					}
				}
				y[k-1] = temp / ak
			}
		} else {
			for k := 1; k <= n; k++ {
				var temp float64
				if k >= 3 {
					temp = y[k-1] - b[k-2]*y[k-2] - d[k-3]*y[k-3]
				} else if k == 2 {
					temp = y[k-1] - b[k-2]*y[k-2]
				} else {
					temp = y[k-1]
				}
				ak := a[k-1]
				pert := math.Copysign(tol, ak)
				for {
					absak := math.Abs(ak)
					if absak < 1 {
						if absak < sfmin {
							if absak == 0 || math.Abs(temp)*sfmin > absak {
								ak += pert
								pert *= 2
								continue
							}
							temp *= bignum
							ak *= bignum
						} else if math.Abs(temp) > absak*bignum {
							ak += pert
							pert *= 2
							continue
						}
					}
					break
				}
				y[k-1] = temp / ak
			}
		}
		for k := n; k >= 2; k-- {
			if in[k-2] == 0 {
				y[k-2] = y[k-2] - c[k-2]*y[k-1]
			} else {
				temp := y[k-2]
				y[k-2] = y[k-1]
				y[k-1] = temp - c[k-2]*y[k-1]
			}
		}
	}
	return 0
}
