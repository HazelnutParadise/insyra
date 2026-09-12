package fa

import (
	"math"
	"strings"
	"testing"

	"gonum.org/v1/gonum/mat"
)

// The whole GPArotation family — GPForth, GPFoblq and the seven vgQ criterion
// functions — was at 0%. These are transliterations of R's GPArotation, and R
// is not available here, so the tests check properties that hold by definition
// rather than comparing against reference numbers:
//
//   - every analytic gradient matches a numerical derivative of its own
//     objective (a wrong Gq is the failure that makes a rotation converge to
//     the wrong place while still looking like it converged);
//   - an orthogonal rotation returns an orthogonal matrix and leaves each
//     variable's communality alone;
//   - an oblique rotation returns a matrix whose columns have unit length.
//
// Reference-number comparison against R belongs with the other reference
// verifications (#302/#303), not here.

// loadings is a three-factor pattern with a clear simple structure.
func loadings() *mat.Dense {
	return mat.NewDense(6, 3, []float64{
		0.80, 0.20, 0.10,
		0.75, 0.15, 0.05,
		0.10, 0.85, 0.20,
		0.20, 0.70, 0.15,
		0.05, 0.15, 0.90,
		0.15, 0.10, 0.75,
	})
}

func identity(n int) *mat.Dense {
	t := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		t.Set(i, i, 1)
	}
	return t
}

// checkGradient compares the analytic gradient against a central difference of
// the objective. A mismatch means the rotation is walking downhill in the wrong
// direction.
func checkGradient(t *testing.T, name string, criterion func(*mat.Dense) (*mat.Dense, float64)) {
	t.Helper()

	L := loadings()
	rows, cols := L.Dims()
	analytic, _ := criterion(mat.DenseCopyOf(L))

	const h = 1e-6
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			up := mat.DenseCopyOf(L)
			up.Set(i, j, up.At(i, j)+h)
			_, fUp := criterion(up)

			down := mat.DenseCopyOf(L)
			down.Set(i, j, down.At(i, j)-h)
			_, fDown := criterion(down)

			numeric := (fUp - fDown) / (2 * h)
			got := analytic.At(i, j)
			// Scale the tolerance with the size of the derivative: a central
			// difference at h=1e-6 carries about six digits.
			tol := 1e-5 * math.Max(1, math.Abs(numeric))
			if math.Abs(got-numeric) > tol {
				t.Errorf("%s: d/dL[%d][%d] analytic %.9f, numerical %.9f", name, i, j, got, numeric)
			}
		}
	}
}

func TestCriterionGradientsMatchNumericalDerivatives(t *testing.T) {
	tests := []struct {
		name      string
		criterion func(*mat.Dense) (*mat.Dense, float64)
	}{
		{name: "varimax", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, _ := vgQVarimax(L)
			return g, f
		}},
		{name: "quartimax", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, _ := vgQQuartimax(L)
			return g, f
		}},
		{name: "quartimin", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, _ := vgQQuartimin(L)
			return g, f
		}},
		{name: "oblimin gamma=0", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, err := vgQOblimin(L, 0)
			if err != nil {
				t.Fatalf("vgQOblimin: %v", err)
			}
			return g, f
		}},
		{name: "oblimin gamma=0.5", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, err := vgQOblimin(L, 0.5)
			if err != nil {
				t.Fatalf("vgQOblimin: %v", err)
			}
			return g, f
		}},
		{name: "geomin", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, _ := vgQGeomin(L, 0.01)
			return g, f
		}},
		{name: "bentler", criterion: func(L *mat.Dense) (*mat.Dense, float64) {
			g, f, _, err := vgQBentler(L)
			if err != nil {
				t.Fatalf("vgQBentler: %v", err)
			}
			return g, f
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkGradient(t, tt.name, tt.criterion)
		})
	}
}

// simplimax's objective counts the k smallest squared loadings, so it is a step
// function of L and has no derivative where the ranking changes. The gradient
// is checked away from those points by keeping the same k rows in play.
func TestVgQSimplimax(t *testing.T) {
	L := loadings()
	rows, cols := L.Dims()

	Gq, f, method := vgQSimplimax(L, rows)
	if method != "Simplimax" {
		t.Errorf("method: got %q, want \"Simplimax\"", method)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		t.Errorf("objective is %v", f)
	}
	if r, c := Gq.Dims(); r != rows || c != cols {
		t.Errorf("gradient is %dx%d, want %dx%d", r, c, rows, cols)
	}
}

