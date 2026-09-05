package quant

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// ---------------------------------------------------------------------------
// fixtures and closed-form references
// ---------------------------------------------------------------------------

// interiorMean and interiorCov are chosen so that both closed-form solutions
// below fall strictly inside (0, 1); the tests assert that precondition.
var (
	interiorMean = []float64{0.010, 0.014, 0.008}
	interiorCov  = [][]float64{
		{0.10, 0.02, 0.01},
		{0.02, 0.12, 0.03},
		{0.01, 0.03, 0.15},
	}
)

// correlatedCov has a global minimum-variance portfolio with a negative
// weight, so the long-only solution sits on the boundary.
var correlatedCov = [][]float64{
	{0.020, 0.036, 0.008},
	{0.036, 0.090, 0.012},
	{0.008, 0.012, 0.050},
}

func denseFromRows(rows [][]float64) *mat.Dense {
	n := len(rows)
	m := mat.NewDense(n, n, nil)
	for i := range rows {
		for j := range rows[i] {
			m.Set(i, j, rows[i][j])
		}
	}
	return m
}

// closedFormMinVariance returns Sigma^-1 1 / (1' Sigma^-1 1).
func closedFormMinVariance(t *testing.T, cov [][]float64) []float64 {
	t.Helper()
	n := len(cov)
	var inv mat.Dense
	if err := inv.Inverse(denseFromRows(cov)); err != nil {
		t.Fatalf("inverse: %v", err)
	}
	ones := mat.NewVecDense(n, nil)
	for i := range n {
		ones.SetVec(i, 1)
	}
	var z mat.VecDense
	z.MulVec(&inv, ones)
	total := mat.Dot(&z, ones)
	out := make([]float64, n)
	for i := range n {
		out[i] = z.AtVec(i) / total
	}
	return out
}

// closedFormTargetReturn solves min w'Sigma w subject to 1'w = 1 and mu'w = r
// through the two-multiplier Lagrangian system.
func closedFormTargetReturn(t *testing.T, cov [][]float64, mean []float64, target float64) []float64 {
	t.Helper()
	n := len(cov)
	var inv mat.Dense
	if err := inv.Inverse(denseFromRows(cov)); err != nil {
		t.Fatalf("inverse: %v", err)
	}
	ones := mat.NewVecDense(n, nil)
	for i := range n {
		ones.SetVec(i, 1)
	}
	mu := mat.NewVecDense(n, append([]float64(nil), mean...))
	var invOnes, invMu mat.VecDense
	invOnes.MulVec(&inv, ones)
	invMu.MulVec(&inv, mu)

	a := mat.Dot(ones, &invOnes) // 1' S^-1 1
	b := mat.Dot(ones, &invMu)   // 1' S^-1 mu
	c := mat.Dot(mu, &invMu)     // mu' S^-1 mu

	det := a*c - b*b
	// w = S^-1 (lambda1 * 1 + lambda2 * mu)
	lambda1 := (c - b*target) / det
	lambda2 := (a*target - b) / det
	out := make([]float64, n)
	for i := range n {
		out[i] = lambda1*invOnes.AtVec(i) + lambda2*invMu.AtVec(i)
	}
	return out
}

func portfolioVariance(cov [][]float64, w []float64) float64 {
	total := 0.0
	for i := range w {
		for j := range w {
			total += w[i] * cov[i][j] * w[j]
		}
	}
	return total
}

func assertInterior(t *testing.T, w []float64) {
	t.Helper()
	for i, v := range w {
		if v <= 0 || v >= 1 {
			t.Fatalf("precondition failed: closed-form weight %d = %.6f is not interior", i, v)
		}
	}
}

// returnsTable builds a rows x assets DataTable with the given column names.
func returnsTable(rows [][]float64, names []string) insyra.IDataTable {
	cols := make([]*insyra.DataList, len(names))
	for j := range names {
		vals := make([]any, len(rows))
		for i := range rows {
			vals[i] = rows[i][j]
		}
		dl := insyra.NewDataList(vals...)
		dl.SetName(names[j])
		cols[j] = dl
	}
	return insyra.NewDataTable(cols...)
}

