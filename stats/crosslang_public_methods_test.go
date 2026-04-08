package stats_test

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func dataListFromFloat64(xs []float64) *insyra.DataList {
	return insyra.NewDataList(xs)
}

func dataTableFromRows(rows [][]float64) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	nCols := len(rows[0])
	dt := insyra.NewDataTable()
	for c := 0; c < nCols; c++ {
		col := insyra.NewDataList().SetName(fmt.Sprintf("C%d", c+1))
		for r := 0; r < len(rows); r++ {
			col.Append(rows[r][c])
		}
		dt.AppendCols(col)
	}
	return dt
}

func tableToFloatMatrix(dt *insyra.DataTable) [][]float64 {
	r, c := dt.Size()
	out := make([][]float64, r)
	for i := 0; i < r; i++ {
		out[i] = make([]float64, c)
		row := dt.GetRow(i)
		for j := 0; j < c; j++ {
			v, ok := insyra.ToFloat64Safe(row.Get(j))
			if !ok {
				panic(fmt.Sprintf("non-float cell at row=%d col=%d: %v", i, j, row.Get(j)))
			}
			out[i][j] = v
		}
	}
	return out
}

func TestCrossLangSingleSampleTTest(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		mu   float64
		cl   float64
	}{
		{name: "case_a", x: []float64{52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8}, mu: 50, cl: 0.95},
		{name: "case_b", x: []float64{10.2, 12.3, 11.7, 13.1, 9.8, 14.2, 12.0, 11.5}, mu: 12, cl: 0.95},
		{name: "case_c", x: []float64{-5.2, -2.3, -6.1, -3.7, -4.8, -1.9, -2.7}, mu: 0, cl: 0.95},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.SingleSampleTTest(dataListFromFloat64(tc.x), tc.mu, tc.cl)
			if err != nil {
				t.Fatalf("SingleSampleTTest error: %v", err)
			}

			payload := map[string]any{"x": tc.x, "mu": tc.mu, "cl": tc.cl}
			rb := runRBaseline(t, "single_t", payload)
			pb := runPythonBaseline(t, "single_t", payload)

			assertCloseToBoth(t, "t", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-8)
			rCI := baselineFloatSlice(t, rb, "ci")
			pCI := baselineFloatSlice(t, pb, "ci")
			assertCloseToBoth(t, "ci.low", got.CI[0], rCI[0], pCI[0], 1e-7)
			assertCloseToBoth(t, "ci.high", got.CI[1], rCI[1], pCI[1], 1e-7)
			assertCloseToBoth(t, "mean", *got.Mean, baselineFloat(t, rb, "mean"), baselineFloat(t, pb, "mean"), 1e-10)
			assertCloseToBoth(t, "effect", got.EffectSizes[0].Value, baselineFloat(t, rb, "effect"), baselineFloat(t, pb, "effect"), 1e-8)
		})
	}
}

func TestCrossLangTwoSampleTTest(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name     string
		x        []float64
		y        []float64
		equalVar bool
		cl       float64
	}{
		{
			name:     "equal_var_diff",
			x:        []float64{55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7},
			y:        []float64{46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9},
			equalVar: true,
			cl:       0.95,
		},
		{
			name:     "welch_diff",
			x:        []float64{60.0, 59.5, 58.8, 60.2, 59.9, 58.7},
			y:        []float64{55.1, 52.3, 58.4, 53.6, 54.2, 53.1, 51.9},
			equalVar: false,
			cl:       0.95,
		},
		{
			name:     "close_means",
			x:        []float64{10.1, 10.3, 10.0, 9.9, 10.2, 10.4},
			y:        []float64{9.8, 9.7, 9.9, 10.0, 9.6, 9.5},
			equalVar: true,
			cl:       0.95,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.TwoSampleTTest(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y), tc.equalVar, tc.cl)
			if err != nil {
				t.Fatalf("TwoSampleTTest error: %v", err)
			}

			payload := map[string]any{
				"x": tc.x, "y": tc.y, "equal_var": tc.equalVar, "cl": tc.cl,
			}
			rb := runRBaseline(t, "two_t", payload)
			pb := runPythonBaseline(t, "two_t", payload)

			assertCloseToBoth(t, "t", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-7)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-7)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-7)
			rCI := baselineFloatSlice(t, rb, "ci")
			pCI := baselineFloatSlice(t, pb, "ci")
			assertCloseToBoth(t, "ci.low", got.CI[0], rCI[0], pCI[0], 1e-7)
			assertCloseToBoth(t, "ci.high", got.CI[1], rCI[1], pCI[1], 1e-7)
			assertCloseToBoth(t, "mean1", *got.Mean, baselineFloat(t, rb, "mean1"), baselineFloat(t, pb, "mean1"), 1e-10)
			assertCloseToBoth(t, "mean2", *got.Mean2, baselineFloat(t, rb, "mean2"), baselineFloat(t, pb, "mean2"), 1e-10)
			assertCloseToBoth(t, "effect", got.EffectSizes[0].Value, baselineFloat(t, rb, "effect"), baselineFloat(t, pb, "effect"), 1e-7)
		})
	}
}

func TestCrossLangPairedTTest(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		y    []float64
		cl   float64
	}{
		{name: "case_a", x: []float64{88, 92, 85, 91, 87, 90, 86}, y: []float64{84, 89, 82, 88, 84, 87, 83}, cl: 0.95},
		{name: "case_b", x: []float64{10.2, 11.0, 10.8, 11.3, 10.9, 11.1}, y: []float64{10.0, 10.7, 10.5, 11.0, 10.6, 10.8}, cl: 0.95},
		{name: "case_c", x: []float64{5.5, 6.1, 5.8, 6.4, 5.9, 6.2, 6.0}, y: []float64{5.2, 5.9, 5.6, 6.0, 5.7, 5.9, 5.8}, cl: 0.95},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.PairedTTest(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y), tc.cl)
			if err != nil {
				t.Fatalf("PairedTTest error: %v", err)
			}

			payload := map[string]any{"x": tc.x, "y": tc.y, "cl": tc.cl}
			rb := runRBaseline(t, "paired_t", payload)
			pb := runPythonBaseline(t, "paired_t", payload)

			assertCloseToBoth(t, "t", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-8)
			rCI := baselineFloatSlice(t, rb, "ci")
			pCI := baselineFloatSlice(t, pb, "ci")
			assertCloseToBoth(t, "ci.low", got.CI[0], rCI[0], pCI[0], 1e-7)
			assertCloseToBoth(t, "ci.high", got.CI[1], rCI[1], pCI[1], 1e-7)
			assertCloseToBoth(t, "mean_diff", *got.MeanDiff, baselineFloat(t, rb, "mean_diff"), baselineFloat(t, pb, "mean_diff"), 1e-10)
			assertCloseToBoth(t, "effect", got.EffectSizes[0].Value, baselineFloat(t, rb, "effect"), baselineFloat(t, pb, "effect"), 1e-8)
		})
	}
}

