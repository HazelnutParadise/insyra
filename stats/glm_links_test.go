package stats

import (
	"math"
	"testing"
)

func TestGLMLinksRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		link glmLink
		mu   float64
	}{
		{"logit", logitLink{}, 0.25},
		{"log", logLink{}, 2.5},
		{"identity", identityLink{}, -3.25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.link.mu(tc.link.eta(tc.mu))
			if math.Abs(got-tc.mu) > 1e-10 {
				t.Fatalf("round trip = %.15g, want %.15g", got, tc.mu)
			}
			if tc.link.muEta(tc.link.eta(tc.mu)) <= 0 {
				t.Fatalf("muEta must be positive")
			}
		})
	}
}
