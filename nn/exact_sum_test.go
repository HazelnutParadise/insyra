package nn

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// exactPow2 builds 2^n as a float32, exactly.
func exactPow2(n int) float32 { return float32(math.Ldexp(1, n)) }

// exactSumOracle is the reference the accumulator must match bit for bit. The
// product of two float32 values is exact in float64, so a 4096-bit big.Float
// accumulates the products without rounding, and its Float32 rounds the exact
// total once, to nearest with ties to even.
func exactSumOracle(pairs [][2]float32) float32 {
	sum := new(big.Float).SetPrec(4096)
	for _, p := range pairs {
		sum.Add(sum, new(big.Float).SetFloat64(float64(p[0])*float64(p[1])))
	}
	if sum.Sign() == 0 {
		return math.Float32frombits(0)
	}
	total, _ := sum.Float32() // the rounding it reports is the one under test
	return total
}

// exactSumOf accumulates pairs and rounds the total once.
func exactSumOf(pairs [][2]float32) float32 {
	var a exactAccumulator
	for _, p := range pairs {
		a.addProduct(p[0], p[1])
	}
	return a.float32()
}

// exactSumOfNormalizing accumulates pairs, normalizing after every one.
func exactSumOfNormalizing(pairs [][2]float32, every int) float32 {
	var a exactAccumulator
	for i, p := range pairs {
		a.addProduct(p[0], p[1])
		if (i+1)%every == 0 {
			a.normalize()
		}
	}
	return a.float32()
}

// exactRandomPairs draws n products of the shape EdgeSum feeds the
// accumulator: a normal value moved by a power of two, widened to float32.
func exactRandomPairs(r *rand.Rand, n int) [][2]float32 {
	pairs := make([][2]float32, n)
	for i := range pairs {
		pairs[i] = [2]float32{
			float32(math.Ldexp(r.NormFloat64(), r.Intn(121)-60)),
			float32(math.Ldexp(r.NormFloat64(), r.Intn(121)-60)),
		}
	}
	return pairs
}

func TestExactAccumulatorKnownValues(t *testing.T) {
	tests := []struct {
		name  string
		pairs [][2]float32
		want  uint32
	}{
		{
			name:  "an empty accumulator is positive zero",
			pairs: nil,
			want:  0x00000000,
		},
		{
			name:  "one product is its own rounding",
			pairs: [][2]float32{{0.25, 0.3}},
			want:  math.Float32bits(float32(0.25) * float32(0.3)),
		},
		{
			name: "a huge pair cancels and leaves the small one",
			pairs: [][2]float32{
				{float32(1e30), 1},
				{float32(-1e30), 1},
				{float32(1e-30), 1},
			},
			want: math.Float32bits(float32(1e-30)),
		},
		{
			name: "one ulp below one is half way to even",
			pairs: [][2]float32{
				{1, 1},
				{exactPow2(-24), 1},
			},
			want: 0x3f800000,
		},
		{
			name: "one and a half ulps above one is half way to even",
			pairs: [][2]float32{
				{1, 1},
				{exactPow2(-23), 1},
				{exactPow2(-24), 1},
			},
			want: 0x3f800002,
		},
		{
			name:  "the smallest subnormal times one and a half is half way to even",
			pairs: [][2]float32{{math.Float32frombits(1), 1.5}},
			want:  0x00000002,
		},
		{
			name:  "a quarter of the smallest subnormal rounds to zero",
			pairs: [][2]float32{{exactPow2(-75), exactPow2(-75)}},
			want:  0x00000000,
		},
		{
			name:  "three quarters of the smallest subnormal rounds up",
			pairs: [][2]float32{{exactPow2(-75), float32(math.Ldexp(1.5, -75))}},
			want:  0x00000001,
		},
		{
			name: "twice the largest float32 overflows",
			pairs: [][2]float32{
				{math.MaxFloat32, 1},
				{math.MaxFloat32, 1},
			},
			want: 0x7f800000,
		},
		{
			name: "the largest float32 plus half an ulp of it overflows",
			pairs: [][2]float32{
				{math.MaxFloat32, 1},
				{float32(math.Ldexp(1, 103)), 1},
			},
			want: 0x7f800000,
		},
		{
			name: "the largest float32 plus a quarter ulp of it does not",
			pairs: [][2]float32{
				{math.MaxFloat32, 1},
				{float32(math.Ldexp(1, 102)), 1},
			},
			want: 0x7f7fffff,
		},
		{
			name: "a negative total is exactly negative",
			pairs: [][2]float32{
				{-3, 1},
				{1, 1},
			},
			want: math.Float32bits(float32(-2)),
		},
		{
			name: "a total that cancels to nothing is positive zero",
			pairs: [][2]float32{
				{1, 1},
				{-1, 1},
			},
			want: 0x00000000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var a exactAccumulator
			for _, p := range tc.pairs {
				a.addProduct(p[0], p[1])
			}
			if got := math.Float32bits(a.float32()); got != tc.want {
				t.Fatalf("bits = %#08x, want %#08x", got, tc.want)
			}
		})
	}

	t.Run("reset returns the accumulator to the empty state", func(t *testing.T) {
		var a exactAccumulator
		a.addProduct(math.MaxFloat32, 2)
		a.addProduct(1, 1)
		a.reset()
		if got := math.Float32bits(a.float32()); got != 0x00000000 {
			t.Fatalf("bits after reset = %#08x, want %#08x", got, uint32(0x00000000))
		}
	})
}

