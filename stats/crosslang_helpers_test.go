package stats_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type crossLangBaseline map[string]any

func requireCrossLangTools(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("python"); err != nil {
		t.Skipf("python not found: %v", err)
	}
	if _, err := exec.LookPath("Rscript"); err != nil {
		t.Skipf("Rscript not found: %v", err)
	}

	checkPy := exec.Command("python", "-c", "import scipy, numpy, statsmodels, sklearn")
	if out, err := checkPy.CombinedOutput(); err != nil {
		t.Skipf("python scientific stack unavailable: %v, out=%s", err, string(out))
	}

	checkR := exec.Command("Rscript", "-e", "pkgs <- c('jsonlite','cluster','dbscan'); ok <- all(sapply(pkgs, function(p) requireNamespace(p, quietly=TRUE))); if (!ok) quit(status=1)")
	if out, err := checkR.CombinedOutput(); err != nil {
		t.Skipf("R jsonlite unavailable: %v, out=%s", err, string(out))
	}
}

func runPythonBaseline(t *testing.T, method string, payload any) crossLangBaseline {
	t.Helper()
	return runBaselineScript(t, "python", filepath.Join("testdata", "crosslang_baseline.py"), method, payload)
}

func runRBaseline(t *testing.T, method string, payload any) crossLangBaseline {
	t.Helper()
	return runBaselineScript(t, "Rscript", filepath.Join("testdata", "crosslang_baseline.R"), method, payload)
}

func runBaselineScript(t *testing.T, exe, scriptPath, method string, payload any) crossLangBaseline {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	cmd := exec.Command(exe, scriptPath, method, string(raw))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s baseline failed (method=%s): %v\nstderr=%s\nstdout=%s", exe, method, err, stderr.String(), stdout.String())
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		t.Fatalf("%s baseline returned empty output (method=%s)", exe, method)
	}

	m := crossLangBaseline{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("parse %s baseline JSON failed (method=%s): %v\noutput=%s", exe, method, err, out)
	}
	return m
}

func baselineFloat(t *testing.T, m crossLangBaseline, key string) float64 {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	switch vv := v.(type) {
	case float64:
		return vv
	case string:
		f, err := strconv.ParseFloat(vv, 64)
		if err != nil {
			t.Fatalf("baseline key %q parse float failed: %v (raw=%v)", key, err, vv)
		}
		return f
	default:
		t.Fatalf("baseline key %q has non-float type %T", key, v)
		return math.NaN()
	}
}

func baselineFloatSlice(t *testing.T, m crossLangBaseline, key string) []float64 {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("baseline key %q has non-array type %T", key, v)
	}
	out := make([]float64, len(arr))
	for i, item := range arr {
		switch vv := item.(type) {
		case float64:
			out[i] = vv
		case string:
			f, err := strconv.ParseFloat(vv, 64)
			if err != nil {
				t.Fatalf("baseline key %q index %d parse float failed: %v (raw=%v)", key, i, err, vv)
			}
			out[i] = f
		default:
			t.Fatalf("baseline key %q index %d has non-float type %T", key, i, item)
		}
	}
	return out
}

func baselineFloatMatrix(t *testing.T, m crossLangBaseline, key string) [][]float64 {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	rows, ok := v.([]any)
	if !ok {
		t.Fatalf("baseline key %q has non-matrix type %T", key, v)
	}
	mat := make([][]float64, len(rows))
	for i, r := range rows {
		rowAny, ok := r.([]any)
		if !ok {
			t.Fatalf("baseline key %q row %d has non-array type %T", key, i, r)
		}
		mat[i] = make([]float64, len(rowAny))
		for j, cell := range rowAny {
			switch vv := cell.(type) {
			case float64:
				mat[i][j] = vv
			case string:
				f, err := strconv.ParseFloat(vv, 64)
				if err != nil {
					t.Fatalf("baseline key %q row=%d col=%d parse float failed: %v (raw=%v)", key, i, j, err, vv)
				}
				mat[i][j] = f
			default:
				t.Fatalf("baseline key %q row=%d col=%d has non-float type %T", key, i, j, cell)
			}
		}
	}
	return mat
}