func TestCrossLangSingleSampleZTest(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		mu   float64
		sigma float64
		alt  stats.AlternativeHypothesis
		cl   float64
	}{
		{name: "two_sided", x: []float64{52, 55, 49, 51, 53, 50, 54}, mu: 50, sigma: 10, alt: stats.TwoSided, cl: 0.95},
		{name: "greater", x: []float64{105, 108, 102, 110, 107, 106, 109}, mu: 100, sigma: 12, alt: stats.Greater, cl: 0.95},
		{name: "less", x: []float64{18, 19, 17, 16, 20, 18, 17}, mu: 20, sigma: 5, alt: stats.Less, cl: 0.95},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.SingleSampleZTest(dataListFromFloat64(tc.x), tc.mu, tc.sigma, tc.alt, tc.cl)
			if err != nil {
				t.Fatalf("SingleSampleZTest error: %v", err)
			}
			payload := map[string]any{"x": tc.x, "mu": tc.mu, "sigma": tc.sigma, "alt": string(tc.alt), "cl": tc.cl}
			rb := runRBaseline(t, "single_z", payload)
			pb := runPythonBaseline(t, "single_z", payload)

			assertCloseToBoth(t, "z", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			rCI := baselineFloatSlice(t, rb, "ci")
			pCI := baselineFloatSlice(t, pb, "ci")
			assertCloseToBoth(t, "ci.low", got.CI[0], rCI[0], pCI[0], 1e-7)
			assertCloseToBoth(t, "ci.high", got.CI[1], rCI[1], pCI[1], 1e-7)
			assertCloseToBoth(t, "mean", got.Mean, baselineFloat(t, rb, "mean"), baselineFloat(t, pb, "mean"), 1e-10)
			assertCloseToBoth(t, "effect", got.EffectSizes[0].Value, baselineFloat(t, rb, "effect"), baselineFloat(t, pb, "effect"), 1e-8)
		})
	}
}

func TestCrossLangTwoSampleZTest(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		y    []float64
		s1   float64
		s2   float64
		alt  stats.AlternativeHypothesis
		cl   float64
	}{
		{
			name: "two_sided", x: []float64{65, 48, 58, 52, 64, 60, 50, 59},
			y: []float64{41, 44, 43, 56, 61, 54, 40, 52}, s1: 10, s2: 12, alt: stats.TwoSided, cl: 0.95,
		},
		{
			name: "greater", x: []float64{49, 66, 53, 55, 56, 63, 55, 51},
			y: []float64{41, 48, 41, 43, 40, 52, 45, 54}, s1: 10, s2: 12, alt: stats.Greater, cl: 0.95,
		},
		{
			name: "less", x: []float64{45, 46, 44, 43, 47, 45, 44, 46},
			y: []float64{48, 49, 50, 47, 51, 49, 48, 50}, s1: 9, s2: 11, alt: stats.Less, cl: 0.95,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.TwoSampleZTest(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y), tc.s1, tc.s2, tc.alt, tc.cl)
			if err != nil {
				t.Fatalf("TwoSampleZTest error: %v", err)
			}
			payload := map[string]any{
				"x": tc.x, "y": tc.y, "sigma1": tc.s1, "sigma2": tc.s2, "alt": string(tc.alt), "cl": tc.cl,
			}
			rb := runRBaseline(t, "two_z", payload)
			pb := runPythonBaseline(t, "two_z", payload)

			assertCloseToBoth(t, "z", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			rCI := baselineFloatSlice(t, rb, "ci")
			pCI := baselineFloatSlice(t, pb, "ci")
			assertCloseToBoth(t, "ci.low", got.CI[0], rCI[0], pCI[0], 1e-7)
			assertCloseToBoth(t, "ci.high", got.CI[1], rCI[1], pCI[1], 1e-7)
			assertCloseToBoth(t, "mean1", got.Mean, baselineFloat(t, rb, "mean1"), baselineFloat(t, pb, "mean1"), 1e-10)
			assertCloseToBoth(t, "mean2", *got.Mean2, baselineFloat(t, rb, "mean2"), baselineFloat(t, pb, "mean2"), 1e-10)
			assertCloseToBoth(t, "effect", got.EffectSizes[0].Value, baselineFloat(t, rb, "effect"), baselineFloat(t, pb, "effect"), 1e-8)
		})
	}
}

func TestCrossLangChiSquareGoodnessOfFit(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name    string
		values  []string
		p       []float64
		rescale bool
	}{
		{name: "uniform_nil_p", values: []string{"A", "A", "B", "C", "C", "C", "B", "A"}, p: nil, rescale: false},
		{name: "provided_probs", values: []string{"X", "X", "X", "Y", "Y", "Z", "X", "Y", "Z", "Z"}, p: []float64{0.5, 0.3, 0.2}, rescale: false},
		{name: "rescale_probs", values: []string{"R", "R", "S", "T", "T", "T", "S", "R", "S"}, p: []float64{2, 3, 5}, rescale: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dl := insyra.NewDataList(tc.values)
			got, err := stats.ChiSquareGoodnessOfFit(dl, tc.p, tc.rescale)
			if err != nil {
				t.Fatalf("ChiSquareGoodnessOfFit error: %v", err)
			}

			payload := map[string]any{"values": tc.values, "p": tc.p, "rescale": tc.rescale}
			rb := runRBaseline(t, "chi_gof", payload)
			pb := runPythonBaseline(t, "chi_gof", payload)

			assertCloseToBoth(t, "chi", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-8)
		})
	}
}

func TestCrossLangChiSquareIndependence(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		rows []string
		cols []string
	}{
		{name: "case_a", rows: []string{"A", "A", "B", "B", "B", "C", "C", "A"}, cols: []string{"X", "Y", "X", "Y", "Y", "Y", "X", "X"}},
		{name: "case_b", rows: []string{"M", "M", "M", "F", "F", "F", "F", "M", "F", "M"}, cols: []string{"Yes", "No", "Yes", "No", "No", "Yes", "No", "Yes", "No", "No"}},
		{name: "case_c", rows: []string{"G1", "G1", "G2", "G2", "G3", "G3", "G3", "G1", "G2"}, cols: []string{"L", "H", "L", "H", "L", "H", "H", "L", "L"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.ChiSquareIndependenceTest(insyra.NewDataList(tc.rows), insyra.NewDataList(tc.cols))
			if err != nil {
				t.Fatalf("ChiSquareIndependenceTest error: %v", err)
			}
			payload := map[string]any{"rows": tc.rows, "cols": tc.cols}
			rb := runRBaseline(t, "chi_ind", payload)
			pb := runPythonBaseline(t, "chi_ind", payload)

			assertCloseToBoth(t, "chi", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-8)
		})
	}
}

