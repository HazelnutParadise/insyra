package stats_test

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

var longRef = &refTable{path: "testdata/long_format_reference.txt"}

func buildLongTable(t *testing.T, prefix string, level func(string) (any, error)) *insyra.DataTable {
	t.Helper()
	longRef.load(t)

	valueStr := longRef.getString(t, prefix+".value")
	valueParts := strings.Split(valueStr, ",")
	values := make([]any, len(valueParts))
	for i, s := range valueParts {
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			t.Fatalf("parse value %d: %v", i, err)
		}
		values[i] = v
	}

	aStr := longRef.getString(t, prefix+".A")
	aParts := strings.Split(aStr, ",")
	aCells := make([]any, len(aParts))
	var err error
	for i, s := range aParts {
		aCells[i], err = level(strings.TrimSpace(s))
		if err != nil {
			t.Fatalf("parse A %d: %v", i, err)
		}
	}

	bStr := longRef.getString(t, prefix+".B")
	bParts := strings.Split(bStr, ",")
	bCells := make([]any, len(bParts))
	for i, s := range bParts {
		bCells[i], err = level(strings.TrimSpace(s))
		if err != nil {
			t.Fatalf("parse B %d: %v", i, err)
		}
	}

	return insyra.NewDataTable(
		insyra.NewDataList(values...).SetName("value"),
		insyra.NewDataList(aCells...).SetName("A"),
		insyra.NewDataList(bCells...).SetName("B"),
	)
}

