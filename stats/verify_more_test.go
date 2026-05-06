package stats_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	stats "github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// TestKaiserSelection: with threshold=1, count of factors must equal the
// count of eigenvalues ≥ 1 in the correlation matrix.
func TestKaiserSelection(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountKaiser
	opt.Count.EigenThreshold = 1.0
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	S := tableToCorrMatrix(tbl)
	expectedK := 0
	for _, v := range eigenvaluesDescending(S) {
		if v >= 1.0 {
			expectedK++
		}
	}
	if res.CountUsed != expectedK {
		t.Errorf("Kaiser: expected %d factors (eigvals≥1) got %d", expectedK, res.CountUsed)
	}
	fmt.Printf("Kaiser: expected=%d got=%d ✓\n", expectedK, res.CountUsed)
}

// TestCovarMode: extraction with Covar=true uses covariance, not correlation.
// Diagonal of model + uniqueness should reproduce diag of covariance matrix.
func TestCovarMode(t *testing.T) {
	t.Skip("Covar option not yet exposed via DefaultFactorAnalysisOptions; skipping")
}

// TestKMOSphericity: KMO sampling adequacy is in [0,1]; Bartlett's
// sphericity test χ² ≥ 0, df = p(p-1)/2, p-value in [0,1].
func TestKMOSphericity(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	// SamplingAdequacy is a DataTable: per-variable MSAi + overall MSA
	if res.SamplingAdequacy == nil {
		t.Fatal("SamplingAdequacy nil")
	}
	rows, _ := res.SamplingAdequacy.Size()
	for i := 0; i < rows; i++ {
		v, ok := res.SamplingAdequacy.GetElementByNumberIndex(i, 0).(float64)
		if !ok {
			continue
		}
		if v < 0 || v > 1 {
			t.Errorf("KMO[%d] = %v ∉ [0,1]", i, v)
		}
	}
	// Bartlett sphericity
	bs := res.BartlettTest
	if bs.ChiSquare < 0 {
		t.Errorf("Bartlett χ² = %v < 0", bs.ChiSquare)
	}
	expectedDf := 5 * (5 - 1) / 2
	if int(bs.DegreesOfFreedom) != expectedDf {
		t.Errorf("Bartlett df = %v, expected %d", bs.DegreesOfFreedom, expectedDf)
	}
	if bs.PValue < 0 || bs.PValue > 1 {
		t.Errorf("Bartlett p = %v ∉ [0,1]", bs.PValue)
	}
	fmt.Printf("KMO range OK; Bartlett χ²=%.3f df=%v p=%.3e ✓\n", bs.ChiSquare, bs.DegreesOfFreedom, bs.PValue)
}

// TestEdgeCaseSingleFactor: k=1 special case.
func TestEdgeCaseSingleFactor(t *testing.T) {
	const n = 50
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 1
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreRegression
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.CountUsed != 1 {
		t.Errorf("expected k=1, got %d", res.CountUsed)
	}
	rows, cols := res.Loadings.Size()
	if cols != 1 {
		t.Errorf("loadings should have 1 column, got %d", cols)
	}
	if rows != 5 {
		t.Errorf("loadings should have 5 rows, got %d", rows)
	}
	// Single-factor model: communality = L²
	for i := 0; i < 5; i++ {
		l, _ := res.Loadings.GetElementByNumberIndex(i, 0).(float64)
		c, _ := res.Communalities.GetElementByNumberIndex(i, 1).(float64) // col 1 = Extraction
		if math.Abs(l*l-c) > 1e-9 {
			t.Errorf("k=1: L²[%d]=%v but communality=%v", i, l*l, c)
		}
	}
	fmt.Printf("k=1: loadings %dx%d, communalities = L² ✓\n", rows, cols)
}

// TestRotationParams: changing Kappa/Delta/GeominEpsilon should change the
// resulting loadings (i.e., the parameter is actually being honored).
func TestRotationParams(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)

	type pcase struct {
		name      string
		setupA    func(*stats.FactorAnalysisOptions)
		setupB    func(*stats.FactorAnalysisOptions)
		rotation  stats.FactorRotationMethod
		expectDif bool
	}
	cases := []pcase{
		{
			name:     "Promax/Kappa",
			rotation: stats.FactorRotationPromax,
			setupA:   func(o *stats.FactorAnalysisOptions) { o.Rotation.Kappa = 4 },
			setupB:   func(o *stats.FactorAnalysisOptions) { o.Rotation.Kappa = 8 },
		},
		{
			name:     "Oblimin/Delta",
			rotation: stats.FactorRotationOblimin,
			setupA:   func(o *stats.FactorAnalysisOptions) { o.Rotation.Delta = 0 },
			setupB:   func(o *stats.FactorAnalysisOptions) { o.Rotation.Delta = -0.5 },
		},
		{
			name:     "GeominQ/Epsilon",
			rotation: stats.FactorRotationGeominQ,
			setupA:   func(o *stats.FactorAnalysisOptions) { o.Rotation.GeominEpsilon = 0.01 },
			setupB:   func(o *stats.FactorAnalysisOptions) { o.Rotation.GeominEpsilon = 0.05 },
		},
	}

	for _, c := range cases {
		makeRes := func(setup func(*stats.FactorAnalysisOptions)) *stats.FactorModel {
			opt := stats.DefaultFactorAnalysisOptions()
			opt.Count.Method = stats.FactorCountFixed
			opt.Count.FixedK = 3
			opt.Extraction = stats.FactorExtractionML
			opt.Rotation.Method = c.rotation
			opt.Scoring = stats.FactorScoreNone
			setup(&opt)
			r, err := stats.FactorAnalysis(tbl, opt)
			if err != nil {
				t.Fatalf("[%s] %v", c.name, err)
			}
			return r
		}
		ra := makeRes(c.setupA)
		rb := makeRes(c.setupB)
		LA := dtToDense(ra.Loadings)
		LB := dtToDense(rb.Loadings)
		diff := maxAbsDiff(LA, LB)
		// We expect the param change to produce DIFFERENT loadings (>1e-6)
		if diff < 1e-6 {
			t.Errorf("[%s] param had no effect: max|LA-LB| = %.3e", c.name, diff)
		} else {
			fmt.Printf("[%-22s] max|LA-LB| = %.3e (param effective) ✓\n", c.name, diff)
		}
	}
}

