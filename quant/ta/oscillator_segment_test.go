package ta

import (
	"math"
	"testing"
)

func weightedCloses(h, l, c []float64) []float64 {
	out := make([]float64, len(c))
	for i := range c {
		out[i] = (h[i] + l[i] + 2*c[i]) / 4
	}
	return out
}

// Each CR segment is CR on its own, for every midpoint, and a segment shows NaN
// until it has n rows of its own plus the one that gives the first midpoint.
func TestCRSegmentStartsEqualSeparateSegments(t *testing.T) {
	h, l, c := kdTestBars(70)
	op := weightedCloses(h, l, c)
	for _, mid := range []CRMidpoint{CRMidHighLow, CRMidWeightedClose, CRMidTypical, CRMidOHLC} {
		for _, n := range []int{1, 4, 9} {
			for _, starts := range [][]int{{20}, {15, 40}, {10, 12, 13, 50}, {68}} {
				got, err := CR(prices(h...), prices(l...), prices(c...), n, CROptions{Midpoint: mid, Open: prices(op...), SegmentStarts: starts})
				if err != nil {
					t.Fatal(err)
				}
				var want []float64
				for _, b := range segmentBounds(len(c), starts) {
					seg, err := CR(prices(h[b[0]:b[1]]...), prices(l[b[0]:b[1]]...), prices(c[b[0]:b[1]]...), n, CROptions{Midpoint: mid, Open: prices(op[b[0]:b[1]]...)})
					if err != nil {
						t.Fatal(err)
					}
					want = append(want, floats(t, seg)...)
				}
				if !sameFloats(floats(t, got), want) {
					t.Fatalf("%v n=%d starts=%v: segmented CR differs from separate segments", mid, n, starts)
				}
			}
		}
	}
}

// Without SegmentStarts, or with only row 0, CR is bit-identical to before.
func TestCRZeroSegmentStartsChangesNothing(t *testing.T) {
	h, l, c := kdTestBars(50)
	base, err := CR(prices(h...), prices(l...), prices(c...), 6)
	if err != nil {
		t.Fatal(err)
	}
	for _, starts := range [][]int{nil, {}, {0}} {
		got, err := CR(prices(h...), prices(l...), prices(c...), 6, CROptions{SegmentStarts: starts})
		if err != nil {
			t.Fatal(err)
		}
		if !sameFloats(floats(t, got), floats(t, base)) {
			t.Fatalf("starts %v changed CR", starts)
		}
	}
}

func TestCRRefusesASegmentStartOutsideTheSeries(t *testing.T) {
	h, l, c := kdTestBars(10)
	for _, s := range []int{-1, 10} {
		if _, err := CR(prices(h...), prices(l...), prices(c...), 3, CROptions{SegmentStarts: []int{s}}); err == nil {
			t.Errorf("segment start %d: want an error", s)
		}
	}
}

func stochRSIPrices(n int) []float64 {
	_, _, c := kdTestBars(n)
	return c
}

// Each StochRSI segment is StochRSI on its own, inner RSI included, for both
// smoothing methods.
func TestStochRSISegmentStartsEqualSeparateSegments(t *testing.T) {
	p := stochRSIPrices(90)
	for _, method := range []KDMethod{KDRecursive, KDMovingAverage} {
		for _, ns := range [][2]int{{3, 3}, {5, 4}, {1, 1}} {
			for _, starts := range [][]int{{30}, {20, 55}, {10, 12, 13, 70}, {88}} {
				o := KDOptions{Method: method, SegmentStarts: starts}
				got, err := StochRSI(prices(p...), ns[0], ns[1], o)
				if err != nil {
					t.Fatal(err)
				}
				var wantK, wantD []float64
				for _, b := range segmentBounds(len(p), starts) {
					seg, err := StochRSI(prices(p[b[0]:b[1]]...), ns[0], ns[1], KDOptions{Method: method})
					if err != nil {
						t.Fatal(err)
					}
					wantK, wantD = append(wantK, floats(t, seg.K)...), append(wantD, floats(t, seg.D)...)
				}
				if !sameFloats(floats(t, got.K), wantK) || !sameFloats(floats(t, got.D), wantD) {
					t.Fatalf("%v n=%d stochN=%d starts=%v: segmented StochRSI differs from separate segments", method, ns[0], ns[1], starts)
				}
			}
		}
	}
}