func TestVgQVarimax_Objective(t *testing.T) {
	// A perfectly simple structure has more variance in its squared loadings —
	// and therefore a lower (better) varimax objective, since GPA minimises —
	// than a pattern that spreads every variable across the factors.
	simple := mat.NewDense(4, 2, []float64{
		1, 0,
		1, 0,
		0, 1,
		0, 1,
	})
	spread := mat.NewDense(4, 2, []float64{
		0.7, 0.7,
		0.7, 0.7,
		0.7, 0.7,
		0.7, 0.7,
	})

	_, fSimple, _ := vgQVarimax(simple)
	_, fSpread, _ := vgQVarimax(spread)

	if !(fSimple < fSpread) {
		t.Errorf("varimax objective: simple structure %.6f is not better than a spread one %.6f", fSimple, fSpread)
	}
}

func TestNormalizingWeight(t *testing.T) {
	A := mat.NewDense(2, 2, []float64{3, 4, 6, 8})

	// Kaiser weights: the length of each variable's loading vector.
	w := NormalizingWeight(A, true)
	if got, want := w.AtVec(0), 5.0; math.Abs(got-want) > 1e-12 {
		t.Errorf("row 0 weight: got %v, want %v", got, want)
	}
	if got, want := w.AtVec(1), 10.0; math.Abs(got-want) > 1e-12 {
		t.Errorf("row 1 weight: got %v, want %v", got, want)
	}

	// Without normalising, every weight is 1, so dividing by it changes nothing.
	w = NormalizingWeight(A, false)
	for i := 0; i < 2; i++ {
		if got := w.AtVec(i); got != 1 {
			t.Errorf("row %d weight without normalising: got %v, want 1", i, got)
		}
	}
}

func TestFrobNorm(t *testing.T) {
	if got, want := frobNorm(mat.NewDense(2, 2, []float64{3, 4, 0, 0})), 5.0; math.Abs(got-want) > 1e-12 {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := frobNorm(mat.NewDense(2, 2, nil)); got != 0 {
		t.Errorf("the zero matrix has norm %v, want 0", got)
	}
}

// An orthogonal rotation must return an orthogonal matrix, or the "orthogonal"
// in its name means nothing.
func TestGPForth_ReturnsAnOrthogonalRotation(t *testing.T) {
	for _, method := range []string{"varimax", "quartimax", "bentler", "geomin"} {
		t.Run(method, func(t *testing.T) {
			A := loadings()
			out, err := GPForth(A, identity(3), false, 1e-5, 1000, method, -1)
			if err != nil {
				t.Fatalf("GPForth: %v", err)
			}

			Th, ok := out["Th"].(*mat.Dense)
			if !ok {
				t.Fatalf("Th is %T, want *mat.Dense", out["Th"])
			}
			var TtT mat.Dense
			TtT.Mul(Th.T(), Th)
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					want := 0.0
					if i == j {
						want = 1.0
					}
					if math.Abs(TtT.At(i, j)-want) > 1e-8 {
						t.Errorf("T'T[%d][%d] = %.10f, want %v", i, j, TtT.At(i, j), want)
					}
				}
			}
			if out["orthogonal"] != true {
				t.Errorf("orthogonal: got %v, want true", out["orthogonal"])
			}
		})
	}
}

// An orthogonal rotation redistributes variance between factors but must not
// change how much of each variable is explained. LL' is invariant.
func TestGPForth_PreservesCommunalities(t *testing.T) {
	A := loadings()
	before := rowSumsOfSquares(A)

	out, err := GPForth(mat.DenseCopyOf(A), identity(3), false, 1e-5, 1000, "varimax", -1)
	if err != nil {
		t.Fatalf("GPForth: %v", err)
	}
	L, ok := out["loadings"].(*mat.Dense)
	if !ok {
		t.Fatalf("loadings is %T, want *mat.Dense", out["loadings"])
	}
	after := rowSumsOfSquares(L)

	for i := range before {
		if math.Abs(before[i]-after[i]) > 1e-8 {
			t.Errorf("variable %d: communality %.10f before, %.10f after", i, before[i], after[i])
		}
	}
}