func TestCrossLangOneWayANOVA(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name   string
		groups [][]float64
	}{
		{name: "case_a", groups: [][]float64{{10, 12, 9, 11}, {20, 19, 21, 22}, {30, 29, 28, 32}}},
		{name: "case_b", groups: [][]float64{{5.1, 5.3, 5.2, 5.4}, {6.0, 6.2, 6.1, 6.3}, {6.8, 6.9, 7.0, 6.7}}},
		{name: "case_c", groups: [][]float64{{100, 102, 101, 99, 98}, {95, 96, 94, 93, 97}, {104, 106, 105, 103, 102}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := make([]insyra.IDataList, 0, len(tc.groups))
			for _, g := range tc.groups {
				args = append(args, dataListFromFloat64(g))
			}
			got, err := stats.OneWayANOVA(args...)
			if err != nil {
				t.Fatalf("OneWayANOVA error: %v", err)
			}
			payload := map[string]any{"groups": tc.groups}
			rb := runRBaseline(t, "oneway_anova", payload)
			pb := runPythonBaseline(t, "oneway_anova", payload)

			assertCloseToBoth(t, "ssb", got.Factor.SumOfSquares, baselineFloat(t, rb, "ssb"), baselineFloat(t, pb, "ssb"), 1e-8)
			assertCloseToBoth(t, "ssw", got.Within.SumOfSquares, baselineFloat(t, rb, "ssw"), baselineFloat(t, pb, "ssw"), 1e-8)
			assertCloseToBoth(t, "f", got.Factor.F, baselineFloat(t, rb, "f"), baselineFloat(t, pb, "f"), 1e-8)
			assertCloseToBoth(t, "p", got.Factor.P, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
			assertCloseToBoth(t, "eta", got.Factor.EtaSquared, baselineFloat(t, rb, "eta"), baselineFloat(t, pb, "eta"), 1e-8)
			assertCloseToBoth(t, "total_ss", got.TotalSS, baselineFloat(t, rb, "total_ss"), baselineFloat(t, pb, "total_ss"), 1e-8)
		})
	}
}

func TestCrossLangTwoWayANOVA(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name    string
		aLevels int
		bLevels int
		cells   [][]float64
	}{
		{
			name: "case_a", aLevels: 2, bLevels: 2,
			cells: [][]float64{{5, 6, 5}, {7, 8, 9}, {4, 3, 4}, {10, 11, 9}},
		},
		{
			name: "case_b", aLevels: 2, bLevels: 3,
			cells: [][]float64{{12, 11, 13}, {14, 15, 14}, {13, 12, 14}, {10, 9, 11}, {12, 11, 10}, {11, 10, 12}},
		},
		{
			name: "case_c", aLevels: 3, bLevels: 2,
			cells: [][]float64{{8, 9, 7}, {10, 11, 9}, {7, 8, 6}, {9, 10, 8}, {11, 12, 10}, {13, 14, 12}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := make([]insyra.IDataList, 0, len(tc.cells))
			for _, c := range tc.cells {
				args = append(args, dataListFromFloat64(c))
			}
			got, err := stats.TwoWayANOVA(tc.aLevels, tc.bLevels, args...)
			if err != nil {
				t.Fatalf("TwoWayANOVA error: %v", err)
			}
			payload := map[string]any{"a_levels": tc.aLevels, "b_levels": tc.bLevels, "cells": tc.cells}
			rb := runRBaseline(t, "twoway_anova", payload)
			pb := runPythonBaseline(t, "twoway_anova", payload)

			assertCloseToBoth(t, "ssa", got.FactorA.SumOfSquares, baselineFloat(t, rb, "ssa"), baselineFloat(t, pb, "ssa"), 1e-7)
			assertCloseToBoth(t, "ssb", got.FactorB.SumOfSquares, baselineFloat(t, rb, "ssb"), baselineFloat(t, pb, "ssb"), 1e-7)
			assertCloseToBoth(t, "ssab", got.Interaction.SumOfSquares, baselineFloat(t, rb, "ssab"), baselineFloat(t, pb, "ssab"), 1e-7)
			assertCloseToBoth(t, "ssw", got.Within.SumOfSquares, baselineFloat(t, rb, "ssw"), baselineFloat(t, pb, "ssw"), 1e-7)
			assertCloseToBoth(t, "fa", got.FactorA.F, baselineFloat(t, rb, "fa"), baselineFloat(t, pb, "fa"), 1e-7)
			assertCloseToBoth(t, "fb", got.FactorB.F, baselineFloat(t, rb, "fb"), baselineFloat(t, pb, "fb"), 1e-7)
			assertCloseToBoth(t, "fab", got.Interaction.F, baselineFloat(t, rb, "fab"), baselineFloat(t, pb, "fab"), 1e-7)
		})
	}
}

func TestCrossLangRepeatedMeasuresANOVA(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name     string
		subjects [][]float64
	}{
		{name: "case_a", subjects: [][]float64{{10, 15, 14}, {12, 14, 13}, {11, 13, 13}, {13, 15, 14}, {12, 13, 15}}},
		{name: "case_b", subjects: [][]float64{{20, 22, 24}, {18, 19, 21}, {21, 23, 25}, {19, 20, 22}, {22, 24, 26}}},
		{name: "case_c", subjects: [][]float64{{5, 6, 7}, {4, 5, 6.2}, {6.1, 7, 8}, {5, 6.3, 7}, {7, 8.2, 9}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := make([]insyra.IDataList, 0, len(tc.subjects))
			for _, s := range tc.subjects {
				args = append(args, dataListFromFloat64(s))
			}
			got, err := stats.RepeatedMeasuresANOVA(args...)
			if err != nil {
				t.Fatalf("RepeatedMeasuresANOVA error: %v", err)
			}
			payload := map[string]any{"subjects": tc.subjects}
			rb := runRBaseline(t, "rm_anova", payload)
			pb := runPythonBaseline(t, "rm_anova", payload)

			assertCloseToBoth(t, "ss_factor", got.Factor.SumOfSquares, baselineFloat(t, rb, "ss_factor"), baselineFloat(t, pb, "ss_factor"), 1e-7)
			assertCloseToBoth(t, "ss_subject", got.Subject.SumOfSquares, baselineFloat(t, rb, "ss_subject"), baselineFloat(t, pb, "ss_subject"), 1e-7)
			assertCloseToBoth(t, "ss_within", got.Within.SumOfSquares, baselineFloat(t, rb, "ss_within"), baselineFloat(t, pb, "ss_within"), 1e-7)
			assertCloseToBoth(t, "f", got.Factor.F, baselineFloat(t, rb, "f"), baselineFloat(t, pb, "f"), 1e-7)
			assertCloseToBoth(t, "p", got.Factor.P, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-7)
			assertCloseToBoth(t, "eta", got.Factor.EtaSquared, baselineFloat(t, rb, "eta"), baselineFloat(t, pb, "eta"), 1e-7)
		})
	}
}

