package nn

import (
	"fmt"
	"math"
	"math/big"

	"github.com/HazelnutParadise/insyra/accel"
)

// adamRoundedScalars returns the float32 constants of Adam step number step
// (counting from 1). Each is computed once, here, so the host and the device
// read the same bits: the bias corrections are 1 - beta^step evaluated in
// extended precision from the float64 betas torch uses, then rounded once.
func adamRoundedScalars(rate, weightDecay float32, step uint64, decay bool) accel.AdamScalars {
	const beta1, beta2, eps = 0.9, 0.999, 1e-8
	b1, b2 := float32(beta1), float32(beta2)
	return accel.AdamScalars{
		Beta1:         b1,
		OneMinusBeta1: float32(1) - b1,
		Beta2:         b2,
		OneMinusBeta2: float32(1) - b2,
		Bias1:         oneMinusPow(beta1, step),
		Bias2:         oneMinusPow(beta2, step),
		Epsilon:       float32(eps),
		NegRate:       -rate,
		NegDecay:      -(rate * weightDecay),
		Decay:         decay,
	}
}

// oneMinusPow returns 1 - base^n rounded once to float32.
func oneMinusPow(base float64, n uint64) float32 {
	const precision = 512
	result := new(big.Float).SetPrec(precision).SetInt64(1)
	factor := new(big.Float).SetPrec(precision).SetFloat64(base)
	for e := n; e > 0; e >>= 1 {
		if e&1 == 1 {
			result.Mul(result, factor)
		}
		factor.Mul(factor, factor)
	}
	value, _ := new(big.Float).SetPrec(precision).Sub(big.NewFloat(1).SetPrec(precision), result).Float32()
	return value
}

// sum2Rounded returns a*b + c*d, the exact value rounded once to float32.
func sum2Rounded(a, b, c, d float32) float32 {
	var fast float64ProductSum
	fast.add(a, b)
	fast.add(c, d)
	if value, ok := fast.result(); ok {
		return value
	}
	var exact exactAccumulator
	exact.addProduct(a, b)
	exact.addProduct(c, d)
	return exact.float32()
}

// adamRoundedElement applies one step to one element and returns the new
// weight, first moment and second moment. Each line is one correctly rounded
// operation or an exact two-term sum rounded once.
func adamRoundedElement(w, g, m, v float32, s accel.AdamScalars) (float32, float32, float32) {
	if s.Decay {
		w = sum2Rounded(w, 1, s.NegDecay, w)
	}
	mn := sum2Rounded(s.Beta1, m, s.OneMinusBeta1, g)
	g2 := g * g
	vn := sum2Rounded(s.Beta2, v, s.OneMinusBeta2, g2)
	mhat := mn / s.Bias1
	vhat := vn / s.Bias2
	den := sum2Rounded(float32(math.Sqrt(float64(vhat))), 1, s.Epsilon, 1)
	q := mhat / den
	return sum2Rounded(w, 1, s.NegRate, q), mn, vn
}

// adamRounded steps every parameter that has a gradient, on the host or on a
// device, in PrecisionFloat32Rounded.
func (t *Tape) adamRounded(rate, weightDecay float32, decay bool) error {
	for _, param := range t.params {
		value := param.value
		gradient := t.grads[value]
		if gradient == nil {
			continue
		}
		if err := requireFloat32Any(value, "adam parameter"); err != nil {
			return err
		}
		if (value.device != nil) != (gradient.device != nil) {
			return fmt.Errorf("adam parameter and gradient must both be on the host or on the same device")
		}
		param.adamStep++
		scalars := adamRoundedScalars(rate, weightDecay, param.adamStep, decay)
		if value.device == nil {
			if len(param.adamM) == 0 {
				param.adamM = make([]float32, len(value.data))
				param.adamV = make([]float32, len(value.data))
			}
			if len(param.adamM) != len(value.data) || len(param.adamV) != len(value.data) {
				return fmt.Errorf("adam state shape does not match parameter shape %v", value.shape)
			}
			for i := range value.data {
				value.data[i], param.adamM[i], param.adamV[i] = adamRoundedElement(value.data[i], gradient.data[i], param.adamM[i], param.adamV[i], scalars)
			}
			continue
		}
		if param.adamMDev == nil {
			var err error
			if param.adamMDev, err = accel.NewZeroDeviceBuffer(value.Device(), value.device.Len()); err != nil {
				return err
			}
			if param.adamVDev, err = accel.NewZeroDeviceBuffer(value.Device(), value.device.Len()); err != nil {
				return err
			}
		}
		if err := accel.RoundedAdam(value.device, gradient.device, param.adamMDev, param.adamVDev, scalars); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parameter) releaseState() {
	if p.adamMDev != nil {
		p.adamMDev.Release()
		p.adamVDev.Release()
		p.adamMDev, p.adamVDev = nil, nil
	}
}