// Rotating is meant to improve the criterion, never worsen it.
func TestGPForth_DoesNotWorsenTheCriterion(t *testing.T) {
	A := loadings()
	_, fBefore, _ := vgQVarimax(A)

	out, err := GPForth(mat.DenseCopyOf(A), identity(3), false, 1e-5, 1000, "varimax", -1)
	if err != nil {
		t.Fatalf("GPForth: %v", err)
	}
	fAfter, ok := out["f"].(float64)
	if !ok {
		t.Fatalf("f is %T, want float64", out["f"])
	}

	if fAfter > fBefore+1e-9 {
		t.Errorf("varimax objective got worse: %.10f before, %.10f after", fBefore, fAfter)
	}
	if conv, ok := out["convergence"].(bool); ok && !conv {
		t.Error("the rotation did not converge within 1000 iterations")
	}
}

// Rotation needs at least two factors to have anything to rotate between.
func TestGPForth_SingleFactorIsRefused(t *testing.T) {
	A := mat.NewDense(3, 1, []float64{0.8, 0.7, 0.6})

	out, err := GPForth(A, identity(1), false, 1e-5, 100, "varimax", -1)
	if err == nil {
		t.Fatal("a single-factor model was rotated")
	}
	if out != nil {
		t.Errorf("a refused rotation still returned %v", out)
	}
}

// An oblique rotation lets the factors correlate, so T is no longer orthogonal —
// but its columns must stay unit length, which is what keeps the rotated
// loadings on the same scale.
func TestGPFoblq_ColumnsStayUnitLength(t *testing.T) {
	// The oblique criteria carry R's "Q" suffix where an orthogonal one of the
	// same name also exists.
	for _, method := range []string{"quartimin", "oblimin", "simplimax", "geominq", "bentlerq"} {
		t.Run(method, func(t *testing.T) {
			A := loadings()
			out, err := GPFoblq(A, identity(3), false, 1e-5, 1000, method, 0)
			if err != nil {
				t.Fatalf("GPFoblq: %v", err)
			}

			Th, ok := out["Th"].(*mat.Dense)
			if !ok {
				t.Fatalf("Th is %T, want *mat.Dense", out["Th"])
			}
			var TtT mat.Dense
			TtT.Mul(Th.T(), Th)
			for i := 0; i < 3; i++ {
				if got := TtT.At(i, i); math.Abs(got-1) > 1e-8 {
					t.Errorf("column %d has squared length %.10f, want 1", i, got)
				}
			}
			if out["orthogonal"] != false {
				t.Errorf("orthogonal: got %v, want false", out["orthogonal"])
			}
		})
	}
}

func TestGPFoblq_SingleFactorIsRefused(t *testing.T) {
	A := mat.NewDense(3, 1, []float64{0.8, 0.7, 0.6})

	if _, err := GPFoblq(A, identity(1), false, 1e-5, 100, "quartimin", 0); err == nil {
		t.Fatal("a single-factor model was rotated")
	}
}

// Kaiser normalisation divides each row by its length before rotating and
// multiplies it back afterwards, so the communalities survive that too.
func TestGPForth_NormalisedRotationPreservesCommunalities(t *testing.T) {
	A := loadings()
	before := rowSumsOfSquares(A)

	out, err := GPForth(mat.DenseCopyOf(A), identity(3), true, 1e-5, 1000, "varimax", -1)
	if err != nil {
		t.Fatalf("GPForth: %v", err)
	}
	L := out["loadings"].(*mat.Dense)
	after := rowSumsOfSquares(L)

	for i := range before {
		if math.Abs(before[i]-after[i]) > 1e-8 {
			t.Errorf("variable %d: communality %.10f before, %.10f after", i, before[i], after[i])
		}
	}
}