func TestCrossLangFTests(t *testing.T) {
	requireCrossLangTools(t)

	t.Run("variance_equality", func(t *testing.T) {
		cases := []struct {
			name string
			x    []float64
			y    []float64
		}{
			{name: "case_a", x: []float64{10, 12, 9, 11}, y: []float64{20, 19, 21, 22}},
			{name: "case_b", x: []float64{5, 6, 7, 8, 9}, y: []float64{3, 3.5, 4, 4.5, 5}},
			{name: "case_c", x: []float64{100, 98, 102, 101, 99}, y: []float64{90, 91, 89, 92, 88}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.FTestForVarianceEquality(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y))
				if err != nil {
					t.Fatalf("FTestForVarianceEquality error: %v", err)
				}
				payload := map[string]any{"x": tc.x, "y": tc.y}
				rb := runRBaseline(t, "f_var", payload)
				pb := runPythonBaseline(t, "f_var", payload)
				assertCloseToBoth(t, "f", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
				assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
				assertCloseToBoth(t, "df1", *got.DF, baselineFloat(t, rb, "df1"), baselineFloat(t, pb, "df1"), 1e-8)
				assertCloseToBoth(t, "df2", got.DF2, baselineFloat(t, rb, "df2"), baselineFloat(t, pb, "df2"), 1e-8)
			})
		}
	})

	t.Run("levene", func(t *testing.T) {
		cases := []struct {
			name   string
			groups [][]float64
		}{
			{name: "case_a", groups: [][]float64{{10, 12, 9, 11}, {20, 19, 21, 22}, {30, 29, 28, 32}}},
			{name: "case_b", groups: [][]float64{{5.1, 5.3, 5.2, 5.4}, {6.0, 6.2, 6.1, 6.3}, {6.8, 6.9, 7.0, 6.7}}},
			{name: "case_c", groups: [][]float64{{100, 102, 101, 99}, {95, 96, 94, 93}, {104, 106, 105, 103}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				args := make([]insyra.IDataList, 0, len(tc.groups))
				for _, g := range tc.groups {
					args = append(args, dataListFromFloat64(g))
				}
				got, err := stats.LeveneTest(args)
				if err != nil {
					t.Fatalf("LeveneTest error: %v", err)
				}
				payload := map[string]any{"groups": tc.groups}
				rb := runRBaseline(t, "levene", payload)
				pb := runPythonBaseline(t, "levene", payload)
				assertCloseToBoth(t, "f", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
				assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
				assertCloseToBoth(t, "df1", *got.DF, baselineFloat(t, rb, "df1"), baselineFloat(t, pb, "df1"), 1e-8)
				assertCloseToBoth(t, "df2", got.DF2, baselineFloat(t, rb, "df2"), baselineFloat(t, pb, "df2"), 1e-8)
			})
		}
	})

	t.Run("bartlett", func(t *testing.T) {
		cases := []struct {
			name   string
			groups [][]float64
		}{
			{name: "case_a", groups: [][]float64{{10, 12, 9, 11}, {20, 19, 21, 22}, {30, 29, 28, 32}}},
			{name: "case_b", groups: [][]float64{{5, 5.5, 6, 6.5}, {4, 4.2, 4.4, 4.6}, {7, 7.3, 7.1, 7.4}}},
			{name: "case_c", groups: [][]float64{{100, 102, 101, 99}, {98, 97, 99, 96}, {103, 104, 102, 105}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				args := make([]insyra.IDataList, 0, len(tc.groups))
				for _, g := range tc.groups {
					args = append(args, dataListFromFloat64(g))
				}
				got, err := stats.BartlettTest(args)
				if err != nil {
					t.Fatalf("BartlettTest error: %v", err)
				}
				payload := map[string]any{"groups": tc.groups}
				rb := runRBaseline(t, "bartlett", payload)
				pb := runPythonBaseline(t, "bartlett", payload)
				assertCloseToBoth(t, "chi", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
				assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
				assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-8)
			})
		}
	})

	t.Run("f_regression", func(t *testing.T) {
		cases := []struct {
			name string
			ssr  float64
			sse  float64
			df1  int
			df2  int
		}{
			{name: "case_a", ssr: 500, sse: 200, df1: 3, df2: 16},
			{name: "case_b", ssr: 180, sse: 220, df1: 2, df2: 25},
			{name: "case_c", ssr: 90, sse: 140, df1: 4, df2: 30},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.FTestForRegression(tc.ssr, tc.sse, tc.df1, tc.df2)
				if err != nil {
					t.Fatalf("FTestForRegression error: %v", err)
				}
				payload := map[string]any{"ssr": tc.ssr, "sse": tc.sse, "df1": tc.df1, "df2": tc.df2}
				rb := runRBaseline(t, "f_reg", payload)
				pb := runPythonBaseline(t, "f_reg", payload)
				assertCloseToBoth(t, "f", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-10)
				assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-10)
				assertCloseToBoth(t, "df1", *got.DF, baselineFloat(t, rb, "df1"), baselineFloat(t, pb, "df1"), 1e-10)
				assertCloseToBoth(t, "df2", got.DF2, baselineFloat(t, rb, "df2"), baselineFloat(t, pb, "df2"), 1e-10)
			})
		}
	})

	t.Run("f_nested", func(t *testing.T) {
		cases := []struct {
			name      string
			rssR      float64
			rssF      float64
			dfReduced int
			dfFull    int
		}{
			{name: "case_a", rssR: 300, rssF: 200, dfReduced: 18, dfFull: 16},
			{name: "case_b", rssR: 220, rssF: 180, dfReduced: 30, dfFull: 27},
			{name: "case_c", rssR: 140, rssF: 120, dfReduced: 25, dfFull: 23},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.FTestForNestedModels(tc.rssR, tc.rssF, tc.dfReduced, tc.dfFull)
				if err != nil {
					t.Fatalf("FTestForNestedModels error: %v", err)
				}
				payload := map[string]any{"rss_reduced": tc.rssR, "rss_full": tc.rssF, "df_reduced": tc.dfReduced, "df_full": tc.dfFull}
				rb := runRBaseline(t, "f_nested", payload)
				pb := runPythonBaseline(t, "f_nested", payload)
				assertCloseToBoth(t, "f", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-10)
				assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-10)
				assertCloseToBoth(t, "df1", *got.DF, baselineFloat(t, rb, "df1"), baselineFloat(t, pb, "df1"), 1e-10)
				assertCloseToBoth(t, "df2", got.DF2, baselineFloat(t, rb, "df2"), baselineFloat(t, pb, "df2"), 1e-10)
			})
		}
	})
}

