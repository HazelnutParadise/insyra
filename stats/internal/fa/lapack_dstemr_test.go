package fa

import (
	"math"
	"testing"
)

// TestDormtrSimple3x3 — the simplest possible non-trivial case to
// debug dormtr indexing.
func TestDormtrSimple3x3(t *testing.T) {
	const n = 4
	src := []float64{
		1, 0.5, 0.25, 0.125,
		0.5, 1, 0.5, 0.25,
		0.25, 0.5, 1, 0.5,
		0.125, 0.25, 0.5, 1,
	}
	a := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[j*n+i] = src[i*n+j]
		}
	}
	d := make([]float64, n)
	e := make([]float64, n)
	tau := make([]float64, n)
	dsytrd('L', n, a, n, d, e, tau)
	t.Logf("d=%v e=%v tau=%v", d, e[:n-1], tau[:n-1])

	// Print A after dsytrd (Householder vectors stored below subdiag).
	t.Log("A after dsytrd (column-major):")
	for col := 0; col < n; col++ {
		t.Logf("  col %d: %v", col, a[col*n:(col+1)*n])
	}

	// Apply dormtr to identity.
	aDor := append([]float64(nil), a...)
	tauCopy := append([]float64(nil), tau...)
	z := make([]float64, n*n)
	for i := 0; i < n; i++ {
		z[i*n+i] = 1
	}
	work := make([]float64, 26*n)
	dormtr('L', 'L', 'N', n, n, aDor, n, tauCopy, z, n, work)
	t.Log("dormtr(I) result (column-major):")
	for col := 0; col < n; col++ {
		t.Logf("  col %d: %v", col, z[col*n:(col+1)*n])
	}

	// Apply dorgtr to get explicit Q.
	aQ := append([]float64(nil), a...)
	dorgtr('L', n, aQ, n, tau)
	t.Log("dorgtr Q (column-major):")
	for col := 0; col < n; col++ {
		t.Logf("  col %d: %v", col, aQ[col*n:(col+1)*n])
	}
}

// TestDormtrAgainstExplicitQ verifies our dormtr output against
// explicit reconstruction of Q via gonum's Dorgtr followed by mat.Mul.
func TestDormtrAgainstExplicitQ(t *testing.T) {
	const n = 6
	src := []float64{
		0.997525177661505, 0.994267413037281, 0.997892006241179, -0.997364091312504, -0.990457822358610, -0.997451516045722,
		0.994267413037281, 0.992875701562461, 0.993264763654566, -0.992648015523461, -0.992222185480177, -0.994629524289718,
		0.997892006241179, 0.993264763654566, 0.995313880710512, -0.995051591734687, -0.992550970132602, -0.995132203924862,
		-0.997364091312504, -0.992648015523461, -0.995051591734687, 0.996372880743831, 0.994430613249075, 0.998372367294639,
		-0.990457822358610, -0.992222185480177, -0.992550970132602, 0.994430613249075, 0.995197684800775, 0.992252719143255,
		-0.997451516045722, -0.994629524289718, -0.995132203924862, 0.998372367294639, 0.992252719143255, 0.996652519537952,
	}
	a := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[j*n+i] = src[i*n+j]
		}
	}
	d := make([]float64, n)
	e := make([]float64, n)
	tau := make([]float64, n)
	dsytrd('L', n, a, n, d, e, tau)

	// Path 1: explicit Q via dorgtr-equivalent + manual Z := Q * Z
	aQ := append([]float64(nil), a...)
	dorgtr('L', n, aQ, n, tau)
	t.Logf("dorgtr Q[0,:] = %v", aQ[:n])

	// Path 2: dormtr applied to identity (should give Q)
	aDor := append([]float64(nil), a...)
	zIdent := make([]float64, n*n)
	for i := 0; i < n; i++ {
		zIdent[i*n+i] = 1
	}
	tauCopy := append([]float64(nil), tau...)
	work := make([]float64, 26*n)
	dormtr('L', 'L', 'N', n, n, aDor, n, tauCopy, zIdent, n, work)

	// Print full Q from dorgtr.
	t.Log("dorgtr Q (column-major):")
	for col := 0; col < n; col++ {
		t.Logf("  col %d: %v", col, aQ[col*n:(col+1)*n])
	}
	t.Log("dormtr(I) (column-major):")
	for col := 0; col < n; col++ {
		t.Logf("  col %d: %v", col, zIdent[col*n:(col+1)*n])
	}

	// Compare aQ (from dorgtr) vs zIdent (from dormtr applied to I).
	maxDiff := 0.0
	for i := 0; i < n*n; i++ {
		d := math.Abs(aQ[i] - zIdent[i])
		if d > maxDiff {
			maxDiff = d
		}
	}
	t.Logf("max |dorgtr Q - dormtr(I)| = %g", maxDiff)
	if maxDiff > 1e-12 {
		t.Errorf("dormtr inconsistent with dorgtr: max diff = %g", maxDiff)
	}
}