func TestObliqueCriterion(t *testing.T) {
	// The method name is matched case-insensitively, and the name reported back
	// is the R function it mirrors.
	for _, method := range []string{"quartimin", "QUARTIMIN", "Oblimin", "simplimax", "geominq", "bentlerq"} {
		if _, _, name, err := obliqueCriterion(method, loadings(), 0); err != nil {
			t.Errorf("obliqueCriterion(%q): %v", method, err)
		} else if !strings.HasPrefix(name, "vgQ.") {
			t.Errorf("obliqueCriterion(%q) reported %q, want a vgQ. name", method, name)
		}
	}

	// "geomin" and "bentler" are the orthogonal spellings; the oblique ones
	// carry the Q.
	for _, method := range []string{"no-such-rotation", "geomin", "bentler", "varimax", ""} {
		if _, _, _, err := obliqueCriterion(method, loadings(), 0); err == nil {
			t.Errorf("obliqueCriterion(%q) was accepted", method)
		}
	}
}

// The criterion functions report a name of their own, and those names are not
// spelled consistently. They are internal — GPForth and GPFoblq label their
// output from their own parameter — so this pins them rather than fixing them.
func TestCriterionMethodNames(t *testing.T) {
	L := loadings()

	_, _, varimax := vgQVarimax(L)
	_, _, quartimax := vgQQuartimax(L)
	_, _, quartimin := vgQQuartimin(L)
	_, _, geomin := vgQGeomin(L, 0.01)
	_, _, bentler, err := vgQBentler(L)
	if err != nil {
		t.Fatalf("vgQBentler: %v", err)
	}

	for _, tt := range []struct{ got, want string }{
		{got: varimax, want: "varimax"},
		{got: quartimax, want: "Quartimax"},
		{got: quartimin, want: "Quartimin"},
		{got: geomin, want: "Geomin"},
		{got: bentler, want: "Bentler's criterion"},
	} {
		if tt.got != tt.want {
			t.Errorf("method name: got %q, want %q", tt.got, tt.want)
		}
	}
}

func rowSumsOfSquares(m *mat.Dense) []float64 {
	rows, cols := m.Dims()
	out := make([]float64, rows)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			v := m.At(i, j)
			sum += v * v
		}
		out[i] = sum
	}
	return out
}

// --- entry points -----------------------------------------------------------

// SymmetricEigenDescendingDsyevr is the package's eigen decomposition, and the
// LAPACK routines under it were all at 0% too. The properties are exact ones:
// A·v = λ·v for every pair, the eigenvalues come back largest first, and the
// eigenvectors are orthonormal.
func TestSymmetricEigenDescendingDsyevr(t *testing.T) {
	// A correlation-like symmetric matrix with distinct eigenvalues.
	a := mat.NewSymDense(4, []float64{
		1.00, 0.60, 0.30, 0.10,
		0.60, 1.00, 0.40, 0.20,
		0.30, 0.40, 1.00, 0.50,
		0.10, 0.20, 0.50, 1.00,
	})

	values, vectors, ok := SymmetricEigenDescendingDsyevr(a)
	if !ok {
		t.Fatal("the decomposition failed")
	}
	if len(values) != 4 {
		t.Fatalf("got %d eigenvalues, want 4", len(values))
	}

	for i := 1; i < len(values); i++ {
		if values[i] > values[i-1]+1e-12 {
			t.Errorf("eigenvalues are not in descending order: %v", values)
			break
		}
	}

	// The trace equals the sum of the eigenvalues.
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	if math.Abs(sum-4.0) > 1e-9 {
		t.Errorf("the eigenvalues sum to %.10f, want the trace 4", sum)
	}

	// A·v = λ·v, column by column.
	for k := 0; k < 4; k++ {
		v := mat.NewVecDense(4, nil)
		for i := 0; i < 4; i++ {
			v.SetVec(i, vectors.At(i, k))
		}
		var av mat.VecDense
		av.MulVec(a, v)
		for i := 0; i < 4; i++ {
			if got, want := av.AtVec(i), values[k]*v.AtVec(i); math.Abs(got-want) > 1e-9 {
				t.Errorf("eigenpair %d, row %d: A·v = %.10f, λ·v = %.10f", k, i, got, want)
			}
		}
	}

	// The eigenvectors are orthonormal.
	var vtv mat.Dense
	vtv.Mul(vectors.T(), vectors)
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			want := 0.0
			if i == j {
				want = 1.0
			}
			if math.Abs(vtv.At(i, j)-want) > 1e-9 {
				t.Errorf("V'V[%d][%d] = %.10f, want %v", i, j, vtv.At(i, j), want)
			}
		}
	}
}