// sampleReturns draws a deterministic 3-asset return panel with a stable
// covariance structure.
func sampleReturns(t *testing.T) ([][]float64, []string) {
	t.Helper()
	rng := rand.New(rand.NewSource(20260905))
	const n = 240
	rows := make([][]float64, n)
	for i := range n {
		common := rng.NormFloat64() * 0.010
		rows[i] = []float64{
			0.0006 + common + rng.NormFloat64()*0.008,
			0.0009 + 0.8*common + rng.NormFloat64()*0.013,
			0.0003 + 0.2*common + rng.NormFloat64()*0.020,
		}
	}
	return rows, []string{"A", "B", "C"}
}

// momentsOf computes the column means and the (n-1) sample covariance with
// gonum, independently of the implementation under test.
func momentsOf(rows [][]float64) ([]float64, [][]float64) {
	n := len(rows)
	k := len(rows[0])
	flat := make([]float64, 0, n*k)
	for _, row := range rows {
		flat = append(flat, row...)
	}
	data := mat.NewDense(n, k, flat)
	mean := make([]float64, k)
	for j := range k {
		mean[j] = stat.Mean(mat.Col(nil, j, data), nil)
	}
	var cov mat.SymDense
	stat.CovarianceMatrix(&cov, data, nil)
	out := make([][]float64, k)
	for i := range k {
		out[i] = make([]float64, k)
		for j := range k {
			out[i][j] = cov.At(i, j)
		}
	}
	return mean, out
}

// ---------------------------------------------------------------------------
// Requirement: mean-variance optimisation under sum-to-one and box constraints
// ---------------------------------------------------------------------------

func TestPortfolioInteriorMinimumVarianceMatchesClosedForm(t *testing.T) {
	want := closedFormMinVariance(t, interiorCov)
	assertInterior(t, want)

	got, err := OptimizePortfolioMoments(interiorMean, interiorCov, []string{"A", "B", "C"}, PortfolioConfig{
		Objective: MinimumVariance,
	})
	if err != nil {
		t.Fatalf("OptimizePortfolioMoments: %v", err)
	}
	if !got.Converged {
		t.Fatalf("Converged = false after %d iterations", got.Iterations)
	}
	for i := range want {
		assertClose(t, got.Weights[i], want[i], 1e-8)
	}
	assertClose(t, got.Variance, portfolioVariance(interiorCov, want), 1e-12)
	assertClose(t, got.Volatility, math.Sqrt(got.Variance), 1e-15)
}

func TestPortfolioInteriorTargetReturnMatchesClosedForm(t *testing.T) {
	minVar := closedFormMinVariance(t, interiorCov)
	base := 0.0
	for i := range minVar {
		base += minVar[i] * interiorMean[i]
	}
	target := base + 0.0008
	want := closedFormTargetReturn(t, interiorCov, interiorMean, target)
	assertInterior(t, want)

	got, err := OptimizePortfolioMoments(interiorMean, interiorCov, nil, PortfolioConfig{
		Objective:    TargetReturn,
		TargetReturn: target,
	})
	if err != nil {
		t.Fatalf("OptimizePortfolioMoments: %v", err)
	}
	if !got.Converged {
		t.Fatalf("Converged = false after %d iterations", got.Iterations)
	}
	for i := range want {
		assertClose(t, got.Weights[i], want[i], 1e-8)
	}
	assertClose(t, got.ExpectedReturn, target, 1e-10)
}

func TestPortfolioLongOnlyMatchesGridSearch(t *testing.T) {
	unconstrained := closedFormMinVariance(t, correlatedCov)
	negative := false
	for _, v := range unconstrained {
		if v < 0 {
			negative = true
		}
	}
	if !negative {
		t.Fatalf("precondition failed: closed form %v has no negative weight", unconstrained)
	}

	got, err := OptimizePortfolioMoments([]float64{0.01, 0.012, 0.009}, correlatedCov, nil, PortfolioConfig{
		Objective: MinimumVariance,
	})
	if err != nil {
		t.Fatalf("OptimizePortfolioMoments: %v", err)
	}

	// Exhaustive simplex grid at step 1e-3.
	const steps = 1000
	best := math.Inf(1)
	for i := 0; i <= steps; i++ {
		for j := 0; i+j <= steps; j++ {
			w := []float64{float64(i) / steps, float64(j) / steps, float64(steps-i-j) / steps}
			if v := portfolioVariance(correlatedCov, w); v < best {
				best = v
			}
		}
	}
	if got.Variance > best {
		t.Errorf("solver variance %.17g exceeds grid minimum %.17g", got.Variance, best)
	}
	sum := 0.0
	for i, v := range got.Weights {
		if v < -1e-12 || v > 1+1e-12 {
			t.Errorf("weight %d = %.17g outside [0, 1]", i, v)
		}
		sum += v
	}
	assertClose(t, sum, 1, 1e-12)
}