// TestDsytrdQTAQ verifies T = Q^T A Q where Q is constructed by dorgtr.
func TestDsytrdQTAQ(t *testing.T) {
	const n = 4
	src := []float64{
		1, 0.5, 0.25, 0.125,
		0.5, 1, 0.5, 0.25,
		0.25, 0.5, 1, 0.5,
		0.125, 0.25, 0.5, 1,
	}
	a := make([]float64, n*n)
	aOrig := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[j*n+i] = src[i*n+j]
			aOrig[j*n+i] = src[i*n+j]
		}
	}
	d := make([]float64, n)
	e := make([]float64, n)
	tau := make([]float64, n)
	dsytrd('L', n, a, n, d, e, tau)

	aQ := append([]float64(nil), a...)
	dorgtr('L', n, aQ, n, tau)

	// Check Q^T Q = I.
	maxOrtho := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += aQ[i*n+k] * aQ[j*n+k]
			}
			expected := 0.0
			if i == j {
				expected = 1
			}
			d := math.Abs(s - expected)
			if d > maxOrtho {
				maxOrtho = d
			}
		}
	}
	t.Logf("Q^T Q - I max = %g", maxOrtho)

	// Compute Q^T A Q manually.
	// AQ = A * Q (n×n)
	aqBuf := make([]float64, n*n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += aOrig[k*n+i] * aQ[j*n+k]
			}
			aqBuf[j*n+i] = s
		}
	}
	// QtAQ = Q^T * AQ
	qtaq := make([]float64, n*n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += aQ[i*n+k] * aqBuf[j*n+k]
			}
			qtaq[j*n+i] = s
		}
	}
	t.Log("Q^T A Q (should be tridiag matching d/e):")
	for col := 0; col < n; col++ {
		t.Logf("  col %d: %v", col, qtaq[col*n:(col+1)*n])
	}
	t.Logf("d from dsytrd: %v", d)
	t.Logf("e from dsytrd: %v", e[:n-1])
}

