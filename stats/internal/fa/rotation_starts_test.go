package fa

import (
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/mat"
)

// A rotation is only a rotation if it leaves the common part of the model
// alone: L·L' = Lu·Lu' for the orthogonal family, L·Φ·L' = Lu·Lu' for the
// oblique one. That invariant follows from the rotation matrix staying on the
// criterion's own manifold, which in turn needs the *start* to be on it — the
// gradient projection algorithms only project relative to where they began.
//
// #373: two of the heuristic starts added when Restarts > 1 were oblique
// matrices, so whichever start happened to win on criterion value decided
// whether the answer described the fitted model.

var orthogonalMethods = []string{"varimax", "quartimax", "bentlerT", "geominT"}

var obliqueMethods = []string{"quartimin", "oblimin", "bentlerQ", "geominQ", "simplimax", "promax"}

// simpleStructure is the clean pattern the gradient tests use.
func simpleStructure() *mat.Dense { return loadings() }

// noisyStructure is a pattern on which the Promax and TargetRot starts used to
// win, which is what made #373 visible. Generated once from a fixed seed and
// pinned here so the test does not depend on the generator.
func noisyStructure() *mat.Dense {
	rnd := rand.New(rand.NewSource(20260912))
	p, nf := 6, 3
	d := make([]float64, p*nf)
	for i := range d {
		d[i] = rnd.NormFloat64() * 0.5
	}
	L := mat.NewDense(p, nf, d)
	for i := 0; i < p; i++ {
		L.Set(i, i%nf, L.At(i, i%nf)+0.8)
	}
	return L
}

func maxAbsDiff(a, b mat.Matrix) float64 {
	r, c := a.Dims()
	worst := 0.0
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if d := math.Abs(a.At(i, j) - b.At(i, j)); d > worst {
				worst = d
			}
		}
	}
	return worst
}

// reproduced returns max|L·L' − Lu·Lu'| when phi is nil, and
// max|L·Φ·L' − Lu·Lu'| when it is not.
func reproduced(rotated, unrotated, phi *mat.Dense) float64 {
	var want, got mat.Dense
	want.Mul(unrotated, unrotated.T())
	if phi == nil {
		got.Mul(rotated, rotated.T())
	} else {
		var tmp mat.Dense
		tmp.Mul(rotated, phi)
		got.Mul(&tmp, rotated.T())
	}
	return maxAbsDiff(&got, &want)
}

func departureFromOrthogonal(m *mat.Dense) float64 {
	n, _ := m.Dims()
	var mtm mat.Dense
	mtm.Mul(m.T(), m)
	return maxAbsDiff(&mtm, identity(n))
}

const invariantTol = 1e-10

func TestRotationPreservesTheModelAcrossRestarts(t *testing.T) {
	patterns := map[string]*mat.Dense{
		"simple": simpleStructure(),
		"noisy":  noisyStructure(),
	}

	for name, Lu := range patterns {
		for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
			for _, restarts := range []int{1, 2, 5, 20} {
				t.Run(name+"/"+method+"/"+itoa(restarts), func(t *testing.T) {
					rotated, rotMat, phi, _, err := Rotate(mat.DenseCopyOf(Lu), method,
						&RotOpts{Eps: 1e-5, MaxIter: 1000, PromaxPower: 4, Restarts: restarts})
					if err != nil {
						t.Fatalf("Rotate: %v", err)
					}

					if got := reproduced(rotated, Lu, phi); got > invariantTol {
						t.Errorf("the rotation changed the model: max|reproduced − original| = %.4e, want <= %.0e", got, invariantTol)
					}

					if isOrthogonalMethod(method) {
						if phi != nil {
							t.Errorf("an orthogonal rotation returned a factor correlation matrix")
						}
						if got := departureFromOrthogonal(rotMat); got > invariantTol {
							t.Errorf("the rotation matrix is not orthogonal: max|R'R − I| = %.4e, want <= %.0e", got, invariantTol)
						}
					}
				})
			}
		}
	}
}

