package finance

import (
	"errors"
	"fmt"

	"github.com/TimLai666/go-decimal/decimal"
)

// NPV returns the net present value of a stream of cashflows where
// cashflows[0] is the cashflow at t=0, cashflows[1] at t=1, and so on.
//
//	NPV = Σ cashflows[i] / (1+rate)^i
//
// This convention (t=0 first) matches numpy_financial.npv and is the
// natural input for IRR. It differs from Excel's NPV, which assumes
// the first value is at t=1; see NPVExcel for that variant.
func NPV(rate decimal.Decimal, cashflows []decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if len(cashflows) == 0 {
		return decimal.Decimal{}, errors.New("NPV: cashflows is empty")
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	v, err := npvAt(work, rate, cashflows, 0)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// NPVExcel matches Excel's NPV(rate, value1, value2, ...): the first
// cashflow is treated as occurring at t=1 (not t=0). Provided for
// users porting spreadsheets verbatim.
func NPVExcel(rate decimal.Decimal, cashflows []decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if len(cashflows) == 0 {
		return decimal.Decimal{}, errors.New("NPV: cashflows is empty")
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	v, err := npvAt(work, rate, cashflows, 1)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// IRR computes the internal rate of return for a cashflow stream
// starting at t=0. Uses Newton's method seeded at guess (0.1 if guess
// is the zero Decimal, matching Excel's default).
//
// Returns an error if the iteration fails to converge — most often
// because the cashflow stream has no real-valued IRR or because the
// initial guess is far from any root. Pass a custom guess close to the
// expected rate to recover.
//
// Excel equivalent: IRR(values, [guess])
func IRR(cashflows []decimal.Decimal, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if len(cashflows) < 2 {
		return decimal.Decimal{}, errors.New("IRR: need at least 2 cashflows")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	r := guess
	if isZero(r) {
		r = decimal.MustParse(work, "0.1")
	}

	tol := tenToMinus(work, o.Scale+8)
	maxIter := 100

	for range maxIter {
		f, err := npvAt(work, r, cashflows, 0)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if absCmp(f, tol) <= 0 {
			return o.outCtx().Normalize(r), nil
		}

		// Analytic derivative:
		//   d/dr Σ cf[i]/(1+r)^i = -Σ i*cf[i]/(1+r)^(i+1)
		df, err := npvDeriv(work, r, cashflows)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if isZero(df) {
			return decimal.Decimal{}, errors.New("IRR failed: derivative vanished during Newton iteration")
		}

		delta, err := decimal.Div(work, f, df)
		if err != nil {
			return decimal.Decimal{}, err
		}
		newR := decimal.Sub(work, r, delta)

		// Guard against diverging into r ≤ -1 territory where
		// (1+r)^i is undefined for non-integer powers and becomes
		// numerically explosive even for integer powers.
		minusOne := decimal.MustParse(work, "-0.999999999")
		if decimal.Cmp(newR, minusOne) <= 0 {
			// Halve the step instead of taking the disastrous full
			// jump, then continue.
			half := decimal.MustParse(work, "0.5")
			delta = decimal.Mul(work, delta, half)
			newR = decimal.Sub(work, r, delta)
			if decimal.Cmp(newR, minusOne) <= 0 {
				return decimal.Decimal{}, errors.New("IRR failed: iterate fell below -1")
			}
		}
		r = newR

		if absCmp(delta, tol) <= 0 {
			return o.outCtx().Normalize(r), nil
		}
	}
	return decimal.Decimal{}, fmt.Errorf("IRR did not converge in %d iterations", maxIter)
}

// npvAt returns Σ cf[i] / (1+rate)^(i+offset).
func npvAt(ctx decimal.Context, rate decimal.Decimal, cashflows []decimal.Decimal, offset int) (decimal.Decimal, error) {
	one := decimal.NewFromInt64(ctx, 1)
	onePlusR := decimal.Add(ctx, one, rate)
	if decimal.Cmp(onePlusR, decimal.NewFromInt64(ctx, 0)) == 0 {
		return decimal.Decimal{}, errors.New("NPV: rate cannot be -1")
	}

	sum := decimal.NewFromInt64(ctx, 0)
	// factor accumulates 1/(1+r)^(i+offset) iteratively to avoid
	// re-running Pow at every step.
	factor, err := powInt(ctx, onePlusR, offset)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if decimal.Cmp(factor, decimal.NewFromInt64(ctx, 0)) == 0 {
		return decimal.Decimal{}, errors.New("NPV: 1+rate raised to offset is zero")
	}
	// Convert factor into a *divisor* for cf[0], then multiply by (1+r)
	// to advance to the next period — equivalent to dividing cf[i] by
	// (1+r)^(i+offset) without per-step Pow calls.
	for _, cf := range cashflows {
		quo, err := decimal.Div(ctx, cf, factor)
		if err != nil {
			return decimal.Decimal{}, err
		}
		sum = decimal.Add(ctx, sum, quo)
		factor = decimal.Mul(ctx, factor, onePlusR)
	}
	return sum, nil
}

// npvDeriv returns d/dr [Σ cf[i] / (1+r)^i] = -Σ i*cf[i] / (1+r)^(i+1)
// — the exact derivative used by IRR's Newton step.
func npvDeriv(ctx decimal.Context, rate decimal.Decimal, cashflows []decimal.Decimal) (decimal.Decimal, error) {
	one := decimal.NewFromInt64(ctx, 1)
	onePlusR := decimal.Add(ctx, one, rate)

	sum := decimal.NewFromInt64(ctx, 0)
	// factor = (1+r)^(i+1), starting at (1+r)^1.
	factor := onePlusR
	for i := 1; i < len(cashflows); i++ {
		iDec := decimal.NewFromInt64(ctx, int64(i))
		num := decimal.Mul(ctx, iDec, cashflows[i])
		quo, err := decimal.Div(ctx, num, factor)
		if err != nil {
			return decimal.Decimal{}, err
		}
		sum = decimal.Add(ctx, sum, quo)
		factor = decimal.Mul(ctx, factor, onePlusR)
	}
	return neg(sum), nil
}
