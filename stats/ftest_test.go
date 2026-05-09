package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
	"github.com/HazelnutParadise/insyra/stats"
)

const (
	tolF    = 1e-12
	tolFP   = 1e-12
	tolBart = 1e-12
)

// ============================================================
// F-test for variance equality (two-tailed F = max(v1,v2) / min(v1,v2))
// ============================================================

type fVarCase struct {
	name        string
	data1, data2 []float64
	prefix      string
}

func TestFTestForVarianceEquality_R(t *testing.T) {
	cases := []fVarCase{
		{name: "fv_basic",
			data1: []float64{10, 12, 9, 11},
			data2: []float64{20, 19, 21, 22},
			prefix: "fv_basic",
		},
		{name: "fv_unequal",
			data1: []float64{1, 2, 3, 4, 5},
			data2: []float64{10, 30, 50, 70, 90, 100, 5},
			prefix: "fv_unequal",
		},
		{name: "fv_largeN",
			data1: anovaDump.get(t, "fv_largeN_1"),
			data2: anovaDump.get(t, "fv_largeN_2"),
			prefix: "fv_largeN",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.FTestForVarianceEquality(
				insyra.NewDataList(c.data1),
				insyra.NewDataList(c.data2))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expF := anovaRef.get(t, c.prefix+".F")
			expP := anovaRef.get(t, c.prefix+".P")
			expDF1 := anovaRef.get(t, c.prefix+".DF1")
			expDF2 := anovaRef.get(t, c.prefix+".DF2")
			if !aClose(r.Statistic, expF, tolF) {
				t.Errorf("F: got %.17g, want %.17g", r.Statistic, expF)
			}
			if !aClose(r.PValue, expP, tolFP) {
				t.Errorf("P: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || *r.DF != expDF1 {
				t.Errorf("DF1: got %v, want %v", r.DF, expDF1)
			}
			if r.DF2 != expDF2 {
				t.Errorf("DF2: got %v, want %v", r.DF2, expDF2)
			}
		})
	}
}

func TestFTestForVarianceEquality_Errors(t *testing.T) {
	d1 := insyra.NewDataList([]float64{1})
	d2 := insyra.NewDataList([]float64{1, 2})
	if _, err := stats.FTestForVarianceEquality(d1, d2); err == nil {
		t.Error("expected error for n1<2")
	}
	if _, err := stats.FTestForVarianceEquality(d2, d1); err == nil {
		t.Error("expected error for n2<2")
	}
	const_ := insyra.NewDataList([]float64{5, 5, 5})
	other := insyra.NewDataList([]float64{1, 2, 3})
	if _, err := stats.FTestForVarianceEquality(const_, other); err == nil {
		t.Error("expected error for zero variance")
	}
}

// ============================================================
// Levene's test (median-centered, also known as Brown-Forsythe)
// ============================================================

type leveneCase struct {
	name   string
	groups [][]float64
	prefix string
}

