package quant

import (
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func randomBars(n int, seed int64) (o, h, l, c, v []float64) {
	rng := rand.New(rand.NewSource(seed))
	o, h, l, c, v = make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	prev := 100.0
	for i := 0; i < n; i++ {
		o[i] = prev * (1 + 0.01*rng.NormFloat64())
		c[i] = o[i] * (1 + 0.015*rng.NormFloat64())
		h[i] = math.Max(o[i], c[i]) * (1 + 0.005*rng.Float64())
		l[i] = math.Min(o[i], c[i]) * (1 - 0.005*rng.Float64())
		v[i] = 1000 + float64(i) // unique, so a bar can be recognised by its volume
		prev = c[i]
	}
	return
}

func shuffled(t *testing.T, o, h, l, c, v []float64, opts ...ShuffleOptions) (so, sh, sl, sc, sv []float64) {
	t.Helper()
	var opt ShuffleOptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	if v != nil {
		opt.Volume = prices(v...)
	}
	dt, err := ShuffleBars(prices(o...), prices(h...), prices(l...), prices(c...), opt)
	if err != nil {
		t.Fatal(err)
	}
	col := func(name string) []float64 {
		dl := dt.GetColByName(name)
		if dl == nil {
			return nil
		}
		return floats(t, dl)
	}
	return col("Open"), col("High"), col("Low"), col("Close"), col("Volume")
}

// Every bar of the result is a real bar rebuilt at a new price level: it
// keeps its own gap, shape and volume, and the multiset of them is unchanged.
func TestShuffleBarsKeepsTheBarsAndReordersThem(t *testing.T) {
	o, h, l, c, v := randomBars(300, 1)
	so, sh, sl, sc, sv := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 7})
	if len(sc) != 300 {
		t.Fatalf("rows: %d", len(sc))
	}
	if so[0] != o[0] || sh[0] != h[0] || sl[0] != l[0] || sc[0] != c[0] || sv[0] != v[0] {
		t.Error("the first bar must stay as it was")
	}
	if math.Abs(sc[299]/c[299]-1) > 1e-9 {
		t.Errorf("the path must end on the real last close: %v vs %v", sc[299], c[299])
	}
	// Recognise each rebuilt bar by its volume and check it has its source's ratios.
	source := map[float64]int{}
	for i := 1; i < 300; i++ {
		source[v[i]] = i
	}
	moved := 0
	for i := 1; i < 300; i++ {
		src, ok := source[sv[i]]
		if !ok {
			t.Fatalf("row %d carries a volume %v that is not a real bar's", i, sv[i])
		}
		if src != i {
			moved++
		}
		for name, pair := range map[string][2]float64{
			"gap":   {so[i] / sc[i-1], o[src] / c[src-1]},
			"high":  {sh[i] / so[i], h[src] / o[src]},
			"low":   {sl[i] / so[i], l[src] / o[src]},
			"close": {sc[i] / so[i], c[src] / o[src]},
		} {
			if math.Abs(pair[0]/pair[1]-1) > 1e-9 {
				t.Fatalf("row %d (bar %d) %s ratio %v, source %v", i, src, name, pair[0], pair[1])
			}
		}
		if sh[i] < math.Max(so[i], sc[i]) || sl[i] > math.Min(so[i], sc[i]) {
			t.Fatalf("row %d: the bar is no longer consistent: o=%v h=%v l=%v c=%v", i, so[i], sh[i], sl[i], sc[i])
		}
	}
	if moved < 250 {
		t.Errorf("only %d of 299 bars moved; this is not a shuffle", moved)
	}
	if len(source) != 299 {
		t.Fatal("test volumes are not unique")
	}
	sorted := append([]float64(nil), sv[1:]...)
	sort.Float64s(sorted)
	want := append([]float64(nil), v[1:]...)
	sort.Float64s(want)
	if !reflect.DeepEqual(sorted, want) {
		t.Error("the volumes must be the same set")
	}
}

func TestShuffleBarsSeedDecidesThePath(t *testing.T) {
	o, h, l, c, v := randomBars(200, 2)
	_, _, _, a, _ := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 5})
	_, _, _, b, _ := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 5})
	_, _, _, d, _ := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 6})
	_, _, _, z, _ := shuffled(t, o, h, l, c, v)
	_, _, _, z0, _ := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 0})
	if !reflect.DeepEqual(a, b) {
		t.Error("the same seed must give the same path")
	}
	if reflect.DeepEqual(a, d) {
		t.Error("a different seed must give a different path")
	}
	if !reflect.DeepEqual(z, z0) {
		t.Error("no options is seed 0")
	}
}