func TestCrossLangCovarianceCorrelationAndBartlett(t *testing.T) {
	requireCrossLangTools(t)

	t.Run("covariance", func(t *testing.T) {
		cases := []struct {
			name string
			x    []float64
			y    []float64
		}{
			{name: "case_a", x: []float64{1, 2, 3, 4, 5, 6}, y: []float64{2, 4, 6, 8, 10, 12}},
			{name: "case_b", x: []float64{10, 12, 9, 11, 13, 14}, y: []float64{5, 7, 6, 8, 9, 10}},
			{name: "case_c", x: []float64{100, 98, 102, 101, 99, 97}, y: []float64{50, 52, 49, 51, 50, 48}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.Covariance(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y))
				if err != nil {
					t.Fatalf("Covariance error: %v", err)
				}
				payload := map[string]any{"x": tc.x, "y": tc.y}
				rb := runRBaseline(t, "covariance", payload)
				pb := runPythonBaseline(t, "covariance", payload)
				assertCloseToBoth(t, "cov", got, baselineFloat(t, rb, "cov"), baselineFloat(t, pb, "cov"), 1e-10)
			})
		}
	})

	t.Run("correlation", func(t *testing.T) {
		cases := []struct {
			name       string
			x          []float64
			y          []float64
			method     stats.CorrelationMethod
			methodName string
		}{
			{name: "pearson_case", x: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, y: []float64{1.1, 2.2, 2.9, 4.2, 5.1, 6.0, 6.8, 8.1, 9.2}, method: stats.PearsonCorrelation, methodName: "pearson"},
			{name: "spearman_case", x: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90}, y: []float64{11, 25, 24, 44, 49, 62, 68, 79, 92}, method: stats.SpearmanCorrelation, methodName: "spearman"},
			{name: "kendall_case", x: []float64{3, 1, 4, 2, 5, 7, 6, 9, 8}, y: []float64{30, 12, 41, 21, 52, 72, 60, 91, 83}, method: stats.KendallCorrelation, methodName: "kendall"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.Correlation(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y), tc.method)
				if err != nil {
					t.Fatalf("Correlation error: %v", err)
				}
				payload := map[string]any{"x": tc.x, "y": tc.y, "corr_method": tc.methodName}
				rb := runRBaseline(t, "correlation", payload)
				pb := runPythonBaseline(t, "correlation", payload)

				assertCloseToBoth(t, "stat", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-8)
				assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-8)
				if tc.method != stats.KendallCorrelation {
					assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-8)
					rCI := baselineFloatSlice(t, rb, "ci")
					pCI := baselineFloatSlice(t, pb, "ci")
					assertCloseToBoth(t, "ci.low", got.CI[0], rCI[0], pCI[0], 1e-7)
					assertCloseToBoth(t, "ci.high", got.CI[1], rCI[1], pCI[1], 1e-7)
				}
			})
		}
	})

	t.Run("bartlett_sphericity", func(t *testing.T) {
		cases := []struct {
			name string
			rows [][]float64
		}{
			{name: "case_a", rows: [][]float64{{2.5, 3.1, 4.2}, {2.7, 3.0, 4.1}, {2.9, 3.4, 4.4}, {3.2, 3.6, 4.8}, {3.0, 3.3, 4.5}, {2.8, 3.2, 4.3}}},
			{name: "case_b", rows: [][]float64{{10, 20, 30}, {11, 19, 29}, {12, 21, 31}, {13, 22, 33}, {9, 18, 28}, {14, 23, 34}}},
			{name: "case_c", rows: [][]float64{{100, 60, 30}, {98, 59, 31}, {102, 62, 29}, {101, 60, 32}, {99, 58, 30}, {97, 57, 29}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				dt := dataTableFromRows(tc.rows)
				chi, p, df, err := stats.BartlettSphericity(dt)
				if err != nil {
					t.Fatalf("BartlettSphericity error: %v", err)
				}
				payload := map[string]any{"rows": tc.rows}
				rb := runRBaseline(t, "bartlett_sphericity", payload)
				pb := runPythonBaseline(t, "bartlett_sphericity", payload)
				assertCloseToBoth(t, "chi", chi, baselineFloat(t, rb, "chi_square"), baselineFloat(t, pb, "chi_square"), 1e-7)
				assertCloseToBoth(t, "p", p, baselineFloat(t, rb, "p_value"), baselineFloat(t, pb, "p_value"), 1e-7)
				assertCloseToBoth(t, "df", float64(df), baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-7)
			})
		}
	})
}