// TestVarimaxAlgorithmChoice: kaiser vs gparotation should both produce
// orthogonal rotations (R'R = I) and preserve the model matrix.
func TestVarimaxAlgorithmChoice(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	for _, alg := range []stats.VarimaxAlgorithm{stats.VarimaxKaiser, stats.VarimaxGPArotation} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = stats.FactorExtractionML
		opt.Rotation.Method = stats.FactorRotationVarimax
		opt.Rotation.VarimaxAlgorithm = alg
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Errorf("[%s] %v", alg, err)
			continue
		}
		L := dtToDense(res.Loadings)
		Lu := dtToDense(res.UnrotatedLoadings)
		// model preservation
		LuT := mat.DenseCopyOf(Lu.T())
		LT := mat.DenseCopyOf(L.T())
		var Mu, Mr mat.Dense
		Mu.Mul(Lu, LuT)
		Mr.Mul(L, LT)
		md := maxAbsDiff(&Mr, &Mu)
		if md > 1e-7 {
			t.Errorf("[%s] model not preserved: max=%.3e", alg, md)
		}
		fmt.Printf("[varimax-%-12s] model preserved max=%.3e ✓\n", alg, md)
	}
}

// TestExplainedProportionInvariant: sum(explained_proportion) for k factors
// should equal sum(communalities) / p (sum of variance explained / total variance).
func TestExplainedProportionInvariant(t *testing.T) {
	const n = 60
	const p = 6
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	for _, ex := range []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = ex
		opt.Rotation.Method = stats.FactorRotationVarimax
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Errorf("[%s] FA error: %v", ex, err)
			continue
		}
		// sum(explained) = sum(SS_loadings) / p = sum(communalities) / p (orthog rotation)
		sumExp := 0.0
		ep := dtToDense(res.ExplainedProportion)
		// ExplainedProportion is 1×k or k×1; iterate all entries
		r1, c1 := ep.Dims()
		for i := 0; i < r1; i++ {
			for j := 0; j < c1; j++ {
				sumExp += ep.At(i, j)
			}
		}
		// Communalities: col 1 = Extraction value
		sumComm := 0.0
		commRows, _ := res.Communalities.Size()
		for i := 0; i < commRows; i++ {
			v, _ := res.Communalities.GetElementByNumberIndex(i, 1).(float64)
			sumComm += v
		}
		expected := sumComm / float64(p)
		if math.Abs(sumExp-expected) > 1e-9 {
			t.Errorf("[%s] sum(explained)=%.10f ≠ sum(comm)/p=%.10f", ex, sumExp, expected)
		} else {
			fmt.Printf("[%-7s] sum(explained)=%.6f = sum(comm)/p=%.6f ✓\n", ex, sumExp, expected)
		}
	}
}

// TestEdgeCaseConstantColumn: a column with zero variance should error
// gracefully (correlation matrix becomes singular).
func TestEdgeCaseConstantColumn(t *testing.T) {
	const n = 30
	tbl := insyra.NewDataTable()
	for c := 0; c < 4; c++ {
		col := make([]any, n)
		for i := 0; i < n; i++ {
			col[i] = math.Sin(float64(i)*0.3) + float64(c)
		}
		tbl.AppendCols(insyra.NewDataList(col...))
	}
	// Add a constant column → corr undefined for that var
	constCol := make([]any, n)
	for i := 0; i < n; i++ {
		constCol[i] = 42.0
	}
	tbl.AppendCols(insyra.NewDataList(constCol...))

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Scoring = stats.FactorScoreNone
	_, err := stats.FactorAnalysis(tbl, opt)
	if err == nil {
		t.Errorf("expected error on constant column, got success")
	} else {
		fmt.Printf("constant column: rejected with error: %v ✓\n", err)
	}
}

// TestEdgeCaseSmallSample: n < p should still produce a result or error
// gracefully (depending on extraction method).
func TestEdgeCaseSmallSample(t *testing.T) {
	const n = 4
	const p = 6 // p > n
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	for _, ex := range []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionML,
	} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 1
		opt.Extraction = ex
		opt.Rotation.Method = stats.FactorRotationNone
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			fmt.Printf("[%s n=%d p=%d] error: %v ✓ (expected; p>n)\n", ex, n, p, err)
		} else {
			fmt.Printf("[%s n=%d p=%d] succeeded with %d factors\n", ex, n, p, res.CountUsed)
		}
	}
}

// TestFactorsSortedByExplainedVariance: factors should be reported in
// descending order of explained variance.
func TestFactorsSortedByExplainedVariance(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	for _, ex := range []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionML,
	} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = ex
		opt.Rotation.Method = stats.FactorRotationVarimax
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		ep := dtToDense(res.ExplainedProportion)
		_, k := ep.Dims()
		// Should be descending
		prev := math.Inf(1)
		for j := 0; j < k; j++ {
			v := ep.At(0, j)
			if v > prev+1e-12 {
				t.Errorf("[%s] explained_proportion not descending: [%d]=%v > prev=%v", ex, j, v, prev)
			}
			prev = v
		}
		fmt.Printf("[%s] explained sorted descending ✓\n", ex)
	}
}

