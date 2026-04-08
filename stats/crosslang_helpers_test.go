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

	checkPy := exec.Command("python", "-c", "import scipy, numpy, statsmodels")
	if out, err := checkPy.CombinedOutput(); err != nil {
		t.Skipf("python scientific stack unavailable: %v, out=%s", err, string(out))
	}

	checkR := exec.Command("Rscript", "-e", "if (!requireNamespace('jsonlite', quietly=TRUE)) quit(status=1)")
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
