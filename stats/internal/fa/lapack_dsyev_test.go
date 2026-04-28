package fa

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

// TestDsyevAgainstGonum verifies our pure-Go dsyev matches gonum's
// EigenSym (which calls lapack64.Syev internally) on a representative
// symmetric matrix. They should agree to machine precision since both
// implement the same algorithm.
func TestDsyevAgainstGonum(t *testing.T) {
	// Use the same matrix structure as the failing parity test cases.
	const n = 6
	src := []float64{
		0.997525177661505, 0.994267413037281, 0.997892006241179, -0.997364091312504, -0.990457822358610, -0.997451516045722,
		0.994267413037281, 0.992875701562461, 0.993264763654566, -0.992648015523461, -0.992222185480177, -0.994629524289718,
		0.997892006241179, 0.993264763654566, 0.995313880710512, -0.995051591734687, -0.992550970132602, -0.995132203924862,
		-0.997364091312504, -0.992648015523461, -0.995051591734687, 0.996372880743831, 0.994430613249075, 0.998372367294639,
		-0.990457822358610, -0.992222185480177, -0.992550970132602, 0.994430613249075, 0.995197684800775, 0.992252719143255,
		-0.997451516045722, -0.994629524289718, -0.995132203924862, 0.998372367294639, 0.992252719143255, 0.996652519537952,
	}
	// Column-major copy for our dsyev.
	aMine := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			aMine[j*n+i] = src[i*n+j] // src is row-major; transpose to col-major
		}
	}
	wMine := make([]float64, n)
	if info := dsyev('V', 'U', n, aMine, n, wMine); info != 0 {
		t.Fatalf("dsyev info=%d", info)
	}

	// Gonum reference.
	sym := mat.NewSymDense(n, src)
	var eig mat.EigenSym
	if !eig.Factorize(sym, true) {
		t.Fatalf("gonum EigenSym failed")
	}
	wGonum := eig.Values(nil)

	// Compare eigenvalues (both should be sorted ascending).
	maxDiff := 0.0
	for i := 0; i < n; i++ {
		d := math.Abs(wMine[i] - wGonum[i])
		if d > maxDiff {
			maxDiff = d
		}
	}
	t.Logf("eigenvalue max diff vs gonum = %g", maxDiff)
	if maxDiff > 1e-10 {
		t.Errorf("eigenvalues differ by %g (want < 1e-10)", maxDiff)
		for i := 0; i < n; i++ {
			t.Logf("  w[%d]: mine=%.17e gonum=%.17e", i, wMine[i], wGonum[i])
		}
	}
}