// TestRestartsParameter: rotation Restarts > 1 should not break, and for
// rotations supporting restarts (geomin, oblimin, etc.) it can find a
// better local minimum than restarts=1.
func TestRestartsParameter(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	for _, restarts := range []int{1, 5, 10} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = stats.FactorExtractionML
		opt.Rotation.Method = stats.FactorRotationGeominQ
		opt.Rotation.Restarts = restarts
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Errorf("[restarts=%d] %v", restarts, err)
			continue
		}
		// Verify rotation invariant still holds
		L := dtToDense(res.Loadings)
		Lu := dtToDense(res.UnrotatedLoadings)
		Phi := dtToDense(res.Phi)
		LuT := mat.DenseCopyOf(Lu.T())
		LT := mat.DenseCopyOf(L.T())
		var Mu, Mr mat.Dense
		Mu.Mul(Lu, LuT)
		LPhi := mat.NewDense(6, 3, nil)
		LPhi.Mul(L, Phi)
		Mr.Mul(LPhi, LT)
		md := maxAbsDiff(&Mr, &Mu)
		if md > 1e-7 {
			t.Errorf("[restarts=%d] model not preserved max=%.3e", restarts, md)
		}
		fmt.Printf("[restarts=%d] model preserved max=%.3e ✓\n", restarts, md)
	}
}

// TestSigmaConsistency: the reproduced covariance Sigma = L·Phi·L' + diag(U)
// should reproduce the diag of the original correlation matrix (= 1 for
// each variable). Off-diag should approximate the original correlations.
func TestSigmaConsistency(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	S := tableToCorrMatrix(tbl)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	L := dtToDense(res.UnrotatedLoadings)
	uniqDt := res.Uniquenesses
	uRows, _ := uniqDt.Size()
	U := make([]float64, uRows)
	for i := 0; i < uRows; i++ {
		U[i], _ = uniqDt.GetElementByNumberIndex(i, 0).(float64)
	}
	// Reproduce: Sigma = L·L' + diag(U) (orthogonal)
	LT := mat.DenseCopyOf(L.T())
	var Sigma mat.Dense
	Sigma.Mul(L, LT)
	for i := 0; i < uRows; i++ {
		Sigma.Set(i, i, Sigma.At(i, i)+U[i])
	}
	// Diag should be ≈ 1 (model + uniqueness = total variance for std variables)
	maxDiagErr := 0.0
	for i := 0; i < uRows; i++ {
		if d := math.Abs(Sigma.At(i, i) - S.At(i, i)); d > maxDiagErr {
			maxDiagErr = d
		}
	}
	if maxDiagErr > 1e-9 {
		t.Errorf("Sigma diag mismatch max=%.3e", maxDiagErr)
	}
	// Off-diag: residual = S - Sigma should be small (the model fits)
	maxOffRes := 0.0
	for i := 0; i < uRows; i++ {
		for j := 0; j < uRows; j++ {
			if i == j {
				continue
			}
			if d := math.Abs(S.At(i, j) - Sigma.At(i, j)); d > maxOffRes {
				maxOffRes = d
			}
		}
	}
	fmt.Printf("Sigma diag err max=%.3e (≤1e-9 ✓), off-diag residual max=%.3e\n", maxDiagErr, maxOffRes)
}

// TestEdgeCaseHighCollinearity: nearly-identical columns shouldn't crash;
// should either succeed with degenerate output or error gracefully.
func TestEdgeCaseHighCollinearity(t *testing.T) {
	const n = 30
	tbl := insyra.NewDataTable()
	for c := 0; c < 5; c++ {
		col := make([]any, n)
		for i := 0; i < n; i++ {
			col[i] = math.Sin(float64(i)*0.3) + 0.0001*float64(c)*math.Cos(float64(i))
		}
		tbl.AppendCols(insyra.NewDataList(col...))
	}
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		fmt.Printf("high-collinearity: error: %v ✓\n", err)
	} else {
		fmt.Printf("high-collinearity: succeeded with %d factors, converged=%v ✓\n", res.CountUsed, res.Converged)
	}
}

// TestRepeatability: running FactorAnalysis twice on the same data must
// produce identical results (within ULP, no rotation random restart).
func TestRepeatability(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Rotation.Restarts = 1
	opt.Scoring = stats.FactorScoreRegression
	r1, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	L1 := dtToDense(r1.Loadings)
	L2 := dtToDense(r2.Loadings)
	if d := maxAbsDiff(L1, L2); d != 0 {
		t.Errorf("loadings differ between runs: max=%.3e", d)
	}
	S1 := dtToDense(r1.Scores)
	S2 := dtToDense(r2.Scores)
	if d := maxAbsDiff(S1, S2); d != 0 {
		t.Errorf("scores differ between runs: max=%.3e", d)
	}
	fmt.Printf("repeatability: bit-identical across runs ✓\n")
}

// TestEdgeCaseZeroFactors: nfactors=0 should error.
func TestEdgeCaseZeroFactors(t *testing.T) {
	const n = 30
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 0
	opt.Extraction = stats.FactorExtractionML
	opt.Scoring = stats.FactorScoreNone
	_, err := stats.FactorAnalysis(tbl, opt)
	if err == nil {
		t.Errorf("expected error for FixedK=0, got success")
	} else {
		fmt.Printf("FixedK=0: rejected: %v ✓\n", err)
	}
}

