package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestKruskalWallis_RExample(t *testing.T) {
	// R baseline: kruskal.test(list(g1, g2, g3))
	//   H = 0.7714285714285722
	//   p = 0.6799647735788936
	//   df = 2
	g1 := insyra.NewDataList(2.9, 3.0, 2.5, 2.6, 3.2)
	g2 := insyra.NewDataList(3.8, 2.7, 4.0, 2.4)
	g3 := insyra.NewDataList(2.8, 3.4, 3.7, 2.2, 2.0)
	res, err := stats.KruskalWallis(g1, g2, g3)
	if err != nil {
		t.Fatalf("KruskalWallis error: %v", err)
	}
	if math.Abs(res.Statistic-0.7714285714285722) > 1e-9 {
		t.Errorf("H mismatch: got %v, want 0.7714285714285722", res.Statistic)
	}
	if math.Abs(res.PValue-0.6799647735788936) > 1e-9 {
		t.Errorf("p mismatch: got %v, want 0.6799647735788936", res.PValue)
	}
	if res.DF == nil || *res.DF != 2 {
		t.Errorf("DF mismatch: got %v, want 2", res.DF)
	}
	if res.NTotal != 14 {
		t.Errorf("NTotal mismatch: got %d, want 14", res.NTotal)
	}
}

func TestFriedman_RExample(t *testing.T) {
	// R baseline: friedman.test(matrix)
	//   Q = 2.512820512820513
	//   p = 0.2846741015315938
	//   df = 2
	subjects := []insyra.IDataList{
		insyra.NewDataList(5.40, 5.50, 5.55),
		insyra.NewDataList(5.85, 5.70, 5.75),
		insyra.NewDataList(5.20, 5.60, 5.50),
		insyra.NewDataList(5.55, 5.50, 5.40),
		insyra.NewDataList(5.90, 5.85, 5.70),
		insyra.NewDataList(5.45, 5.55, 5.60),
		insyra.NewDataList(5.40, 5.40, 5.35),
		insyra.NewDataList(5.45, 5.50, 5.35),
		insyra.NewDataList(5.25, 5.15, 5.00),
		insyra.NewDataList(5.85, 5.80, 5.70),
	}
	res, err := stats.FriedmanTest(subjects...)
	if err != nil {
		t.Fatalf("FriedmanTest error: %v", err)
	}
	if math.Abs(res.Statistic-2.512820512820513) > 1e-9 {
		t.Errorf("Q mismatch: got %v, want 2.512820512820513", res.Statistic)
	}
	if math.Abs(res.PValue-0.2846741015315938) > 1e-9 {
		t.Errorf("p mismatch: got %v, want 0.2846741015315938", res.PValue)
	}
	if res.DF == nil || *res.DF != 2 {
		t.Errorf("DF mismatch: got %v, want 2", res.DF)
	}
	if res.NSubjects != 10 {
		t.Errorf("NSubjects mismatch: got %d, want 10", res.NSubjects)
	}
	if res.KConditions != 3 {
		t.Errorf("KConditions mismatch: got %d, want 3", res.KConditions)
	}
}

func TestMannWhitneyU_RExactExample(t *testing.T) {
	// R baseline: wilcox.test(x, y, conf.int=TRUE)
	//   W (= U1) = 59
	//   p.value = 0.02739613327848622
	//   estimate (HL shift) = 7
	//   conf.int = [1, 14]
	x := insyra.NewDataList(15.0, 18, 22, 11, 30, 14, 26, 25)
	y := insyra.NewDataList(10.0, 9, 13, 17, 7, 12, 19, 8, 20)
	res, err := stats.MannWhitneyU(x, y, stats.TwoSided)
	if err != nil {
		t.Fatalf("MannWhitneyU error: %v", err)
	}
	if math.Abs(res.U1-59) > 1e-9 {
		t.Errorf("U1 mismatch: got %v, want 59", res.U1)
	}
	if math.Abs(res.U2-13) > 1e-9 {
		t.Errorf("U2 mismatch: got %v, want 13 (8*9 - 59)", res.U2)
	}
	if math.Abs(res.Statistic-13) > 1e-9 {
		t.Errorf("Statistic (min) mismatch: got %v, want 13", res.Statistic)
	}
	if math.Abs(res.PValue-0.02739613327848622) > 1e-9 {
		t.Errorf("p-value mismatch: got %v, want 0.02739613327848622", res.PValue)
	}
	if res.Method != "exact" {
		t.Errorf("Method mismatch: got %q, want exact", res.Method)
	}
	if res.CI == nil {
		t.Fatal("CI is nil")
	}
	if math.Abs(res.CI[0]-1) > 1e-9 {
		t.Errorf("CI lower mismatch: got %v, want 1", res.CI[0])
	}
	if math.Abs(res.CI[1]-14) > 1e-9 {
		t.Errorf("CI upper mismatch: got %v, want 14", res.CI[1])
	}
}

func TestPairedWilcoxon_RExactExample(t *testing.T) {
	// R baseline: wilcox.test(x, y, paired=TRUE, conf.int=TRUE)
	//   statistic = 40
	//   p.value  = 0.0390625
	//   estimate (pseudo-median) = 0.46
	//   conf.int = [0.01, 0.786]
	x := insyra.NewDataList(1.83, 0.50, 1.62, 2.48, 1.68, 1.88, 1.55, 3.06, 1.30)
	y := insyra.NewDataList(0.878, 0.647, 0.598, 2.05, 1.06, 1.29, 1.06, 3.14, 1.29)

	res, err := stats.PairedWilcoxon(x, y, stats.TwoSided)
	if err != nil {
		t.Fatalf("PairedWilcoxon error: %v", err)
	}
	if math.Abs(res.Statistic-40) > 1e-9 {
		t.Errorf("W+ mismatch: got %v, want 40", res.Statistic)
	}
	if math.Abs(res.PValue-0.0390625) > 1e-9 {
		t.Errorf("p-value mismatch: got %v, want 0.0390625", res.PValue)
	}
	if res.Method != "exact" {
		t.Errorf("Method mismatch: got %q, want exact", res.Method)
	}
	if res.NEffective != 9 {
		t.Errorf("NEffective mismatch: got %d, want 9", res.NEffective)
	}
	if res.CI == nil {
		t.Fatal("CI is nil")
	}
	if math.Abs(res.CI[0]-0.01) > 1e-9 {
		t.Errorf("CI lower mismatch: got %v, want 0.01", res.CI[0])
	}
	if math.Abs(res.CI[1]-0.786) > 1e-9 {
		t.Errorf("CI upper mismatch: got %v, want 0.786", res.CI[1])
	}
}
