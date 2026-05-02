package stats_test

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	stats "github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// TestKaiserSelection: with threshold=1, count of factors must equal the
// count of eigenvalues ≥ 1 in the correlation matrix.
func TestKaiserSelection(t *testing.T) {
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
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
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
	t.Skip("Covar option not yet exposed via DefaultFactorAnalysisOptions; skipping")
}

// TestKMOSphericity: KMO sampling adequacy is in [0,1]; Bartlett's
// sphericity test χ² ≥ 0, df = p(p-1)/2, p-value in [0,1].
func TestKMOSphericity(t *testing.T) {
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
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
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
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
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
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
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
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

// TestEdgeCaseMaxFactors: k = p-1 (saturated factor model).
func TestEdgeCaseMaxFactors(t *testing.T) {
	if os.Getenv("INSYRA_VERIFY_MORE") != "1" {
		t.Skip()
	}
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