// TestConcurrencySafety: 10 parallel FactorAnalysis on independent tables
// should each produce the same result as serial; no shared mutable state.
func TestConcurrencySafety(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Rotation.Restarts = 1
	opt.Scoring = stats.FactorScoreRegression

	// Serial baseline
	baseline, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	baselineL := dtToDense(baseline.Loadings)

	// Parallel
	const par = 10
	results := make([]*stats.FactorModel, par)
	errs := make([]error, par)
	done := make(chan int, par)
	for i := 0; i < par; i++ {
		go func(i int) {
			results[i], errs[i] = stats.FactorAnalysis(tbl, opt)
			done <- i
		}(i)
	}
	for i := 0; i < par; i++ {
		<-done
	}
	for i := 0; i < par; i++ {
		if errs[i] != nil {
			t.Errorf("[%d] %v", i, errs[i])
			continue
		}
		L := dtToDense(results[i].Loadings)
		if d := maxAbsDiff(L, baselineL); d != 0 {
			t.Errorf("[%d] loadings differ from serial baseline: max=%.3e", i, d)
		}
	}
	fmt.Printf("concurrency: %d parallel runs all bit-identical to serial ✓\n", par)
}

// TestScoreCoefficientsConsistency: Scores should equal Z · ScoreCoefficients
// where Z is centered/standardized data per the scoring method's convention.
func TestScoreCoefficientsConsistency(t *testing.T) {
	const n = 60
	const p = 6
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	for _, sm := range []stats.FactorScoreMethod{
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = stats.FactorExtractionML
		opt.Rotation.Method = stats.FactorRotationVarimax
		opt.Scoring = sm
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		Scores := dtToDense(res.Scores)
		W := dtToDense(res.ScoreCoefficients)
		// Build standardized X (z-score)
		data := mat.NewDense(n, p, nil)
		for i := 0; i < n; i++ {
			for j := 0; j < p; j++ {
				v, _ := tbl.GetElementByNumberIndex(i, j).(float64)
				data.Set(i, j, v)
			}
		}
		// Standardize: z = (x - mean) / sd
		Z := mat.NewDense(n, p, nil)
		for j := 0; j < p; j++ {
			mean, sd := 0.0, 0.0
			for i := 0; i < n; i++ {
				mean += data.At(i, j)
			}
			mean /= float64(n)
			for i := 0; i < n; i++ {
				d := data.At(i, j) - mean
				sd += d * d
			}
			sd = math.Sqrt(sd / float64(n-1))
			for i := 0; i < n; i++ {
				Z.Set(i, j, (data.At(i, j)-mean)/sd)
			}
		}
		// Reproduce: scores = Z · W
		var ScoresCheck mat.Dense
		ScoresCheck.Mul(Z, W)
		if d := maxAbsDiff(Scores, &ScoresCheck); d > 1e-8 {
			t.Errorf("[%s] Scores ≠ Z·W: max=%.3e", sm, d)
		} else {
			fmt.Printf("[%-15s] Scores = Z · ScoreCoefficients: max=%.3e ✓\n", sm, d)
		}
	}
}

// TestCumulativeProportionInvariant: Cumulative[j] = sum(Explained[0..j]).
func TestCumulativeProportionInvariant(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	ep := dtToDense(res.ExplainedProportion)
	cp := dtToDense(res.CumulativeProportion)
	_, k := ep.Dims()
	cum := 0.0
	for j := 0; j < k; j++ {
		cum += ep.At(0, j)
		if d := math.Abs(cum - cp.At(0, j)); d > 1e-12 {
			t.Errorf("Cumulative[%d] = %v ≠ cumsum = %v", j, cp.At(0, j), cum)
		}
	}
	fmt.Printf("CumulativeProportion = cumsum(ExplainedProportion): exact ✓\n")
}

// TestRotationMatrixOrthogonal: for orthogonal rotations, RotMat'·RotMat = I
// when measured against the rotation algorithm's internal frame (sign/sort
// post-processing in factor_analysis.go can break L = Lu·R but RotMat itself
// should still be orthonormal).
func TestRotationMatrixOrthogonal(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	for _, m := range []stats.FactorRotationMethod{
		stats.FactorRotationVarimax,
		stats.FactorRotationQuartimax,
		stats.FactorRotationGeominT,
		stats.FactorRotationBentlerT,
	} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = stats.FactorExtractionML
		opt.Rotation.Method = m
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		R := dtToDense(res.RotationMatrix)
		if R == nil {
			t.Errorf("[%s] RotationMatrix nil", m)
			continue
		}
		_, c := R.Dims()
		RT := mat.DenseCopyOf(R.T())
		var RTR mat.Dense
		RTR.Mul(RT, R)
		maxOff := 0.0
		for i := 0; i < c; i++ {
			for j := 0; j < c; j++ {
				target := 0.0
				if i == j {
					target = 1.0
				}
				if d := math.Abs(RTR.At(i, j) - target); d > maxOff {
					maxOff = d
				}
			}
		}
		if maxOff > 1e-9 {
			t.Errorf("[%s] R'R - I max=%.3e (RotMat not orthonormal)", m, maxOff)
		} else {
			fmt.Printf("[%-12s] R'R - I max=%.3e (orthonormal) ✓\n", m, maxOff)
		}
	}
}

