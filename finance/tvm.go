package finance

import (
	"errors"
	"fmt"

	"github.com/TimLai666/go-decimal/decimal"
)

// PMT computes the periodic payment of an annuity given an interest
// rate per period, total number of periods, present value, future
// value, and payment timing. Result follows Excel's sign convention:
// PMT comes back with the opposite sign of (PV + FV).
//
//	rate := finance.MustNew("0.005")        // 0.5% / month
//	pv   := finance.MustNew("100000")       // borrow 100,000
//	pmt, _ := finance.PMT(rate, 360, pv, finance.Zero, finance.PaymentEnd)
//	// pmt ≈ -599.5505...   (you pay ~599.55 per month)
//
// Excel equivalent: PMT(rate, nper, pv, fv, type)
func PMT(rate decimal.Decimal, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	if err := validateNper(nper); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()
	nperD := decimal.NewFromInt64(work, int64(nper))

	if isZero(rate) {
		// PV + PMT*n + FV = 0  =>  PMT = -(PV+FV)/n
		sum := decimal.Add(work, pv, fv)
		quo, err := decimal.Div(work, neg(sum), nperD)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return o.outCtx().Normalize(quo), nil
	}

	q, err := powInt(work, onePlus(work, rate), nper)
	if err != nil {
		return decimal.Decimal{}, err
	}
	one := decimal.NewFromInt64(work, 1)
	qMinus1 := decimal.Sub(work, q, one)

	// numerator = -(pv*q + fv)
	pvq := decimal.Mul(work, pv, q)
	num := neg(decimal.Add(work, pvq, fv))

	// denominator = (1 + r*t) * (q - 1) / r
	tFactor := timingFactor(work, rate, timing)
	tQm1, err := decimal.Div(work, qMinus1, rate)
	if err != nil {
		return decimal.Decimal{}, err
	}
	den := decimal.Mul(work, tFactor, tQm1)

	pmt, err := decimal.Div(work, num, den)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(pmt), nil
}

// PV computes the present value of an annuity (or loan) given the
// periodic rate, number of periods, periodic payment, future value,
// and payment timing.
//
// Excel equivalent: PV(rate, nper, pmt, fv, type)
func PV(rate decimal.Decimal, nper int, pmt, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	if err := validateNper(nper); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()
	nperD := decimal.NewFromInt64(work, int64(nper))

	if isZero(rate) {
		// PV + PMT*n + FV = 0  =>  PV = -(FV + PMT*n)
		sum := decimal.Add(work, fv, decimal.Mul(work, pmt, nperD))
		return o.outCtx().Normalize(neg(sum)), nil
	}

	q, err := powInt(work, onePlus(work, rate), nper)
	if err != nil {
		return decimal.Decimal{}, err
	}
	one := decimal.NewFromInt64(work, 1)
	qMinus1 := decimal.Sub(work, q, one)

	// PV = -(FV + PMT*(1+r*t)*(q-1)/r) / q
	tFactor := timingFactor(work, rate, timing)
	tQm1, err := decimal.Div(work, qMinus1, rate)
	if err != nil {
		return decimal.Decimal{}, err
	}
	annuityPart := decimal.Mul(work, decimal.Mul(work, pmt, tFactor), tQm1)
	num := neg(decimal.Add(work, fv, annuityPart))
	pv, err := decimal.Div(work, num, q)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(pv), nil
}

// FV computes the future value of an annuity given the periodic rate,
// number of periods, periodic payment, present value, and payment
// timing.
//
// Excel equivalent: FV(rate, nper, pmt, pv, type)
func FV(rate decimal.Decimal, nper int, pmt, pv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	if err := validateNper(nper); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()
	nperD := decimal.NewFromInt64(work, int64(nper))

	if isZero(rate) {
		// PV + PMT*n + FV = 0  =>  FV = -(PV + PMT*n)
		sum := decimal.Add(work, pv, decimal.Mul(work, pmt, nperD))
		return o.outCtx().Normalize(neg(sum)), nil
	}

	q, err := powInt(work, onePlus(work, rate), nper)
	if err != nil {
		return decimal.Decimal{}, err
	}
	one := decimal.NewFromInt64(work, 1)
	qMinus1 := decimal.Sub(work, q, one)

	// FV = -(PV*q + PMT*(1+r*t)*(q-1)/r)
	tFactor := timingFactor(work, rate, timing)
	tQm1, err := decimal.Div(work, qMinus1, rate)
	if err != nil {
		return decimal.Decimal{}, err
	}
	pvq := decimal.Mul(work, pv, q)
	annuityPart := decimal.Mul(work, decimal.Mul(work, pmt, tFactor), tQm1)
	fv2 := neg(decimal.Add(work, pvq, annuityPart))
	return o.outCtx().Normalize(fv2), nil
}