func TestExactAccumulatorSpecialValues(t *testing.T) {
	nan := float32(math.NaN())
	posInf := float32(math.Inf(1))
	negInf := float32(math.Inf(-1))
	negZero := float32(math.Copysign(0, -1))

	tests := []struct {
		name  string
		pairs [][2]float32
		want  uint32
	}{
		{name: "NaN times one", pairs: [][2]float32{{nan, 1}}, want: 0x7fc00000},
		{name: "one times NaN", pairs: [][2]float32{{1, nan}}, want: 0x7fc00000},
		{name: "NaN times zero then positive infinity", pairs: [][2]float32{{nan, 0}, {posInf, 1}}, want: 0x7fc00000},
		{name: "positive infinity times zero", pairs: [][2]float32{{posInf, 0}}, want: 0x7fc00000},
		{name: "zero times negative infinity", pairs: [][2]float32{{0, negInf}}, want: 0x7fc00000},
		{name: "positive infinity times negative zero", pairs: [][2]float32{{posInf, negZero}}, want: 0x7fc00000},
		{name: "positive infinity times two", pairs: [][2]float32{{posInf, 2}}, want: 0x7f800000},
		{name: "positive infinity times negative two", pairs: [][2]float32{{posInf, -2}}, want: 0xff800000},
		{name: "negative infinity times negative two", pairs: [][2]float32{{negInf, -2}}, want: 0x7f800000},
		{name: "infinities of both signs", pairs: [][2]float32{{posInf, 1}, {negInf, 1}}, want: 0x7fc00000},
		{name: "infinity with finite products", pairs: [][2]float32{{posInf, 1}, {1e38, 10}, {-3, 1}}, want: 0x7f800000},
		{name: "negative zero times one", pairs: [][2]float32{{negZero, 1}}, want: 0x00000000},
		{name: "negative one times zero", pairs: [][2]float32{{-1, 0}}, want: 0x00000000},
		{name: "negative zero times negative zero", pairs: [][2]float32{{negZero, negZero}}, want: 0x00000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var a exactAccumulator
			for _, pair := range tc.pairs {
				a.addProduct(pair[0], pair[1])
			}
			if got := math.Float32bits(a.float32()); got != tc.want {
				t.Fatalf("bits = %#08x, want %#08x", got, tc.want)
			}
		})
	}

	t.Run("reset clears special values", func(t *testing.T) {
		var a exactAccumulator
		a.addProduct(nan, 1)
		a.reset()
		a.addProduct(2, 3)
		if got, want := math.Float32bits(a.float32()), math.Float32bits(float32(6)); got != want {
			t.Fatalf("bits = %#08x, want %#08x", got, want)
		}
	})
}

func TestExactAccumulatorMatchesOracle(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for round := 0; round < 300; round++ {
		pairs := exactRandomPairs(r, 1+r.Intn(500))
		want := math.Float32bits(exactSumOracle(pairs))
		got := math.Float32bits(exactSumOf(pairs))
		if got != want {
			t.Fatalf("round %d of %d pairs: bits = %#08x, want %#08x",
				round, len(pairs), got, want)
		}
	}
}