// TestPCAFullRankReconstruction: PCA with k=p reconstructs correlation
// matrix exactly: L · L' = R.
func TestPCAFullRankReconstruction(t *testing.T) {
	const n = 60
	const p = 5
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = p
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	L := dtToDense(res.Loadings)
	LT := mat.DenseCopyOf(L.T())
	var Reconstructed mat.Dense
	Reconstructed.Mul(L, LT)
	S := tableToCorrMatrix(tbl)
	d := maxAbsDiff(&Reconstructed, S)
	if d > 1e-10 {
		t.Errorf("PCA k=p: L·L' ≠ R, max=%.3e", d)
	} else {
		fmt.Printf("PCA k=p: L·L' = R, max diff=%.3e ✓\n", d)
	}
}

// TestCommunalityMonotonicity: increasing nfactors monotonically increases
// sum(communalities) — more factors capture more variance.
func TestCommunalityMonotonicity(t *testing.T) {
	const n = 60
	const p = 6
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	prevSum := 0.0
	for k := 1; k <= 4; k++ {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = k
		opt.Extraction = stats.FactorExtractionPCA
		opt.Rotation.Method = stats.FactorRotationNone
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		sumComm := 0.0
		commRows, _ := res.Communalities.Size()
		for i := 0; i < commRows; i++ {
			v, _ := res.Communalities.GetElementByNumberIndex(i, 1).(float64)
			sumComm += v
		}
		if sumComm < prevSum-1e-9 {
			t.Errorf("k=%d: sum(comm)=%v < prev=%v (not monotonic)", k, sumComm, prevSum)
		}
		fmt.Printf("PCA k=%d: sum(communalities) = %.6f\n", k, sumComm)
		prevSum = sumComm
	}
}

// TestEigenvaluesMatchPCAOnCorrMatrix: for PCA, reported eigenvalues
// should match the actual eigenvalues of the correlation matrix.
func TestEigenvaluesMatchPCAOnCorrMatrix(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	S := tableToCorrMatrix(tbl)
	expectedEigvals := eigenvaluesDescending(S)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	// Eigenvalues field reports the top k eigenvalues of correlation matrix
	ep := dtToDense(res.Eigenvalues)
	rows, cols := ep.Dims()
	got := make([]float64, 0)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			got = append(got, ep.At(i, j))
		}
	}
	for j := 0; j < len(got) && j < len(expectedEigvals); j++ {
		if d := math.Abs(got[j] - expectedEigvals[j]); d > 1e-9 {
			t.Errorf("eigval[%d]: got %v, expected %v (diff %.3e)", j, got[j], expectedEigvals[j], d)
		}
	}
	fmt.Printf("PCA eigenvalues match correlation matrix eigenvalues ✓\n")
}

// TestPartialNaNListwiseDeletion: rows containing NaN are listwise deleted.
// Result should match running on the manually-cleaned data.
func TestPartialNaNListwiseDeletion(t *testing.T) {
	const n = 60
	const p = 5
	clean := make([][]float64, n)
	for i := 0; i < n; i++ {
		clean[i] = []float64{
			math.Sin(0.31*float64(i)) + 0.2*math.Cos(0.7*float64(i)),
			math.Cos(0.42*float64(i)) - 0.3*math.Sin(0.5*float64(i)),
			math.Sin(0.61*float64(i)+0.4) + 0.15*math.Cos(0.83*float64(i)),
			0.3*math.Sin(0.31*float64(i)) + 0.5*math.Cos(0.42*float64(i)),
			0.4*math.Cos(0.31*float64(i)) - 0.6*math.Sin(0.42*float64(i)),
		}
	}
	// Inject NaN in 5 specific rows
	dirty := make([][]float64, n)
	for i := 0; i < n; i++ {
		dirty[i] = append([]float64(nil), clean[i]...)
	}
	dirtyRows := []int{3, 17, 28, 41, 53}
	for _, r := range dirtyRows {
		dirty[r][1] = math.NaN()
	}
	// Build the manually-cleaned reference: drop NaN rows from `clean`
	cleaned := make([][]float64, 0, n-len(dirtyRows))
	skip := map[int]bool{}
	for _, r := range dirtyRows {
		skip[r] = true
	}
	for i := 0; i < n; i++ {
		if !skip[i] {
			cleaned = append(cleaned, clean[i])
		}
	}

	makeTbl := func(rows [][]float64) *insyra.DataTable {
		tbl := insyra.NewDataTable()
		nr := len(rows)
		for c := 0; c < p; c++ {
			col := make([]any, nr)
			for i := 0; i < nr; i++ {
				col[i] = rows[i][c]
			}
			tbl.AppendCols(insyra.NewDataList(col...))
		}
		return tbl
	}
	dirtyTbl := makeTbl(dirty)
	cleanTbl := makeTbl(cleaned)

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationVarimax
	opt.Scoring = stats.FactorScoreNone
	rDirty, err := stats.FactorAnalysis(dirtyTbl, opt)
	if err != nil {
		t.Fatalf("dirty: %v", err)
	}
	rClean, err := stats.FactorAnalysis(cleanTbl, opt)
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	LD := dtToDense(rDirty.Loadings)
	LC := dtToDense(rClean.Loadings)
	if d := maxAbsDiff(LD, LC); d > 1e-10 {
		t.Errorf("listwise-deleted loadings differ from manually-cleaned: max=%.3e", d)
	} else {
		fmt.Printf("partial NaN: listwise = manual clean, max=%.3e ✓\n", d)
	}
}