func TestPortfolioBoxBoundsRespected(t *testing.T) {
	minWeight := []float64{0.1, 0, 0}
	maxWeight := []float64{0.5, 0.5, 0.5}
	got, err := OptimizePortfolioMoments(interiorMean, interiorCov, nil, PortfolioConfig{
		Objective: MinimumVariance,
		MinWeight: minWeight,
		MaxWeight: maxWeight,
	})
	if err != nil {
		t.Fatalf("OptimizePortfolioMoments: %v", err)
	}
	sum := 0.0
	for i, v := range got.Weights {
		if v < minWeight[i]-1e-12 || v > maxWeight[i]+1e-12 {
			t.Errorf("weight %d = %.17g outside [%g, %g]", i, v, minWeight[i], maxWeight[i])
		}
		sum += v
	}
	assertClose(t, sum, 1, 1e-12)
}

func TestPortfolioMaximumSharpeIsNoWorseThanFrontier(t *testing.T) {
	rows, names := sampleReturns(t)
	table := returnsTable(rows, names)
	cfg := PortfolioConfig{Objective: MaximumSharpe, RiskFreeRate: 0.0001}

	best, err := OptimizePortfolio(table, cfg)
	if err != nil {
		t.Fatalf("OptimizePortfolio: %v", err)
	}
	frontier, err := EfficientFrontier(table, 50, cfg)
	if err != nil {
		t.Fatalf("EfficientFrontier: %v", err)
	}
	sweepBest := math.Inf(-1)
	for _, point := range frontier {
		if point.SharpeRatio > sweepBest {
			sweepBest = point.SharpeRatio
		}
	}
	if best.SharpeRatio < sweepBest-1e-6 {
		t.Errorf("MaximumSharpe %.17g is worse than the 50-point sweep best %.17g", best.SharpeRatio, sweepBest)
	}
	assertClose(t, best.SharpeRatio, (best.ExpectedReturn-cfg.RiskFreeRate)/best.Volatility, 1e-12)
}

func TestPortfolioMomentsEntryAgreesWithTableEntry(t *testing.T) {
	rows, names := sampleReturns(t)
	table := returnsTable(rows, names)
	cfg := PortfolioConfig{Objective: MinimumVariance}

	fromTable, err := OptimizePortfolio(table, cfg)
	if err != nil {
		t.Fatalf("OptimizePortfolio: %v", err)
	}
	mean, cov := momentsOf(rows)
	fromMoments, err := OptimizePortfolioMoments(mean, cov, names, cfg)
	if err != nil {
		t.Fatalf("OptimizePortfolioMoments: %v", err)
	}
	for i := range fromTable.Weights {
		assertClose(t, fromMoments.Weights[i], fromTable.Weights[i], 1e-12)
	}
	assertClose(t, fromMoments.ExpectedReturn, fromTable.ExpectedReturn, 1e-15)
	assertClose(t, fromMoments.Variance, fromTable.Variance, 1e-15)
}

func TestPortfolioWeightLookup(t *testing.T) {
	rows, names := sampleReturns(t)
	got, err := OptimizePortfolio(returnsTable(rows, names), PortfolioConfig{Objective: MinimumVariance})
	if err != nil {
		t.Fatalf("OptimizePortfolio: %v", err)
	}
	value, ok := got.Weight("B")
	if !ok {
		t.Fatalf(`Weight("B") reported not found`)
	}
	assertClose(t, value, got.Weights[1], 0)
	if value, ok := got.Weight("Z"); ok || value != 0 {
		t.Errorf(`Weight("Z") = %v, %v; want 0, false`, value, ok)
	}
	var nilResult *PortfolioResult
	if value, ok := nilResult.Weight("A"); ok || value != 0 {
		t.Errorf("nil receiver Weight = %v, %v; want 0, false", value, ok)
	}
}

