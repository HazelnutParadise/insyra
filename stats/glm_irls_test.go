package stats

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestFitIRLSGaussianIdentityMatchesOLS(t *testing.T) {
	y := []float64{1, 2, 1.3, 3.75, 2.25, 5.4}
	xs := [][]float64{
		{0, 1, 2, 3, 4, 5},
		{2, 1, 0, 1, 2, 3},
	}
	X := buildDesignMatrix(xs, len(y))
	fit, err := fitIRLS(X, y, gaussianFamily{}, identityLink{}, irlsOptions{maxIter: 25, tolerance: 1e-10})
	if err != nil {
		t.Fatalf("fitIRLS error: %v", err)
	}
	olsBeta, xtxInv := solveOLS(X, y)
	if len(fit.beta) != len(olsBeta) {
		t.Fatalf("beta length mismatch")
	}
	for i := range fit.beta {
		if math.Abs(fit.beta[i]-olsBeta[i]) > 1e-9 {
			t.Fatalf("beta[%d] = %.15g, want %.15g", i, fit.beta[i], olsBeta[i])
		}
		for j := range fit.covUnscaled[i] {
			if math.Abs(fit.covUnscaled[i][j]-xtxInv[i][j]) > 1e-9 {
				t.Fatalf("cov[%d][%d] = %.15g, want %.15g", i, j, fit.covUnscaled[i][j], xtxInv[i][j])
			}
		}
	}
}

func TestFitIRLSLogisticAndPoissonConverge(t *testing.T) {
	xs := [][]float64{{-3, -2, -1, 0, 1, 2, 3, 4}}
	X := buildDesignMatrix(xs, 8)
	logitY := []float64{0, 0, 0, 1, 0, 1, 1, 1}
	logitFit, err := fitIRLS(X, logitY, binomialFamily{}, logitLink{}, irlsOptions{maxIter: 100, tolerance: 1e-10})
	if err != nil {
		t.Fatalf("logistic fitIRLS error: %v", err)
	}
	if !logitFit.converged || logitFit.iterations > 25 {
		t.Fatalf("logistic convergence = %v iterations=%d", logitFit.converged, logitFit.iterations)
	}

	poisY := []float64{1, 1, 2, 3, 4, 6, 8, 11}
	poisFit, err := fitIRLS(X, poisY, poissonFamily{}, logLink{}, irlsOptions{maxIter: 100, tolerance: 1e-10})
	if err != nil {
		t.Fatalf("poisson fitIRLS error: %v", err)
	}
	if !poisFit.converged || poisFit.iterations > 25 {
		t.Fatalf("poisson convergence = %v iterations=%d", poisFit.converged, poisFit.iterations)
	}
}

func TestSolveWeightedNormalEquations(t *testing.T) {
	X := mat.NewDense(3, 2, []float64{
		1, 0,
		1, 1,
		1, 2,
	})
	beta, err := solveWeightedNormalEquations(X, []float64{1, 1, 1}, []float64{1, 3, 5}, 0)
	if err != nil {
		t.Fatalf("solveWeightedNormalEquations error: %v", err)
	}
	if math.Abs(beta[0]-1) > 1e-12 || math.Abs(beta[1]-2) > 1e-12 {
		t.Fatalf("beta = %v, want [1 2]", beta)
	}
}