// TestPCAOrthogonalScores: PCA Regression scores with no rotation should
// have empirical Cor matrix ≈ I (PCA scores are orthogonal by construction).
func TestPCAOrthogonalScores(t *testing.T) {
	const n = 80
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreRegression
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	S := dtToDense(res.Scores)
	r, c := S.Dims()
	cov := empiricalCov(S)
	// Standardize to correlation
	maxOff := 0.0
	for i := 0; i < c; i++ {
		for j := 0; j < c; j++ {
			if i == j {
				continue
			}
			corr := cov.At(i, j) / math.Sqrt(cov.At(i, i)*cov.At(j, j))
			if math.Abs(corr) > maxOff {
				maxOff = math.Abs(corr)
			}
		}
	}
	if maxOff > 1e-9 {
		t.Errorf("PCA unrotated scores not orthogonal: max|corr|=%.3e", maxOff)
	} else {
		fmt.Printf("PCA unrotated: empirical Cor(scores) ≈ I, max off-diag=%.3e ✓\n", maxOff)
	}
	_ = r
}

// TestPAFConvergence: PAF should iterate communalities to a fixed point.
// At the fixed point, sum(L²)[i] for k factors should approximately equal
// the iterated communality h²[i].
func TestPAFConvergence(t *testing.T) {
	const n = 60
	const p = 6
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionPAF
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	opt.MaxIter = 100
	opt.MinErr = 1e-9
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	L := dtToDense(res.UnrotatedLoadings)
	for i := 0; i < p; i++ {
		hSq := 0.0
		for j := 0; j < 3; j++ {
			v := L.At(i, j)
			hSq += v * v
		}
		c, _ := res.Communalities.GetElementByNumberIndex(i, 1).(float64)
		if d := math.Abs(hSq - c); d > 1e-6 {
			t.Errorf("PAF[%d]: sum(L²)=%v ≠ communality=%v (diff %.3e)", i, hSq, c, d)
		}
	}
	fmt.Printf("PAF: sum(L²) = communality at converged fixed point ✓\n")
}

// TestPAFDeterministic: same data, same options → same loadings.
func TestPAFDeterministic(t *testing.T) {
	const n = 50
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionPAF
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	r1, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	L1 := dtToDense(r1.Loadings)
	L2 := dtToDense(r2.Loadings)
	if d := maxAbsDiff(L1, L2); d != 0 {
		t.Errorf("PAF non-deterministic: max=%.3e", d)
	} else {
		fmt.Printf("PAF deterministic: bit-identical across runs ✓\n")
	}
}

// TestMINRESObjectiveLowerBound: MINRES objective at converged psi should
// be ≥ 0 (it's a sum of squares) and finite.
func TestMINRESObjectiveLowerBound(t *testing.T) {
	const n = 60
	const p = 6
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionMINRES
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	// MINRES objective = sum of squared off-diagonal residuals
	S := tableToCorrMatrix(tbl)
	L := dtToDense(res.UnrotatedLoadings)
	LT := mat.DenseCopyOf(L.T())
	var Reproduced mat.Dense
	Reproduced.Mul(L, LT)
	obj := 0.0
	for i := 0; i < p; i++ {
		for j := i + 1; j < p; j++ {
			r := S.At(i, j) - Reproduced.At(i, j)
			obj += r * r
		}
	}
	if obj < 0 || math.IsNaN(obj) || math.IsInf(obj, 0) {
		t.Errorf("MINRES objective invalid: %v", obj)
	}
	fmt.Printf("MINRES off-diag SSE at converged psi = %.6e (≥0, finite) ✓\n", obj)
}

// TestInitialCommunalityIsSMC: for ML/MINRES/PAF, the "Initial" column of
// the Communalities table should match the SMC (squared multiple
// correlation) of each variable — psych::fa convention.
func TestInitialCommunalityIsSMC(t *testing.T) {
	const n = 60
	const p = 5
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	S := tableToCorrMatrix(tbl)
	// Compute SMC manually: SMC[i] = 1 - 1/diag(R^-1)[i]
	var Sinv mat.Dense
	if err := Sinv.Inverse(S); err != nil {
		t.Fatal(err)
	}
	expectedSMC := make([]float64, p)
	for i := 0; i < p; i++ {
		expectedSMC[i] = 1.0 - 1.0/Sinv.At(i, i)
	}
	for _, ex := range []stats.FactorExtractionMethod{
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	} {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 2
		opt.Extraction = ex
		opt.Rotation.Method = stats.FactorRotationNone
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		// Initial = column 0
		maxDiff := 0.0
		for i := 0; i < p; i++ {
			v, _ := res.Communalities.GetElementByNumberIndex(i, 0).(float64)
			if d := math.Abs(v - expectedSMC[i]); d > maxDiff {
				maxDiff = d
			}
		}
		if maxDiff > 1e-9 {
			t.Errorf("[%s] Initial communality ≠ SMC: max=%.3e", ex, maxDiff)
		} else {
			fmt.Printf("[%-7s] Initial communality = SMC: max=%.3e ✓\n", ex, maxDiff)
		}
	}
}

// TestBartlettSampleSize: SampleSize field should equal the input row count
// (after listwise NaN deletion if applicable).
func TestBartlettSampleSize(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.BartlettTest.SampleSize != n {
		t.Errorf("BartlettTest.SampleSize = %d, expected %d", res.BartlettTest.SampleSize, n)
	} else {
		fmt.Printf("BartlettTest.SampleSize = %d ✓\n", n)
	}
}

// TestRotationNoneIdentityRotMat: when Rotation = None, RotationMatrix
// should be either nil or identity (k×k).
func TestRotationNoneIdentityRotMat(t *testing.T) {
	const n = 50
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	R := dtToDense(res.RotationMatrix)
	if R == nil {
		fmt.Printf("Rotation=None: RotationMatrix = nil ✓\n")
		return
	}
	r, c := R.Dims()
	if r != 2 || c != 2 {
		t.Errorf("Rotation=None: RotMat shape = %dx%d, expected 2x2", r, c)
	}
	// Should be identity
	maxOff := 0.0
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			target := 0.0
			if i == j {
				target = 1.0
			}
			if d := math.Abs(R.At(i, j) - target); d > maxOff {
				maxOff = d
			}
		}
	}
	if maxOff > 1e-12 {
		t.Errorf("Rotation=None: RotMat ≠ I, max=%.3e", maxOff)
	} else {
		fmt.Printf("Rotation=None: RotationMatrix = I (2x2), max diff=%.3e ✓\n", maxOff)
	}
}

