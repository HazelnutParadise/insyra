package wgpu

import (
	"context"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

const oraclePrec = 600

func bigF(x float32) *big.Float { return new(big.Float).SetPrec(oraclePrec).SetFloat64(float64(x)) }

func oracleRound(x *big.Float) float32 {
	if x.Sign() == 0 {
		return 0
	}
	f, _ := x.Float32()
	return f
}

func oracleExp(x float32) float32 {
	if x < -110 {
		return 0
	}
	xb := bigF(x)
	term := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	sum := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	for n := int64(1); n < 3000; n++ {
		term.Mul(term, xb)
		term.Quo(term, new(big.Float).SetPrec(oraclePrec).SetInt64(n))
		next := new(big.Float).SetPrec(oraclePrec).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	return oracleRound(sum)
}

func oracleAtanh2(m *big.Float) *big.Float {
	one := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	z := new(big.Float).SetPrec(oraclePrec).Quo(new(big.Float).SetPrec(oraclePrec).Sub(m, one), new(big.Float).SetPrec(oraclePrec).Add(m, one))
	z2 := new(big.Float).SetPrec(oraclePrec).Mul(z, z)
	term := new(big.Float).SetPrec(oraclePrec).Set(z)
	sum := new(big.Float).SetPrec(oraclePrec).Set(z)
	for k := int64(1); k < 3000; k++ {
		term.Mul(term, z2)
		next := new(big.Float).SetPrec(oraclePrec).Add(sum, new(big.Float).SetPrec(oraclePrec).Quo(term, new(big.Float).SetPrec(oraclePrec).SetInt64(2*k+1)))
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	return sum.Mul(sum, new(big.Float).SetPrec(oraclePrec).SetInt64(2))
}

var oracleLn2Cache *big.Float

func oracleLog(x float32) float32 {
	if oracleLn2Cache == nil {
		oracleLn2Cache = oracleAtanh2(new(big.Float).SetPrec(oraclePrec).SetInt64(2))
	}
	v := bigF(x)
	m := new(big.Float).SetPrec(oraclePrec)
	e := v.MantExp(m)
	m.Mul(m, new(big.Float).SetPrec(oraclePrec).SetInt64(2))
	e--
	r := new(big.Float).SetPrec(oraclePrec).Add(oracleAtanh2(m), new(big.Float).SetPrec(oraclePrec).Mul(oracleLn2Cache, new(big.Float).SetPrec(oraclePrec).SetInt64(int64(e))))
	return oracleRound(r)
}

func expLogInputs(r *rand.Rand, forExp bool) []float32 {
	var in []float32
	if forExp {
		in = append(in, []float32{0, 1, -1, 0.5, -0.5, 88.72283, 88.72284, 88.7229, -87.33655, -87.4, -100, -103.9, -103.97208, -103.9721, -104, 1e-8, -1e-8, 1e-5, 0x1p-25, -0x1p-25, 0x1p-24, -0x1p-24, 10, -10, 50, -50}...)
		for i := 0; i < 400; i++ {
			in = append(in, float32(r.Float64()*193-104))
		}
		for i := 0; i < 200; i++ {
			in = append(in, float32(math.Ldexp(r.Float64()*2-1, -r.Intn(30))))
		}
		return in
	}
	in = append(in, []float32{1, 2, 0.5, 10, 0x1p-149, 0x1p-126, math.MaxFloat32, 1.0000001, 0.99999994, 1.4142135, 1.4142137, 0.70710677, 3, 100, 1e-10, 1e10, 0.9, 1.1}...)
	for i := 0; i < 300; i++ {
		in = append(in, math.Float32frombits(r.Uint32()&0x7fffffff))
	}
	for i := 0; i < 200; i++ {
		in = append(in, math.Float32frombits(0x3f800000+uint32(r.Intn(2001))-1000))
	}
	for i := 0; i < 100; i++ {
		in = append(in, math.Float32frombits(uint32(r.Intn(1<<23))))
	}
	return in
}

func TestExpAndLogRoundedMatchTheOracle(t *testing.T) {
	requireResidentDevice(t)
	r := rand.New(rand.NewSource(77))
	for _, forExp := range []bool{true, false} {
		in := expLogInputs(r, forExp)
		buf, err := Upload(in)
		if err != nil {
			t.Fatal(err)
		}
		var out *Buffer
		if forExp {
			out, err = ExpRounded(buf)
		} else {
			out, err = LogRounded(buf)
		}
		if err != nil {
			t.Fatal(err)
		}
		got, err := out.Download(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		bad := 0
		for i, x := range in {
			var want float32
			switch {
			case math.IsNaN(float64(x)):
				if math.IsNaN(float64(got[i])) {
					continue
				}
				want = 0
			case math.IsInf(float64(x), 1):
				want = x
			case forExp:
				want = oracleExp(x)
			default:
				want = oracleLog(x)
			}
			if x == 0 && !forExp {
				want = float32(math.Inf(-1))
			}
			if math.Float32bits(got[i]) != math.Float32bits(want) {
				if bad < 12 {
					t.Errorf("exp=%v x=%g (%#x) got %g (%#x) want %g (%#x)", forExp, x, math.Float32bits(x), got[i], math.Float32bits(got[i]), want, math.Float32bits(want))
				}
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("exp=%v: %d of %d wrong", forExp, bad, len(in))
		}
		buf.Release()
		out.Release()
	}
}