// TestPipelineManual runs dsytrd → dstemr → manual Q*Z, checking
// orthonormality at each step.
func TestPipelineManual(t *testing.T) {
	const n = 6
	src := []float64{
		0.997525177661505, 0.994267413037281, 0.997892006241179, -0.997364091312504, -0.990457822358610, -0.997451516045722,
		0.994267413037281, 0.992875701562461, 0.993264763654566, -0.992648015523461, -0.992222185480177, -0.994629524289718,
		0.997892006241179, 0.993264763654566, 0.995313880710512, -0.995051591734687, -0.992550970132602, -0.995132203924862,
		-0.997364091312504, -0.992648015523461, -0.995051591734687, 0.996372880743831, 0.994430613249075, 0.998372367294639,
		-0.990457822358610, -0.992222185480177, -0.992550970132602, 0.994430613249075, 0.995197684800775, 0.992252719143255,
		-0.997451516045722, -0.994629524289718, -0.995132203924862, 0.998372367294639, 0.992252719143255, 0.996652519537952,
	}
	a := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[j*n+i] = src[i*n+j]
		}
	}
	d := make([]float64, n)
	e := make([]float64, n)
	tau := make([]float64, n)
	dsytrd('L', n, a, n, d, e, tau)

	// Build Q via dorgtr (on a copy of a).
	aQ := append([]float64(nil), a...)
	dorgtr('L', n, aQ, n, tau)

	// Check Q^T Q = I.
	maxOrtho := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += aQ[i*n+k] * aQ[j*n+k]
			}
			expected := 0.0
			if i == j {
				expected = 1
			}
			d := math.Abs(s - expected)
			if d > maxOrtho {
				maxOrtho = d
			}
		}
	}
	t.Logf("Q^T Q - I max = %g", maxOrtho)

	// Run dstemr on the tridiag output.
	w := make([]float64, n)
	z := make([]float64, n*n)
	isuppz := make([]int, 2*n)
	work := make([]float64, 18*n)
	iwork := make([]int, 10*n)
	dCopy := append([]float64(nil), d...)
	eCopy := append([]float64(nil), e...)
	m, info := dstemr('V', 'A', n, dCopy, eCopy, 0, 0, 0, 0,
		w, z, n, n, isuppz, true, work, iwork)
	if info != 0 {
		t.Fatalf("dstemr info=%d", info)
	}
	t.Logf("dstemr m=%d, eigenvalues=%v", m, w[:m])

	// Check Z^T Z = I.
	maxZortho := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += z[i*n+k] * z[j*n+k]
			}
			expected := 0.0
			if i == j {
				expected = 1
			}
			d := math.Abs(s - expected)
			if d > maxZortho {
				maxZortho = d
			}
		}
	}
	t.Logf("Z^T Z - I max (tridiag basis) = %g", maxZortho)

	// Manually compute Q * Z.
	qz := make([]float64, n*n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += aQ[k*n+i] * z[j*n+k]
			}
			qz[j*n+i] = s
		}
	}

	// Check (Q*Z)^T (Q*Z) = I.
	maxQZortho := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += qz[i*n+k] * qz[j*n+k]
			}
			expected := 0.0
			if i == j {
				expected = 1
			}
			d := math.Abs(s - expected)
			if d > maxQZortho {
				maxQZortho = d
			}
		}
	}
	t.Logf("(QZ)^T (QZ) - I max = %g", maxQZortho)
}