func TestExactAccumulatorOrderAndNormalizationDoNotMatter(t *testing.T) {
	pairs := exactRandomPairs(rand.New(rand.NewSource(7)), 2000)
	shuffled := append([][2]float32(nil), pairs...)
	rand.New(rand.NewSource(11)).Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	orderings := []struct {
		name string
		got  float32
	}{
		{"in the order given", exactSumOf(pairs)},
		{"shuffled", exactSumOf(shuffled)},
		{"normalized after every seven", exactSumOfNormalizing(pairs, 7)},
		{"normalized after every product", exactSumOfNormalizing(pairs, 1)},
	}
	want := math.Float32bits(exactSumOracle(pairs))
	for _, o := range orderings {
		if got := math.Float32bits(o.got); got != want {
			t.Fatalf("%s: bits = %#08x, want %#08x", o.name, got, want)
		}
	}
}

func TestExactAccumulatorCanceledTotalsAreExact(t *testing.T) {
	// Every pair appears with x and with -x, so the exact total is the last
	// product alone. A float32 accumulator loses it to the pairs around it
	// long before this; the exact one must not.
	var a exactAccumulator
	r := rand.New(rand.NewSource(3))
	for _, p := range exactRandomPairs(r, 1000) {
		a.addProduct(p[0], p[1])
		a.addProduct(-p[0], p[1])
	}
	a.addProduct(exactPow2(-140), 1)

	want := exactPow2(-140)
	if got := a.float32(); got != want {
		t.Fatalf("got %v (%#08x), want %v (%#08x)", got, math.Float32bits(got), want, math.Float32bits(want))
	}
}

func exactSumWindowBitwise(d *[exactSumDigits]int64, lo, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v |= uint64(exactSumBit(d, lo+i)) << uint(i)
	}
	return v
}

func exactSumStickyBitwise(d *[exactSumDigits]int64, lo int) bool {
	for k := 0; k < lo; k++ {
		if exactSumBit(d, k) != 0 {
			return true
		}
	}
	return false
}

func TestExactSumHelpersMatchBitwiseReference(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	for round := 0; round < 10000; round++ {
		var digits [exactSumDigits]int64
		for i := range digits {
			if r.Intn(3) != 0 {
				digits[i] = int64(r.Uint32())
			}
		}
		lo := r.Intn(exactSumDigits * 32)
		n := 1 + r.Intn(24)

		if got, want := exactSumWindow(&digits, lo, n), exactSumWindowBitwise(&digits, lo, n); got != want {
			t.Fatalf("round %d: exactSumWindow(%d, %d) = %#x, want %#x", round, lo, n, got, want)
		}
		if got, want := exactSumSticky(&digits, lo), exactSumStickyBitwise(&digits, lo); got != want {
			t.Fatalf("round %d: exactSumSticky(%d) = %t, want %t", round, lo, got, want)
		}
	}
}

// float64FastThenExact is the choice EdgeSum makes for one output: the fast
// path's answer when it can prove the rounding, the exact accumulator's
// otherwise.
func float64FastThenExact(pairs [][2]float32) float32 {
	var fast float64ProductSum
	for _, p := range pairs {
		fast.add(p[0], p[1])
	}
	if out, ok := fast.result(); ok {
		return out
	}
	return exactSumOf(pairs)
}

func TestFloat64ProductSumAgreesWithExact(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	accepted, rounds := 0, 0
	for round := 0; round < 2000; round++ {
		pairs := exactRandomPairs(r, 1+r.Intn(300))
		if round%4 == 0 {
			// Every group also arrives as (-x, y), so the float64 total loses
			// each one to the pair beside it and only the small leftovers
			// survive, while abs keeps every magnitude. The bound then dwarfs
			// the total and the fast path has to refuse.
			canceling := make([][2]float32, 0, 3*len(pairs))
			for _, p := range pairs {
				canceling = append(canceling, p, [2]float32{-p[0], p[1]}, [2]float32{exactPow2(-140), 1})
			}
			pairs = canceling
		}
		rounds++

		var fast float64ProductSum
		fast.reset()
		for _, p := range pairs {
			fast.add(p[0], p[1])
		}
		out, ok := fast.result()
		if ok {
			accepted++
			if got, want := math.Float32bits(out), math.Float32bits(exactSumOf(pairs)); got != want {
				t.Fatalf("round %d of %d pairs: fast path bits = %#08x, exact bits = %#08x", round, len(pairs), got, want)
			}
		}

		if got, want := math.Float32bits(float64FastThenExact(pairs)), math.Float32bits(exactSumOracle(pairs)); got != want {
			t.Fatalf("round %d of %d pairs: fast-then-exact bits = %#08x, oracle bits = %#08x", round, len(pairs), got, want)
		}
	}
	t.Logf("fast path proved the rounding in %d of %d rounds (%.1f%%)", accepted, rounds, 100*float64(accepted)/float64(rounds))
}

