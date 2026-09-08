package quant

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

// This file checks OptimizePortfolioMoments against cvxpy, an independent
// convex solver, on random long-only and box-bounded minimum-variance
// problems. It is opt-in because cvxpy is usually absent:
//
//	INSYRA_RUN_CVXPY=1 go test ./quant/ -run CVXPY
//
// It looks for Python in the crosslang virtual environment first and falls
// back to python3 on PATH. To install the dependency there:
//
//	~/.cache/insyra-crosslang-venv/bin/python3 -m pip install cvxpy
//
// Nothing is recorded into the repository, so `go test ./...` on a machine
// with no Python is unaffected.

const cvxpyPortfolioScript = `
import json, sys
import numpy as np
import cvxpy as cp

problems = json.load(sys.stdin)
out = []
for problem in problems:
    sigma = np.array(problem["cov"], dtype=float)
    lo = np.array(problem["lo"], dtype=float)
    hi = np.array(problem["hi"], dtype=float)
    n = sigma.shape[0]
    w = cp.Variable(n)
    objective = cp.Minimize(cp.quad_form(w, cp.psd_wrap(sigma)))
    constraints = [cp.sum(w) == 1, w >= lo, w <= hi]
    cp.Problem(objective, constraints).solve(solver=cp.CLARABEL)
    weights = np.array(w.value, dtype=float)
    out.append(float(weights @ sigma @ weights))
print(json.dumps(out))
`

type cvxpyPortfolioProblem struct {
	Cov [][]float64 `json:"cov"`
	Lo  []float64   `json:"lo"`
	Hi  []float64   `json:"hi"`
}

// randomPSDCovariance builds A'A/m plus a ridge, which is symmetric, positive
// definite, and conditioned well enough that both solvers agree to well under
// the compared tolerance.
func randomPSDCovariance(rng *rand.Rand, n int) [][]float64 {
	m := n + 4
	factors := make([][]float64, m)
	for i := range m {
		factors[i] = make([]float64, n)
		for j := range n {
			factors[i][j] = rng.NormFloat64() * 0.2
		}
	}
	cov := make([][]float64, n)
	for i := range n {
		cov[i] = make([]float64, n)
	}
	for i := range n {
		for j := i; j < n; j++ {
			sum := 0.0
			for k := range m {
				sum += factors[k][i] * factors[k][j]
			}
			value := sum / float64(m)
			if i == j {
				value += 0.01
			}
			cov[i][j] = value
			cov[j][i] = value
		}
	}
	return cov
}

func TestPortfolioAgreesWithCVXPY(t *testing.T) {
	const verification = "the portfolio-optimisation comparison against cvxpy"
	// Opt-in because cvxpy is usually absent. Strict mode says it is present,
	// so the reason to stay opt-in no longer holds.
	if os.Getenv("INSYRA_RUN_CVXPY") != "1" && !reftest.Strict() {
		t.Skipf("set INSYRA_RUN_CVXPY=1 or %s=1 to run %s", reftest.StrictEnv, verification)
	}

	python := filepath.Join(os.Getenv("HOME"), ".cache", "insyra-crosslang-venv", "bin", "python3")
	if _, err := os.Stat(python); err != nil {
		found, lookErr := exec.LookPath("python3")
		if lookErr != nil {
			reftest.Missing(t, "python3", verification, lookErr)
			return
		}
		python = found
	}
	probe := exec.Command(python, "-c", "import cvxpy, numpy")
	var probeErr bytes.Buffer
	probe.Stderr = &probeErr
	if err := probe.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with cvxpy", verification, err, probeErr.Bytes())
		return
	}

	rng := rand.New(rand.NewSource(20260905))
	const cases = 20
	problems := make([]cvxpyPortfolioProblem, 0, 2*cases)
	kinds := make([]string, 0, 2*cases)
	for kind := range 2 {
		for range cases {
			n := 5 + rng.Intn(4)
			cov := randomPSDCovariance(rng, n)
			lo := make([]float64, n)
			hi := make([]float64, n)
			for i := range n {
				if kind == 0 {
					lo[i], hi[i] = 0, 1
				} else {
					lo[i] = 0.01 + rng.Float64()*0.02
					hi[i] = 1/float64(n) + 0.1 + rng.Float64()*0.3
				}
			}
			problems = append(problems, cvxpyPortfolioProblem{Cov: cov, Lo: lo, Hi: hi})
			if kind == 0 {
				kinds = append(kinds, "long-only")
			} else {
				kinds = append(kinds, "box-bounded")
			}
		}
	}

	payload, err := json.Marshal(problems)
	if err != nil {
		t.Fatalf("marshal problems: %v", err)
	}
	command := exec.Command(python, "-c", cvxpyPortfolioScript)
	command.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("cvxpy run failed: %v: %s", err, stderr.String())
	}
	var reference []float64
	if err := json.Unmarshal(stdout.Bytes(), &reference); err != nil {
		t.Fatalf("decode cvxpy output %q: %v", stdout.String(), err)
	}
	if len(reference) != len(problems) {
		t.Fatalf("cvxpy returned %d objectives for %d problems", len(reference), len(problems))
	}

	worst := 0.0
	for i, problem := range problems {
		mean := make([]float64, len(problem.Lo))
		result, err := OptimizePortfolioMoments(mean, problem.Cov, nil, PortfolioConfig{
			Objective: MinimumVariance,
			MinWeight: problem.Lo,
			MaxWeight: problem.Hi,
		})
		if err != nil {
			t.Fatalf("problem %d (%s): %v", i, kinds[i], err)
		}
		if !result.Converged {
			t.Errorf("problem %d (%s): Converged = false after %d iterations", i, kinds[i], result.Iterations)
		}
		difference := math.Abs(result.Variance - reference[i])
		if difference > worst {
			worst = difference
		}
		if difference > 1e-6 {
			t.Errorf("problem %d (%s): objective %.17g vs cvxpy %.17g, difference %.3g", i, kinds[i], result.Variance, reference[i], difference)
		}
	}
	t.Logf("largest objective difference against cvxpy over %d problems: %.3g", len(problems), worst)
}
