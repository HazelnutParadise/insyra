package finance

import (
	"errors"

	"github.com/TimLai666/go-decimal/decimal"
)

// SLN returns the straight-line depreciation per period:
//
//	SLN = (cost - salvage) / life
//
// Excel equivalent: SLN(cost, salvage, life)
func SLN(cost, salvage decimal.Decimal, life int, opts ...Options) (decimal.Decimal, error) {
	if life <= 0 {
		return decimal.Decimal{}, errors.New("life must be positive")
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	depreciable := decimal.Sub(work, cost, salvage)
	v, err := decimal.Div(work, depreciable, decimal.NewFromInt64(work, int64(life)))
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// SYD returns the depreciation in the per-th period under the
// sum-of-years-digits method:
//
//	SYD = (cost - salvage) · (life - per + 1) / (life · (life + 1) / 2)
//
// Excel equivalent: SYD(cost, salvage, life, per)
func SYD(cost, salvage decimal.Decimal, life, per int, opts ...Options) (decimal.Decimal, error) {
	if life <= 0 {
		return decimal.Decimal{}, errors.New("life must be positive")
	}
	if per < 1 || per > life {
		return decimal.Decimal{}, errors.New("per must satisfy 1 <= per <= life")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	depreciable := decimal.Sub(work, cost, salvage)
	num := decimal.Mul(work, depreciable, decimal.NewFromInt64(work, int64(life-per+1)))
	// Use life·(life+1)/2 (sum of integers 1..life) as the denominator.
	sumYears := int64(life) * int64(life+1) / 2
	v, err := decimal.Div(work, num, decimal.NewFromInt64(work, sumYears))
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// DDB returns the depreciation in the per-th period under the
// double-declining-balance method (or any declining-balance method
// determined by factor). The rate is factor/life applied to the
// remaining book value, capped so the asset never depreciates below
// salvage.
//
// factor=2 is the conventional double-declining choice (Excel's
// default). Pass finance.MustNew("1.5") for 150%-DB, etc.
//
// Excel equivalent: DDB(cost, salvage, life, period, factor)
func DDB(cost, salvage decimal.Decimal, life, per int, factor decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	if life <= 0 {
		return decimal.Decimal{}, errors.New("life must be positive")
	}
	if per < 1 || per > life {
		return decimal.Decimal{}, errors.New("per must satisfy 1 <= per <= life")
	}
	if isZero(factor) {
		return decimal.Decimal{}, errors.New("factor must be non-zero")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	rate, err := decimal.Div(work, factor, decimal.NewFromInt64(work, int64(life)))
	if err != nil {
		return decimal.Decimal{}, err
	}

	bookValue := cost
	zero := decimal.NewFromInt64(work, 0)
	var depr decimal.Decimal
	for k := 1; k <= per; k++ {
		raw := decimal.Mul(work, bookValue, rate)
		// Cap at book_value - salvage so the asset never falls below
		// salvage (and never depreciates by a negative amount).
		cap := decimal.Sub(work, bookValue, salvage)
		if decimal.Cmp(cap, zero) < 0 {
			cap = zero
		}
		if decimal.Cmp(raw, cap) > 0 {
			raw = cap
		}
		if decimal.Cmp(raw, zero) < 0 {
			raw = zero
		}
		depr = raw
		bookValue = decimal.Sub(work, bookValue, raw)
	}
	return o.outCtx().Normalize(depr), nil
}

// VDB returns the cumulative depreciation between startPeriod and
// endPeriod under the variable declining-balance method.
//
// startPeriod and endPeriod are real numbers in [0, life]; partial
// periods are pro-rated linearly within whatever depreciation method
// applies in that period. With noSwitch=false (Excel's default), the
// method automatically switches to straight-line for the remaining
// life as soon as straight-line would yield more depreciation than
// declining balance — which is what most US tax schedules require.
//
// Excel equivalent: VDB(cost, salvage, life, start_period, end_period, factor, no_switch)
func VDB(cost, salvage decimal.Decimal, life int, startPeriod, endPeriod, factor decimal.Decimal, noSwitch bool, opts ...Options) (decimal.Decimal, error) {
	if life <= 0 {
		return decimal.Decimal{}, errors.New("life must be positive")
	}
	if isZero(factor) {
		return decimal.Decimal{}, errors.New("factor must be non-zero")
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	zero := decimal.NewFromInt64(work, 0)
	lifeDec := decimal.NewFromInt64(work, int64(life))
	if decimal.Cmp(startPeriod, zero) < 0 || decimal.Cmp(endPeriod, lifeDec) > 0 ||
		decimal.Cmp(startPeriod, endPeriod) > 0 {
		return decimal.Decimal{}, errors.New("VDB: require 0 <= startPeriod <= endPeriod <= life")
	}

	cumEnd, err := vdbCumulative(work, cost, salvage, life, factor, noSwitch, endPeriod)
	if err != nil {
		return decimal.Decimal{}, err
	}
	cumStart, err := vdbCumulative(work, cost, salvage, life, factor, noSwitch, startPeriod)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(decimal.Sub(work, cumEnd, cumStart)), nil
}

// vdbCumulative returns total depreciation accumulated from time 0 up
// to time t (0 ≤ t ≤ life). Whole periods are processed iteratively;
// when t lands inside period k (k-1 < t < k) the period's full
// depreciation is pro-rated by (t - (k-1)).
func vdbCumulative(ctx decimal.Context, cost, salvage decimal.Decimal, life int, factor decimal.Decimal, noSwitch bool, t decimal.Decimal) (decimal.Decimal, error) {
	zero := decimal.NewFromInt64(ctx, 0)
	if decimal.Cmp(t, zero) <= 0 {
		return zero, nil
	}
	rate, err := decimal.Div(ctx, factor, decimal.NewFromInt64(ctx, int64(life)))
	if err != nil {
		return decimal.Decimal{}, err
	}

	bookValue := cost
	cum := zero
	inSL := false
	var slDepr decimal.Decimal

	for k := 1; k <= life; k++ {
		kDec := decimal.NewFromInt64(ctx, int64(k))
		kPrev := decimal.NewFromInt64(ctx, int64(k-1))

		var fullDepr decimal.Decimal
		switch {
		case inSL:
			fullDepr = slDepr
		default:
			// Declining-balance amount, capped to keep book ≥ salvage.
			ddb := decimal.Mul(ctx, bookValue, rate)
			cap := decimal.Sub(ctx, bookValue, salvage)
			if decimal.Cmp(cap, zero) < 0 {
				cap = zero
			}
			if decimal.Cmp(ddb, cap) > 0 {
				ddb = cap
			}
			if decimal.Cmp(ddb, zero) < 0 {
				ddb = zero
			}
			fullDepr = ddb

			if !noSwitch {
				// Compare to straight-line over remaining life.
				remainLife := decimal.NewFromInt64(ctx, int64(life-k+1))
				if decimal.Cmp(remainLife, zero) > 0 {
					sl, err := decimal.Div(ctx, decimal.Sub(ctx, bookValue, salvage), remainLife)
					if err != nil {
						return decimal.Decimal{}, err
					}
					if decimal.Cmp(sl, zero) < 0 {
						sl = zero
					}
					if decimal.Cmp(sl, ddb) > 0 {
						inSL = true
						slDepr = sl
						fullDepr = sl
					}
				}
			}
		}

		switch {
		case decimal.Cmp(t, kDec) >= 0:
			// Period k fits entirely within [0, t].
			cum = decimal.Add(ctx, cum, fullDepr)
			bookValue = decimal.Sub(ctx, bookValue, fullDepr)
		case decimal.Cmp(t, kPrev) > 0:
			// t lands inside period k → partial.
			frac := decimal.Sub(ctx, t, kPrev)
			partial := decimal.Mul(ctx, fullDepr, frac)
			cum = decimal.Add(ctx, cum, partial)
			return cum, nil
		default:
			// t ≤ k-1, nothing more to do.
			return cum, nil
		}
	}
	return cum, nil
}
