package stats_test

import (
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Tolerances for ANOVA results. SS values can lose precision through repeated
// summation; the most demanding cases (rm_basic, ow_largeN) achieve 1e-12 only
// at relative scale. p-values go through gonum's distuv.F which agrees with
// R's pf() to a few ULPs.
const (
	tolANOVA   = 1e-10 // statistic / SS (relative)
	tolANOVAP  = 1e-10 // p-value (relative)
	tolANOVAEta = 1e-12
)

// shared file-backed dump for rnorm-generated arrays
var anovaDump = &labelledFloats{path: "testdata/anova_data_dump.txt"}

func aClose(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 0) && math.IsInf(b, 0) && math.Signbit(a) == math.Signbit(b) {
		return true
	}
	if math.IsNaN(a) != math.IsNaN(b) || math.IsInf(a, 0) != math.IsInf(b, 0) {
		return false
	}
	if b == 0 {
		return math.Abs(a) <= tol
	}
	return math.Abs(a-b) <= tol*math.Max(1, math.Abs(b))
}

// loadLabelledFile is a stand-alone parser for ANOVA-shape testdata files
// — distinct from the labelledFloats helper above (which only supports CSV
// numeric arrays). This one parses `key = value` per line for scalar refs.
type refTable struct {
	once sync.Once
	data map[string]float64
	path string
}

func (r *refTable) get(t *testing.T, key string) float64 {
	t.Helper()
	r.once.Do(func() {
		raw, err := os.ReadFile(r.path)
		if err != nil {
			t.Fatalf("read %s: %v", r.path, err)
		}
		r.data = map[string]float64{}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq < 0 {
				continue
			}
			k := strings.TrimSpace(line[:eq])
			v := strings.TrimSpace(line[eq+1:])
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				switch v {
				case "Inf":
					f = math.Inf(1)
				case "-Inf":
					f = math.Inf(-1)
				case "NaN":
					f = math.NaN()
				default:
					t.Fatalf("parse %s=%q: %v", k, v, err)
				}
			}
			r.data[k] = f
		}
	})
	v, ok := r.data[key]
	if !ok {
		t.Fatalf("ref key %q not found in %s", key, r.path)
	}
	return v
}

var anovaRef = &refTable{path: "testdata/anova_reference.txt"}

func owDLs(values ...[]float64) []insyra.IDataList {
	out := make([]insyra.IDataList, len(values))
	for i, v := range values {
		out[i] = insyra.NewDataList(v)
	}
	return out
}

// ============================================================
// One-way ANOVA
// ============================================================

type oneWayCase struct {
	name   string
	groups [][]float64
	prefix string // ref key prefix in anova_reference.txt
}

