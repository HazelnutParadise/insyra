package fa

import (
	"bytes"
	"log"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
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
		if got := len(buildStarts(L, nf, restarts)); got != restarts {
			t.Errorf("Restarts %d produced %d starts", restarts, got)
		}
	}
}

func TestEveryStartIsOrthogonal(t *testing.T) {
	for name, L := range map[string]*mat.Dense{"simple": simpleStructure(), "noisy": noisyStructure()} {
		_, nf := L.Dims()
		for _, s := range buildStarts(L, nf, 20) {
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
// (stats/factor_analysis.go, rotateFactors), so there is no public call that
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
		starts := buildStarts(L, nf, restarts)
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
		vm := Varimax(L, true, 1e-08, 5000)
		if rot, ok := vm["rotmat"].(*mat.Dense); ok && isOrthonormal(rot) {
			t.Errorf("%s: Varimax now produces a usable start, so this fixture no longer exercises the rejection", name)
			continue
		}
		const want = 5
		starts := buildStarts(L, 3, want)
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
// because it is the one fixture on which the identity start stops in a basin a
// random start escapes: the quartimin criterion reaches f = 0.0444 from the
// identity and from the informed Varimax start, and f = 0.00094 from the first
// random start. Oblimin at gamma = 0 is the same criterion, but on this line it
// builds its own identity start and ignores the ones it is handed, so it stays
// at 0.0444 however many starts are asked for.
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

// The random starts used to be seeded from a hash of every bit of the loadings,
// and extraction does not reproduce those bits across architectures: ML
// extraction of the synthetic table gives loading [1,1] = 0.39670957381899075
// on arm64 and 0.39670959426082741 on amd64. That 2e-8 drew an unrelated set of
// random starts, so the same FactorAnalysis call could reach a different basin
// on Linux than on a Mac. Measured over the 20 datasets of
// stats/factor_analysis_test.go and all four extractions, 197 of the 2400
// Restarts >= 2 combinations disagreed between the two architectures by more
// than 1e-5, the worst of them by 2.1.
//
// Quartimin rather than oblimin: at gamma = 0 they are the same criterion, but
// oblimin ignores the starts it is given on this line.
func TestRandomStartsDoNotDependOnTheLoadingsBits(t *testing.T) {
	arm64 := overFactoredStructure()
	amd64 := mat.DenseCopyOf(arm64)
	amd64.Set(1, 1, 0.39670959426082741)
	_, nf := arm64.Dims()

	const restarts = 20
	a := buildStarts(arm64, nf, restarts)
	b := buildStarts(amd64, nf, restarts)
	if len(a) != restarts || len(b) != restarts {
		t.Fatalf("%d and %d starts, want %d", len(a), len(b), restarts)
	}
	// Start 0 is the identity and start 1 the Varimax solution, which follows
	// the loadings by design; the random ones after it must not.
	for i := 2; i < restarts; i++ {
		if d := maxAbsDiff(a[i], b[i]); d != 0 {
			t.Errorf("random start %d differs by %.3e between loadings 2e-8 apart", i, d)
		}
	}

	fArm, fAmd := criterionOf(t, "quartimin", arm64, 5), criterionOf(t, "quartimin", amd64, 5)
	if math.Abs(fArm-fAmd) > 1e-6 {
		t.Errorf("quartimin at five starts: f = %.9f and %.9f for loadings 2e-8 apart; the search reached different basins", fArm, fAmd)
	}
}

// captureWarnings routes the logger into a buffer at warning level for the
// duration of the test and returns the buffer.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	prevWriter := log.Writer()
	prevLevel := insyra.Config.GetLogLevel()
	log.SetOutput(&out)
	insyra.Config.SetLogLevel(insyra.LogLevelWarning)
	t.Cleanup(func() {
		log.SetOutput(prevWriter)
		insyra.Config.SetLogLevel(prevLevel)
	})
	return &out
}

// A multi-start search reports non-convergence once, and only when the
// solution it chose did not converge. GPForth and GPFoblq used to warn on
// every start that hit the cap, so five starts could log five warnings — and
// push as many entries into the global error buffer — and the informed
// Varimax start warned as well.
func TestUnconvergedRotationWarnsOnce(t *testing.T) {
	out := captureWarnings(t)
	before := insyra.GetErrorCount()
	_, _, _, converged, err := Rotate(noisyStructure(), "quartimin",
		&RotOpts{Eps: 1e-12, MaxIter: 1, PromaxPower: 4, Restarts: 5})
	if err != nil {
		t.Fatal(err)
	}
	if converged {
		t.Fatal("the fixture converged, so it no longer exercises the case")
	}
	if n := strings.Count(out.String(), "[insyra - Warning]"); n != 1 {
		t.Errorf("%d warnings logged for one unconverged rotation over 5 starts, want 1:\n%s", n, out.String())
	}
	if !strings.Contains(out.String(), "quartimin") || !strings.Contains(out.String(), "5 starts") {
		t.Errorf("the warning does not name the method and the number of starts:\n%s", out.String())
	}
	if got := insyra.GetErrorCount() - before; got != 1 {
		t.Errorf("%d entries pushed into the error buffer, want 1", got)
	}
}

func TestConvergedRotationDoesNotWarn(t *testing.T) {
	out := captureWarnings(t)
	before := insyra.GetErrorCount()
	_, _, _, converged, err := Rotate(simpleStructure(), "quartimin",
		&RotOpts{Eps: 1e-5, MaxIter: 1000, PromaxPower: 4, Restarts: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !converged {
		t.Fatal("the fixture did not converge, so it no longer exercises the case")
	}
	if out.Len() != 0 {
		t.Errorf("a converged rotation logged:\n%s", out.String())
	}
	if got := insyra.GetErrorCount() - before; got != 0 {
		t.Errorf("%d entries pushed into the error buffer for a converged rotation", got)
	}
}