// ---------------------------------------------------------------------------
// Requirement: efficient frontier sweep
// ---------------------------------------------------------------------------

func TestEfficientFrontierIsMonotone(t *testing.T) {
	rows, names := sampleReturns(t)
	points, err := EfficientFrontier(returnsTable(rows, names), 20, PortfolioConfig{})
	if err != nil {
		t.Fatalf("EfficientFrontier: %v", err)
	}
	if len(points) != 20 {
		t.Fatalf("got %d points, want 20", len(points))
	}
	for i := 1; i < len(points); i++ {
		if points[i].ExpectedReturn <= points[i-1].ExpectedReturn {
			t.Errorf("ExpectedReturn not strictly increasing at %d: %.17g then %.17g", i, points[i-1].ExpectedReturn, points[i].ExpectedReturn)
		}
		if points[i].Variance < points[i-1].Variance {
			t.Errorf("Variance decreased at %d: %.17g then %.17g", i, points[i-1].Variance, points[i].Variance)
		}
	}
}

func TestEfficientFrontierEndpoints(t *testing.T) {
	rows, names := sampleReturns(t)
	table := returnsTable(rows, names)
	points, err := EfficientFrontier(table, 20, PortfolioConfig{})
	if err != nil {
		t.Fatalf("EfficientFrontier: %v", err)
	}
	minVar, err := OptimizePortfolio(table, PortfolioConfig{Objective: MinimumVariance})
	if err != nil {
		t.Fatalf("OptimizePortfolio: %v", err)
	}
	assertClose(t, points[0].Variance, minVar.Variance, 1e-8)

	// The attainable maximum under the default [0, 1] bounds is the largest
	// column mean.
	mean, _ := momentsOf(rows)
	maxReturn := math.Inf(-1)
	for _, m := range mean {
		if m > maxReturn {
			maxReturn = m
		}
	}
	assertClose(t, points[len(points)-1].ExpectedReturn, maxReturn, 1e-8)
}

func TestEfficientFrontierRejectsTooFewPoints(t *testing.T) {
	rows, names := sampleReturns(t)
	if _, err := EfficientFrontier(returnsTable(rows, names), 1, PortfolioConfig{}); err == nil {
		t.Fatal("expected an error for points < 2")
	}
}

// ---------------------------------------------------------------------------
// Requirement: input validation and refusal
// ---------------------------------------------------------------------------

func TestPortfolioInfeasibleBounds(t *testing.T) {
	_, err := OptimizePortfolioMoments(interiorMean, interiorCov, nil, PortfolioConfig{
		MaxWeight: []float64{0.3, 0.3, 0.3},
	})
	if err == nil {
		t.Fatal("expected an infeasibility error")
	}
	if !strings.Contains(err.Error(), "MaxWeight") && !strings.Contains(strings.ToLower(err.Error()), "infeasible") {
		t.Errorf("error does not report infeasibility: %v", err)
	}
}

func TestPortfolioUnattainableTargetReturn(t *testing.T) {
	_, err := OptimizePortfolioMoments(interiorMean, interiorCov, nil, PortfolioConfig{
		Objective:    TargetReturn,
		TargetReturn: 0.5,
	})
	if err == nil {
		t.Fatal("expected an unattainable-target error")
	}
	if !strings.Contains(err.Error(), "TargetReturn") {
		t.Errorf("error does not name TargetReturn: %v", err)
	}
	if !strings.Contains(err.Error(), "0.014") {
		t.Errorf("error does not report the attainable range: %v", err)
	}
}