func TestLeveneTest_R(t *testing.T) {
	cases := []leveneCase{
		{name: "levene_basic", prefix: "levene_basic",
			groups: [][]float64{{10, 12, 9, 11}, {20, 19, 21, 22}, {30, 29, 28, 32}}},
		{name: "levene_2grp", prefix: "levene_2grp",
			groups: [][]float64{{1, 2, 3, 4, 5}, {2, 5, 8, 11, 14}}},
		{name: "levene_4grp", prefix: "levene_4grp",
			groups: [][]float64{
				{10, 11, 9}, {20, 21, 22, 23},
				{30, 32}, {40, 39, 41, 42, 38},
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			groups := isr.DLs{}
			for _, g := range c.groups {
				groups = append(groups, insyra.NewDataList(g))
			}
			r, err := stats.LeveneTest(groups)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expF := anovaRef.get(t, c.prefix+".F")
			expP := anovaRef.get(t, c.prefix+".P")
			expDF1 := anovaRef.get(t, c.prefix+".DF1")
			expDF2 := anovaRef.get(t, c.prefix+".DF2")
			if !aClose(r.Statistic, expF, tolF) {
				t.Errorf("F: got %.17g, want %.17g", r.Statistic, expF)
			}
			if !aClose(r.PValue, expP, tolFP) {
				t.Errorf("P: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || *r.DF != expDF1 {
				t.Errorf("DF1: got %v, want %v", r.DF, expDF1)
			}
			if r.DF2 != expDF2 {
				t.Errorf("DF2: got %v, want %v", r.DF2, expDF2)
			}
		})
	}
}

func TestLeveneTest_Errors(t *testing.T) {
	if _, err := stats.LeveneTest(isr.DLs{insyra.NewDataList([]float64{1, 2, 3})}); err == nil {
		t.Error("expected error for fewer than two groups")
	}
}

// ============================================================
// Bartlett's test
// ============================================================

type bartlettCase struct {
	name   string
	groups [][]float64
	prefix string
}

func TestBartlettTest_R(t *testing.T) {
	cases := []bartlettCase{
		{name: "bartlett_basic", prefix: "bartlett_basic",
			groups: [][]float64{{10, 12, 9, 11}, {20, 19, 21, 22}, {30, 29, 28, 32}}},
		{name: "bartlett_4grp", prefix: "bartlett_4grp",
			groups: [][]float64{
				{1, 2, 3, 4}, {2, 4, 6, 8},
				{5, 6, 7, 8}, {10, 11, 12, 13},
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			groups := isr.DLs{}
			for _, g := range c.groups {
				groups = append(groups, insyra.NewDataList(g))
			}
			r, err := stats.BartlettTest(groups)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expStat := anovaRef.get(t, c.prefix+".Stat")
			expP := anovaRef.get(t, c.prefix+".P")
			expDF := anovaRef.get(t, c.prefix+".DF")
			if !aClose(r.Statistic, expStat, tolBart) {
				t.Errorf("Stat: got %.17g, want %.17g", r.Statistic, expStat)
			}
			if !aClose(r.PValue, expP, tolFP) {
				t.Errorf("P: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || *r.DF != expDF {
				t.Errorf("DF: got %v, want %v", r.DF, expDF)
			}
		})
	}
}

func TestBartlettTest_Errors(t *testing.T) {
	if _, err := stats.BartlettTest(isr.DLs{insyra.NewDataList([]float64{1, 2, 3})}); err == nil {
		t.Error("expected error for fewer than two groups")
	}
	if _, err := stats.BartlettTest(isr.DLs{
		insyra.NewDataList([]float64{1}),
		insyra.NewDataList([]float64{2, 3}),
	}); err == nil {
		t.Error("expected error for n<2 group")
	}
}

// ============================================================
// F-test for regression
// ============================================================

func TestFTestForRegression_R(t *testing.T) {
	cases := []struct {
		name      string
		ssr, sse  float64
		df1, df2  int
		prefix    string
	}{
		{"freg_basic", 500, 200, 3, 16, "freg_basic"},
		{"freg_large", 1000, 50, 4, 95, "freg_large"},
		{"freg_small", 50, 100, 2, 20, "freg_small"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.FTestForRegression(c.ssr, c.sse, c.df1, c.df2)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expF := anovaRef.get(t, c.prefix+".F")
			expP := anovaRef.get(t, c.prefix+".P")
			if !aClose(r.Statistic, expF, tolF) {
				t.Errorf("F: got %.17g, want %.17g", r.Statistic, expF)
			}
			if !aClose(r.PValue, expP, tolFP) {
				t.Errorf("P: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || *r.DF != float64(c.df1) {
				t.Errorf("DF1: got %v, want %d", r.DF, c.df1)
			}
			if r.DF2 != float64(c.df2) {
				t.Errorf("DF2: got %v, want %d", r.DF2, c.df2)
			}
		})
	}
}

func TestFTestForRegression_Errors(t *testing.T) {
	if _, err := stats.FTestForRegression(100, 50, 0, 5); err == nil {
		t.Error("expected error for df1=0")
	}
	if _, err := stats.FTestForRegression(100, 50, 1, 0); err == nil {
		t.Error("expected error for df2=0")
	}
	if _, err := stats.FTestForRegression(-1, 50, 1, 5); err == nil {
		t.Error("expected error for ssr<0")
	}
	if _, err := stats.FTestForRegression(100, 0, 1, 5); err == nil {
		t.Error("expected error for sse=0")
	}
}

// ============================================================
// F-test for nested models
// ============================================================

func TestFTestForNestedModels_R(t *testing.T) {
	cases := []struct {
		name                   string
		rssReduced, rssFull    float64
		dfReduced, dfFull      int
		prefix                 string
	}{
		{"fnest_basic", 300, 200, 18, 16, "fnest_basic"},
		{"fnest_2", 250, 200, 20, 18, "fnest_2"},
		{"fnest_5", 800, 500, 30, 25, "fnest_5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.FTestForNestedModels(c.rssReduced, c.rssFull, c.dfReduced, c.dfFull)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expF := anovaRef.get(t, c.prefix+".F")
			expP := anovaRef.get(t, c.prefix+".P")
			expDF1 := anovaRef.get(t, c.prefix+".DF1")
			expDF2 := anovaRef.get(t, c.prefix+".DF2")
			if !aClose(r.Statistic, expF, tolF) {
				t.Errorf("F: got %.17g, want %.17g", r.Statistic, expF)
			}
			if !aClose(r.PValue, expP, tolFP) {
				t.Errorf("P: got %.17g, want %.17g (Δ=%g)", r.PValue, expP, math.Abs(r.PValue-expP))
			}
			if r.DF == nil || *r.DF != expDF1 {
				t.Errorf("DF1: got %v, want %v", r.DF, expDF1)
			}
			if r.DF2 != expDF2 {
				t.Errorf("DF2: got %v, want %v", r.DF2, expDF2)
			}
		})
	}
}

func TestFTestForNestedModels_Errors(t *testing.T) {
	if _, err := stats.FTestForNestedModels(300, 200, 16, 16); err == nil {
		t.Error("expected error for dfReduced<=dfFull")
	}
	if _, err := stats.FTestForNestedModels(300, 200, 18, 0); err == nil {
		t.Error("expected error for dfFull<=0")
	}
	if _, err := stats.FTestForNestedModels(100, 200, 18, 16); err == nil {
		t.Error("expected error for rssReduced<rssFull")
	}
	if _, err := stats.FTestForNestedModels(300, 0, 18, 16); err == nil {
		t.Error("expected error for rssFull<=0")
	}
}
