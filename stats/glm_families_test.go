package stats

import (
	"math"
	"testing"
)

func TestGLMFamiliesCoreValues(t *testing.T) {
	bin := binomialFamily{}
	if got := bin.devianceResidualSq(1, 0.8, 1); math.Abs(got-0.4462871026284194) > 1e-12 {
		t.Fatalf("binomial deviance = %.15g", got)
	}
	if got := bin.logLikContrib(1, 0.8, 1); math.Abs(got-math.Log(0.8)) > 1e-12 {
		t.Fatalf("binomial logLik = %.15g", got)
	}

	pois := poissonFamily{}
	if got := pois.devianceResidualSq(3, 2.5, 1); math.Abs(got-0.09392934076372731) > 1e-12 {
		t.Fatalf("poisson deviance = %.15g", got)
	}

	gauss := gaussianFamily{}
	if got := gauss.devianceResidualSq(4, 2.5, 2); got != 4.5 {
		t.Fatalf("gaussian deviance = %.15g", got)
	}
}