func TestFloat64ProductSumRefusesNearMidpoints(t *testing.T) {
	tests := []struct {
		name  string
		pairs [][2]float32
	}{
		{
			name: "a float64 total sitting exactly on a midpoint",
			pairs: [][2]float32{
				{1, 1},
				{exactPow2(-24), 1},
				{exactPow2(-80), 1},
			},
		},
		{
			name: "the same total with the tiny term below it",
			pairs: [][2]float32{
				{1, 1},
				{exactPow2(-24), 1},
				{-exactPow2(-80), 1},
			},
		},
		{
			name: "a total that cancels to nothing",
			pairs: [][2]float32{
				{1, 1},
				{-1, 1},
			},
		},
		{
			name: "the largest float32 plus a product too small to move it",
			pairs: [][2]float32{
				{math.MaxFloat32, 1},
				{1, 1},
			},
		},
		{
			name: "a huge pair cancelling around a small one",
			pairs: [][2]float32{
				{float32(1e30), 1},
				{float32(-1e30), 1},
				{float32(1e-30), 1},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fast float64ProductSum
			fast.reset()
			for _, p := range tc.pairs {
				fast.add(p[0], p[1])
			}
			if out, ok := fast.result(); ok {
				t.Fatalf("fast path accepted %#08x, want a refusal", math.Float32bits(out))
			}
			if got, want := math.Float32bits(float64FastThenExact(tc.pairs)), math.Float32bits(exactSumOracle(tc.pairs)); got != want {
				t.Fatalf("bits = %#08x, want %#08x", got, want)
			}
		})
	}
}

func TestFloat64ProductSumSingleProduct(t *testing.T) {
	tests := []struct {
		name string
		pair [2]float32
		want uint32
	}{
		{
			name: "a quarter times three tenths",
			pair: [2]float32{0.25, 0.3},
			want: math.Float32bits(float32(0.25) * float32(0.3)),
		},
		{
			name: "negative zero times one is positive zero",
			pair: [2]float32{float32(math.Copysign(0, -1)), 1},
			want: 0x00000000,
		},
		{
			name: "the largest float32 times two overflows",
			pair: [2]float32{math.MaxFloat32, 2},
			want: 0x7f800000,
		},
		{
			name: "the smallest subnormal times a half ties to even",
			pair: [2]float32{math.Float32frombits(1), 0.5},
			want: 0x00000000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fast float64ProductSum
			fast.reset()
			fast.add(tc.pair[0], tc.pair[1])
			out, ok := fast.result()
			if !ok {
				t.Fatal("fast path refused a single product, want ok")
			}
			if got := math.Float32bits(out); got != tc.want {
				t.Fatalf("bits = %#08x, want %#08x", got, tc.want)
			}
			if got, want := math.Float32bits(exactSumOf([][2]float32{tc.pair})), tc.want; got != want {
				t.Fatalf("exact accumulator bits = %#08x, want %#08x", got, want)
			}
		})
	}
}

var exactAccumulatorFloat32BenchmarkSink float32
var exactAccumulatorAddProductBenchmarkSink int

func BenchmarkExactAccumulatorFloat32(b *testing.B) {
	var a exactAccumulator
	for i := 0; i < 10; i++ {
		a.addProduct(0.37, float32(i)+0.5)
	}
	b.ResetTimer()

	var sum float32
	for i := 0; i < b.N; i++ {
		sum += a.float32()
	}
	exactAccumulatorFloat32BenchmarkSink = sum
}

func BenchmarkExactAccumulatorAddProduct(b *testing.B) {
	var a exactAccumulator
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.addProduct(0.37, 1.25)
	}
	exactAccumulatorAddProductBenchmarkSink = a.pending
}
