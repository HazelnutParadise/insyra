package clustering

import "testing"

func TestKMeansGeneratedCasesSatisfySSEIdentities(t *testing.T) {
	for _, tc := range generatedInternalKMeansCases() {
		t.Run(tc.name, func(t *testing.T) {
			got, err := KMeans(tc.rows, tc.k, KMeansOptions{NStart: tc.nstart, IterMax: tc.itermax, Seed: &tc.seed})
			if err != nil {
				t.Fatalf("KMeans error: %v", err)
			}
			assertKMeansSSEIdentities(t, tc.rows, got)
		})
	}
}

func TestLanceWilliamsFormulas(t *testing.T) {
	for _, tc := range lanceWilliamsFormulaCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := updatedDistance(tc.method, tc.other, tc.a, tc.b, tc.dik, tc.djk, tc.dij)
			if !almostEqual(got, tc.want) {
				t.Fatalf("updatedDistance(%s)=%v want=%v", tc.method, got, tc.want)
			}
		})
	}
}

func TestReducibleLinkagesHaveMonotoneHeights(t *testing.T) {
	for _, tc := range reducibleHierarchyCases() {
		t.Run(tc.name, func(t *testing.T) {
			labels := make([]string, len(tc.rows))
			for i := range labels {
				labels[i] = string(rune('a' + i))
			}
			got, err := Hierarchical(tc.rows, labels, tc.method)
			if err != nil {
				t.Fatalf("Hierarchical error: %v", err)
			}
			for i := 1; i < len(got.Height); i++ {
				if got.Height[i] < got.Height[i-1] && !almostEqual(got.Height[i], got.Height[i-1]) {
					t.Fatalf("height[%d]=%v < height[%d]=%v for method %s", i, got.Height[i], i-1, got.Height[i-1], tc.method)
				}
			}
		})
	}
}