func baselineIntSlice(t *testing.T, m crossLangBaseline, key string) []int {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("baseline key %q has non-array type %T", key, v)
	}
	out := make([]int, len(arr))
	for i, item := range arr {
		f, ok := toFloat64FromAny(item)
		if !ok {
			t.Fatalf("baseline key %q index %d has non-numeric type %T", key, i, item)
		}
		out[i] = int(f)
	}
	return out
}

func baselineBoolSlice(t *testing.T, m crossLangBaseline, key string) []bool {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("baseline key %q has non-array type %T", key, v)
	}
	out := make([]bool, len(arr))
	for i, item := range arr {
		b, ok := item.(bool)
		if !ok {
			t.Fatalf("baseline key %q index %d has non-bool type %T", key, i, item)
		}
		out[i] = b
	}
	return out
}

func baselineStringSlice(t *testing.T, m crossLangBaseline, key string) []string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("baseline key %q has non-array type %T", key, v)
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("baseline key %q index %d has non-string type %T", key, i, item)
		}
		out[i] = s
	}
	return out
}

func baselineIntMatrix(t *testing.T, m crossLangBaseline, key string) [][2]int {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	rows, ok := v.([]any)
	if !ok {
		t.Fatalf("baseline key %q has non-matrix type %T", key, v)
	}
	out := make([][2]int, len(rows))
	for i, row := range rows {
		rowAny, ok := row.([]any)
		if !ok || len(rowAny) != 2 {
			t.Fatalf("baseline key %q row %d invalid: %T %#v", key, i, row, row)
		}
		a, okA := toFloat64FromAny(rowAny[0])
		b, okB := toFloat64FromAny(rowAny[1])
		if !okA || !okB {
			t.Fatalf("baseline key %q row %d has non-numeric values", key, i)
		}
		out[i] = [2]int{int(a), int(b)}
	}
	return out
}

func assertCloseToBoth(t *testing.T, label string, got, rWant, pyWant, tol float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsNaN(rWant) || math.IsNaN(pyWant) {
		t.Fatalf("%s has NaN (go=%v r=%v py=%v)", label, got, rWant, pyWant)
	}
	if math.IsInf(got, 0) || math.IsInf(rWant, 0) || math.IsInf(pyWant, 0) {
		if !(math.IsInf(got, 1) && math.IsInf(rWant, 1) && math.IsInf(pyWant, 1)) &&
			!(math.IsInf(got, -1) && math.IsInf(rWant, -1) && math.IsInf(pyWant, -1)) {
			t.Fatalf("%s infinity mismatch (go=%v r=%v py=%v)", label, got, rWant, pyWant)
		}
		return
	}
	if math.Abs(got-rWant) > tol {
		t.Fatalf("%s mismatch vs R: got=%.15g r=%.15g tol=%g", label, got, rWant, tol)
	}
	if math.Abs(got-pyWant) > tol {
		t.Fatalf("%s mismatch vs Python: got=%.15g py=%.15g tol=%g", label, got, pyWant, tol)
	}
	if math.Abs(rWant-pyWant) > tol*10 {
		t.Fatalf("%s baseline drift R vs Python too large: r=%.15g py=%.15g tol=%g", label, rWant, pyWant, tol*10)
	}
}

func assertSliceCloseToBoth(t *testing.T, label string, got, rWant, pyWant []float64, tol float64) {
	t.Helper()
	if len(got) != len(rWant) || len(got) != len(pyWant) {
		t.Fatalf("%s length mismatch got=%d r=%d py=%d", label, len(got), len(rWant), len(pyWant))
	}
	for i := range got {
		assertCloseToBoth(t, fmt.Sprintf("%s[%d]", label, i), got[i], rWant[i], pyWant[i], tol)
	}
}

func assertMatrixCloseToBoth(t *testing.T, label string, got, rWant, pyWant [][]float64, tol float64) {
	t.Helper()
	if len(got) != len(rWant) || len(got) != len(pyWant) {
		t.Fatalf("%s row length mismatch got=%d r=%d py=%d", label, len(got), len(rWant), len(pyWant))
	}
	for i := range got {
		if len(got[i]) != len(rWant[i]) || len(got[i]) != len(pyWant[i]) {
			t.Fatalf("%s row %d length mismatch got=%d r=%d py=%d", label, i, len(got[i]), len(rWant[i]), len(pyWant[i]))
		}
		for j := range got[i] {
			assertCloseToBoth(t, fmt.Sprintf("%s[%d,%d]", label, i, j), got[i][j], rWant[i][j], pyWant[i][j], tol)
		}
	}
}