// Nothing is carried across a segment start: rows before it do not change when
// the prices after it do.
func TestStochRSICarriesNothingAcrossASegmentStart(t *testing.T) {
	p := stochRSIPrices(60)
	q := append([]float64(nil), p...)
	for i := 30; i < len(q); i++ {
		q[i] += 500
	}
	a, err := StochRSI(prices(p...), 5, 5, KDOptions{SegmentStarts: []int{30}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := StochRSI(prices(q...), 5, 5, KDOptions{SegmentStarts: []int{30}})
	if err != nil {
		t.Fatal(err)
	}
	if !sameFloats(floats(t, a.K)[:30], floats(t, b.K)[:30]) || !sameFloats(floats(t, a.D)[:30], floats(t, b.D)[:30]) {
		t.Fatal("rows before the start changed when the later segment shifted")
	}
	// A level shift does not change a segment's oscillator, since the RSI sees
	// only changes inside the segment.
	ak, bk := floats(t, a.K), floats(t, b.K)
	for i := 30; i < 60; i++ {
		if math.IsNaN(ak[i]) != math.IsNaN(bk[i]) || (!math.IsNaN(ak[i]) && math.Abs(ak[i]-bk[i]) > 1e-6) {
			t.Fatalf("row %d: %v vs %v", i, ak[i], bk[i])
		}
	}
}

func TestStochRSIZeroSegmentStartsChangesNothing(t *testing.T) {
	p := stochRSIPrices(60)
	base, err := StochRSI(prices(p...), 5, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, starts := range [][]int{nil, {0}} {
		got, err := StochRSI(prices(p...), 5, 4, KDOptions{SegmentStarts: starts})
		if err != nil {
			t.Fatal(err)
		}
		if !sameFloats(floats(t, got.K), floats(t, base.K)) || !sameFloats(floats(t, got.D), floats(t, base.D)) {
			t.Fatalf("starts %v changed StochRSI", starts)
		}
	}
}

func TestStochRSIRefusesASegmentStartOutsideTheSeries(t *testing.T) {
	p := stochRSIPrices(10)
	for _, s := range []int{-1, 10} {
		if _, err := StochRSI(prices(p...), 3, 3, KDOptions{SegmentStarts: []int{s}}); err == nil {
			t.Errorf("segment start %d: want an error", s)
		}
	}
}

// A flat stretch at the start leaves the RSI undefined; the smoothing must begin
// when the stochastic value does instead of staying NaN for good.
func TestStochRSIRecoversAfterUndefinedRSI(t *testing.T) {
	p := prices(10, 10, 10, 10, 11, 12, 11, 13, 12, 14)
	got, err := StochRSI(p, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	k, d := floats(t, got.K), floats(t, got.D)
	// The raw stochastic value is 50 on row 5, 0 on row 6 and 100 on row 7;
	// the recursion starts from 50 with both periods 3.
	k6 := 50.0 * 2 / 3
	d6 := 50.0*2/3 + k6/3
	k7 := k6*2/3 + 100.0/3
	d7 := d6*2/3 + k7/3
	for i := 0; i < 5; i++ {
		if !math.IsNaN(k[i]) || !math.IsNaN(d[i]) {
			t.Fatalf("row %d should be NaN, got K %v D %v", i, k[i], d[i])
		}
	}
	for i, w := range [][2]float64{{50, 50}, {k6, d6}, {k7, d7}} {
		if math.Abs(k[5+i]-w[0]) > 1e-9 || math.Abs(d[5+i]-w[1]) > 1e-9 {
			t.Errorf("row %d: K %v want %v, D %v want %v", 5+i, k[5+i], w[0], d[5+i], w[1])
		}
	}
	for i := 5; i < len(k); i++ {
		if math.IsNaN(k[i]) || math.IsNaN(d[i]) {
			t.Fatalf("row %d is still NaN", i)
		}
	}
}

// A window with no true range adds nothing to the ultimate oscillator while the
// weights still total 7, as in TA-Lib, instead of making the whole value NaN.
func TestUltimateOscillatorZeroRangeWindowAddsNothing(t *testing.T) {
	c := []float64{10, 11, 12, 11, 13, 14, 12, 13, 13, 13, 13, 13}
	p := prices(c...)
	got, err := UltimateOscillator(p, p, p, 2, 4, 6)
	if err != nil {
		t.Fatal(err)
	}
	v := floats(t, got)
	avg := func(i, n int) float64 {
		var b, r float64
		for j := i - n + 1; j <= i; j++ {
			b += math.Max(c[j]-c[j-1], 0)
			r += math.Abs(c[j] - c[j-1])
		}
		if r == 0 {
			return 0
		}
		return b / r
	}
	// row 10: the short window (rows 9, 10) is flat, the others are not.
	// row 11: the medium window (rows 8 to 11) is flat too.
	for _, i := range []int{10, 11} {
		want := 100 * (4*avg(i, 2) + 2*avg(i, 4) + avg(i, 6)) / 7
		if math.IsNaN(v[i]) || math.Abs(v[i]-want) > 1e-9 {
			t.Errorf("row %d: %v, want %v", i, v[i], want)
		}
	}
	flat := prices(5, 5, 5, 5, 5, 5, 5, 5, 5, 5)
	z, err := UltimateOscillator(flat, flat, flat, 2, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	for i, x := range floats(t, z)[4:] {
		if x != 0 {
			t.Errorf("flat series row %d: %v, want 0", i+4, x)
		}
	}
}

// A window whose typical price does not move is NaN even when the price is a
// decimal whose running mean is not exactly the price.
func TestCCIFlatDecimalWindowIsNaN(t *testing.T) {
	flat := make([]float64, 30)
	for i := range flat {
		flat[i] = 100.1
	}
	p := prices(flat...)
	got, err := CCI(p, p, p, 20)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range floats(t, got) {
		if !math.IsNaN(v) {
			t.Fatalf("row %d of a flat series is %v, want NaN", i, v)
		}
	}
}