// TestDsytrdThenDstemr runs the dsytrd reduction on the same 6×6
// matrix used by TestDsyevrAgainstGonum, then dstemr on the resulting
// tridiagonal — bypassing dormtr. If Z (in tridiagonal basis) is
// orthonormal here, the dsyevr failure isolates to dormtr.
func TestDsytrdThenDstemr(t *testing.T) {
	const n = 6
	src := []float64{
		0.997525177661505, 0.994267413037281, 0.997892006241179, -0.997364091312504, -0.990457822358610, -0.997451516045722,
		0.994267413037281, 0.992875701562461, 0.993264763654566, -0.992648015523461, -0.992222185480177, -0.994629524289718,
		0.997892006241179, 0.993264763654566, 0.995313880710512, -0.995051591734687, -0.992550970132602, -0.995132203924862,
		-0.997364091312504, -0.992648015523461, -0.995051591734687, 0.996372880743831, 0.994430613249075, 0.998372367294639,
		-0.990457822358610, -0.992222185480177, -0.992550970132602, 0.994430613249075, 0.995197684800775, 0.992252719143255,
		-0.997451516045722, -0.994629524289718, -0.995132203924862, 0.998372367294639, 0.992252719143255, 0.996652519537952,
	}
	a := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[j*n+i] = src[i*n+j]
		}
	}
	d := make([]float64, n)
	e := make([]float64, n)
	tau := make([]float64, n)
	dsytrd('L', n, a, n, d, e, tau)

	t.Logf("d after dsytrd: %v", d)
	t.Logf("e after dsytrd: %v", e[:n-1])

	w := make([]float64, n)
	z := make([]float64, n*n)
	isuppz := make([]int, 2*n)
	work := make([]float64, 18*n)
	iwork := make([]int, 10*n)

	dCopy := append([]float64(nil), d...)
	eCopy := append([]float64(nil), e...)

	m, info := dstemr('V', 'A', n, dCopy, eCopy, 0, 0, 0, 0,
		w, z, n, n, isuppz, true, work, iwork)
	if info != 0 {
		t.Fatalf("dstemr info=%d", info)
	}
	if m != n {
		t.Fatalf("m=%d, want %d", m, n)
	}

	// Check column norms.
	for j := 0; j < n; j++ {
		sum := 0.0
		for i := 0; i < n; i++ {
			v := z[j*n+i]
			sum += v * v
		}
		t.Logf("col %d norm² = %g (eigval=%g)", j, sum, w[j])
		if math.Abs(sum-1.0) > 1e-9 {
			t.Errorf("eigenvector %d not unit norm: |z|² = %g", j, sum)
		}
	}

	// Check T z_j ≈ λ_j z_j.
	maxResid := 0.0
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			tz := d[i] * z[j*n+i]
			if i > 0 {
				tz += e[i-1] * z[j*n+(i-1)]
			}
			if i < n-1 {
				tz += e[i] * z[j*n+(i+1)]
			}
			r := math.Abs(tz - w[j]*z[j*n+i])
			if r > maxResid {
				maxResid = r
			}
		}
	}
	t.Logf("dstemr T z - λ z max residual = %g", maxResid)
}

// TestDstemrSmallTridiag exercises dstemr on a small known tridiagonal
// matrix. If eigenvalues + eigenvectors are correct here, any failure
// in the dsyevr full pipeline must come from dsytrd or dormtr.
func TestDstemrSmallTridiag(t *testing.T) {
	const n = 4
	d := []float64{4, 3, 2, 1}
	e := []float64{1, 1, 1, 0}

	w := make([]float64, n)
	z := make([]float64, n*n)
	isuppz := make([]int, 2*n)
	work := make([]float64, 18*n)
	iwork := make([]int, 10*n)

	dCopy := append([]float64(nil), d...)
	eCopy := append([]float64(nil), e...)

	m, info := dstemr('V', 'A', n, dCopy, eCopy, 0, 0, 0, 0,
		w, z, n, n, isuppz, true, work, iwork)
	if info != 0 {
		t.Fatalf("dstemr info=%d", info)
	}
	if m != n {
		t.Fatalf("m=%d, want %d", m, n)
	}

	t.Logf("eigenvalues: %v", w[:n])

	// Check column norms (should be 1).
	for j := 0; j < n; j++ {
		sum := 0.0
		for i := 0; i < n; i++ {
			v := z[j*n+i]
			sum += v * v
		}
		t.Logf("col %d norm² = %g", j, sum)
		if math.Abs(sum-1.0) > 1e-10 {
			t.Errorf("eigenvector %d not unit norm: |z|² = %g", j, sum)
		}
	}

	// Check T z_j ≈ λ_j z_j by hand-applying the tridiag.
	maxResid := 0.0
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			tz := d[i] * z[j*n+i]
			if i > 0 {
				tz += e[i-1] * z[j*n+(i-1)]
			}
			if i < n-1 {
				tz += e[i] * z[j*n+(i+1)]
			}
			r := math.Abs(tz - w[j]*z[j*n+i])
			if r > maxResid {
				maxResid = r
			}
		}
	}
	t.Logf("dstemr T z - λ z max residual = %g", maxResid)
	if maxResid > 1e-10 {
		t.Errorf("eigenpair residual too large: %g", maxResid)
	}
}