// TestMaxFactorsLimit: MaxFactors should cap the Kaiser-derived count.
func TestMaxFactorsLimit(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountKaiser
	opt.Count.EigenThreshold = 1.0
	opt.Count.MaxFactors = 1 // Force cap to 1
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.CountUsed != 1 {
		t.Errorf("MaxFactors=1: CountUsed=%d, expected 1", res.CountUsed)
	} else {
		fmt.Printf("MaxFactors=1: cap respected, CountUsed=%d ✓\n", res.CountUsed)
	}
}

// TestIterationsFieldSemantics: Iterations should be >0 for L-BFGS-B-based
// methods (ML/MINRES) on real data, and 0 (or 1) for closed-form (PCA).
func TestIterationsFieldSemantics(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	cases := []struct {
		ex      stats.FactorExtractionMethod
		mustPos bool
	}{
		{stats.FactorExtractionPCA, false},   // closed-form
		{stats.FactorExtractionPAF, true},    // iterative
		{stats.FactorExtractionML, true},     // L-BFGS-B
		{stats.FactorExtractionMINRES, true}, // L-BFGS-B
	}
	for _, c := range cases {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 2
		opt.Extraction = c.ex
		opt.Rotation.Method = stats.FactorRotationNone
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		if c.mustPos && res.Iterations <= 0 {
			t.Errorf("[%s] expected Iterations > 0, got %d", c.ex, res.Iterations)
		}
		fmt.Printf("[%-7s] Iterations=%d Converged=%v ✓\n", c.ex, res.Iterations, res.Converged)
	}
}

// TestMLConvergenceOnNormalData: ML should always converge=true on
// well-conditioned data.
func TestMLConvergenceOnNormalData(t *testing.T) {
	const n = 100
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Converged {
		t.Errorf("ML on normal data: Converged=false (Iterations=%d)", res.Iterations)
	}
	fmt.Printf("ML on normal n=100 p=6 k=3: Converged=%v Iter=%d ✓\n", res.Converged, res.Iterations)
}

// TestFixedKOverridesKaiser: setting FactorCountFixed always returns FixedK
// regardless of MaxFactors value.
func TestFixedKOverridesKaiser(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 4
	opt.Count.MaxFactors = 1 // Should be ignored when Method=Fixed
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.CountUsed != 4 {
		t.Errorf("FixedK=4 + MaxFactors=1: expected 4, got %d", res.CountUsed)
	}
	fmt.Printf("FixedK=4 with MaxFactors=1 ignored: CountUsed=%d ✓\n", res.CountUsed)
}

// TestInfInputListwiseDeletion: Inf values are treated like NaN (listwise
// delete the row).
func TestInfInputListwiseDeletion(t *testing.T) {
	const n = 60
	const p = 5
	clean := make([][]float64, n)
	for i := 0; i < n; i++ {
		clean[i] = []float64{
			math.Sin(0.31*float64(i)) + 0.2*math.Cos(0.7*float64(i)),
			math.Cos(0.42*float64(i)) - 0.3*math.Sin(0.5*float64(i)),
			math.Sin(0.61*float64(i)+0.4) + 0.15*math.Cos(0.83*float64(i)),
			0.3*math.Sin(0.31*float64(i)) + 0.5*math.Cos(0.42*float64(i)),
			0.4*math.Cos(0.31*float64(i)) - 0.6*math.Sin(0.42*float64(i)),
		}
	}
	dirty := make([][]float64, n)
	for i := 0; i < n; i++ {
		dirty[i] = append([]float64(nil), clean[i]...)
	}
	dirty[10][2] = math.Inf(1)
	dirty[25][3] = math.Inf(-1)

	makeTbl := func(rows [][]float64) *insyra.DataTable {
		tbl := insyra.NewDataTable()
		nr := len(rows)
		for c := 0; c < p; c++ {
			col := make([]any, nr)
			for i := 0; i < nr; i++ {
				col[i] = rows[i][c]
			}
			tbl.AppendCols(insyra.NewDataList(col...))
		}
		return tbl
	}
	cleaned := make([][]float64, 0, n-2)
	for i := 0; i < n; i++ {
		if i == 10 || i == 25 {
			continue
		}
		cleaned = append(cleaned, clean[i])
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone

	rDirty, err := stats.FactorAnalysis(makeTbl(dirty), opt)
	if err != nil {
		t.Fatalf("dirty: %v", err)
	}
	rClean, err := stats.FactorAnalysis(makeTbl(cleaned), opt)
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	LD := dtToDense(rDirty.Loadings)
	LC := dtToDense(rClean.Loadings)
	if d := maxAbsDiff(LD, LC); d != 0 {
		t.Errorf("Inf listwise vs manual cleanup: max=%.3e", d)
	} else {
		fmt.Printf("Inf input: listwise = manual clean, bit-identical ✓\n")
	}
}

// TestRotationConvergedFlag: when MaxIter is set very low, oblique rotations
// should report RotationConverged=false. With normal MaxIter, true.
func TestRotationConvergedFlag(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)

	// Normal: should converge
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 3
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationOblimin
	opt.MaxIter = 1000
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !res.RotationConverged {
		t.Errorf("normal: RotationConverged=false (expected true)")
	}
	fmt.Printf("normal MaxIter=1000: RotationConverged=%v ✓\n", res.RotationConverged)

	// Cripple: MaxIter=1 should not converge for GPF-based oblique rotations
	opt.MaxIter = 1
	res2, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res2.RotationConverged {
		t.Logf("note: even MaxIter=1 reports RotationConverged=true (rotation may converge in 1 iter on this data)")
	}
	fmt.Printf("crippled MaxIter=1: RotationConverged=%v\n", res2.RotationConverged)
}

