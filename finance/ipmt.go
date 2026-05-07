package finance

import (
	"errors"
	"fmt"

	"github.com/TimLai666/go-decimal/decimal"
)

// IPMT returns the interest portion of the per-th payment.
//
// Excel equivalent: IPMT(rate, per, nper, pv, fv, type)
//
// per is 1-indexed. With PaymentBegin (type=1), the first payment has
// IPMT=0 because no interest has accrued by t=0.
func IPMT(rate decimal.Decimal, per, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	if err := validatePer(per, nper); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()

	if timing == PaymentBegin && per == 1 {
		return o.outCtx().Normalize(decimal.NewFromInt64(work, 0)), nil
	}

	pmt, err := pmtInternal(work, rate, nper, pv, fv, timing)
	if err != nil {
		return decimal.Decimal{}, err
	}
	fvPrev, err := fvInternal(work, rate, per-1, pmt, pv, timing)
	if err != nil {
		return decimal.Decimal{}, err
	}

	ipmt := decimal.Mul(work, fvPrev, rate)
	if timing == PaymentBegin {
		ipmt, err = decimal.Div(work, ipmt, onePlus(work, rate))
		if err != nil {
			return decimal.Decimal{}, err
		}
	}
	return o.outCtx().Normalize(ipmt), nil
}

// PPMT returns the principal portion of the per-th payment. By
// definition PMT = IPMT + PPMT for every period, so PPMT is computed
// as PMT - IPMT in the same units.
//
// Excel equivalent: PPMT(rate, per, nper, pv, fv, type)
func PPMT(rate decimal.Decimal, per, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	if err := validatePer(per, nper); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()

	pmt, err := pmtInternal(work, rate, nper, pv, fv, timing)
	if err != nil {
		return decimal.Decimal{}, err
	}

	var ipmt decimal.Decimal
	if timing == PaymentBegin && per == 1 {
		ipmt = decimal.NewFromInt64(work, 0)
	} else {
		fvPrev, err := fvInternal(work, rate, per-1, pmt, pv, timing)
		if err != nil {
			return decimal.Decimal{}, err
		}
		ipmt = decimal.Mul(work, fvPrev, rate)
		if timing == PaymentBegin {
			ipmt, err = decimal.Div(work, ipmt, onePlus(work, rate))
			if err != nil {
				return decimal.Decimal{}, err
			}
		}
	}

	return o.outCtx().Normalize(decimal.Sub(work, pmt, ipmt)), nil
}

// CumIPMT returns the cumulative interest paid between periods
// startPeriod and endPeriod (inclusive, 1-indexed).
//
// Excel equivalent: CUMIPMT(rate, nper, pv, start_period, end_period, type)
func CumIPMT(rate decimal.Decimal, nper int, pv decimal.Decimal, startPeriod, endPeriod int, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	return cumulate(rate, nper, pv, startPeriod, endPeriod, timing, true, opts)
}

// CumPPMT returns the cumulative principal paid between periods
// startPeriod and endPeriod (inclusive, 1-indexed).
//
// Excel equivalent: CUMPRINC(rate, nper, pv, start_period, end_period, type)
func CumPPMT(rate decimal.Decimal, nper int, pv decimal.Decimal, startPeriod, endPeriod int, timing PaymentTiming, opts ...Options) (decimal.Decimal, error) {
	return cumulate(rate, nper, pv, startPeriod, endPeriod, timing, false, opts)
}

// cumulate is the shared body of CumIPMT/CumPPMT — sums interest or
// principal across [startPeriod, endPeriod].
func cumulate(rate decimal.Decimal, nper int, pv decimal.Decimal, startPeriod, endPeriod int, timing PaymentTiming, interest bool, opts []Options) (decimal.Decimal, error) {
	if err := validateNper(nper); err != nil {
		return decimal.Decimal{}, err
	}
	if startPeriod < 1 || endPeriod > nper || startPeriod > endPeriod {
		return decimal.Decimal{}, fmt.Errorf("invalid period range [%d, %d] for nper=%d", startPeriod, endPeriod, nper)
	}
	if err := validateTiming(timing); err != nil {
		return decimal.Decimal{}, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()
	zero := decimal.NewFromInt64(work, 0)

	pmt, err := pmtInternal(work, rate, nper, pv, zero, timing)
	if err != nil {
		return decimal.Decimal{}, err
	}

	total := zero
	for per := startPeriod; per <= endPeriod; per++ {
		var ipmt decimal.Decimal
		if timing == PaymentBegin && per == 1 {
			ipmt = zero
		} else {
			fvPrev, err := fvInternal(work, rate, per-1, pmt, pv, timing)
			if err != nil {
				return decimal.Decimal{}, err
			}
			ipmt = decimal.Mul(work, fvPrev, rate)
			if timing == PaymentBegin {
				ipmt, err = decimal.Div(work, ipmt, onePlus(work, rate))
				if err != nil {
					return decimal.Decimal{}, err
				}
			}
		}
		if interest {
			total = decimal.Add(work, total, ipmt)
		} else {
			total = decimal.Add(work, total, decimal.Sub(work, pmt, ipmt))
		}
	}
	return o.outCtx().Normalize(total), nil
}

func validatePer(per, nper int) error {
	if err := validateNper(nper); err != nil {
		return err
	}
	if per < 1 || per > nper {
		return errors.New("per must satisfy 1 <= per <= nper")
	}
	return nil
}

// pmtInternal and fvInternal live in tvm.go — the same TVM core that
// public PMT / FV use, so the formula is defined exactly once.