func TestOneWayANOVA_R(t *testing.T) {
	cases := []oneWayCase{
		{
			name:   "ow_basic",
			groups: [][]float64{{10, 12, 9, 11}, {20, 19, 21, 22}, {30, 29, 28, 32}},
			prefix: "ow_basic",
		},
		// ---- Diverse cases ----
		{
			name:   "two_groups",
			groups: [][]float64{{1, 2, 3, 4, 5}, {2, 4, 6, 8, 10}},
			prefix: "ow_2grp",
		},
		{
			name: "five_groups",
			groups: [][]float64{
				{1, 2, 3}, {2, 3, 4}, {3, 4, 5}, {4, 5, 6}, {5, 6, 7},
			},
			prefix: "ow_5grp",
		},
		{
			name: "unequal_sizes",
			groups: [][]float64{
				{10, 11, 12},
				{20, 21, 22, 23, 24},
				{30, 31, 32, 33},
			},
			prefix: "ow_unequal",
		},
		{
			name: "large_n_three_groups",
			groups: [][]float64{
				anovaDump.get(t, "ow_largeN_a"),
				anovaDump.get(t, "ow_largeN_b"),
				anovaDump.get(t, "ow_largeN_c"),
			},
			prefix: "ow_largeN",
		},
		{
			name: "huge_magnitude",
			groups: [][]float64{
				{1.0e9, 1.0001e9, 0.9999e9},
				{1.0001e9, 1.0002e9, 1.0e9},
				{0.9998e9, 0.9999e9, 1.0e9},
			},
			prefix: "ow_huge",
		},
		{
			name: "near_zero_f",
			groups: [][]float64{
				{10, 11, 12}, {11, 12, 10}, {12, 10, 11},
			},
			prefix: "ow_zero_f",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.OneWayANOVA(owDLs(c.groups...)...)
			if err != nil {
				t.Fatalf("OneWayANOVA error: %v", err)
			}
			expSSB := anovaRef.get(t, c.prefix+".SSB")
			expSSW := anovaRef.get(t, c.prefix+".SSW")
			expF := anovaRef.get(t, c.prefix+".F")
			expP := anovaRef.get(t, c.prefix+".P")
			expEta := anovaRef.get(t, c.prefix+".eta")
			expDFB := int(anovaRef.get(t, c.prefix+".DFB"))
			expDFW := int(anovaRef.get(t, c.prefix+".DFW"))
			expTotal := anovaRef.get(t, c.prefix+".totalSS")

			if !aClose(r.Factor.SumOfSquares, expSSB, tolANOVA) {
				t.Errorf("SSB: got %.17g, want %.17g", r.Factor.SumOfSquares, expSSB)
			}
			if !aClose(r.Within.SumOfSquares, expSSW, tolANOVA) {
				t.Errorf("SSW: got %.17g, want %.17g", r.Within.SumOfSquares, expSSW)
			}
			if r.Factor.DF != expDFB {
				t.Errorf("DFB: got %d, want %d", r.Factor.DF, expDFB)
			}
			if r.Within.DF != expDFW {
				t.Errorf("DFW: got %d, want %d", r.Within.DF, expDFW)
			}
			if !aClose(r.Factor.F, expF, tolANOVA) {
				t.Errorf("F: got %.17g, want %.17g", r.Factor.F, expF)
			}
			if !aClose(r.Factor.P, expP, tolANOVAP) {
				t.Errorf("P: got %.17g, want %.17g", r.Factor.P, expP)
			}
			if !aClose(r.Factor.EtaSquared, expEta, tolANOVAEta) {
				t.Errorf("eta: got %.17g, want %.17g", r.Factor.EtaSquared, expEta)
			}
			if !aClose(r.TotalSS, expTotal, tolANOVA) {
				t.Errorf("TotalSS: got %.17g, want %.17g", r.TotalSS, expTotal)
			}
			// Within component must report NaN for F/P/Eta (not part of inference).
			if !math.IsNaN(r.Within.F) {
				t.Errorf("Within.F should be NaN, got %v", r.Within.F)
			}
			if !math.IsNaN(r.Within.P) {
				t.Errorf("Within.P should be NaN, got %v", r.Within.P)
			}
			if !math.IsNaN(r.Within.EtaSquared) {
				t.Errorf("Within.EtaSquared should be NaN, got %v", r.Within.EtaSquared)
			}
		})
	}
}

func TestOneWayANOVA_Errors(t *testing.T) {
	if _, err := stats.OneWayANOVA(insyra.NewDataList([]float64{1, 2, 3})); err == nil {
		t.Error("expected error for fewer than two groups")
	}
	if _, err := stats.OneWayANOVA(insyra.NewDataList([]float64{}), insyra.NewDataList([]float64{1, 2})); err == nil {
		t.Error("expected error for empty group")
	}
}

// ============================================================
// Two-way ANOVA (balanced designs only — Type I SS matches insyra)
// ============================================================

type twoWayCase struct {
	name           string
	aLevels, bLevels int
	cells          [][]float64
	prefix         string
}