func TestTwoWayANOVAFromTableMatchesR(t *testing.T) {
	for _, prefix := range []string{"tw", "tw_int"} {
		t.Run(prefix, func(t *testing.T) {
			var level func(string) (any, error)
			if prefix == "tw_int" {
				level = func(s string) (any, error) {
					return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
				}
			} else {
				level = func(s string) (any, error) {
					return strings.TrimSpace(s), nil
				}
			}
			dt := buildLongTable(t, prefix, level)

			r, err := stats.TwoWayANOVAFromTable(dt, insyra.Name("value"), insyra.Name("A"), insyra.Name("B"))
			if err != nil {
				t.Fatalf("TwoWayANOVAFromTable error: %v", err)
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

			check("SSA", r.FactorA.SumOfSquares, longRef.get(t, prefix+".SSA"))
			check("SSB", r.FactorB.SumOfSquares, longRef.get(t, prefix+".SSB"))
			check("SSAB", r.Interaction.SumOfSquares, longRef.get(t, prefix+".SSAB"))
			check("SSW", r.Within.SumOfSquares, longRef.get(t, prefix+".SSW"))

			if r.FactorA.DF != int(longRef.get(t, prefix+".DFA")) {
				t.Errorf("DFA: got %d, want %v", r.FactorA.DF, longRef.get(t, prefix+".DFA"))
			}
			if r.FactorB.DF != int(longRef.get(t, prefix+".DFB")) {
				t.Errorf("DFB: got %d, want %v", r.FactorB.DF, longRef.get(t, prefix+".DFB"))
			}
			if r.Interaction.DF != int(longRef.get(t, prefix+".DFAB")) {
				t.Errorf("DFAB: got %d, want %v", r.Interaction.DF, longRef.get(t, prefix+".DFAB"))
			}
			if r.Within.DF != int(longRef.get(t, prefix+".DFW")) {
				t.Errorf("DFW: got %d, want %v", r.Within.DF, longRef.get(t, prefix+".DFW"))
			}

			check("FA", r.FactorA.F, longRef.get(t, prefix+".FA"))
			check("FB", r.FactorB.F, longRef.get(t, prefix+".FB"))
			check("FAB", r.Interaction.F, longRef.get(t, prefix+".FAB"))

			checkP("PA", r.FactorA.P, longRef.get(t, prefix+".PA"))
			checkP("PB", r.FactorB.P, longRef.get(t, prefix+".PB"))
			checkP("PAB", r.Interaction.P, longRef.get(t, prefix+".PAB"))
		})
	}
}

// sameResult reports the first field where got and want differ, walking
// both structs with reflection: floats must be == (two NaNs count as
// equal), every other comparable value must be ==, slices are compared
// element by element, pointers by what they point to. It returns "" when
// they are identical.
func sameResult(got, want any) string {
	return sameResultPath(reflect.ValueOf(got), reflect.ValueOf(want), "")
}

func sameResultPath(got, want reflect.Value, path string) string {
	if !got.IsValid() && !want.IsValid() {
		return ""
	}
	if !got.IsValid() || !want.IsValid() {
		return fmt.Sprintf("%s: %v != %v", path, got, want)
	}
	if got.Type() != want.Type() {
		return fmt.Sprintf("%s: type %v != %v", path, got.Type(), want.Type())
	}

	switch got.Kind() {
	case reflect.Pointer, reflect.Interface:
		if got.IsNil() && want.IsNil() {
			return ""
		}
		if got.IsNil() || want.IsNil() {
			return fmt.Sprintf("%s: %v != %v", path, got, want)
		}
		nextPath := path
		if nextPath != "" {
			nextPath += "."
		}
		return sameResultPath(got.Elem(), want.Elem(), nextPath)

	case reflect.Struct:
		for i := 0; i < got.NumField(); i++ {
			fieldName := got.Type().Field(i).Name
			nextPath := path
			if nextPath != "" {
				nextPath += "."
			}
			nextPath += fieldName
			if diff := sameResultPath(got.Field(i), want.Field(i), nextPath); diff != "" {
				return diff
			}
		}
		return ""

	case reflect.Slice, reflect.Array:
		if got.IsNil() && want.IsNil() {
			return ""
		}
		if got.IsNil() || want.IsNil() {
			return fmt.Sprintf("%s: %v != %v", path, got, want)
		}
		if got.Len() != want.Len() {
			return fmt.Sprintf("%s: len %d != %d", path, got.Len(), want.Len())
		}
		for i := 0; i < got.Len(); i++ {
			nextPath := fmt.Sprintf("%s[%d]", path, i)
			if diff := sameResultPath(got.Index(i), want.Index(i), nextPath); diff != "" {
				return diff
			}
		}
		return ""

	case reflect.Float32, reflect.Float64:
		g := got.Float()
		w := want.Float()
		if math.IsNaN(g) && math.IsNaN(w) {
			return ""
		}
		if g != w {
			return fmt.Sprintf("%s: %.17g != %.17g", path, g, w)
		}
		return ""

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if got.Int() != want.Int() {
			return fmt.Sprintf("%s: %d != %d", path, got.Int(), want.Int())
		}
		return ""

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if got.Uint() != want.Uint() {
			return fmt.Sprintf("%s: %d != %d", path, got.Uint(), want.Uint())
		}
		return ""

	case reflect.Bool:
		if got.Bool() != want.Bool() {
			return fmt.Sprintf("%s: %v != %v", path, got.Bool(), want.Bool())
		}
		return ""

	case reflect.String:
		if got.String() != want.String() {
			return fmt.Sprintf("%s: %q != %q", path, got.String(), want.String())
		}
		return ""

	default:
		// For other comparable types, try direct comparison via Interface
		gi := got.Interface()
		wi := want.Interface()
		if gi != wi {
			return fmt.Sprintf("%s: %v != %v", path, gi, wi)
		}
		return ""
	}
}

func TestTwoWayANOVAFromTableEqualsCells(t *testing.T) {
	dt := buildLongTable(t, "tw", func(s string) (any, error) {
		return strings.TrimSpace(s), nil
	})

	r1, err := stats.TwoWayANOVAFromTable(dt, insyra.Name("value"), insyra.Name("A"), insyra.Name("B"))
	if err != nil {
		t.Fatalf("TwoWayANOVAFromTable error: %v", err)
	}

	longRef.load(t)
	valueStr := longRef.getString(t, "tw.value")
	valueParts := strings.Split(valueStr, ",")
	values := make([]any, len(valueParts))
	for i, s := range valueParts {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		values[i] = v
	}

	aStr := longRef.getString(t, "tw.A")
	aParts := strings.Split(aStr, ",")
	aCells := make([]string, len(aParts))
	for i, s := range aParts {
		aCells[i] = strings.TrimSpace(s)
	}

	bStr := longRef.getString(t, "tw.B")
	bParts := strings.Split(bStr, ",")
	bCells := make([]string, len(bParts))
	for i, s := range bParts {
		bCells[i] = strings.TrimSpace(s)
	}

	levelsA := make([]string, 0)
	seenA := make(map[string]bool)
	for _, v := range aCells {
		if !seenA[v] {
			seenA[v] = true
			levelsA = append(levelsA, v)
		}
	}
	levelsB := make([]string, 0)
	seenB := make(map[string]bool)
	for _, v := range bCells {
		if !seenB[v] {
			seenB[v] = true
			levelsB = append(levelsB, v)
		}
	}

	aIndex := make(map[string]int)
	for i, v := range levelsA {
		aIndex[v] = i
	}
	bIndex := make(map[string]int)
	for i, v := range levelsB {
		bIndex[v] = i
	}

	cellVals := make([][]any, len(levelsA)*len(levelsB))
	for i := range cellVals {
		cellVals[i] = make([]any, 0)
	}
	for r := range values {
		ai := aIndex[aCells[r]]
		bi := bIndex[bCells[r]]
		idx := ai*len(levelsB) + bi
		cellVals[idx] = append(cellVals[idx], values[r])
	}

	cellLists := make([]insyra.IDataList, len(cellVals))
	for i, cv := range cellVals {
		cellLists[i] = insyra.NewDataList(cv...)
	}

	r2, err := stats.TwoWayANOVA(len(levelsA), len(levelsB), cellLists)
	if err != nil {
		t.Fatalf("TwoWayANOVA error: %v", err)
	}

	if diff := sameResult(r1, r2); diff != "" {
		t.Errorf("results differ: %s", diff)
	}
}

func TestTwoWayANOVAFromTableSelectors(t *testing.T) {
	dt := buildLongTable(t, "tw", func(s string) (any, error) {
		return strings.TrimSpace(s), nil
	})

	rName, err := stats.TwoWayANOVAFromTable(dt, insyra.Name("value"), insyra.Name("A"), insyra.Name("B"))
	if err != nil {
		t.Fatalf("Name selectors error: %v", err)
	}

	rIdx, err := stats.TwoWayANOVAFromTable(dt, "A", "B", "C")
	if err != nil {
		t.Fatalf("Index selectors error: %v", err)
	}

	rInt, err := stats.TwoWayANOVAFromTable(dt, 0, 1, 2)
	if err != nil {
		t.Fatalf("Int selectors error: %v", err)
	}

	if diff := sameResult(rName, rIdx); diff != "" {
		t.Errorf("Name vs Index selectors differ: %s", diff)
	}
	if diff := sameResult(rName, rInt); diff != "" {
		t.Errorf("Name vs Int selectors differ: %s", diff)
	}
}

func TestLongFormatErrors(t *testing.T) {
	tests := []struct {
		name      string
		dt        insyra.IDataTable
		valueCol  any
		factorA   any
		factorB   any
		wantError string
	}{
		{
			name:      "nil table untyped",
			dt:        nil,
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "table is nil",
		},
		{
			name:      "nil table typed",
			dt:        (*insyra.DataTable)(nil),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "table is nil",
		},
		{
			name:      "value column not found",
			dt:        insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("a"), insyra.NewDataList(3, 4).SetName("b"), insyra.NewDataList(5, 6).SetName("c")),
			valueCol:  "Z",
			factorA:   insyra.Name("a"),
			factorB:   insyra.Name("b"),
			wantError: "value column: ",
		},
		{
			name:      "value column non-numeric",
			dt:        insyra.NewDataTable(insyra.NewDataList(1.0, "x", 3.0).SetName("value"), insyra.NewDataList("a", "b", "c").SetName("A"), insyra.NewDataList("p", "q", "r").SetName("B")),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "value column value contains a non-numeric value at row 2: x",
		},
		{
			name:      "value column Inf",
			dt:        insyra.NewDataTable(insyra.NewDataList(math.Inf(1), 2.0, 3.0).SetName("value"), insyra.NewDataList("a", "b", "c").SetName("A"), insyra.NewDataList("p", "q", "r").SetName("B")),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "value column value contains a non-finite value at row 1: +Inf",
		},
		{
			name:      "factor A nil cell",
			dt:        insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("value"), insyra.NewDataList("a", "b", nil, "d").SetName("A"), insyra.NewDataList("p", "q", "r", "s").SetName("B")),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "factor A column A has no level at row 3",
		},
		{
			name:      "factor B NaN cell",
			dt:        insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("value"), insyra.NewDataList("a", "b", "c", "d").SetName("A"), insyra.NewDataList("p", "q", math.NaN(), "s").SetName("B")),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "factor B column B has no level at row 3",
		},
		{
			name:      "factor A one level",
			dt:        insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("value"), insyra.NewDataList("a", "a", "a", "a").SetName("A"), insyra.NewDataList("p", "q", "r", "s").SetName("B")),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "factor A has fewer than two levels",
		},
		{
			name:      "missing cell combination",
			dt:        insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0).SetName("value"), insyra.NewDataList("x", "x", "y").SetName("A"), insyra.NewDataList("p", "q", "p").SetName("B")),
			valueCol:  insyra.Name("value"),
			factorA:   insyra.Name("A"),
			factorB:   insyra.Name("B"),
			wantError: "no observations for A=y, B=q",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := stats.TwoWayANOVAFromTable(tc.dt, tc.valueCol, tc.factorA, tc.factorB)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantError)
			}
			if tc.name == "value column not found" {
				if !strings.HasPrefix(err.Error(), tc.wantError) {
					t.Errorf("error prefix mismatch: got %q, want prefix %q", err.Error(), tc.wantError)
				}
				if len(err.Error()) <= len(tc.wantError) {
					t.Errorf("error message %q is not longer than prefix %q", err.Error(), tc.wantError)
				}
			} else {
				if err.Error() != tc.wantError {
					t.Errorf("error mismatch: got %q, want %q", err.Error(), tc.wantError)
				}
			}
			if tc.dt != nil {
				if dt, ok := tc.dt.(*insyra.DataTable); ok && dt != nil {
					if dt.Err() != nil {
						t.Errorf("original table Err() should be nil, got %v", dt.Err())
					}
				}
			}
		})
	}
}

