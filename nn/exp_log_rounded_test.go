package nn

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

const expLogOraclePrec = 512

func expLogBig(x float32) *big.Float {
	return new(big.Float).SetPrec(expLogOraclePrec).SetMode(big.ToNearestEven).SetFloat64(float64(x))
}

func expLogRound(x *big.Float) float32 {
	r, _ := new(big.Float).SetPrec(expLogOraclePrec).SetMode(big.ToNearestEven).Set(x).Float32()
	return r
}

// expLogOracleExp evaluates the Taylor series at high precision.
func expLogOracleExp(x float32) float32 {
	if x > 100 {
		return float32(math.Inf(1))
	}
	if x < -110 {
		return 0 // e^-110 < 2^-150, half the smallest subnormal
	}
	p := uint(expLogOraclePrec)
	xb := expLogBig(x)
	term := new(big.Float).SetPrec(p).SetInt64(1)
	sum := new(big.Float).SetPrec(p).SetInt64(1)
	for n := int64(1); n < 2000; n++ {
		term.Mul(term, xb)
		term.Quo(term, new(big.Float).SetPrec(p).SetInt64(n))
		next := new(big.Float).SetPrec(p).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	return expLogRound(sum)
}

func expLogOracleLn2() *big.Float {
	return expLogOracleAtanh(expLogBig(2))
}

func expLogOracleAtanh(x *big.Float) *big.Float {
	p := uint(expLogOraclePrec)
	one := new(big.Float).SetPrec(p).SetInt64(1)
	z := new(big.Float).SetPrec(p).Quo(new(big.Float).SetPrec(p).Sub(x, one), new(big.Float).SetPrec(p).Add(x, one))
	z2 := new(big.Float).SetPrec(p).Mul(z, z)
	term := new(big.Float).SetPrec(p).Set(z)
	sum := new(big.Float).SetPrec(p).Set(z)
	for k := int64(1); k < 2000; k++ {
		term.Mul(term, z2)
		add := new(big.Float).SetPrec(p).Quo(term, new(big.Float).SetPrec(p).SetInt64(2*k+1))
		next := new(big.Float).SetPrec(p).Add(sum, add)
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	return new(big.Float).SetPrec(p).Mul(sum, new(big.Float).SetPrec(p).SetInt64(2))
}

// log oracle scales x to [1,2) so the atanh series converges rapidly.
func expLogOracleLog(x float32) float32 {
	value := new(big.Float).SetPrec(expLogOraclePrec).SetFloat64(float64(x))
	m := new(big.Float).SetPrec(expLogOraclePrec)
	e := value.MantExp(m)
	// MantExp returns m in [0.5,1); doubling makes it [1,2) and reduces e by one.
	m.Mul(m, new(big.Float).SetPrec(expLogOraclePrec).SetInt64(2))
	e--
	lnM := expLogOracleAtanh(m)
	ln2 := expLogOracleLn2()
	result := new(big.Float).SetPrec(expLogOraclePrec).Add(lnM, new(big.Float).SetPrec(expLogOraclePrec).Mul(ln2, new(big.Float).SetPrec(expLogOraclePrec).SetInt64(int64(e))))
	return expLogRound(result)
}

func TestExpLogOracle(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  float32
		want uint32
	}{
		{"exp(1)", expLogOracleExp(1), 0x402df854},
		{"exp(-1)", expLogOracleExp(-1), 0x3ebc5ab2},
		{"log(2)", expLogOracleLog(2), 0x3f317218},
		{"log(10)", expLogOracleLog(10), 0x40135d8e},
	} {
		if got := math.Float32bits(tc.got); got != tc.want {
			t.Errorf("%s oracle = %#08x, want %#08x", tc.name, got, tc.want)
		}
	}
}

func checkExpFloat32(t *testing.T, x float32) {
	t.Helper()
	got := expFloat32(x)
	want := expLogOracleExp(x)
	if math.IsNaN(float64(want)) {
		if !math.IsNaN(float64(got)) {
			t.Fatalf("expFloat32(%g) = %g, want NaN", x, got)
		}
		return
	}
	if math.Float32bits(got) != math.Float32bits(want) {
		t.Fatalf("expFloat32(%g) = %g (%#08x), want %g (%#08x)", x, got, math.Float32bits(got), want, math.Float32bits(want))
	}
}

func checkLogFloat32(t *testing.T, x float32) {
	t.Helper()
	got := logFloat32(x)
	want := expLogOracleLog(x)
	if math.IsNaN(float64(want)) {
		if !math.IsNaN(float64(got)) {
			t.Fatalf("logFloat32(%g) = %g, want NaN", x, got)
		}
		return
	}
	if math.Float32bits(got) != math.Float32bits(want) {
		t.Fatalf("logFloat32(%g) = %g (%#08x), want %g (%#08x)", x, got, math.Float32bits(got), want, math.Float32bits(want))
	}
}

func TestExpFloat32SpecialAndDocumentedValues(t *testing.T) {
	for _, tc := range []struct {
		x    float32
		want uint32
	}{
		{float32(math.NaN()), 0}, {float32(math.Inf(1)), 0x7f800000}, {float32(math.Inf(-1)), 0},
		{0, 0x3f800000}, {math.Float32frombits(0x33000000), 0x3f800000}, {-math.Float32frombits(0x33000000), 0x3f800000},
		{88.72284, 0x7f800000},
	} {
		got := expFloat32(tc.x)
		if math.IsNaN(float64(tc.x)) {
			if !math.IsNaN(float64(got)) {
				t.Errorf("exp(NaN) = %g", got)
			}
			continue
		}
		if math.Float32bits(got) != tc.want {
			t.Errorf("exp(%g) bits=%#08x want %#08x", tc.x, math.Float32bits(got), tc.want)
		}
	}
	for _, x := range []float32{88.72283, math.Nextafter32(88.72284, float32(math.Inf(-1))), math.Nextafter32(-103.97208, float32(math.Inf(-1))), -103.97208, -87.33655, math.Float32frombits(1)} {
		checkExpFloat32(t, x)
	}
}

func TestLogFloat32SpecialAndDocumentedValues(t *testing.T) {
	for _, tc := range []struct {
		x    float32
		want uint32
	}{
		{0, 0xff800000}, {math.Float32frombits(0x80000000), 0xff800000}, {float32(math.Inf(1)), 0x7f800000},
		{-1, 0x7fc00000}, {float32(math.Inf(-1)), 0x7fc00000}, {1, 0},
	} {
		got := logFloat32(tc.x)
		if tc.want == 0x7fc00000 {
			if !math.IsNaN(float64(got)) {
				t.Errorf("log(%g) = %g, want NaN", tc.x, got)
			}
			continue
		}
		if math.Float32bits(got) != tc.want {
			t.Errorf("log(%g) bits=%#08x want %#08x", tc.x, math.Float32bits(got), tc.want)
		}
	}
	for _, x := range []float32{math.Float32frombits(1), math.Float32frombits(2), math.Float32frombits(0x007fffff), math.Nextafter32(1, 0), math.Nextafter32(1, 2)} {
		checkLogFloat32(t, x)
	}
}

func TestExpFloat32RoundedSampling(t *testing.T) {
	rng := rand.New(rand.NewSource(20261021))
	for i := 0; i < 1500; i++ {
		x := math.Float32frombits(rng.Uint32())
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			continue
		}
		checkExpFloat32(t, x)
	}
	for i := 0; i < 1500; i++ {
		checkExpFloat32(t, float32(rng.Float64()*193-104))
	}
	for _, x := range expLogBoundaryValues() {
		checkExpFloat32(t, x)
	}
	for i := 0; i < 500; i++ {
		x := float32(math.Exp(rng.Float64() * math.Log(89)))
		if rng.Intn(2) == 0 {
			x = -x
		}
		checkExpFloat32(t, x)
	}
}

