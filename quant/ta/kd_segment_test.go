package ta

import (
	"math"
	"math/rand"
	"testing"
)

func kdTestBars(n int) (h, l, c []float64) {
	rng := rand.New(rand.NewSource(7))
	p := 100.0
	for i := 0; i < n; i++ {
		p += rng.NormFloat64()
		hi, lo := p+rng.Float64()*2, p-rng.Float64()*2
		h, l, c = append(h, hi), append(l, lo), append(c, lo+(hi-lo)*rng.Float64())
	}
	return h, l, c
}

func sameFloats(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !(a[i] == b[i] || (math.IsNaN(a[i]) && math.IsNaN(b[i]))) {
			return false
		}
	}
	return true
}

// Each segment is KD on its own: the result equals KD of every segment
// separately, laid end to end.
func TestKDSegmentStartsEqualSeparateSegments(t *testing.T) {
	h, l, c := kdTestBars(60)
	for _, method := range []KDMethod{KDRecursive, KDMovingAverage} {
		for _, n := range []int{1, 5, 9} {
			for _, starts := range [][]int{{20}, {15, 40}, {10, 12, 13, 50}, {58}} {
				o := KDOptions{Method: method, SegmentStarts: starts}
				got, err := KD(prices(h...), prices(l...), prices(c...), n, o)
				if err != nil {
					t.Fatal(err)
				}
				var wantK, wantD []float64
				for _, b := range segmentBounds(len(c), starts) {
					seg, err := KD(prices(h[b[0]:b[1]]...), prices(l[b[0]:b[1]]...), prices(c[b[0]:b[1]]...), n, KDOptions{Method: method})
					if err != nil {
						t.Fatal(err)
					}
					wantK, wantD = append(wantK, floats(t, seg.K)...), append(wantD, floats(t, seg.D)...)
				}
				if !sameFloats(floats(t, got.K), wantK) || !sameFloats(floats(t, got.D), wantD) {
					t.Fatalf("%v n=%d starts=%v: segmented result differs from separate segments", method, n, starts)
				}
			}
		}
	}
}

// A segment shorter than the warm-up shows NaN throughout, and the segment
// after it still starts clean.
func TestKDSegmentShorterThanWarmUp(t *testing.T) {
	h, l, c := kdTestBars(30)
	for _, method := range []KDMethod{KDRecursive, KDMovingAverage} {
		got, err := KD(prices(h...), prices(l...), prices(c...), 9, KDOptions{Method: method, SegmentStarts: []int{10, 14}})
		if err != nil {
			t.Fatal(err)
		}
		k, d := floats(t, got.K), floats(t, got.D)
		for i := 10; i < 14; i++ {
			if !math.IsNaN(k[i]) || !math.IsNaN(d[i]) {
				t.Fatalf("%v row %d inside a short segment: K %v D %v", method, i, k[i], d[i])
			}
		}
		if math.IsNaN(k[29]) || math.IsNaN(d[29]) {
			t.Fatalf("%v: last segment never produced a value", method)
		}
	}
}

// Without SegmentStarts the result is bit-for-bit what KD gave before the option.
func TestKDZeroSegmentStartsChangesNothing(t *testing.T) {
	h, l, c := kdTestBars(80)
	for _, method := range []KDMethod{KDRecursive, KDMovingAverage} {
		base, err := KD(prices(h...), prices(l...), prices(c...), 9, KDOptions{Method: method})
		if err != nil {
			t.Fatal(err)
		}
		for _, starts := range [][]int{nil, {}, {0}} {
			got, err := KD(prices(h...), prices(l...), prices(c...), 9, KDOptions{Method: method, SegmentStarts: starts})
			if err != nil {
				t.Fatal(err)
			}
			if !sameFloats(floats(t, got.K), floats(t, base.K)) || !sameFloats(floats(t, got.D), floats(t, base.D)) {
				t.Fatalf("%v starts=%v changed the result", method, starts)
			}
		}
	}
}

func TestKDRefusesBadSegmentStarts(t *testing.T) {
	h, l, c := kdTestBars(20)
	for _, starts := range [][]int{{-1}, {20}, {5, 99}} {
		if _, err := KD(prices(h...), prices(l...), prices(c...), 5, KDOptions{SegmentStarts: starts}); err == nil {
			t.Errorf("segment starts %v should be an error", starts)
		}
	}
}
