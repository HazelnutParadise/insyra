package wgpu

import (
	"context"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// log1pOracle is log(1+x) by the Mercator series on a reduced argument.
func log1pOracle(x float32) float32 {
	prec := uint(oraclePrec)
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	v := new(big.Float).SetPrec(prec).SetFloat64(float64(x))
	v.Add(v, one)
	m := new(big.Float).SetPrec(prec)
	e := v.MantExp(m)
	if m.Cmp(new(big.Float).SetPrec(prec).SetFloat64(0.75)) < 0 {
		m.Mul(m, new(big.Float).SetPrec(prec).SetInt64(2))
		e--
	}
	y := new(big.Float).SetPrec(prec).Sub(m, one)
	sum := new(big.Float).SetPrec(prec)
	pow := new(big.Float).SetPrec(prec).Set(y)
	for n := int64(1); n < 4000; n++ {
		term := new(big.Float).SetPrec(prec).Quo(pow, new(big.Float).SetPrec(prec).SetInt64(n))
		if n%2 == 0 {
			term.Neg(term)
		}
		next := new(big.Float).SetPrec(prec).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum = next
		pow.Mul(pow, y)
	}
	if oracleLn2Cache == nil {
		oracleLn2Cache = oracleAtanh2(new(big.Float).SetPrec(oraclePrec).SetInt64(2))
	}
	sum.Add(sum, new(big.Float).SetPrec(prec).Mul(oracleLn2Cache, new(big.Float).SetPrec(prec).SetInt64(int64(e))))
	return oracleRound(sum)
}

func expm1Oracle(x float32) float32 {
	prec := uint(oraclePrec)
	xb := bigF(x)
	term := new(big.Float).SetPrec(prec).Set(xb)
	sum := new(big.Float).SetPrec(prec).Set(xb)
	for n := int64(2); n < 6000; n++ {
		term.Mul(term, xb)
		term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(n))
		next := new(big.Float).SetPrec(prec).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum = next
	}
	return oracleRound(sum)
}

func log1pExpm1Inputs(r *rand.Rand, forExpm1 bool) []float32 {
	in := []float32{0, 0x1p-24, -0x1p-24, 0x1p-23, -0x1p-23, 1e-10, -1e-10, 0.1, -0.1, 0.29, -0.29, 0.4140625, -0.2890625, 0.41421356, -0.2928932, 0.5, -0.5, 1, 2, 10, 100, 1e7, 16777215, 16777216, 33554432, 1e10, 1e30, math.MaxFloat32, -0.9, -0.99, -0.999999, -0.99999994, 1.4142135, 0.70710677}
	if forExpm1 {
		in = append(in, 88.72, 88.7228, 88.723, -17, -17.3, -17.33, -17.4, -18, -88, 11.09, 11.0904, 12, -12, 20, -20)
	}
	for i := 0; i < 600; i++ {
		in = append(in, float32(r.NormFloat64()*3))
		in = append(in, float32(math.Ldexp(r.Float64()*2-1, -r.Intn(48))))
		in = append(in, float32(math.Ldexp(r.Float64(), r.Intn(110))))
	}
	if forExpm1 {
		for i := 0; i < 600; i++ {
			in = append(in, float32(r.Float64()*107-18))
		}
	} else {
		for i := 0; i < 600; i++ {
			in = append(in, float32(r.Float64()-1))
		}
	}
	return in
}

func TestLog1pAndExpm1RoundedMatchTheOracle(t *testing.T) {
	requireResidentDevice(t)
	r := rand.New(rand.NewSource(91))
	for _, forExpm1 := range []bool{true, false} {
		in := log1pExpm1Inputs(r, forExpm1)
		buf, err := Upload(in)
		if err != nil {
			t.Fatal(err)
		}
		var out *Buffer
		if forExpm1 {
			out, err = Expm1Rounded(buf)
		} else {
			out, err = Log1pRounded(buf)
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
			case forExpm1 && x > 89:
				want = float32(math.Inf(1))
			case forExpm1 && x < -104:
				want = -1
			case forExpm1 && x == 0, !forExpm1 && x == 0:
				want = x
			case forExpm1:
				want = expm1Oracle(x)
			case x > -1:
				want = log1pOracle(x)
			default:
				continue
			}
			if math.Float32bits(got[i]) != math.Float32bits(want) {
				if bad < 12 {
					t.Errorf("expm1=%v x=%g (%#x) got %g (%#x) want %g (%#x)", forExpm1, x, math.Float32bits(x), got[i], math.Float32bits(got[i]), want, math.Float32bits(want))
				}
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("expm1=%v: %d of %d wrong", forExpm1, bad, len(in))
		}
		buf.Release()
		out.Release()
	}
}