func TestLogFloat32RoundedSampling(t *testing.T) {
	rng := rand.New(rand.NewSource(20261022))
	for i := 0; i < 1500; i++ {
		x := math.Float32frombits(rng.Uint32() & 0x7fffffff)
		if x > 0 && !math.IsInf(float64(x), 0) {
			checkLogFloat32(t, x)
		}
	}
	for i := 0; i < 1500; i++ {
		checkLogFloat32(t, math.Float32frombits(uint32(rng.Intn(255))<<23|uint32(rng.Intn(1<<23))|1))
	}
	for _, x := range expLogBoundaryValues() {
		if x > 0 {
			checkLogFloat32(t, x)
		}
	}
	for i := 0; i < 500; i++ {
		e := rng.Intn(277) - 149
		mantissa := 1 + rng.Float64()
		x := float32(math.Ldexp(mantissa, e))
		if x == 0 {
			x = math.Float32frombits(1)
		}
		if math.IsInf(float64(x), 0) {
			continue
		}
		checkLogFloat32(t, x)
	}
}

func expLogBoundaryValues() []float32 {
	values := []float32{88.72283, 88.72284, -103.97208, -87.33655, 1, 2, math.Float32frombits(1), math.Float32frombits(0x7f7fffff)}
	for _, center := range []float32{88.72283, 88.72284, -103.97208, -87.33655, 1} {
		if center == 1 {
			for x, i := center, 0; i <= 100; i++ {
				values = append(values, x)
				x = math.Nextafter32(x, float32(math.Inf(-1)))
			}
			for x, i := center, 0; i <= 100; i++ {
				values = append(values, x)
				x = math.Nextafter32(x, float32(math.Inf(1)))
			}
		} else {
			for x, i := center, 0; i <= 8; i++ {
				values = append(values, x)
				x = math.Nextafter32(x, float32(math.Inf(-1)))
			}
			for x, i := center, 0; i <= 8; i++ {
				values = append(values, x)
				x = math.Nextafter32(x, float32(math.Inf(1)))
			}
		}
	}
	for e := -149; e <= 127; e++ {
		values = append(values, math.Float32frombits(uint32(e+127)<<23))
	}
	return values
}

// Exhaustive enumeration is intentionally deferred: the owner will add a
// bounded undecided-input exhaustive test once expFloat32Undecided and
// logFloat32Undecided are part of the implementation.