func TestTwoWayANOVA_R(t *testing.T) {
	cases := []twoWayCase{
		{
			name: "tw_2x2_basic", aLevels: 2, bLevels: 2,
			cells: [][]float64{
				{5, 6, 5}, {7, 8, 9},
				{4, 3, 4}, {10, 11, 9},
			},
			prefix: "tw_2x2",
		},
		{
			name: "tw_2x3", aLevels: 2, bLevels: 3,
			cells: [][]float64{
				{2, 3}, {4, 5}, {6, 7},
				{3, 2}, {5, 6}, {7, 8},
			},
			prefix: "tw_2x3",
		},
		{
			name: "tw_3x3", aLevels: 3, bLevels: 3,
			cells: [][]float64{
				{1, 2}, {3, 4}, {5, 6},
				{2, 3}, {4, 5}, {6, 7},
				{3, 4}, {5, 6}, {7, 8},
			},
			prefix: "tw_3x3",
		},
		{
			name: "tw_strong_interaction", aLevels: 2, bLevels: 2,
			cells: [][]float64{
				{10, 11}, {20, 21},
				{20, 21}, {10, 11},
			},
			prefix: "tw_interact",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.TwoWayANOVA(c.aLevels, c.bLevels, owDLs(c.cells...)...)
			if err != nil {
				t.Fatalf("TwoWayANOVA error: %v", err)
			}
			check := func(label string, got, want float64) {
				if !aClose(got, want, tolANOVA) {
					t.Errorf("%s: got %.17g, want %.17g", label, got, want)
				}
			}
			checkP := func(label string, got, want float64) {
				if !aClose(got, want, tolANOVAP) {
					t.Errorf("%s: got %.17g, want %.17g", label, got, want)
				}
			}
			check("SSA", r.FactorA.SumOfSquares, anovaRef.get(t, c.prefix+".SSA"))
			check("SSB", r.FactorB.SumOfSquares, anovaRef.get(t, c.prefix+".SSB"))
			check("SSAB", r.Interaction.SumOfSquares, anovaRef.get(t, c.prefix+".SSAB"))
			check("SSW", r.Within.SumOfSquares, anovaRef.get(t, c.prefix+".SSW"))
			if r.FactorA.DF != int(anovaRef.get(t, c.prefix+".DFA")) {
				t.Errorf("DFA: got %d, want %v", r.FactorA.DF, anovaRef.get(t, c.prefix+".DFA"))
			}
			if r.FactorB.DF != int(anovaRef.get(t, c.prefix+".DFB")) {
				t.Errorf("DFB: got %d, want %v", r.FactorB.DF, anovaRef.get(t, c.prefix+".DFB"))
			}
			if r.Interaction.DF != int(anovaRef.get(t, c.prefix+".DFAB")) {
				t.Errorf("DFAB: got %d, want %v", r.Interaction.DF, anovaRef.get(t, c.prefix+".DFAB"))
			}
			if r.Within.DF != int(anovaRef.get(t, c.prefix+".DFW")) {
				t.Errorf("DFW: got %d, want %v", r.Within.DF, anovaRef.get(t, c.prefix+".DFW"))
			}
			check("FA", r.FactorA.F, anovaRef.get(t, c.prefix+".FA"))
			check("FB", r.FactorB.F, anovaRef.get(t, c.prefix+".FB"))
			check("FAB", r.Interaction.F, anovaRef.get(t, c.prefix+".FAB"))
			checkP("PA", r.FactorA.P, anovaRef.get(t, c.prefix+".PA"))
			checkP("PB", r.FactorB.P, anovaRef.get(t, c.prefix+".PB"))
			checkP("PAB", r.Interaction.P, anovaRef.get(t, c.prefix+".PAB"))
			check("EA", r.FactorA.EtaSquared, anovaRef.get(t, c.prefix+".EA"))
			check("EB", r.FactorB.EtaSquared, anovaRef.get(t, c.prefix+".EB"))
			check("EAB", r.Interaction.EtaSquared, anovaRef.get(t, c.prefix+".EAB"))
			check("totalSS", r.TotalSS, anovaRef.get(t, c.prefix+".totalSS"))
		})
	}
}