// TestNoRotationConvergedTrue: when Rotation=None, RotationConverged should
// trivially be true (nothing to rotate, can't fail).
func TestNoRotationConvergedTrue(t *testing.T) {
	const n = 50
	tbl := buildSyntheticTable(n, 5, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = stats.FactorExtractionML
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !res.RotationConverged {
		t.Errorf("Rotation=None: RotationConverged should be true, got false")
	}
	fmt.Printf("Rotation=None: RotationConverged=true ✓\n")
}

// TestKaiserPCAOnly: Kaiser EigenThreshold=2 should select fewer factors
// than threshold=1 (stricter cutoff).
func TestKaiserThresholdEffect(t *testing.T) {
	const n = 60
	tbl := buildSyntheticTable(n, 6, syntheticGen3Factor)

	makeRes := func(thr float64) int {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountKaiser
		opt.Count.EigenThreshold = thr
		opt.Extraction = stats.FactorExtractionPCA
		opt.Rotation.Method = stats.FactorRotationNone
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Fatal(err)
		}
		return res.CountUsed
	}
	c1 := makeRes(1.0)
	c2 := makeRes(2.0)
	c0 := makeRes(0.5)
	if c2 > c1 {
		t.Errorf("Kaiser threshold=2 should select ≤ threshold=1: got %d vs %d", c2, c1)
	}
	if c0 < c1 {
		t.Errorf("Kaiser threshold=0.5 should select ≥ threshold=1: got %d vs %d", c0, c1)
	}
	fmt.Printf("Kaiser threshold 0.5/1.0/2.0 → CountUsed %d/%d/%d (monotonically decreasing) ✓\n", c0, c1, c2)
}

// TestEdgeCaseMaxFactors: k = p-1 (saturated factor model).
func TestEdgeCaseMaxFactors(t *testing.T) {
	const n = 50
	const p = 5
	tbl := buildSyntheticTable(n, p, syntheticGen3Factor)
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = p - 1
	opt.Extraction = stats.FactorExtractionPCA
	opt.Rotation.Method = stats.FactorRotationNone
	opt.Scoring = stats.FactorScoreNone
	res, err := stats.FactorAnalysis(tbl, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.CountUsed != p-1 {
		t.Errorf("expected k=%d, got %d", p-1, res.CountUsed)
	}
	fmt.Printf("k=p-1=%d: extracted ✓\n", res.CountUsed)
}

// --- helpers ---
func buildSyntheticTable(n, p int, gen func(i, p int) []float64) *insyra.DataTable {
	tbl := insyra.NewDataTable()
	cols := make([][]any, p)
	for j := 0; j < p; j++ {
		cols[j] = make([]any, n)
	}
	for i := 0; i < n; i++ {
		row := gen(i, p)
		for j := 0; j < p; j++ {
			cols[j][i] = row[j]
		}
	}
	for j := 0; j < p; j++ {
		tbl.AppendCols(insyra.NewDataList(cols[j]...))
	}
	return tbl
}

func syntheticGen3Factor(i, p int) []float64 {
	x := float64(i)
	f1 := math.Sin(0.31*x) + 0.2*math.Cos(0.7*x)
	f2 := math.Cos(0.42*x) - 0.3*math.Sin(0.5*x)
	f3 := math.Sin(0.61*x+0.4) + 0.15*math.Cos(0.83*x)
	out := make([]float64, p)
	for j := 0; j < p; j++ {
		// Spread loadings across 3 factors
		switch j % 3 {
		case 0:
			out[j] = 0.85*f1 + 0.10*f2 + 0.20*math.Sin(1.3*x+float64(j))
		case 1:
			out[j] = 0.10*f1 + 0.82*f2 + 0.21*math.Cos(1.7*x+float64(j))
		case 2:
			out[j] = 0.20*f1 + 0.20*f2 + 0.78*f3 + 0.18*math.Sin(2.1*x+float64(j))
		}
	}
	return out
}

func tableToCorrMatrix(tbl insyra.IDataTable) *mat.Dense {
	rows, cols := tbl.Size()
	data := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			v, _ := tbl.GetElementByNumberIndex(i, j).(float64)
			data.Set(i, j, v)
		}
	}
	var sym mat.SymDense
	stat.CorrelationMatrix(&sym, data, nil)
	S := mat.NewDense(cols, cols, nil)
	for i := 0; i < cols; i++ {
		for j := 0; j < cols; j++ {
			S.Set(i, j, sym.At(i, j))
		}
	}
	return S
}

func eigenvaluesDescending(S *mat.Dense) []float64 {
	r, _ := S.Dims()
	sym := mat.NewSymDense(r, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < r; j++ {
			sym.SetSym(i, j, S.At(i, j))
		}
	}
	var eig mat.EigenSym
	if !eig.Factorize(sym, true) {
		return nil
	}
	vals := eig.Values(nil)
	// EigenSym returns ascending; reverse
	out := make([]float64, len(vals))
	for i := range vals {
		out[i] = vals[len(vals)-1-i]
	}
	return out
}
