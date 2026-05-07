package finance

import (
	"errors"

	"github.com/TimLai666/go-decimal/decimal"
)

// EffectiveRate converts a nominal annual rate compounded periodsPerYear
// times per year into the effective annual rate.
//
//	effective = (1 + nominal/m)^m - 1
//
// Excel equivalent: EFFECT(nominal_rate, npery)
func EffectiveRate(nominal decimal.Decimal, periodsPerYear int, opts ...Options) (decimal.Decimal, error) {
	if periodsPerYear < 1 {
		return decimal.Decimal{}, errors.New("periodsPerYear must be >= 1")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	m := decimal.NewFromInt64(work, int64(periodsPerYear))
	periodic, err := decimal.Div(work, nominal, m)
	if err != nil {
		return decimal.Decimal{}, err
	}
	pow, err := powInt(work, onePlus(work, periodic), periodsPerYear)
	if err != nil {
		return decimal.Decimal{}, err
	}
	one := decimal.NewFromInt64(work, 1)
	return o.outCtx().Normalize(decimal.Sub(work, pow, one)), nil
}

// NominalRate converts an effective annual rate into the equivalent
// nominal annual rate that, when compounded periodsPerYear times per
// year, yields the same effective annual rate.
//
//	nominal = m * ((1 + effective)^(1/m) - 1)
//
// Excel equivalent: NOMINAL(effect_rate, npery)
func NominalRate(effective decimal.Decimal, periodsPerYear int, opts ...Options) (decimal.Decimal, error) {
	if periodsPerYear < 1 {
		return decimal.Decimal{}, errors.New("periodsPerYear must be >= 1")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	m := decimal.NewFromInt64(work, int64(periodsPerYear))
	one := decimal.NewFromInt64(work, 1)
	base := decimal.Add(work, one, effective)

	// 1/m as a Decimal so Pow takes the fractional-exponent path
	// (Exp(exp * Log(base))).
	expDec, err := decimal.Div(work, one, m)
	if err != nil {
		return decimal.Decimal{}, err
	}
	root, err := decimal.Pow(work, base, expDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	nominal := decimal.Mul(work, m, decimal.Sub(work, root, one))
	return o.outCtx().Normalize(nominal), nil
}

// ContinuousFromAnnual converts an effective annual rate r into the
// continuously compounded rate that yields the same growth, ln(1+r).
func ContinuousFromAnnual(effective decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	o := resolveOpts(opts)
	work := o.workCtx()
	one := decimal.NewFromInt64(work, 1)
	base := decimal.Add(work, one, effective)
	v, err := decimal.Log(work, base)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// AnnualFromContinuous converts a continuously compounded rate ρ into
// the equivalent effective annual rate, exp(ρ) - 1.
func AnnualFromContinuous(continuous decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	o := resolveOpts(opts)
	work := o.workCtx()
	one := decimal.NewFromInt64(work, 1)
	v := decimal.Exp(work, continuous)
	return o.outCtx().Normalize(decimal.Sub(work, v, one)), nil
}