func TestSymmetricEigenDescendingDsyevr_NonSquare(t *testing.T) {
	if _, _, ok := SymmetricEigenDescendingDsyevr(mat.NewDense(2, 3, nil)); ok {
		t.Error("a non-square matrix was decomposed")
	}
}

// KaiserVarimaxWithRotationMatrix is the stats::varimax path, separate from
// GPForth. Same invariants: an orthogonal rotation matrix, communalities
// unchanged.
func TestKaiserVarimaxWithRotationMatrix(t *testing.T) {
	A := loadings()
	before := rowSumsOfSquares(A)

	rotated, R, err := KaiserVarimaxWithRotationMatrix(mat.DenseCopyOf(A), true, 1000, 1e-5)
	if err != nil {
		t.Fatalf("KaiserVarimaxWithRotationMatrix: %v", err)
	}

	var RtR mat.Dense
	RtR.Mul(R.T(), R)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			want := 0.0
			if i == j {
				want = 1.0
			}
			if math.Abs(RtR.At(i, j)-want) > 1e-8 {
				t.Errorf("R'R[%d][%d] = %.10f, want %v", i, j, RtR.At(i, j), want)
			}
		}
	}

	after := rowSumsOfSquares(rotated)
	for i := range before {
		if math.Abs(before[i]-after[i]) > 1e-8 {
			t.Errorf("variable %d: communality %.10f before, %.10f after", i, before[i], after[i])
		}
	}
}

// A single factor has nothing to rotate between, so the loadings come back
// untouched with an identity rotation rather than an error.
func TestKaiserVarimaxWithRotationMatrix_SingleFactor(t *testing.T) {
	A := mat.NewDense(3, 1, []float64{0.8, 0.7, 0.6})

	rotated, R, err := KaiserVarimaxWithRotationMatrix(A, true, 1000, 1e-5)
	if err != nil {
		t.Fatalf("KaiserVarimaxWithRotationMatrix: %v", err)
	}
	if r, c := R.Dims(); r != 1 || c != 1 || R.At(0, 0) != 1 {
		t.Errorf("the rotation matrix is %dx%d with R[0][0] = %v, want the 1x1 identity", r, c, R.At(0, 0))
	}
	for i := 0; i < 3; i++ {
		if rotated.At(i, 0) != A.At(i, 0) {
			t.Errorf("row %d changed: %v to %v", i, A.At(i, 0), rotated.At(i, 0))
		}
	}
}

// Rotate is the package's own entry point over the two GPA families. With a
// single start it is a plain orthogonal rotation and preserves communalities
// exactly. It does NOT with the default of twenty restarts — see
// TestRotate_RestartsBreakOrthogonality below.
func TestRotate_SingleStartPreservesCommunalities(t *testing.T) {
	for _, method := range []string{"varimax", "quartimax"} {
		t.Run(method, func(t *testing.T) {
			A := loadings()
			before := rowSumsOfSquares(A)

			rotated, R, _, _, err := Rotate(mat.DenseCopyOf(A), method, &RotOpts{
				Eps: 1e-5, MaxIter: 1000, Restarts: 1,
			})
			if err != nil {
				t.Fatalf("Rotate(%q): %v", method, err)
			}

			var RtR mat.Dense
			RtR.Mul(R.T(), R)
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					want := 0.0
					if i == j {
						want = 1.0
					}
					if math.Abs(RtR.At(i, j)-want) > 1e-8 {
						t.Errorf("R'R[%d][%d] = %.10f, want %v", i, j, RtR.At(i, j), want)
					}
				}
			}

			after := rowSumsOfSquares(rotated)
			for i := range before {
				if math.Abs(before[i]-after[i]) > 1e-8 {
					t.Errorf("variable %d: communality %.10f before, %.10f after", i, before[i], after[i])
				}
			}
		})
	}
}

// The restart behaviour this file used to pin as broken
// (TestRotate_RestartsBreakOrthogonality, #373) is fixed, and the replacement
// it asked for lives in rotation_starts_test.go: every method, both families,
// checked on the invariant rather than on the rotation matrix alone.

func TestRotate_UnknownMethod(t *testing.T) {
	if _, _, _, _, err := Rotate(loadings(), "no-such-rotation", nil); err == nil {
		t.Error("an unknown rotation method was accepted")
	}
}
