package wgpu

import (
	"context"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// tanhOracle is (1 - e^(-2|x|)) / (1 + e^(-2|x|)) at 600 bits, with x's sign.
func tanhOracle(x float32) float32 {
	a := float32(math.Abs(float64(x)))
	z := new(big.Float).SetPrec(oraclePrec).Mul(bigF(a), new(big.Float).SetPrec(oraclePrec).SetInt64(-2))
	term := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	e := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	for n := int64(1); n < 3000; n++ {
		term.Mul(term, z)
		term.Quo(term, new(big.Float).SetPrec(oraclePrec).SetInt64(n))
		next := new(big.Float).SetPrec(oraclePrec).Add(e, term)
		if next.Cmp(e) == 0 {
			break
		}
		e.Set(next)
	}
	one := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	num := new(big.Float).SetPrec(oraclePrec).Sub(one, e)
	den := new(big.Float).SetPrec(oraclePrec).Add(one, e)
	y := oracleRound(new(big.Float).SetPrec(oraclePrec).Quo(num, den))
	if x < 0 {
		return -y
	}
	return y
}

func TestTanhRoundedMatchesTheOracle(t *testing.T) {
	requireResidentDevice(t)
	bits := math.Float32frombits
	in := []float32{
		0, float32(math.Copysign(0, -1)), bits(1), bits(0x80000001), bits(0x007fffff),
		bits(0x38ffffff), bits(0x39000000), bits(0xb9000000), bits(0x39000001),
		bits(0x4117ffff), bits(0xc117ffff), 9.5, -9.5, 20, -1e30,
		float32(math.Inf(1)), float32(math.Inf(-1)),
		// The inputs the host settles from its table of hard cases.
		bits(0x3ac37de2), bits(0x3cd41b91), bits(0x3eee0566), bits(0x40acb4d0),
		bits(0xbac37de2), bits(0xbcd41b91), bits(0xbeee0566), bits(0xc0acb4d0),
		0.5, -0.5, 1, -1, 2, 8, 8.5, 9, 0.001, 0.01, 0.1,
	}
	r := rand.New(rand.NewSource(37))
	for i := 0; i < 1500; i++ {
		in = append(in, float32(r.NormFloat64()*3))
		in = append(in, float32(math.Ldexp(r.Float64()*2-1, -r.Intn(14))))
		in = append(in, bits(0x39000000+uint32(r.Int63n(0x41180000-0x39000000))))
	}
	in = append(in, bits(0x7fc00000), bits(0xffc12345))
	buf, err := Upload(in)
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Release()
	out, err := TanhRounded(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Release()
	got, err := out.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	bad := 0
	for i, x := range in {
		ax := math.Float32bits(x) & 0x7fffffff
		var want float32
		switch {
		case ax > 0x7f800000:
			if math.Float32bits(got[i]) != math.Float32bits(x) {
				t.Errorf("tanh(NaN %#x) = %#x, want the input unchanged", math.Float32bits(x), math.Float32bits(got[i]))
			}
			continue
		case ax < 0x39000000:
			want = x
		case ax >= 0x41180000:
			want = float32(math.Copysign(1, float64(x)))
		default:
			want = tanhOracle(x)
		}
		if math.Float32bits(got[i]) != math.Float32bits(want) {
			if bad < 12 {
				t.Errorf("tanh(%g = %#x) = %#x, want %#x", x, math.Float32bits(x), math.Float32bits(got[i]), math.Float32bits(want))
			}
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d wrong", bad, len(in))
	}
}
