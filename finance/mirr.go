package finance

import (
	"errors"

	"github.com/TimLai666/go-decimal/decimal"
)

// MIRR computes the modified internal rate of return.
//
// Unlike IRR, MIRR distinguishes between the rate at which negative
// (financing) cashflows are discounted and the rate at which positive
// (reinvestment) cashflows are compounded:
//
//	MIRR = (FV(positive_cf @ reinvestRate) / -PV(negative_cf @ financeRate))^(1/n) - 1
//
// where n is the number of periods spanned (len(cashflows) - 1).
// Positive cashflows are compounded forward to t=n at reinvestRate;
// negative cashflows are discounted back to t=0 at financeRate.
//
// Excel equivalent: MIRR(values, finance_rate, reinvest_rate)
func MIRR(cashflows []decimal.Decimal, financeRate, reinvestRate decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if len(cashflows) < 2 {
		return decimal.Decimal{}, errors.New("MIRR: need at least 2 cashflows")
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	one := decimal.NewFromInt64(work, 1)
	zero := decimal.NewFromInt64(work, 0)
	n := len(cashflows) - 1

	// PV of negative cashflows at financeRate, summed at t=0.
	pvNeg := zero
	onePlusFin := decimal.Add(work, one, financeRate)
	finFactor := one
	for _, cf := range cashflows {
		if decimal.Cmp(cf, zero) < 0 {
			quo, err := decimal.Div(work, cf, finFactor)
			if err != nil {
				return decimal.Decimal{}, err
			}
			pvNeg = decimal.Add(work, pvNeg, quo)
		}
		finFactor = decimal.Mul(work, finFactor, onePlusFin)
	}

	// FV of positive cashflows at reinvestRate, summed at t=n. cf[i]
	// is at time i, so it gets compounded by (1+r)^(n-i).
	fvPos := zero
	onePlusRe := decimal.Add(work, one, reinvestRate)
	for i, cf := range cashflows {
		if decimal.Cmp(cf, zero) > 0 {
			power, err := powInt(work, onePlusRe, n-i)
			if err != nil {
				return decimal.Decimal{}, err
			}
			fvPos = decimal.Add(work, fvPos, decimal.Mul(work, cf, power))
		}
	}

	if isZero(pvNeg) {
		return decimal.Decimal{}, errors.New("MIRR: no negative cashflows; ratio is undefined")
	}
	if decimal.Cmp(fvPos, zero) <= 0 {
		return decimal.Decimal{}, errors.New("MIRR: total positive FV is non-positive; nth-root is undefined")
	}

	// MIRR = (fvPos / -pvNeg)^(1/n) - 1
	ratio, err := decimal.Div(work, fvPos, neg(pvNeg))
	if err != nil {
		return decimal.Decimal{}, err
	}
	nDec := decimal.NewFromInt64(work, int64(n))
	invN, err := decimal.Div(work, one, nDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	root, err := decimal.Pow(work, ratio, invN)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(decimal.Sub(work, root, one)), nil
}