func TestCrossLangRegressionFamily(t *testing.T) {
	requireCrossLangTools(t)

	t.Run("linear_regression", func(t *testing.T) {
		cases := []struct {
			name string
			y    []float64
			xs   [][]float64
		}{
			{name: "simple_case", y: []float64{3.5, 7.2, 9.8, 15.1, 17.9, 20.6, 24.2, 27.0}, xs: [][]float64{{1.2, 2.5, 3.1, 4.8, 5.3, 6.1, 7.2, 8.0}}},
			{name: "multi_case_2", y: []float64{12.1, 15.3, 18.7, 20.1, 23.5, 25.0, 27.9, 30.2}, xs: [][]float64{{1.0, 1.5, 2.1, 2.4, 3.0, 3.4, 3.8, 4.2}, {0.8, 1.2, 1.5, 1.9, 2.2, 2.6, 2.9, 3.1}}},
			{name: "multi_case_3", y: []float64{5.2, 6.1, 7.4, 8.0, 9.3, 10.1, 11.5, 12.0, 13.4, 14.1}, xs: [][]float64{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, {2, 1, 3, 2, 4, 3, 5, 4, 6, 5}, {0.5, 0.8, 1.0, 1.2, 1.5, 1.7, 2.0, 2.1, 2.4, 2.6}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				dlY := dataListFromFloat64(tc.y)
				xLists := make([]insyra.IDataList, 0, len(tc.xs))
				for _, x := range tc.xs {
					xLists = append(xLists, dataListFromFloat64(x))
				}
				got, err := stats.LinearRegression(dlY, xLists...)
				if err != nil {
					t.Fatalf("LinearRegression error: %v", err)
				}
				payload := map[string]any{"y": tc.y, "xs": tc.xs}
				rb := runRBaseline(t, "linear_reg", payload)
				pb := runPythonBaseline(t, "linear_reg", payload)

				assertSliceCloseToBoth(t, "coefficients", got.Coefficients, baselineFloatSlice(t, rb, "coefficients"), baselineFloatSlice(t, pb, "coefficients"), 1e-6)
				assertSliceCloseToBoth(t, "standard_errors", got.StandardErrors, baselineFloatSlice(t, rb, "standard_errors"), baselineFloatSlice(t, pb, "standard_errors"), 1e-6)
				assertSliceCloseToBoth(t, "t_values", got.TValues, baselineFloatSlice(t, rb, "t_values"), baselineFloatSlice(t, pb, "t_values"), 1e-5)
				assertSliceCloseToBoth(t, "p_values", got.PValues, baselineFloatSlice(t, rb, "p_values"), baselineFloatSlice(t, pb, "p_values"), 1e-6)
				assertSliceCloseToBoth(t, "residuals", got.Residuals, baselineFloatSlice(t, rb, "residuals"), baselineFloatSlice(t, pb, "residuals"), 1e-6)
				assertCloseToBoth(t, "r_squared", got.RSquared, baselineFloat(t, rb, "r_squared"), baselineFloat(t, pb, "r_squared"), 1e-6)
				assertCloseToBoth(t, "adj_r_squared", got.AdjustedRSquared, baselineFloat(t, rb, "adj_r_squared"), baselineFloat(t, pb, "adj_r_squared"), 1e-6)

				rCI := baselineFloatMatrix(t, rb, "confidence_intervals")
				pCI := baselineFloatMatrix(t, pb, "confidence_intervals")
				gotCI := make([][]float64, len(got.ConfidenceIntervals))
				for i := range got.ConfidenceIntervals {
					gotCI[i] = []float64{got.ConfidenceIntervals[i][0], got.ConfidenceIntervals[i][1]}
				}
				for i := range gotCI {
					assertCloseToBoth(t, fmt.Sprintf("ci[%d].low", i), gotCI[i][0], rCI[i][0], pCI[i][0], 1e-6)
					assertCloseToBoth(t, fmt.Sprintf("ci[%d].high", i), gotCI[i][1], rCI[i][1], pCI[i][1], 1e-6)
				}

				if len(tc.xs) == 1 {
					assertCloseToBoth(t, "simple.intercept", got.Intercept, baselineFloat(t, rb, "intercept"), baselineFloat(t, pb, "intercept"), 1e-6)
					assertCloseToBoth(t, "simple.slope", got.Slope, baselineFloat(t, rb, "slope"), baselineFloat(t, pb, "slope"), 1e-6)
					assertCloseToBoth(t, "simple.se_intercept", got.StandardErrorIntercept, baselineFloat(t, rb, "se_intercept"), baselineFloat(t, pb, "se_intercept"), 1e-6)
					assertCloseToBoth(t, "simple.se_slope", got.StandardError, baselineFloat(t, rb, "se_slope"), baselineFloat(t, pb, "se_slope"), 1e-6)
					assertCloseToBoth(t, "simple.t_intercept", got.TValueIntercept, baselineFloat(t, rb, "t_intercept"), baselineFloat(t, pb, "t_intercept"), 1e-6)
					assertCloseToBoth(t, "simple.t_slope", got.TValue, baselineFloat(t, rb, "t_slope"), baselineFloat(t, pb, "t_slope"), 1e-6)
					assertCloseToBoth(t, "simple.p_intercept", got.PValueIntercept, baselineFloat(t, rb, "p_intercept"), baselineFloat(t, pb, "p_intercept"), 1e-6)
					assertCloseToBoth(t, "simple.p_slope", got.PValue, baselineFloat(t, rb, "p_slope"), baselineFloat(t, pb, "p_slope"), 1e-6)
				}
			})
		}
	})

	t.Run("polynomial_regression", func(t *testing.T) {
		cases := []struct {
			name   string
			x      []float64
			y      []float64
			degree int
		}{
			{name: "quad_case_1", x: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, y: []float64{2.1, 8.9, 20.1, 35.8, 56.2, 81.1, 110.9, 145.2, 184.1, 227.8}, degree: 2},
			{name: "cubic_case", x: []float64{1, 2, 3, 4, 5, 6}, y: []float64{5.4, 17.6, 47.8, 97.2, 178.1, 289.4}, degree: 3},
			{name: "quad_case_2", x: []float64{2, 3, 4, 5, 6, 7, 8, 9}, y: []float64{4.2, 8.8, 16.5, 24.7, 36.1, 48.5, 64.3, 80.6}, degree: 2},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.PolynomialRegression(dataListFromFloat64(tc.y), dataListFromFloat64(tc.x), tc.degree)
				if err != nil {
					t.Fatalf("PolynomialRegression error: %v", err)
				}
				payload := map[string]any{"y": tc.y, "x": tc.x, "degree": tc.degree}
				rb := runRBaseline(t, "poly_reg", payload)
				pb := runPythonBaseline(t, "poly_reg", payload)

				assertSliceCloseToBoth(t, "coefficients", got.Coefficients, baselineFloatSlice(t, rb, "coefficients"), baselineFloatSlice(t, pb, "coefficients"), 1e-6)
				assertSliceCloseToBoth(t, "standard_errors", got.StandardErrors, baselineFloatSlice(t, rb, "standard_errors"), baselineFloatSlice(t, pb, "standard_errors"), 1e-6)
				assertSliceCloseToBoth(t, "t_values", got.TValues, baselineFloatSlice(t, rb, "t_values"), baselineFloatSlice(t, pb, "t_values"), 1e-5)
				assertSliceCloseToBoth(t, "p_values", got.PValues, baselineFloatSlice(t, rb, "p_values"), baselineFloatSlice(t, pb, "p_values"), 1e-6)
				assertSliceCloseToBoth(t, "residuals", got.Residuals, baselineFloatSlice(t, rb, "residuals"), baselineFloatSlice(t, pb, "residuals"), 1e-6)
				assertCloseToBoth(t, "r_squared", got.RSquared, baselineFloat(t, rb, "r_squared"), baselineFloat(t, pb, "r_squared"), 1e-6)
				assertCloseToBoth(t, "adj_r_squared", got.AdjustedRSquared, baselineFloat(t, rb, "adj_r_squared"), baselineFloat(t, pb, "adj_r_squared"), 1e-6)
			})
		}
	})

	t.Run("exponential_regression", func(t *testing.T) {
		cases := []struct {
			name string
			x    []float64
			y    []float64
		}{
			{name: "case_a", x: []float64{1, 2, 3, 4, 5, 6}, y: []float64{2.7, 7.4, 20.1, 54.6, 148.3, 403.2}},
			{name: "case_b", x: []float64{1, 2, 3, 4, 5, 6, 7}, y: []float64{3.1, 4.8, 7.4, 11.2, 16.9, 25.3, 37.7}},
			{name: "case_c", x: []float64{1, 2, 3, 4, 5, 6, 7, 8}, y: []float64{1.8, 2.9, 4.6, 7.4, 11.8, 18.9, 30.1, 48.2}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.ExponentialRegression(dataListFromFloat64(tc.y), dataListFromFloat64(tc.x))
				if err != nil {
					t.Fatalf("ExponentialRegression error: %v", err)
				}
				payload := map[string]any{"y": tc.y, "x": tc.x}
				rb := runRBaseline(t, "exp_reg", payload)
				pb := runPythonBaseline(t, "exp_reg", payload)
				assertCloseToBoth(t, "intercept", got.Intercept, baselineFloat(t, rb, "intercept"), baselineFloat(t, pb, "intercept"), 1e-6)
				assertCloseToBoth(t, "slope", got.Slope, baselineFloat(t, rb, "slope"), baselineFloat(t, pb, "slope"), 1e-6)
				assertCloseToBoth(t, "r_squared", got.RSquared, baselineFloat(t, rb, "r_squared"), baselineFloat(t, pb, "r_squared"), 1e-6)
				assertCloseToBoth(t, "adj_r_squared", got.AdjustedRSquared, baselineFloat(t, rb, "adj_r_squared"), baselineFloat(t, pb, "adj_r_squared"), 1e-6)
				assertCloseToBoth(t, "se_intercept", got.StandardErrorIntercept, baselineFloat(t, rb, "se_intercept"), baselineFloat(t, pb, "se_intercept"), 1e-6)
				assertCloseToBoth(t, "se_slope", got.StandardErrorSlope, baselineFloat(t, rb, "se_slope"), baselineFloat(t, pb, "se_slope"), 1e-6)
				assertCloseToBoth(t, "t_intercept", got.TValueIntercept, baselineFloat(t, rb, "t_intercept"), baselineFloat(t, pb, "t_intercept"), 1e-5)
				assertCloseToBoth(t, "t_slope", got.TValueSlope, baselineFloat(t, rb, "t_slope"), baselineFloat(t, pb, "t_slope"), 1e-5)
				assertCloseToBoth(t, "p_intercept", got.PValueIntercept, baselineFloat(t, rb, "p_intercept"), baselineFloat(t, pb, "p_intercept"), 1e-6)
				assertCloseToBoth(t, "p_slope", got.PValueSlope, baselineFloat(t, rb, "p_slope"), baselineFloat(t, pb, "p_slope"), 1e-6)
				assertSliceCloseToBoth(t, "residuals", got.Residuals, baselineFloatSlice(t, rb, "residuals"), baselineFloatSlice(t, pb, "residuals"), 1e-6)
			})
		}
	})

	t.Run("logarithmic_regression", func(t *testing.T) {
		cases := []struct {
			name string
			x    []float64
			y    []float64
		}{
			{name: "case_a", x: []float64{1, 2, 3, 4, 5, 6, 7, 8}, y: []float64{0.0, 0.69, 1.10, 1.39, 1.61, 1.79, 1.95, 2.08}},
			{name: "case_b", x: []float64{1, 2, 3, 4, 5, 6, 7}, y: []float64{2.0, 2.7, 3.1, 3.4, 3.6, 3.8, 4.0}},
			{name: "case_c", x: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, y: []float64{5.0, 5.8, 6.2, 6.5, 6.7, 6.9, 7.1, 7.2, 7.4}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.LogarithmicRegression(dataListFromFloat64(tc.y), dataListFromFloat64(tc.x))
				if err != nil {
					t.Fatalf("LogarithmicRegression error: %v", err)
				}
				payload := map[string]any{"y": tc.y, "x": tc.x}
				rb := runRBaseline(t, "log_reg", payload)
				pb := runPythonBaseline(t, "log_reg", payload)
				assertCloseToBoth(t, "intercept", got.Intercept, baselineFloat(t, rb, "intercept"), baselineFloat(t, pb, "intercept"), 1e-6)
				assertCloseToBoth(t, "slope", got.Slope, baselineFloat(t, rb, "slope"), baselineFloat(t, pb, "slope"), 1e-6)
				assertCloseToBoth(t, "r_squared", got.RSquared, baselineFloat(t, rb, "r_squared"), baselineFloat(t, pb, "r_squared"), 1e-6)
				assertCloseToBoth(t, "adj_r_squared", got.AdjustedRSquared, baselineFloat(t, rb, "adj_r_squared"), baselineFloat(t, pb, "adj_r_squared"), 1e-6)
				assertCloseToBoth(t, "se_intercept", got.StandardErrorIntercept, baselineFloat(t, rb, "se_intercept"), baselineFloat(t, pb, "se_intercept"), 1e-6)
				assertCloseToBoth(t, "se_slope", got.StandardErrorSlope, baselineFloat(t, rb, "se_slope"), baselineFloat(t, pb, "se_slope"), 1e-6)
				assertCloseToBoth(t, "t_intercept", got.TValueIntercept, baselineFloat(t, rb, "t_intercept"), baselineFloat(t, pb, "t_intercept"), 1e-5)
				assertCloseToBoth(t, "t_slope", got.TValueSlope, baselineFloat(t, rb, "t_slope"), baselineFloat(t, pb, "t_slope"), 1e-5)
				assertCloseToBoth(t, "p_intercept", got.PValueIntercept, baselineFloat(t, rb, "p_intercept"), baselineFloat(t, pb, "p_intercept"), 1e-6)
				assertCloseToBoth(t, "p_slope", got.PValueSlope, baselineFloat(t, rb, "p_slope"), baselineFloat(t, pb, "p_slope"), 1e-6)
				assertSliceCloseToBoth(t, "residuals", got.Residuals, baselineFloatSlice(t, rb, "residuals"), baselineFloatSlice(t, pb, "residuals"), 1e-6)
			})
		}
	})

	t.Run("pca", func(t *testing.T) {
		cases := []struct {
			name        string
			rows        [][]float64
			nComponents int
		}{
			{name: "case_a", rows: [][]float64{{2.5, 2.4, 1.2}, {0.5, 0.7, 0.3}, {2.2, 2.9, 1.1}, {1.9, 2.2, 0.9}, {3.1, 3.0, 1.5}, {2.3, 2.7, 1.3}}, nComponents: 2},
			{name: "case_b", rows: [][]float64{{10, 20, 30}, {11, 19, 29}, {12, 21, 31}, {13, 22, 33}, {9, 18, 28}, {14, 23, 34}}, nComponents: 2},
			{name: "case_c", rows: [][]float64{{100, 60, 30}, {98, 58, 29}, {102, 62, 31}, {101, 61, 32}, {99, 59, 30}, {97, 57, 28}}, nComponents: 3},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				dt := dataTableFromRows(tc.rows)
				got, err := stats.PCA(dt, tc.nComponents)
				if err != nil {
					t.Fatalf("PCA error: %v", err)
				}
				payload := map[string]any{"rows": tc.rows, "n_components": tc.nComponents}
				rb := runRBaseline(t, "pca", payload)
				pb := runPythonBaseline(t, "pca", payload)

				assertSliceCloseToBoth(t, "eigenvalues", got.Eigenvalues, baselineFloatSlice(t, rb, "eigenvalues"), baselineFloatSlice(t, pb, "eigenvalues"), 1e-6)
				assertSliceCloseToBoth(t, "explained", got.ExplainedVariance, baselineFloatSlice(t, rb, "explained"), baselineFloatSlice(t, pb, "explained"), 1e-5)

				rComp := baselineFloatMatrix(t, rb, "components")
				pComp := baselineFloatMatrix(t, pb, "components")
				for pc := 0; pc < tc.nComponents; pc++ {
					col := got.Components.GetColByNumber(pc).ToF64Slice()
					assertSliceCloseToBoth(t, fmt.Sprintf("pc%d", pc+1), col, rComp[pc], pComp[pc], 1e-5)
				}
			})
		}
	})
}