// NPER computes the number of periods required to satisfy the TVM
// equation. May return a non-integer value (the last period is then a
// partial period); round up if you need a whole-period count.
//
// Excel equivalent: NPER(rate, pmt, pv, fv, type)
func NPER(rate, pmt, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()

	if isZero(rate) {
		if isZero(pmt) {
			return decimal.Decimal{}, errors.New("NPER undefined: rate=0 and pmt=0")
		}
		// n = -(pv + fv) / pmt
		num := neg(decimal.Add(work, pv, fv))
		n, err := decimal.Div(work, num, pmt)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return o.outCtx().Normalize(n), nil
	}

	// Solve q = (c - fv) / (pv + c), n = ln(q)/ln(1+r), where
	// c = pmt * (1 + r*t) / r.
	tFactor := timingFactor(work, rate, timing)
	c, err := decimal.Div(work, decimal.Mul(work, pmt, tFactor), rate)
	if err != nil {
		return decimal.Decimal{}, err
	}
	num := decimal.Sub(work, c, fv)
	den := decimal.Add(work, pv, c)
	if isZero(den) {
		return decimal.Decimal{}, errors.New("NPER undefined: pv + pmt*(1+r*t)/r == 0")
	}
	q, err := decimal.Div(work, num, den)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if decimal.Cmp(q, Zero) <= 0 {
		return decimal.Decimal{}, fmt.Errorf("NPER undefined: argument to log is non-positive (q=%s)", q.String())
	}

	logQ, err := decimal.Log(work, q)
	if err != nil {
		return decimal.Decimal{}, err
	}
	logBase, err := decimal.Log(work, onePlus(work, rate))
	if err != nil {
		return decimal.Decimal{}, err
	}
	n, err := decimal.Div(work, logQ, logBase)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(n), nil
}

// RATE solves for the periodic interest rate that satisfies the TVM
// equation, using Newton's method seeded at 10% (Excel's default
// guess). Returns an error if the iteration fails to converge.
//
// Excel equivalent: RATE(nper, pmt, pv, fv, type, [guess])
//
// Pass an initial guess via guess. When guess is omitted, 0.1 is used.
func RATE(nper int, pmt, pv, fv decimal.Decimal, timing PaymentTiming, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if err := validateNper(nper); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()

	r := guess
	if isZero(r) {
		// A guess of exactly 0 makes the TVM equation degenerate; bump
		// it to Excel's default starting point.
		r = decimal.MustParse(work, "0.1")
	}

	// Tolerance: 10^-(Scale + 8). Tighter than caller's output Scale so
	// rounding to Scale doesn't move the answer.
	tol := tenToMinus(work, o.Scale+8)
	// Step size for the symmetric finite-difference derivative.
	h := tenToMinus(work, o.Scale+10)

	maxIter := 60
	for range maxIter {
		f, err := tvmResidual(work, r, nper, pmt, pv, fv, timing)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if absCmp(f, tol) <= 0 {
			return o.outCtx().Normalize(r), nil
		}

		fp, err := tvmResidual(work, decimal.Add(work, r, h), nper, pmt, pv, fv, timing)
		if err != nil {
			return decimal.Decimal{}, err
		}
		fm, err := tvmResidual(work, decimal.Sub(work, r, h), nper, pmt, pv, fv, timing)
		if err != nil {
			return decimal.Decimal{}, err
		}
		two := decimal.NewFromInt64(work, 2)
		df, err := decimal.Div(work, decimal.Sub(work, fp, fm), decimal.Mul(work, two, h))
		if err != nil {
			return decimal.Decimal{}, err
		}
		if isZero(df) {
			return decimal.Decimal{}, errors.New("RATE failed: derivative vanished during Newton iteration")
		}

		delta, err := decimal.Div(work, f, df)
		if err != nil {
			return decimal.Decimal{}, err
		}
		r = decimal.Sub(work, r, delta)

		if absCmp(delta, tol) <= 0 {
			return o.outCtx().Normalize(r), nil
		}
	}
	return decimal.Decimal{}, fmt.Errorf("RATE did not converge in %d iterations", maxIter)
}

// tvmResidual returns the value of the TVM equation
//
//	pv*(1+r)^n + pmt*(1+r*t)*((1+r)^n - 1)/r + fv
//
// (which RATE drives to zero). For r==0 it returns the degenerate form
// pv + pmt*n + fv.
func tvmResidual(ctx decimal.Context, r decimal.Decimal, nper int, pmt, pv, fv decimal.Decimal, timing PaymentTiming) (decimal.Decimal, error) {
	if isZero(r) {
		nperD := decimal.NewFromInt64(ctx, int64(nper))
		return decimal.Add(ctx, decimal.Add(ctx, pv, decimal.Mul(ctx, pmt, nperD)), fv), nil
	}
	q, err := powInt(ctx, onePlus(ctx, r), nper)
	if err != nil {
		return decimal.Decimal{}, err
	}
	one := decimal.NewFromInt64(ctx, 1)
	qMinus1 := decimal.Sub(ctx, q, one)
	tFactor := timingFactor(ctx, r, timing)
	annuity, err := decimal.Div(ctx, decimal.Mul(ctx, decimal.Mul(ctx, pmt, tFactor), qMinus1), r)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return decimal.Add(ctx, decimal.Add(ctx, decimal.Mul(ctx, pv, q), annuity), fv), nil
}

// tenToMinus returns 10^-k as a Decimal at the given context.
func tenToMinus(ctx decimal.Context, k int32) decimal.Decimal {
	if k <= 0 {
		return decimal.NewFromInt64(ctx, 1)
	}
	return decimal.NewFromScaledInt(oneBigInt, k)
}

// absCmp returns the sign of |a| - |b| as -1/0/+1.
func absCmp(a, b decimal.Decimal) int {
	abs := func(d decimal.Decimal) decimal.Decimal {
		if decimal.Cmp(d, Zero) < 0 {
			return decimal.Neg(d)
		}
		return d
	}
	return decimal.Cmp(abs(a), abs(b))
}