func assertIntSliceEqualToBoth(t *testing.T, label string, got, rWant, pyWant []int) {
	t.Helper()
	if len(got) != len(rWant) || len(got) != len(pyWant) {
		t.Fatalf("%s length mismatch got=%d r=%d py=%d", label, len(got), len(rWant), len(pyWant))
	}
	for i := range got {
		if got[i] != rWant[i] {
			t.Fatalf("%s mismatch vs R at %d: got=%d r=%d", label, i, got[i], rWant[i])
		}
		if got[i] != pyWant[i] {
			t.Fatalf("%s mismatch vs Python at %d: got=%d py=%d", label, i, got[i], pyWant[i])
		}
	}
}

func assertBoolSliceEqualToBoth(t *testing.T, label string, got, rWant, pyWant []bool) {
	t.Helper()
	if len(got) != len(rWant) || len(got) != len(pyWant) {
		t.Fatalf("%s length mismatch got=%d r=%d py=%d", label, len(got), len(rWant), len(pyWant))
	}
	for i := range got {
		if got[i] != rWant[i] {
			t.Fatalf("%s mismatch vs R at %d: got=%v r=%v", label, i, got[i], rWant[i])
		}
		if got[i] != pyWant[i] {
			t.Fatalf("%s mismatch vs Python at %d: got=%v py=%v", label, i, got[i], pyWant[i])
		}
	}
}

func assertStringSliceEqualToBoth(t *testing.T, label string, got, rWant, pyWant []string) {
	t.Helper()
	if len(got) != len(rWant) || len(got) != len(pyWant) {
		t.Fatalf("%s length mismatch got=%d r=%d py=%d", label, len(got), len(rWant), len(pyWant))
	}
	for i := range got {
		if got[i] != rWant[i] {
			t.Fatalf("%s mismatch vs R at %d: got=%q r=%q", label, i, got[i], rWant[i])
		}
		if got[i] != pyWant[i] {
			t.Fatalf("%s mismatch vs Python at %d: got=%q py=%q", label, i, got[i], pyWant[i])
		}
	}
}

func assertIntMatrixEqualToBoth(t *testing.T, label string, got, rWant, pyWant [][2]int) {
	t.Helper()
	if len(got) != len(rWant) || len(got) != len(pyWant) {
		t.Fatalf("%s length mismatch got=%d r=%d py=%d", label, len(got), len(rWant), len(pyWant))
	}
	for i := range got {
		if got[i] != rWant[i] {
			t.Fatalf("%s mismatch vs R at %d: got=%v r=%v", label, i, got[i], rWant[i])
		}
		if got[i] != pyWant[i] {
			t.Fatalf("%s mismatch vs Python at %d: got=%v py=%v", label, i, got[i], pyWant[i])
		}
	}
}

func toObservedExpectedPair(t *testing.T, value any) (float64, float64) {
	t.Helper()
	switch v := value.(type) {
	case [2]float64:
		return v[0], v[1]
	case []float64:
		if len(v) != 2 {
			t.Fatalf("expected []float64 length=2, got %d", len(v))
		}
		return v[0], v[1]
	case [2]any:
		a, okA := toFloat64FromAny(v[0])
		b, okB := toFloat64FromAny(v[1])
		if !okA || !okB {
			t.Fatalf("expected numeric [2]any pair, got %T with values %#v", value, value)
		}
		return a, b
	case []any:
		if len(v) != 2 {
			t.Fatalf("expected []any length=2, got %d", len(v))
		}
		a, okA := toFloat64FromAny(v[0])
		b, okB := toFloat64FromAny(v[1])
		if !okA || !okB {
			t.Fatalf("expected numeric []any pair, got %T with values %#v", value, value)
		}
		return a, b
	default:
		t.Fatalf("expected observed/expected pair cell, got %T (%#v)", value, value)
		return math.NaN(), math.NaN()
	}
}

func toFloat64FromAny(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

func assertNaNToBoth(t *testing.T, label string, got, rWant, pyWant float64) {
	t.Helper()
	if !math.IsNaN(got) || !math.IsNaN(rWant) || !math.IsNaN(pyWant) {
		t.Fatalf("%s expected NaN for go/r/py, got go=%v r=%v py=%v", label, got, rWant, pyWant)
	}
}
