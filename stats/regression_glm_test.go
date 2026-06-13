package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestGLMGaussianIdentityMatchesLinearRegression(t *testing.T) {
	y := insyra.NewDataList(1.2, 2.1, 2.9, 4.2, 5.1, 5.8, 7.0, 8.2)
	x := insyra.NewDataList(0.2, 0.7, 1.1, 1.8, 2.2, 2.9, 3.4, 4.1)
	glm, err := GLM(GLMOptions{
		Family:    Gaussian,
		Link:      Identity,
		MaxIter:   100,
		Tolerance: 1e-10,
	}, y, x)
	if err != nil {
		t.Fatalf("GLM error: %v", err)
	}
	lm, err := LinearRegression(y, x)
	if err != nil {
		t.Fatalf("LinearRegression error: %v", err)
	}
	for i := range lm.Coefficients {
		if math.Abs(glm.Coefficients[i]-lm.Coefficients[i]) > 1e-9 {
			t.Fatalf("coef[%d] = %.15g, want %.15g", i, glm.Coefficients[i], lm.Coefficients[i])
		}
		if math.Abs(glm.StandardErrors[i]-lm.StandardErrors[i]) > 1e-9 {
			t.Fatalf("se[%d] = %.15g, want %.15g", i, glm.StandardErrors[i], lm.StandardErrors[i])
		}
	}
}

func TestGLMBinomialMatchesLogisticRegression(t *testing.T) {
	y := insyra.NewDataList(0, 0, 1, 0, 1, 1, 0, 1, 1, 0)
	x := insyra.NewDataList(-2.1, -1.5, -0.4, 0.1, 0.6, 1.2, 1.7, 2.1, 2.8, 3.3)
	glm, err := GLM(GLMOptions{Family: Binomial, Link: Logit, MaxIter: 100, Tolerance: 1e-10}, y, x)
	if err != nil {
		t.Fatalf("GLM error: %v", err)
	}
	logit, err := LogisticRegressionWithOptions(LogisticRegressionOptions{MaxIter: 100, Tolerance: 1e-10}, y, x)
	if err != nil {
		t.Fatalf("LogisticRegression error: %v", err)
	}
	for i := range logit.Coefficients {
		if math.Abs(glm.Coefficients[i]-logit.Coefficients[i]) > 1e-9 {
			t.Fatalf("coef[%d] = %.15g, want %.15g", i, glm.Coefficients[i], logit.Coefficients[i])
		}
	}
}

func TestGLMPredict(t *testing.T) {
	y := insyra.NewDataList(0, 0, 1, 0, 1, 1, 0, 1, 1, 0)
	x := insyra.NewDataList(-2.1, -1.5, -0.4, 0.1, 0.6, 1.2, 1.7, 2.1, 2.8, 3.3)
	logit, err := LogisticRegressionWithOptions(LogisticRegressionOptions{MaxIter: 100, Tolerance: 1e-10}, y, x)
	if err != nil {
		t.Fatalf("LogisticRegression error: %v", err)
	}
	newX := insyra.NewDataList(-1.0, 1.0, 3.0)
	probs, err := logit.Predict(PredictResponse, newX)
	if err != nil {
		t.Fatalf("Predict response error: %v", err)
	}
	if probs.Len() != 3 {
		t.Fatalf("expected 3 probabilities, got %d", probs.Len())
	}
	classes, err := logit.Predict(PredictClass, newX)
	if err != nil {
		t.Fatalf("Predict class error: %v", err)
	}
	if classes.Len() != 3 {
		t.Fatalf("expected 3 classes, got %d", classes.Len())
	}

	pois, err := PoissonRegressionWithOptions(PoissonRegressionOptions{MaxIter: 100, Tolerance: 1e-10}, insyra.NewDataList(1, 2, 3, 5, 8, 13), insyra.NewDataList(0, 1, 2, 3, 4, 5))
	if err != nil {
		t.Fatalf("PoissonRegression error: %v", err)
	}
	if _, err := pois.Predict(PredictClass, newX); err == nil {
		t.Fatalf("expected class prediction error for poisson")
	}
}