// The order of the days is what is destroyed: the lag-1 autocorrelation of an
// AR(1) return series is near zero afterwards.
func TestShuffleBarsDestroysAutocorrelation(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	n := 4000
	o, h, l, c, v := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	prev, r := 100.0, 0.0
	for i := 0; i < n; i++ {
		r = 0.7*r + 0.01*rng.NormFloat64()
		o[i], c[i] = prev, prev*math.Exp(r)
		h[i], l[i], v[i] = math.Max(o[i], c[i])*1.001, math.Min(o[i], c[i])*0.999, 1
		prev = c[i]
	}
	ac := func(c []float64) float64 {
		ret := make([]float64, len(c)-1)
		for i := range ret {
			ret[i] = math.Log(c[i+1] / c[i])
		}
		var m float64
		for _, x := range ret {
			m += x
		}
		m /= float64(len(ret))
		var num, den float64
		for i := range ret {
			den += (ret[i] - m) * (ret[i] - m)
			if i > 0 {
				num += (ret[i] - m) * (ret[i-1] - m)
			}
		}
		return num / den
	}
	_, _, _, sc, _ := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 1})
	if real, fake := ac(c), ac(sc); real < 0.5 || math.Abs(fake) > 0.08 {
		t.Errorf("autocorrelation: real %v, shuffled %v", real, fake)
	}
}

func TestShuffleBarsKeepsEachSegmentAndNoVolume(t *testing.T) {
	o, h, l, c, v := randomBars(300, 3)
	breaks := []int{100, 220}
	so, _, _, sc, sv := shuffled(t, o, h, l, c, v, ShuffleOptions{Seed: 4, SegmentStarts: breaks})
	for _, b := range []int{0, 100, 220} {
		if so[b] != o[b] || sc[b] != c[b] || sv[b] != v[b] {
			t.Errorf("segment start %d must stay as it was", b)
		}
	}
	for _, e := range []int{99, 219, 299} {
		if math.Abs(sc[e]/c[e]-1) > 1e-9 {
			t.Errorf("segment ending at %d must end on the real close: %v vs %v", e, sc[e], c[e])
		}
	}
	// A bar never leaves its segment.
	for i, vol := range sv {
		src := int(vol - 1000)
		if (src >= 100) != (i >= 100) || (src >= 220) != (i >= 220) {
			t.Fatalf("row %d holds bar %d from another segment", i, src)
		}
	}
	_, _, _, _, none := shuffled(t, o, h, l, c, nil)
	if none != nil {
		t.Error("no volume in, no Volume column out")
	}
}

func TestShuffleBarsRefusesBadInput(t *testing.T) {
	o, h, l, c, v := randomBars(10, 4)
	p := func(x []float64) insyra.IDataList { return prices(x...) }
	bad := append([]float64(nil), o...)
	bad[3] = 0
	if _, err := ShuffleBars(p(bad), p(h), p(l), p(c), ShuffleOptions{Volume: p(v)}); err == nil {
		t.Error("a zero open should be an error")
	}
	if _, err := ShuffleBars(p(o), p(h), p(l[:9]), p(c), ShuffleOptions{Volume: p(v)}); err == nil {
		t.Error("different lengths should be an error")
	}
	if _, err := ShuffleBars(p(o), p(l), p(h), p(c), ShuffleOptions{Volume: p(v)}); err == nil {
		t.Error("a high below the low should be an error")
	}
	neg := append([]float64(nil), v...)
	neg[2] = -1
	if _, err := ShuffleBars(p(o), p(h), p(l), p(c), ShuffleOptions{Volume: p(neg)}); err == nil {
		t.Error("negative volume should be an error")
	}
	if _, err := ShuffleBars(p(o), p(h), p(l), p(c), ShuffleOptions{Volume: p(v[:9])}); err == nil {
		t.Error("a volume of a different length should be an error")
	}
	if _, err := ShuffleBars(p(o), p(h), p(l), p(c), ShuffleOptions{Volume: p(v), SegmentStarts: []int{10}}); err == nil {
		t.Error("a break outside the series should be an error")
	}
	if _, err := ShuffleBars(p(o), p(h), p(l), p(c), ShuffleOptions{}, ShuffleOptions{}); err == nil {
		t.Error("two option sets should be an error")
	}
	if _, err := ShuffleBars(nil, p(h), p(l), p(c)); err == nil {
		t.Error("nil open should be an error")
	}
	if dt, err := ShuffleBars(p(nil), p(nil), p(nil), p(nil)); err != nil || dt == nil {
		t.Errorf("empty input: %v, %v", dt, err)
	}
}