func TestPortfolioUnreadableCellNamesTheColumn(t *testing.T) {
	rows, names := sampleReturns(t)
	cols := make([]*insyra.DataList, len(names))
	for j := range names {
		vals := make([]any, len(rows))
		for i := range rows {
			vals[i] = rows[i][j]
		}
		if j == 1 {
			vals[3] = "n/a"
		}
		dl := insyra.NewDataList(vals...)
		dl.SetName(names[j])
		cols[j] = dl
	}
	_, err := OptimizePortfolio(insyra.NewDataTable(cols...), PortfolioConfig{})
	if err == nil {
		t.Fatal("expected an unreadable-cell error")
	}
	if !strings.Contains(err.Error(), "B") || !strings.Contains(err.Error(), "row 4") {
		t.Errorf("error does not name column B and row 4: %v", err)
	}
}

func TestPortfolioNonPSDCovarianceIsRefused(t *testing.T) {
	_, err := OptimizePortfolioMoments([]float64{0.01, 0.02}, [][]float64{{1, 2}, {2, 1}}, nil, PortfolioConfig{})
	if err == nil {
		t.Fatal("expected a non-PSD error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "positive semidefinite") {
		t.Errorf("error does not report a non-PSD covariance: %v", err)
	}
}

func TestPortfolioValidationErrors(t *testing.T) {
	rows, names := sampleReturns(t)
	table := returnsTable(rows, names)

	t.Run("nil table", func(t *testing.T) {
		if _, err := OptimizePortfolio(nil, PortfolioConfig{}); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("single column", func(t *testing.T) {
		single := returnsTable([][]float64{{1}, {2}, {3}}, []string{"A"})
		if _, err := OptimizePortfolio(single, PortfolioConfig{}); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("too few observations", func(t *testing.T) {
		short := returnsTable(rows[:3], names)
		_, err := OptimizePortfolio(short, PortfolioConfig{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "observations") {
			t.Errorf("error does not mention observations: %v", err)
		}
	})
	t.Run("bound length mismatch", func(t *testing.T) {
		_, err := OptimizePortfolio(table, PortfolioConfig{MinWeight: []float64{0, 0}})
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "MinWeight") {
			t.Errorf("error does not name MinWeight: %v", err)
		}
	})
	t.Run("lower above upper", func(t *testing.T) {
		_, err := OptimizePortfolio(table, PortfolioConfig{
			MinWeight: []float64{0.6, 0, 0},
			MaxWeight: []float64{0.5, 1, 1},
		})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("lower bounds sum above one", func(t *testing.T) {
		_, err := OptimizePortfolio(table, PortfolioConfig{MinWeight: []float64{0.5, 0.5, 0.5}})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("unknown objective", func(t *testing.T) {
		_, err := OptimizePortfolio(table, PortfolioConfig{Objective: PortfolioObjective(42)})
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "Objective") {
			t.Errorf("error does not name Objective: %v", err)
		}
	})
	t.Run("moments dimension mismatch", func(t *testing.T) {
		_, err := OptimizePortfolioMoments([]float64{0.01, 0.02}, interiorCov, nil, PortfolioConfig{})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("asymmetric covariance", func(t *testing.T) {
		_, err := OptimizePortfolioMoments([]float64{0.01, 0.02}, [][]float64{{0.1, 0.02}, {0.03, 0.1}}, nil, PortfolioConfig{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "symmetric") {
			t.Errorf("error does not report asymmetry: %v", err)
		}
	})
	t.Run("non-finite moment", func(t *testing.T) {
		_, err := OptimizePortfolioMoments([]float64{0.01, math.NaN()}, [][]float64{{0.1, 0.02}, {0.02, 0.1}}, nil, PortfolioConfig{})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("name count mismatch", func(t *testing.T) {
		_, err := OptimizePortfolioMoments(interiorMean, interiorCov, []string{"A", "B"}, PortfolioConfig{})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestPortfolioNonConvergenceIsReported(t *testing.T) {
	got, err := OptimizePortfolioMoments(interiorMean, interiorCov, nil, PortfolioConfig{
		Objective:     MinimumVariance,
		MaxIterations: 1,
	})
	if err != nil {
		t.Fatalf("OptimizePortfolioMoments: %v", err)
	}
	if got.Converged {
		t.Errorf("Converged = true with MaxIterations = 1")
	}
	if got.Iterations != 1 {
		t.Errorf("Iterations = %d, want 1", got.Iterations)
	}
	sum := 0.0
	for _, v := range got.Weights {
		sum += v
	}
	assertClose(t, sum, 1, 1e-12)
}
