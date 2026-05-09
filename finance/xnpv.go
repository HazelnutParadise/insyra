package finance

import (
	"errors"
	"fmt"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

// XNPV returns the net present value of a stream of cashflows that
// occur on arbitrary dates. dates[0] is treated as the reference
// (t=0) and every other cashflow is discounted by
// (dates[i] - dates[0]) / 365 years, matching Excel's convention.
//
//	XNPV = Σ values[i] / (1 + rate)^((dates[i] - dates[0]) / 365)
//
// Excel equivalent: XNPV(rate, values, dates)
func XNPV(rate decimal.Decimal, values []decimal.Decimal, dates []time.Time, opts ...Options) (decimal.Decimal, error) {
	if err := validateXLengths(values, dates); err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	v, err := xnpvAt(work, rate, values, dates)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// XIRR returns the rate that drives XNPV to zero. Uses Newton's
// method seeded at guess (or 0.1 if guess is the zero Decimal).
//
// Excel equivalent: XIRR(values, dates, [guess])
func XIRR(values []decimal.Decimal, dates []time.Time, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if err := validateXLengths(values, dates); err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	r := guess
	if isZero(r) {
		r = decimal.MustParse(work, "0.1")
	}

	tol := tenToMinus(work, o.Scale+8)
	maxIter := 100
	minusOne := decimal.MustParse(work, "-0.999999999")

	for range maxIter {
		f, err := xnpvAt(work, r, values, dates)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if absCmp(f, tol) <= 0 {
			return o.outCtx().Normalize(r), nil
		}
		df, err := xnpvDeriv(work, r, values, dates)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if isZero(df) {
			return decimal.Decimal{}, errors.New("XIRR failed: derivative vanished")
		}
		delta, err := decimal.Div(work, f, df)
		if err != nil {
			return decimal.Decimal{}, err
		}
		newR := decimal.Sub(work, r, delta)
		if decimal.Cmp(newR, minusOne) <= 0 {
			// Halve the step to avoid jumping past the singularity at -1.
			half := decimal.MustParse(work, "0.5")
			delta = decimal.Mul(work, delta, half)
			newR = decimal.Sub(work, r, delta)
			if decimal.Cmp(newR, minusOne) <= 0 {
				return decimal.Decimal{}, errors.New("XIRR failed: iterate fell below -1")
			}
		}
		r = newR
		if absCmp(delta, tol) <= 0 {
			return o.outCtx().Normalize(r), nil
		}
	}
	return decimal.Decimal{}, fmt.Errorf("XIRR did not converge in %d iterations", maxIter)
}

func validateXLengths(values []decimal.Decimal, dates []time.Time) error {
	if len(values) != len(dates) {
		return fmt.Errorf("values (%d) and dates (%d) must have the same length",
			len(values), len(dates))
	}
	if len(values) < 2 {
		return errors.New("need at least 2 cashflows")
	}
	return nil
}

// xnpvAt evaluates Σ values[i] / (1+r)^((dates[i] - dates[0])/365) at
// the supplied working context.
func xnpvAt(ctx decimal.Context, rate decimal.Decimal, values []decimal.Decimal, dates []time.Time) (decimal.Decimal, error) {
	one := decimal.NewFromInt64(ctx, 1)
	onePlusR := decimal.Add(ctx, one, rate)
	if isZero(onePlusR) {
		return decimal.Decimal{}, errors.New("XNPV: rate cannot be -1")
	}
	d0 := toDateUTC(dates[0])
	year := decimal.NewFromInt64(ctx, 365)

	sum := decimal.NewFromInt64(ctx, 0)
	for i, v := range values {
		days := calendarDays(d0, dates[i])
		exp, err := decimal.Div(ctx, decimal.NewFromInt64(ctx, int64(days)), year)
		if err != nil {
			return decimal.Decimal{}, err
		}
		var factor decimal.Decimal
		if days == 0 {
			factor = one
		} else {
			factor, err = decimal.Pow(ctx, onePlusR, exp)
			if err != nil {
				return decimal.Decimal{}, err
			}
		}
		quo, err := decimal.Div(ctx, v, factor)
		if err != nil {
			return decimal.Decimal{}, err
		}
		sum = decimal.Add(ctx, sum, quo)
	}
	return sum, nil
}

// xnpvDeriv returns d/dr XNPV = Σ -t_i · v_i / (1+r)^(t_i+1) where
// t_i = (dates[i] - dates[0]) / 365.
func xnpvDeriv(ctx decimal.Context, rate decimal.Decimal, values []decimal.Decimal, dates []time.Time) (decimal.Decimal, error) {
	one := decimal.NewFromInt64(ctx, 1)
	onePlusR := decimal.Add(ctx, one, rate)
	d0 := toDateUTC(dates[0])
	year := decimal.NewFromInt64(ctx, 365)

	sum := decimal.NewFromInt64(ctx, 0)
	for i, v := range values {
		days := calendarDays(d0, dates[i])
		if days == 0 {
			continue
		}
		t, err := decimal.Div(ctx, decimal.NewFromInt64(ctx, int64(days)), year)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// (1+r)^(t+1)
		expPlus1 := decimal.Add(ctx, t, one)
		factor, err := decimal.Pow(ctx, onePlusR, expPlus1)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// term = t * v / (1+r)^(t+1)
		num := decimal.Mul(ctx, t, v)
		term, err := decimal.Div(ctx, num, factor)
		if err != nil {
			return decimal.Decimal{}, err
		}
		sum = decimal.Add(ctx, sum, term)
	}
	return decimal.Neg(sum), nil
}
