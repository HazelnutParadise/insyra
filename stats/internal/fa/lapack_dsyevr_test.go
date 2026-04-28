package fa

import (
	"math"
	"sort"
	"testing"

	"gonum.org/v1/gonum/mat"
)

// TestDsyevrAgainstGonum verifies our pure-Go dsyevr (full MRRR pipeline)
// matches gonum's EigenSym (which uses dsyev/QL+QR internally). The two
// algorithms differ in last-bit details but should agree to ~1e-10 on
// well-conditioned 6×6 matrices.
func TestDsyevrAgainstGonum(t *testing.T) {
	const n = 6
	// Same row-major source matrix as the dsyev test.
	src := []float64{
		0.997525177661505, 0.994267413037281, 0.997892006241179, -0.997364091312504, -0.990457822358610, -0.997451516045722,
		0.994267413037281, 0.992875701562461, 0.993264763654566, -0.992648015523461, -0.992222185480177, -0.994629524289718,
		0.997892006241179, 0.993264763654566, 0.995313880710512, -0.995051591734687, -0.992550970132602, -0.995132203924862,
		-0.997364091312504, -0.992648015523461, -0.995051591734687, 0.996372880743831, 0.994430613249075, 0.998372367294639,
		-0.990457822358610, -0.992222185480177, -0.992550970132602, 0.994430613249075, 0.995197684800775, 0.992252719143255,
		-0.997451516045722, -0.994629524289718, -0.995132203924862, 0.998372367294639, 0.992252719143255, 0.996652519537952,
	}
	// Column-major copy for our dsyevr (UPLO='L').
	aMine := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			aMine[j*n+i] = src[i*n+j]
		}
	}
	wMine := make([]float64, n)
	zMine := make([]float64, n*n)
	isuppz := make([]int, 2*n)
	work := make([]float64, 26*n)
	iwork := make([]int, 10*n)
	m, info := dsyevr('V', 'A', 'L', n, aMine, n,
		0, 0, 0, 0, 0,
		wMine, zMine, n, isuppz, work, iwork)
	if info != 0 {
		t.Fatalf("dsyevr info=%d", info)
	}
	if m != n {
		t.Fatalf("dsyevr m=%d, want %d", m, n)
	}

	// Gonum reference.
	sym := mat.NewSymDense(n, src)
	var eig mat.EigenSym
	if !eig.Factorize(sym, true) {
		t.Fatalf("gonum EigenSym failed")
	}
	wGonum := eig.Values(nil)

	// Sort both ascending (dsyevr returns ascending; gonum's also ascending).
	sort.Float64s(wMine)
	sort.Float64s(wGonum)

	maxDiff := 0.0
	for i := 0; i < n; i++ {
		d := math.Abs(wMine[i] - wGonum[i])
		if d > maxDiff {
			maxDiff = d
		}
	}
	t.Logf("dsyevr vs gonum eigenvalue max diff = %g", maxDiff)
	if maxDiff > 1e-10 {
		t.Errorf("eigenvalues differ by %g (want < 1e-10)", maxDiff)
		for i := 0; i < n; i++ {
			t.Logf("  w[%d]: dsyevr=%.17e gonum=%.17e", i, wMine[i], wGonum[i])
		}
	}

	// Verify orthogonality: Z^T Z ≈ I.
	var ortho mat.Dense
	zMat := mat.NewDense(n, n, nil)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			zMat.Set(i, j, zMine[j*n+i])
		}
	}
	ortho.Mul(zMat.T(), zMat)
	maxOrtho := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			d := math.Abs(ortho.At(i, j) - expected)
			if d > maxOrtho {
				maxOrtho = d
			}
		}
	}
	t.Logf("dsyevr Z^T Z - I max element = %g", maxOrtho)
	if maxOrtho > 1e-10 {
		t.Errorf("eigenvectors not orthonormal (max |Z^T Z - I| = %g)", maxOrtho)
	}

	// Verify A z_i ≈ λ_i z_i.
	aMat := mat.NewSymDense(n, src)
	maxResid := 0.0
	for j := 0; j < n; j++ {
		var az mat.VecDense
		zCol := mat.NewVecDense(n, zMine[j*n:j*n+n])
		az.MulVec(aMat, zCol)
		for i := 0; i < n; i++ {
			r := math.Abs(az.AtVec(i) - wMine[j]*zMine[j*n+i])
			if r > maxResid {
				maxResid = r
			}
		}
	}
	t.Logf("dsyevr A z - λ z max residual = %g", maxResid)
	if maxResid > 1e-10 {
		t.Errorf("eigenpair residual too large: %g", maxResid)
	}
}