// buildLongTableRM builds a long-format table for repeated measures / Friedman tests.
func buildLongTableRM(t *testing.T, prefix string) *insyra.DataTable {
	t.Helper()
	longRef.load(t)

	valueStr := longRef.getString(t, prefix+".value")
	valueParts := strings.Split(valueStr, ",")
	values := make([]any, len(valueParts))
	for i, s := range valueParts {
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			t.Fatalf("parse value %d: %v", i, err)
		}
		values[i] = v
	}

	condStr := longRef.getString(t, prefix+".cond")
	condParts := strings.Split(condStr, ",")
	condCells := make([]any, len(condParts))
	for i, s := range condParts {
		condCells[i] = strings.TrimSpace(s)
	}

	subjStr := longRef.getString(t, prefix+".subj")
	subjParts := strings.Split(subjStr, ",")
	subjCells := make([]any, len(subjParts))
	for i, s := range subjParts {
		subjCells[i] = strings.TrimSpace(s)
	}

	return insyra.NewDataTable(
		insyra.NewDataList(values...).SetName("value"),
		insyra.NewDataList(condCells...).SetName("cond"),
		insyra.NewDataList(subjCells...).SetName("subj"),
	)
}

// TestRepeatedMeasuresANOVAFromTableMatchesR tests RepeatedMeasuresANOVAFromTable
// against R reference values (prefix "rm").
func TestRepeatedMeasuresANOVAFromTableMatchesR(t *testing.T) {
	dt := buildLongTableRM(t, "rm")

	r, err := stats.RepeatedMeasuresANOVAFromTable(dt, insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
	if err != nil {
		t.Fatalf("RepeatedMeasuresANOVAFromTable error: %v", err)
	}

	longRef.load(t)

	check := func(label string, got, want, tol float64) {
		if !aClose(got, want, tol) {
			t.Errorf("%s: got %.17g, want %.17g", label, got, want)
		}
	}
	checkP := func(label string, got, want float64) {
		if !aClose(got, want, tolANOVAP) {
			t.Errorf("%s: got %.17g, want %.17g", label, got, want)
		}
	}

	check("SSB", r.Factor.SumOfSquares, longRef.get(t, "rm.SSB"), tolANOVA)
	check("SSS", r.Subject.SumOfSquares, longRef.get(t, "rm.SSS"), tolANOVA)
	check("SSW", r.Within.SumOfSquares, longRef.get(t, "rm.SSW"), tolANOVA)
	if r.Factor.DF != int(longRef.get(t, "rm.DFB")) {
		t.Errorf("DFB: got %d, want %v", r.Factor.DF, longRef.get(t, "rm.DFB"))
	}
	if r.Subject.DF != int(longRef.get(t, "rm.DFS")) {
		t.Errorf("DFS: got %d, want %v", r.Subject.DF, longRef.get(t, "rm.DFS"))
	}
	if r.Within.DF != int(longRef.get(t, "rm.DFW")) {
		t.Errorf("DFW: got %d, want %v", r.Within.DF, longRef.get(t, "rm.DFW"))
	}
	check("F", r.Factor.F, longRef.get(t, "rm.F"), tolANOVA)
	checkP("P", r.Factor.P, longRef.get(t, "rm.P"))
	check("eta", r.Factor.EtaSquared, longRef.get(t, "rm.eta"), tolANOVAEta)
	check("totalSS", r.TotalSS, longRef.get(t, "rm.totalSS"), tolANOVA)
}

// TestFriedmanTestFromTableMatchesR tests FriedmanTestFromTable against R
// reference values (prefix "fr").
func TestFriedmanTestFromTableMatchesR(t *testing.T) {
	dt := buildLongTableRM(t, "fr")

	r, err := stats.FriedmanTestFromTable(dt, insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
	if err != nil {
		t.Fatalf("FriedmanTestFromTable error: %v", err)
	}

	longRef.load(t)

	if !aClose(r.Statistic, longRef.get(t, "fr.stat"), 1e-12) {
		t.Errorf("Statistic: got %.17g, want %.17g", r.Statistic, longRef.get(t, "fr.stat"))
	}
	if r.DF == nil || *r.DF != longRef.get(t, "fr.df") {
		t.Errorf("DF: got %v, want %v", r.DF, longRef.get(t, "fr.df"))
	}
	if !aClose(r.PValue, longRef.get(t, "fr.p"), 1e-10) {
		t.Errorf("PValue: got %.17g, want %.17g", r.PValue, longRef.get(t, "fr.p"))
	}
	if len(r.EffectSizes) == 0 {
		t.Fatalf("EffectSizes is empty")
	}
	if !aClose(r.EffectSizes[0].Value, longRef.get(t, "fr.W"), 1e-12) {
		t.Errorf("Kendall's W: got %.17g, want %.17g", r.EffectSizes[0].Value, longRef.get(t, "fr.W"))
	}
	if r.NSubjects != 7 {
		t.Errorf("NSubjects: got %d, want 7", r.NSubjects)
	}
	if r.KConditions != 3 {
		t.Errorf("KConditions: got %d, want 3", r.KConditions)
	}
}

// TestRepeatedMeasuresLongEqualsLists tests that the long-format entry points
// produce the same results as calling the list-based functions directly.
func TestRepeatedMeasuresLongEqualsLists(t *testing.T) {
	for _, prefix := range []string{"rm", "fr"} {
		t.Run(prefix, func(t *testing.T) {
			dt := buildLongTableRM(t, prefix)

			longRef.load(t)
			valueStr := longRef.getString(t, prefix+".value")
			valueParts := strings.Split(valueStr, ",")
			values := make([]any, len(valueParts))
			for i, s := range valueParts {
				v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
				values[i] = v
			}

			condStr := longRef.getString(t, prefix+".cond")
			condParts := strings.Split(condStr, ",")
			condCells := make([]string, len(condParts))
			for i, s := range condParts {
				condCells[i] = strings.TrimSpace(s)
			}

			subjStr := longRef.getString(t, prefix+".subj")
			subjParts := strings.Split(subjStr, ",")
			subjCells := make([]string, len(subjParts))
			for i, s := range subjParts {
				subjCells[i] = strings.TrimSpace(s)
			}

			// Group manually: subjects and conditions in first-appearance order
			subjects := make([]string, 0)
			seenSubj := make(map[string]bool)
			for _, s := range subjCells {
				if !seenSubj[s] {
					seenSubj[s] = true
					subjects = append(subjects, s)
				}
			}
			conditions := make([]string, 0)
			seenCond := make(map[string]bool)
			for _, c := range condCells {
				if !seenCond[c] {
					seenCond[c] = true
					conditions = append(conditions, c)
				}
			}

			subjIndex := make(map[string]int)
			for i, s := range subjects {
				subjIndex[s] = i
			}
			condIndex := make(map[string]int)
			for i, c := range conditions {
				condIndex[c] = i
			}

			grid := make([][]float64, len(subjects))
			for i := range grid {
				grid[i] = make([]float64, len(conditions))
			}
			for r := range values {
				s := subjIndex[subjCells[r]]
				c := condIndex[condCells[r]]
				grid[s][c] = values[r].(float64)
			}

			lists := make([]insyra.IDataList, len(subjects))
			for i := range lists {
				lists[i] = insyra.NewDataList(grid[i])
			}

			if prefix == "rm" {
				r1, err := stats.RepeatedMeasuresANOVAFromTable(dt, insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
				if err != nil {
					t.Fatalf("RepeatedMeasuresANOVAFromTable error: %v", err)
				}
				r2, err := stats.RepeatedMeasuresANOVA(lists)
				if err != nil {
					t.Fatalf("RepeatedMeasuresANOVA error: %v", err)
				}
				if diff := sameResult(r1, r2); diff != "" {
					t.Errorf("results differ: %s", diff)
				}
			} else {
				r1, err := stats.FriedmanTestFromTable(dt, insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
				if err != nil {
					t.Fatalf("FriedmanTestFromTable error: %v", err)
				}
				r2, err := stats.FriedmanTest(lists)
				if err != nil {
					t.Fatalf("FriedmanTest error: %v", err)
				}
				if diff := sameResult(r1, r2); diff != "" {
					t.Errorf("results differ: %s", diff)
				}
			}
		})
	}
}

// TestRepeatedMeasuresLongErrors tests error cases for the long-format
// repeated measures entry points.
func TestRepeatedMeasuresLongErrors(t *testing.T) {
	tests := []struct {
		name      string
		dt        *insyra.DataTable
		wantError string
	}{
		{
			name: "subject s3 has no t2",
			dt: insyra.NewDataTable(
				insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0).SetName("value"),
				insyra.NewDataList("t1", "t2", "t1", "t2", "t1").SetName("cond"),
				insyra.NewDataList("s1", "s1", "s2", "s2", "s3").SetName("subj"),
			),
			wantError: "subject s3 has no observation for condition t2",
		},
		{
			name: "subject s2 has t1 twice at rows 3 and 5",
			dt: insyra.NewDataTable(
				insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0).SetName("value"),
				insyra.NewDataList("t1", "t2", "t1", "t2", "t1").SetName("cond"),
				insyra.NewDataList("s1", "s2", "s2", "s1", "s2").SetName("subj"),
			),
			wantError: "subject s2 has more than one observation for condition t1 (rows 3 and 5)",
		},
		{
			name: "only condition t1 present",
			dt: insyra.NewDataTable(
				insyra.NewDataList(1.0, 2.0, 3.0).SetName("value"),
				insyra.NewDataList("t1", "t1", "t1").SetName("cond"),
				insyra.NewDataList("s1", "s2", "s3").SetName("subj"),
			),
			wantError: "at least two conditions are required",
		},
		{
			name: "only subject s1 present",
			dt: insyra.NewDataTable(
				insyra.NewDataList(1.0, 2.0).SetName("value"),
				insyra.NewDataList("t1", "t2").SetName("cond"),
				insyra.NewDataList("s1", "s1").SetName("subj"),
			),
			wantError: "at least two subjects are required",
		},
		{
			name: "subject cell nil at row 5",
			dt: insyra.NewDataTable(
				insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0).SetName("value"),
				insyra.NewDataList("t1", "t2", "t1", "t2", "t1").SetName("cond"),
				insyra.NewDataList("s1", "s2", "s1", "s2", nil).SetName("subj"),
			),
			wantError: "subject column subj has no level at row 5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, fn := range []struct {
				name string
				f    func(insyra.IDataTable, any, any, any) (any, error)
			}{
				{"RepeatedMeasuresANOVAFromTable", func(dt insyra.IDataTable, v, c, s any) (any, error) {
					return stats.RepeatedMeasuresANOVAFromTable(dt, v, c, s)
				}},
				{"FriedmanTestFromTable", func(dt insyra.IDataTable, v, c, s any) (any, error) {
					return stats.FriedmanTestFromTable(dt, v, c, s)
				}},
			} {
				t.Run(fn.name, func(t *testing.T) {
					_, err := fn.f(tc.dt, insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
					if err == nil {
						t.Fatalf("expected error %q, got nil", tc.wantError)
					}
					if err.Error() != tc.wantError {
						t.Errorf("error mismatch: got %q, want %q", err.Error(), tc.wantError)
					}
				})
			}
		})
	}
}

func TestLongFormatIntegerLevelsByValue(t *testing.T) {
	values := []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	aCells := []any{int64(1), int(1), int64(2), int(2), int64(1), int(1), int64(2), int(2)}
	bCells := []any{"p", "p", "p", "p", "q", "q", "q", "q"}

	dt := insyra.NewDataTable(
		insyra.NewDataList(values...).SetName("value"),
		insyra.NewDataList(aCells...).SetName("A"),
		insyra.NewDataList(bCells...).SetName("B"),
	)

	r, err := stats.TwoWayANOVAFromTable(dt, insyra.Name("value"), insyra.Name("A"), insyra.Name("B"))
	if err != nil {
		t.Fatalf("TwoWayANOVAFromTable error: %v", err)
	}
	if r.FactorA.DF != 1 {
		t.Errorf("FactorA.DF: got %d, want 1 (two levels, not four)", r.FactorA.DF)
	}
}