func TestCrossLangCorrelationMatrixAndAnalysis(t *testing.T) {
	requireCrossLangTools(t)

	matrixCases := []struct {
		name       string
		rows       [][]float64
		method     stats.CorrelationMethod
		methodName string
	}{
		{
			name: "pearson_case",
			rows: [][]float64{
				{2.5, 3.1, 4.2},
				{2.7, 3.0, 4.1},
				{2.9, 3.4, 4.4},
				{3.2, 3.6, 4.8},
				{3.0, 3.3, 4.5},
				{2.8, 3.2, 4.3},
			},
			method: stats.PearsonCorrelation, methodName: "pearson",
		},
		{
			name: "spearman_case",
			rows: [][]float64{
				{10, 15, 21},
				{12, 17, 23},
				{11, 16, 23},
				{14, 19, 28},
				{13, 18, 26},
				{15, 20, 30},
			},
			method: stats.SpearmanCorrelation, methodName: "spearman",
		},
		{
			name: "kendall_case",
			rows: [][]float64{
				{3, 8, 20},
				{1, 5, 12},
				{4, 9, 24},
				{2, 6, 15},
				{5, 10, 28},
				{7, 12, 35},
				{6, 11, 31},
				{8, 13, 38},
			},
			method: stats.KendallCorrelation, methodName: "kendall",
		},
	}

	t.Run("correlation_matrix", func(t *testing.T) {
		for _, tc := range matrixCases {
			t.Run(tc.name, func(t *testing.T) {
				dt := dataTableFromRows(tc.rows)
				gotCorr, gotP, err := stats.CorrelationMatrix(dt, tc.method)
				if err != nil {
					t.Fatalf("CorrelationMatrix error: %v", err)
				}
				payload := map[string]any{"rows": tc.rows, "corr_method": tc.methodName}
				rb := runRBaseline(t, "corr_matrix", payload)
				pb := runPythonBaseline(t, "corr_matrix", payload)

				gotCorrM := tableToFloatMatrix(gotCorr)
				gotPM := tableToFloatMatrix(gotP)
				rCorr := baselineFloatMatrix(t, rb, "corr_matrix")
				pCorr := baselineFloatMatrix(t, pb, "corr_matrix")
				rPM := baselineFloatMatrix(t, rb, "p_matrix")
				pPM := baselineFloatMatrix(t, pb, "p_matrix")

				for i := range gotCorrM {
					for j := range gotCorrM[i] {
						assertCloseToBoth(t, fmt.Sprintf("corr[%d,%d]", i, j), gotCorrM[i][j], rCorr[i][j], pCorr[i][j], 1e-6)
						assertCloseToBoth(t, fmt.Sprintf("p[%d,%d]", i, j), gotPM[i][j], rPM[i][j], pPM[i][j], 1e-6)
					}
				}
			})
		}
	})

	t.Run("correlation_analysis", func(t *testing.T) {
		analysisCases := []struct {
			name string
			rows [][]float64
		}{
			{
				name: "analysis_a",
				rows: [][]float64{
					{2.5, 3.1, 4.2},
					{2.7, 3.0, 4.1},
					{2.9, 3.4, 4.4},
					{3.2, 3.6, 4.8},
					{3.0, 3.3, 4.5},
					{2.8, 3.2, 4.3},
				},
			},
			{
				name: "analysis_b",
				rows: [][]float64{
					{10, 20, 30},
					{11, 19, 29},
					{12, 21, 31},
					{13, 22, 33},
					{9, 18, 28},
					{14, 23, 34},
				},
			},
			{
				name: "analysis_c",
				rows: [][]float64{
					{100, 60, 30},
					{98, 59, 31},
					{102, 62, 29},
					{101, 60, 32},
					{99, 58, 30},
					{97, 57, 29},
				},
			},
		}
		for _, tc := range analysisCases {
			t.Run(tc.name, func(t *testing.T) {
				dt := dataTableFromRows(tc.rows)
				gotCorr, gotP, chi, p, df, err := stats.CorrelationAnalysis(dt, stats.PearsonCorrelation)
				if err != nil {
					t.Fatalf("CorrelationAnalysis error: %v", err)
				}
				payload := map[string]any{"rows": tc.rows, "corr_method": "pearson"}
				rb := runRBaseline(t, "corr_analysis", payload)
				pb := runPythonBaseline(t, "corr_analysis", payload)

				gotCorrM := tableToFloatMatrix(gotCorr)
				gotPM := tableToFloatMatrix(gotP)
				rCorr := baselineFloatMatrix(t, rb, "corr_matrix")
				pCorr := baselineFloatMatrix(t, pb, "corr_matrix")
				rPM := baselineFloatMatrix(t, rb, "p_matrix")
				pPM := baselineFloatMatrix(t, pb, "p_matrix")

				for i := range gotCorrM {
					for j := range gotCorrM[i] {
						assertCloseToBoth(t, fmt.Sprintf("analysis.corr[%d,%d]", i, j), gotCorrM[i][j], rCorr[i][j], pCorr[i][j], 1e-6)
						assertCloseToBoth(t, fmt.Sprintf("analysis.p[%d,%d]", i, j), gotPM[i][j], rPM[i][j], pPM[i][j], 1e-6)
					}
				}

				assertCloseToBoth(t, "analysis.chi", chi, baselineFloat(t, rb, "chi_square"), baselineFloat(t, pb, "chi_square"), 1e-6)
				assertCloseToBoth(t, "analysis.p", p, baselineFloat(t, rb, "p_value"), baselineFloat(t, pb, "p_value"), 1e-6)
				assertCloseToBoth(t, "analysis.df", float64(df), baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-6)
			})
		}
	})
}