func isOrthogonalMethod(name string) bool {
	for _, m := range orthogonalMethods {
		if m == name {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestRestartsIsTheNumberOfStarts(t *testing.T) {
	// It used to bound only the random starts while three heuristics were
	// added unconditionally, so Restarts: 2 ran four of them.
	L := noisyStructure()
	_, nf := L.Dims()
	for _, restarts := range []int{1, 2, 3, 5, 20} {
		if got := len(buildStarts(L, nf, restarts, 1e-5, 1000)); got != restarts {
			t.Errorf("Restarts %d produced %d starts", restarts, got)
		}
	}
}

func TestEveryStartIsOrthogonal(t *testing.T) {
	for name, L := range map[string]*mat.Dense{"simple": simpleStructure(), "noisy": noisyStructure()} {
		_, nf := L.Dims()
		for _, s := range buildStarts(L, nf, 20, 1e-5, 1000) {
			if d := departureFromOrthogonal(s); d > startOrthogonalityTol {
				t.Errorf("%s: a start is off the orthogonal group by %.4e", name, d)
			}
		}
	}
}

// The two starts #373 removed: Promax and TargetRot are oblique methods, so
// their rotation matrices are not on the orthogonal group and cannot be used
// as starts by either family. This pins why they are gone rather than merely
// that they are.
func TestObliqueMatricesAreRejectedAsStarts(t *testing.T) {
	L := noisyStructure()

	pm := Promax(L, 4, true)
	rot, ok := pm["rotmat"].(*mat.Dense)
	if !ok || rot == nil {
		t.Fatal("Promax returned no rotation matrix")
	}
	if isOrthonormal(rot) {
		t.Error("the Promax rotation matrix is orthogonal on this fixture, so it no longer demonstrates the case")
	}

	if _, trg, _, err := TargetRot(L); err == nil && trg != nil && isOrthonormal(trg) {
		t.Error("the TargetRot matrix is orthogonal on this fixture, so it no longer demonstrates the case")
	}
}

func TestConvergedSolutionBeatsOneThatDidNot(t *testing.T) {
	cases := []struct {
		name               string
		haveBest, bestConv bool
		bestScore          float64
		conv               bool
		score              float64
		want               bool
	}{
		{"first candidate always wins", false, false, math.Inf(1), false, 9, true},
		{"converged beats unconverged with a worse score", true, false, 0.1, true, 9, true},
		{"unconverged never beats converged", true, true, 9, false, 0.1, false},
		{"among converged, lower is better", true, true, 0.5, true, 0.4, true},
		{"among converged, higher is not", true, true, 0.4, true, 0.5, false},
		{"among unconverged, lower is better", true, false, 0.5, false, 0.4, true},
		{"a number beats a NaN", true, true, math.NaN(), true, 0.4, true},
	}
	for _, c := range cases {
		if got := preferCandidate(c.haveBest, c.bestConv, c.bestScore, c.conv, c.score); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// Every rotation wrapper has to hand its convergence flag back, or the
// best-of-restarts rule cannot prefer a converged solution and fa.Rotate
// reports a convergence it never checked.
func TestEveryRotationReportsConvergence(t *testing.T) {
	L := simpleStructure()
	for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
		res := FaRotations(mat.DenseCopyOf(L), nil, method, 0, 5, 4, 0.01, 1e-5, 1000)
		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("%s: FaRotations returned %T", method, res)
		}
		if _, ok := m["convergence"].(bool); !ok {
			t.Errorf("%s: no convergence flag in the chosen candidate", method)
		}
	}
}

// A rotation that runs out of iterations has to say so. fa.Rotate used to
// report converged = true unconditionally, because the candidate the restart
// loop chose never carried the flag, so the default in Rotate was all anyone
// ever saw.
//
// This lives here rather than in the stats package because opt.MaxIter governs
// extraction and is deliberately not passed to the rotation
// (stats/factor_analysis.go, buildRotation), so there is no public call that
// can cripple a rotation.
func TestRotateReportsThatItDidNotConverge(t *testing.T) {
	L := noisyStructure()
	for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
		if method == "promax" {
			continue // Promax is a closed-form target rotation, not an iteration.
		}
		_, _, _, converged, err := Rotate(mat.DenseCopyOf(L), method,
			&RotOpts{Eps: 1e-12, MaxIter: 1, PromaxPower: 4, Restarts: 1})
		if err != nil {
			t.Errorf("%s: %v", method, err)
			continue
		}
		if converged {
			t.Errorf("%s: reported converged after a single iteration at eps 1e-12", method)
		}
	}
}

func TestRotateReportsThatItConverged(t *testing.T) {
	L := simpleStructure()
	for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
		_, _, _, converged, err := Rotate(mat.DenseCopyOf(L), method,
			&RotOpts{Eps: 1e-5, MaxIter: 1000, PromaxPower: 4, Restarts: 5})
		if err != nil {
			t.Errorf("%s: %v", method, err)
			continue
		}
		if !converged {
			t.Errorf("%s: reported not converged on a clean simple structure", method)
		}
	}
}

// One start means the identity and nothing else, which is what makes the
// default path byte-for-byte what it was before multiple starts existed.
func TestOneRestartIsTheIdentityAlone(t *testing.T) {
	L := noisyStructure()
	_, nf := L.Dims()
	for _, restarts := range []int{-1, 0, 1} {
		starts := buildStarts(L, nf, restarts, 1e-5, 1000)
		if len(starts) != 1 {
			t.Fatalf("Restarts %d produced %d starts, want 1", restarts, len(starts))
		}
		if d := maxAbsDiff(starts[0], identity(nf)); d != 0 {
			t.Errorf("Restarts %d: the only start is not the identity (off by %.3e)", restarts, d)
		}
	}
}

// A start is only used if it passes the orthogonality check. On these loadings
// the Varimax that produces the informed start fails outright (its SVD does
// not converge), so that start is dropped — and a random one takes its place
// rather than the search quietly running one start short.
func TestARejectedStartIsReplaced(t *testing.T) {
	degenerate := map[string]*mat.Dense{
		"all zeros":    mat.NewDense(6, 3, make([]float64, 18)),
		"denormalised": mat.NewDense(6, 3, []float64{1e-300, 0, 0, 0, 1e-300, 0, 0, 0, 1e-300, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
	}
	for name, L := range degenerate {
		vm := Varimax(L, true, 1e-5, 1000)
		if rot, ok := vm["rotmat"].(*mat.Dense); ok && isOrthonormal(rot) {
			t.Errorf("%s: Varimax now produces a usable start, so this fixture no longer exercises the rejection", name)
			continue
		}
		const want = 5
		starts := buildStarts(L, 3, want, 1e-5, 1000)
		if len(starts) != want {
			t.Errorf("%s: %d starts after one was rejected, want %d", name, len(starts), want)
		}
		for i, s := range starts {
			if d := departureFromOrthogonal(s); d > startOrthogonalityTol {
				t.Errorf("%s: start %d is off the orthogonal group by %.4e", name, i, d)
			}
		}
	}
}

// overFactoredStructure is what ML extraction of four factors from the
// three-factor synthetic table in stats/verify_more_test.go returns
// (buildSyntheticTable(60, 6, syntheticGen3Factor), FixedK = 4). Pinned here
// because it is the one fixture on which the identity start stops in a basin
// a random start escapes: oblimin at gamma = 0 reaches f = 0.0444 from the
// identity and f = 0.00094 from the third random start.
func overFactoredStructure() *mat.Dense {
	return mat.NewDense(6, 4, []float64{
		0.97371501712956454, 0.0079721325645778496, -0.014459486993830628, -0.21588523283154318,
		0.24494542308991687, 0.39670957381899075, 0.818573580412012, -0.026861909522813428,
		0.24618789823293688, 0.87788500026365501, -0.18411403330678633, 0.10511937386832459,
		0.97328202834432709, -0.049024267698480777, 0.0048744505795983543, 0.21282611261386447,
		0.25480162936163508, 0.40368803304199108, 0.79654947492798456, -0.012421277991481435,
		0.24555549634269103, 0.91633995553871161, -0.205167492487175, 0.096432095362504286,
	})
}

func criterionOf(t *testing.T, method string, L *mat.Dense, restarts int) float64 {
	t.Helper()
	res, ok := FaRotations(mat.DenseCopyOf(L), nil, method, 0, restarts, 4, 0.01, 1e-5, 1000).(map[string]any)
	if !ok {
		t.Fatalf("%s: FaRotations returned a non-map", method)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("%s: %s", method, msg)
	}
	f, ok := res["f"].(float64)
	if !ok {
		t.Fatalf("%s: no criterion value in the chosen candidate", method)
	}
	return f
}

// Oblimin used to build its own identity start on every pass and ignore the
// one it was handed, so Restarts ran the same computation N times. At gamma = 0
// oblimin is the quartimin criterion, and quartimin does use its starts, so the
// two must agree for the same start list — they did not, at Restarts >= 5 on
// the over-factored fixture, where a random start reaches a lower basin.
func TestObliminRunsFromTheStartItIsGiven(t *testing.T) {
	for name, L := range map[string]*mat.Dense{"noisy": noisyStructure(), "over-factored": overFactoredStructure()} {
		for _, restarts := range []int{1, 2, 5, 20} {
			fo := criterionOf(t, "oblimin", L, restarts)
			fq := criterionOf(t, "quartimin", L, restarts)
			if math.Abs(fo-fq) > 1e-12 {
				t.Errorf("%s/restarts=%d: oblimin f = %.12f, quartimin f = %.12f — oblimin is not rotating from the same starts", name, restarts, fo, fq)
			}
		}
	}

	L := overFactoredStructure()
	one, five := criterionOf(t, "oblimin", L, 1), criterionOf(t, "oblimin", L, 5)
	if five >= one/10 {
		t.Errorf("over-factored: f = %.6f at one start and %.6f at five; the search did not leave the identity's basin", one, five)
	}
}