func TestTwoWayANOVA_Errors(t *testing.T) {
	c := insyra.NewDataList([]float64{1, 2})
	if _, err := stats.TwoWayANOVA(1, 2, c, c); err == nil {
		t.Error("expected error for aLevels<2")
	}
	if _, err := stats.TwoWayANOVA(2, 1, c, c); err == nil {
		t.Error("expected error for bLevels<2")
	}
	if _, err := stats.TwoWayANOVA(2, 2, c, c, c); err == nil {
		t.Error("expected error for cell count mismatch")
	}
}

// ============================================================
// Repeated-measures ANOVA
// ============================================================

type rmCase struct {
	name     string
	subjects [][]float64
	prefix   string
}

func TestRepeatedMeasuresANOVA_R(t *testing.T) {
	cases := []rmCase{
		{
			name: "rm_basic",
			subjects: [][]float64{
				{10, 15, 14},
				{12, 14, 13},
				{11, 13, 13},
				{13, 15, 14},
				{12, 13, 15},
			},
			prefix: "rm_basic",
		},
		{
			name: "rm_4cond",
			subjects: [][]float64{
				{5, 7, 6, 8},
				{6, 8, 7, 10},
				{4, 6, 6, 7},
			},
			prefix: "rm_4cond",
		},
		{
			name: "rm_10sub",
			subjects: [][]float64{
				{10, 12, 14}, {11, 13, 15}, {12, 14, 16}, {10, 11, 13},
				{11, 12, 14}, {9, 11, 13}, {10, 13, 15}, {11, 14, 16},
				{12, 13, 15}, {10, 12, 13},
			},
			prefix: "rm_10sub",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.RepeatedMeasuresANOVA(owDLs(c.subjects...)...)
			if err != nil {
				t.Fatalf("RepeatedMeasuresANOVA error: %v", err)
			}
			check := func(label string, got, want, tol float64) {
				if !aClose(got, want, tol) {
					t.Errorf("%s: got %.17g, want %.17g", label, got, want)
				}
			}
			check("SSB", r.Factor.SumOfSquares, anovaRef.get(t, c.prefix+".SSB"), tolANOVA)
			check("SSS", r.Subject.SumOfSquares, anovaRef.get(t, c.prefix+".SSS"), tolANOVA)
			check("SSW", r.Within.SumOfSquares, anovaRef.get(t, c.prefix+".SSW"), tolANOVA)
			if r.Factor.DF != int(anovaRef.get(t, c.prefix+".DFB")) {
				t.Errorf("DFB: got %d, want %v", r.Factor.DF, anovaRef.get(t, c.prefix+".DFB"))
			}
			if r.Subject.DF != int(anovaRef.get(t, c.prefix+".DFS")) {
				t.Errorf("DFS: got %d, want %v", r.Subject.DF, anovaRef.get(t, c.prefix+".DFS"))
			}
			if r.Within.DF != int(anovaRef.get(t, c.prefix+".DFW")) {
				t.Errorf("DFW: got %d, want %v", r.Within.DF, anovaRef.get(t, c.prefix+".DFW"))
			}
			check("F", r.Factor.F, anovaRef.get(t, c.prefix+".F"), tolANOVA)
			check("P", r.Factor.P, anovaRef.get(t, c.prefix+".P"), tolANOVAP)
			check("eta", r.Factor.EtaSquared, anovaRef.get(t, c.prefix+".eta"), tolANOVAEta)
			check("totalSS", r.TotalSS, anovaRef.get(t, c.prefix+".totalSS"), tolANOVA)
		})
	}
}

func TestRepeatedMeasuresANOVA_Errors(t *testing.T) {
	if _, err := stats.RepeatedMeasuresANOVA(insyra.NewDataList([]float64{1, 2})); err == nil {
		t.Error("expected error for fewer than two subjects")
	}
	if _, err := stats.RepeatedMeasuresANOVA(
		insyra.NewDataList([]float64{1, 2}),
		insyra.NewDataList([]float64{1, 2, 3})); err == nil {
		t.Error("expected error for inconsistent condition counts")
	}
	if _, err := stats.RepeatedMeasuresANOVA(
		insyra.NewDataList([]float64{1}),
		insyra.NewDataList([]float64{2})); err == nil {
		t.Error("expected error for fewer than two conditions")
	}
}