func TestCrossLangMomentsSkewnessKurtosis(t *testing.T) {
	requireCrossLangTools(t)

	t.Run("calculate_moment", func(t *testing.T) {
		cases := []struct {
			name    string
			x       []float64
			order   int
			central bool
		}{
			{name: "raw_m3", x: []float64{1, 2, 3, 4, 5, 6}, order: 3, central: false},
			{name: "central_m2", x: []float64{10, 12, 9, 11, 13, 14}, order: 2, central: true},
			{name: "central_m4", x: []float64{2.1, 3.4, 2.9, 4.2, 3.8, 2.7, 3.3}, order: 4, central: true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.CalculateMoment(dataListFromFloat64(tc.x), tc.order, tc.central)
				if err != nil {
					t.Fatalf("CalculateMoment error: %v", err)
				}
				payload := map[string]any{"x": tc.x, "order": tc.order, "central": tc.central}
				rb := runRBaseline(t, "moment", payload)
				pb := runPythonBaseline(t, "moment", payload)
				assertCloseToBoth(t, "moment", got, baselineFloat(t, rb, "value"), baselineFloat(t, pb, "value"), 1e-10)
			})
		}
	})

	t.Run("skewness", func(t *testing.T) {
		cases := []struct {
			name   string
			x      []float64
			method stats.SkewnessMethod
			mode   string
		}{
			{name: "g1_case", x: []float64{2, 3, 4, 5, 7, 9, 12}, method: stats.SkewnessG1, mode: "g1"},
			{name: "adjusted_case", x: []float64{10, 11, 13, 14, 18, 21, 24, 30}, method: stats.SkewnessAdjusted, mode: "adjusted"},
			{name: "bias_adjusted_case", x: []float64{1.2, 1.5, 1.9, 2.1, 2.8, 3.0, 3.7, 4.1}, method: stats.SkewnessBiasAdjusted, mode: "bias_adjusted"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.Skewness(tc.x, tc.method)
				if err != nil {
					t.Fatalf("Skewness error: %v", err)
				}
				payload := map[string]any{"x": tc.x, "mode": tc.mode}
				rb := runRBaseline(t, "skewness", payload)
				pb := runPythonBaseline(t, "skewness", payload)
				assertCloseToBoth(t, "skewness", got, baselineFloat(t, rb, "value"), baselineFloat(t, pb, "value"), 1e-10)
			})
		}
	})

	t.Run("kurtosis", func(t *testing.T) {
		cases := []struct {
			name   string
			x      []float64
			method stats.KurtosisMethod
			mode   string
		}{
			{name: "g2_case", x: []float64{2, 3, 4, 5, 7, 9, 12}, method: stats.KurtosisG2, mode: "g2"},
			{name: "adjusted_case", x: []float64{10, 11, 13, 14, 18, 21, 24, 30}, method: stats.KurtosisAdjusted, mode: "adjusted"},
			{name: "bias_adjusted_case", x: []float64{1.2, 1.5, 1.9, 2.1, 2.8, 3.0, 3.7, 4.1}, method: stats.KurtosisBiasAdjusted, mode: "bias_adjusted"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.Kurtosis(tc.x, tc.method)
				if err != nil {
					t.Fatalf("Kurtosis error: %v", err)
				}
				payload := map[string]any{"x": tc.x, "mode": tc.mode}
				rb := runRBaseline(t, "kurtosis", payload)
				pb := runPythonBaseline(t, "kurtosis", payload)
				assertCloseToBoth(t, "kurtosis", got, baselineFloat(t, rb, "value"), baselineFloat(t, pb, "value"), 1e-10)
			})
		}
	})
}
