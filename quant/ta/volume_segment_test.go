package ta

import (
	"math"
	"testing"
)

func volumeTestBars(n int) (h, l, v []float64) {
	h, l, v = make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range h {
		mid := 50 + 8*math.Sin(float64(i)/5) + float64(i%7)
		h[i] = mid + 1 + float64(i%3)
		l[i] = mid - 1 - float64(i%4)
		v[i] = 1e8 * (1 + float64((i*7)%11))
	}
	return
}

// Each EaseOfMovement segment is EaseOfMovement on its own, a segment shows NaN
// until it has n rows of its own, and an option left at zero changes nothing.
func TestEaseOfMovementSegmentStartsEqualSeparateSegments(t *testing.T) {
	h, l, v := volumeTestBars(70)
	for _, n := range []int{1, 3, 9} {
		for _, starts := range [][]int{{20}, {15, 40}, {10, 12, 13, 50}, {68}, {40, 15, 40, 0}} {
			got, err := EaseOfMovement(prices(h...), prices(l...), prices(v...), n, EaseOfMovementOptions{SegmentStarts: starts})
			if err != nil {
				t.Fatal(err)
			}
			clean, err := cleanSegmentStarts(starts, len(h))
			if err != nil {
				t.Fatal(err)
			}
			var want []float64
			for _, b := range segmentBounds(len(h), clean) {
				seg, err := EaseOfMovement(prices(h[b[0]:b[1]]...), prices(l[b[0]:b[1]]...), prices(v[b[0]:b[1]]...), n)
				if err != nil {
					t.Fatal(err)
				}
				want = append(want, floats(t, seg)...)
			}
			if !sameFloats(floats(t, got), want) {
				t.Fatalf("n=%d starts=%v: segmented EaseOfMovement differs from separate segments", n, starts)
			}
		}
	}
	base, _ := EaseOfMovement(prices(h...), prices(l...), prices(v...), 5)
	for _, starts := range [][]int{nil, {}, {0}} {
		got, err := EaseOfMovement(prices(h...), prices(l...), prices(v...), 5, EaseOfMovementOptions{SegmentStarts: starts})
		if err != nil || !sameFloats(floats(t, got), floats(t, base)) {
			t.Fatalf("starts %v changed EaseOfMovement: %v", starts, err)
		}
	}
	// SegmentStarts alone must not turn the default scale into zero.
	a, _ := EaseOfMovement(prices(h...), prices(l...), prices(v...), 5, EaseOfMovementOptions{SegmentStarts: []int{30}})
	b, _ := EaseOfMovement(prices(h...), prices(l...), prices(v...), 5, EaseOfMovementOptions{Scale: 1e8, SegmentStarts: []int{30}})
	if !sameFloats(floats(t, a), floats(t, b)) {
		t.Fatal("SegmentStarts without Scale did not keep the default scale")
	}
	if _, err := EaseOfMovement(prices(h...), prices(l...), prices(v...), 5, EaseOfMovementOptions{SegmentStarts: []int{70}}); err == nil {
		t.Error("a start outside the rows must be an error")
	}
}

// Each VolumeProfile segment is VolumeProfile on its own: the window restarts
// at a start and the segment shows NaN until its window is full.
func TestVolumeProfileSegmentStartsEqualSeparateSegments(t *testing.T) {
	h, l, v := volumeTestBars(70)
	for _, n := range []int{1, 4, 9} {
		for _, starts := range [][]int{{20}, {15, 40}, {10, 12, 13, 50}, {68}, {40, 15, 40, 0}} {
			o := VolumeProfileOptions{Bins: 10, ValueArea: 0.6, SegmentStarts: starts}
			got, err := VolumeProfile(prices(h...), prices(l...), prices(v...), n, o)
			if err != nil {
				t.Fatal(err)
			}
			clean, _ := cleanSegmentStarts(starts, len(h))
			var poc, vah, val []float64
			for _, b := range segmentBounds(len(h), clean) {
				seg, err := VolumeProfile(prices(h[b[0]:b[1]]...), prices(l[b[0]:b[1]]...), prices(v[b[0]:b[1]]...), n, VolumeProfileOptions{Bins: 10, ValueArea: 0.6})
				if err != nil {
					t.Fatal(err)
				}
				poc = append(poc, floats(t, seg.POC)...)
				vah = append(vah, floats(t, seg.ValueAreaHigh)...)
				val = append(val, floats(t, seg.ValueAreaLow)...)
			}
			if !sameFloats(floats(t, got.POC), poc) || !sameFloats(floats(t, got.ValueAreaHigh), vah) || !sameFloats(floats(t, got.ValueAreaLow), val) {
				t.Fatalf("n=%d starts=%v: segmented VolumeProfile differs from separate segments", n, starts)
			}
		}
	}
	base, _ := VolumeProfile(prices(h...), prices(l...), prices(v...), 6)
	for _, starts := range [][]int{nil, {}, {0}} {
		got, err := VolumeProfile(prices(h...), prices(l...), prices(v...), 6, VolumeProfileOptions{SegmentStarts: starts})
		if err != nil || !sameFloats(floats(t, got.POC), floats(t, base.POC)) || !sameFloats(floats(t, got.ValueAreaLow), floats(t, base.ValueAreaLow)) {
			t.Fatalf("starts %v changed VolumeProfile: %v", starts, err)
		}
	}
	if _, err := VolumeProfile(prices(h...), prices(l...), prices(v...), 6, VolumeProfileOptions{SegmentStarts: []int{-1}}); err == nil {
		t.Error("a negative start must be an error")
	}
	if _, err := VolumeProfile(prices(h...), prices(l...), prices(v[:5]...), 3); err == nil {
		t.Error("volume of another length must be an error")
	}
	if _, err := VolumeProfile(prices(5, 4), prices(6, 3), prices(1, 1), 2); err == nil {
		t.Error("a high below its low must be an error")
	}
}

// A bar covering the whole range spreads its volume evenly, so every bin ties
// and the lowest wins, even when the prices are decimals whose edges round.
func TestVolumeProfileUniformDecimalProfileTiesToLowestBin(t *testing.T) {
	vp, err := VolumeProfile(prices(10.1), prices(10), prices(100), 1)
	if err != nil {
		t.Fatal(err)
	}
	want := 10 + 0.1/24/2
	if got := floats(t, vp.POC)[0]; math.Abs(got-want) > 1e-12 {
		t.Fatalf("POC %v, want the middle of the lowest bin %v", got, want)
	}
}
