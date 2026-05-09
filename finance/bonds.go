package finance

import (
	"errors"
	"fmt"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

// CouponFrequency restricts coupon frequency to the values Excel
// supports: 1 (annual), 2 (semi-annual), 4 (quarterly).
func validateFrequency(freq int) error {
	if freq != 1 && freq != 2 && freq != 4 {
		return errors.New("frequency must be 1, 2, or 4")
	}
	return nil
}

// coupPrevNext returns (prev, next) coupon dates that bracket
// settlement. Coupons land on the same day-of-month as maturity,
// stepped backward by 12/freq months. If settlement falls exactly on
// a coupon date, that date is treated as `prev` and the next coupon
// is one period later.
func coupPrevNext(settlement, maturity time.Time, freq int) (time.Time, time.Time) {
	months := 12 / freq
	settlement = toDateUTC(settlement)
	maturity = toDateUTC(maturity)
	prev := maturity
	next := maturity
	for prev.After(settlement) {
		next = prev
		prev = addMonths(prev, -months)
	}
	return prev, next
}

// coupDaysBS returns the number of days from the previous coupon date
// (or issue, in continuous-accrual mode) to settlement, under the
// given basis. Excel: COUPDAYBS.
func coupDaysBS(settlement, maturity time.Time, freq int, basis DayCountBasis) int {
	prev, _ := coupPrevNext(settlement, maturity, freq)
	return dayDiff(prev, settlement, basis)
}

// coupDaysNC returns the number of days from settlement to the next
// coupon date, under the given basis. Excel: COUPDAYSNC.
func coupDaysNC(settlement, maturity time.Time, freq int, basis DayCountBasis) int {
	_, next := coupPrevNext(settlement, maturity, freq)
	return dayDiff(settlement, next, basis)
}

// coupDays returns the number of days in the current coupon period
// (the one containing settlement), under the given basis.
// Excel: COUPDAYS.
func coupDays(settlement, maturity time.Time, freq int, basis DayCountBasis) int {
	prev, next := coupPrevNext(settlement, maturity, freq)
	if basis == BasisActualActual {
		// For Act/Act, the period length is the actual calendar days.
		return calendarDays(prev, next)
	}
	if basis == BasisActual360 || basis == BasisActual365 {
		// Actual day count for the period.
		return calendarDays(prev, next)
	}
	// 30/360 bases: 360 / freq.
	return 360 / freq
}

// coupNum returns the number of coupons payable between settlement
// and maturity (inclusive of the coupon at maturity).
// Excel: COUPNUM.
func coupNum(settlement, maturity time.Time, freq int) int {
	_, next := coupPrevNext(settlement, maturity, freq)
	months := 12 / freq
	maturity = toDateUTC(maturity)
	count := 0
	c := next
	for !c.After(maturity) {
		count++
		c = addMonths(c, months)
	}
	return count
}

// Price returns the bond's price per $100 face value given its
// annual coupon rate, yield, redemption value, frequency, and
// day-count basis.
//
// Excel equivalent: PRICE(settlement, maturity, rate, yld, redemption, frequency, basis)
func Price(settlement, maturity time.Time, rate, yld, redemption decimal.Decimal,
	freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error) {
	if err := validateBondInputs(settlement, maturity, freq, basis); err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	v, err := priceInternal(work, settlement, maturity, rate, yld, redemption, freq, basis)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// priceInternal does the PRICE calculation in the given working
// context without doing the final normalize. It's split out so YIELD's
// Newton solver and DURATION can reuse it without paying the
// normalize-per-iteration cost.
func priceInternal(ctx decimal.Context, settlement, maturity time.Time,
	rate, yld, redemption decimal.Decimal, freq int, basis DayCountBasis) (decimal.Decimal, error) {

	freqDec := decimal.NewFromInt64(ctx, int64(freq))
	one := decimal.NewFromInt64(ctx, 1)
	hundred := decimal.NewFromInt64(ctx, 100)
	dsc := coupDaysNC(settlement, maturity, freq, basis)
	a := coupDaysBS(settlement, maturity, freq, basis)
	e := coupDays(settlement, maturity, freq, basis)
	if e <= 0 {
		return decimal.Decimal{}, errors.New("PRICE: coupon period length E is non-positive")
	}
	n := coupNum(settlement, maturity, freq)
	if n < 1 {
		return decimal.Decimal{}, errors.New("PRICE: no coupons remain (settlement >= maturity?)")
	}

	dscOverE, err := decimal.Div(ctx,
		decimal.NewFromInt64(ctx, int64(dsc)),
		decimal.NewFromInt64(ctx, int64(e)))
	if err != nil {
		return decimal.Decimal{}, err
	}
	aOverE, err := decimal.Div(ctx,
		decimal.NewFromInt64(ctx, int64(a)),
		decimal.NewFromInt64(ctx, int64(e)))
	if err != nil {
		return decimal.Decimal{}, err
	}

	// Coupon payment = 100·rate/freq.
	coupon, err := decimal.Div(ctx, decimal.Mul(ctx, hundred, rate), freqDec)
	if err != nil {
		return decimal.Decimal{}, err
	}

	// Accrued = coupon · A/E (subtracted at the end to convert from
	// dirty to clean price).
	accrued := decimal.Mul(ctx, coupon, aOverE)

	if n == 1 {
		// Single remaining coupon: simple-interest discounting on the
		// stub period.
		yldOverFreq, err := decimal.Div(ctx, yld, freqDec)
		if err != nil {
			return decimal.Decimal{}, err
		}
		denom := decimal.Add(ctx, one, decimal.Mul(ctx, dscOverE, yldOverFreq))
		num := decimal.Add(ctx, redemption, coupon)
		dirty, err := decimal.Div(ctx, num, denom)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return decimal.Sub(ctx, dirty, accrued), nil
	}

	// Multi-coupon case. We use compound discounting at yld/freq per
	// period; the leading exponent is fractional (DSC/E) for the first
	// stub period.
	yldOverFreq, err := decimal.Div(ctx, yld, freqDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	onePlus := decimal.Add(ctx, one, yldOverFreq)

	// Coupon-strip PV: Σ_{k=1}^{N} coupon / (1+yld/f)^(k-1 + DSC/E)
	couponSum := decimal.NewFromInt64(ctx, 0)
	for k := 1; k <= n; k++ {
		exp := decimal.Add(ctx, decimal.NewFromInt64(ctx, int64(k-1)), dscOverE)
		factor, err := decimal.Pow(ctx, onePlus, exp)
		if err != nil {
			return decimal.Decimal{}, err
		}
		quo, err := decimal.Div(ctx, coupon, factor)
		if err != nil {
			return decimal.Decimal{}, err
		}
		couponSum = decimal.Add(ctx, couponSum, quo)
	}
	// Redemption PV: redemption / (1+yld/f)^(N-1 + DSC/E)
	redempExp := decimal.Add(ctx,
		decimal.NewFromInt64(ctx, int64(n-1)), dscOverE)
	redempFactor, err := decimal.Pow(ctx, onePlus, redempExp)
	if err != nil {
		return decimal.Decimal{}, err
	}
	redempPV, err := decimal.Div(ctx, redemption, redempFactor)
	if err != nil {
		return decimal.Decimal{}, err
	}

	dirty := decimal.Add(ctx, couponSum, redempPV)
	return decimal.Sub(ctx, dirty, accrued), nil
}

// Yield solves for the bond yield that reproduces the given price
// per $100 face. Uses Newton's method with a finite-difference
// derivative seeded at the provided guess (or 0.05 if guess is the
// zero Decimal).
//
// Excel equivalent: YIELD(settlement, maturity, rate, pr, redemption, frequency, basis)
func Yield(settlement, maturity time.Time, rate, pr, redemption decimal.Decimal,
	freq int, basis DayCountBasis, guess decimal.Decimal,
	opts ...Options) (decimal.Decimal, error) {
	if err := validateBondInputs(settlement, maturity, freq, basis); err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	y := guess
	if isZero(y) {
		y = decimal.MustParse(work, "0.05")
	}

	tol := tenToMinus(work, o.Scale+8)
	h := tenToMinus(work, o.Scale+10)

	for range 80 {
		p, err := priceInternal(work, settlement, maturity, rate, y, redemption, freq, basis)
		if err != nil {
			return decimal.Decimal{}, err
		}
		f := decimal.Sub(work, p, pr)
		if absCmp(f, tol) <= 0 {
			return o.outCtx().Normalize(y), nil
		}

		pp, err := priceInternal(work, settlement, maturity, rate,
			decimal.Add(work, y, h), redemption, freq, basis)
		if err != nil {
			return decimal.Decimal{}, err
		}
		pm, err := priceInternal(work, settlement, maturity, rate,
			decimal.Sub(work, y, h), redemption, freq, basis)
		if err != nil {
			return decimal.Decimal{}, err
		}
		two := decimal.NewFromInt64(work, 2)
		df, err := decimal.Div(work,
			decimal.Sub(work, pp, pm),
			decimal.Mul(work, two, h))
		if err != nil {
			return decimal.Decimal{}, err
		}
		if isZero(df) {
			return decimal.Decimal{}, errors.New("YIELD: derivative vanished during Newton iteration")
		}
		delta, err := decimal.Div(work, f, df)
		if err != nil {
			return decimal.Decimal{}, err
		}
		y = decimal.Sub(work, y, delta)

		if absCmp(delta, tol) <= 0 {
			return o.outCtx().Normalize(y), nil
		}
	}
	return decimal.Decimal{}, errors.New("YIELD did not converge in 80 iterations")
}

// Duration returns the Macaulay duration of a bond.
//
//	D = Σ t_k · PV(cf_k) / Σ PV(cf_k)
//
// where t_k is the time to cashflow k in years from settlement.
//
// Excel equivalent: DURATION(settlement, maturity, coupon, yld, frequency, basis)
func Duration(settlement, maturity time.Time, coupon, yld decimal.Decimal,
	freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error) {
	if err := validateBondInputs(settlement, maturity, freq, basis); err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	v, err := durationInternal(work, settlement, maturity, coupon, yld, freq, basis)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

func durationInternal(ctx decimal.Context, settlement, maturity time.Time,
	rate, yld decimal.Decimal, freq int, basis DayCountBasis) (decimal.Decimal, error) {

	freqDec := decimal.NewFromInt64(ctx, int64(freq))
	one := decimal.NewFromInt64(ctx, 1)
	hundred := decimal.NewFromInt64(ctx, 100)

	dsc := coupDaysNC(settlement, maturity, freq, basis)
	e := coupDays(settlement, maturity, freq, basis)
	if e <= 0 {
		return decimal.Decimal{}, errors.New("DURATION: coupon period length E is non-positive")
	}
	n := coupNum(settlement, maturity, freq)
	if n < 1 {
		return decimal.Decimal{}, errors.New("DURATION: no coupons remain")
	}

	dscOverE, err := decimal.Div(ctx,
		decimal.NewFromInt64(ctx, int64(dsc)),
		decimal.NewFromInt64(ctx, int64(e)))
	if err != nil {
		return decimal.Decimal{}, err
	}
	cpn, err := decimal.Div(ctx, decimal.Mul(ctx, hundred, rate), freqDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	yldOverFreq, err := decimal.Div(ctx, yld, freqDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	onePlus := decimal.Add(ctx, one, yldOverFreq)

	// Numerator: Σ t_k · PV(cf_k)  (t_k in years)
	// Denominator: Σ PV(cf_k)  (= dirty price)
	num := decimal.NewFromInt64(ctx, 0)
	den := decimal.NewFromInt64(ctx, 0)

	for k := 1; k <= n; k++ {
		periodOffset := decimal.Add(ctx,
			decimal.NewFromInt64(ctx, int64(k-1)), dscOverE)
		factor, err := decimal.Pow(ctx, onePlus, periodOffset)
		if err != nil {
			return decimal.Decimal{}, err
		}

		var cf decimal.Decimal
		if k == n {
			cf = decimal.Add(ctx, cpn, hundred) // coupon + redemption (par=100)
		} else {
			cf = cpn
		}
		pv, err := decimal.Div(ctx, cf, factor)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// time t_k in years = periodOffset / freq
		tYears, err := decimal.Div(ctx, periodOffset, freqDec)
		if err != nil {
			return decimal.Decimal{}, err
		}
		num = decimal.Add(ctx, num, decimal.Mul(ctx, tYears, pv))
		den = decimal.Add(ctx, den, pv)
	}
	if isZero(den) {
		return decimal.Decimal{}, errors.New("DURATION: zero PV denominator")
	}
	return decimal.Div(ctx, num, den)
}

// MDuration returns the modified duration of a bond:
//
//	MD = Duration / (1 + yld/freq)
//
// Excel equivalent: MDURATION(settlement, maturity, coupon, yld, frequency, basis)
func MDuration(settlement, maturity time.Time, coupon, yld decimal.Decimal,
	freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error) {
	if err := validateBondInputs(settlement, maturity, freq, basis); err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	d, err := durationInternal(work, settlement, maturity, coupon, yld, freq, basis)
	if err != nil {
		return decimal.Decimal{}, err
	}
	freqDec := decimal.NewFromInt64(work, int64(freq))
	yldOverFreq, err := decimal.Div(work, yld, freqDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	denom := decimal.Add(work, decimal.NewFromInt64(work, 1), yldOverFreq)
	v, err := decimal.Div(work, d, denom)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// AccrInt returns the accrued interest of a security that pays
// periodic interest. With calcMethod=true, accrual is computed from
// issue to settlement; with calcMethod=false, accrual starts at
// firstInterest (matching Excel's behavior for bonds purchased after
// the first coupon date but before maturity).
//
// Excel equivalent: ACCRINT(issue, first_interest, settlement, rate, par, frequency, basis, calc_method)
func AccrInt(issue, firstInterest, settlement time.Time, rate, par decimal.Decimal,
	freq int, basis DayCountBasis, calcMethod bool, opts ...Options) (decimal.Decimal, error) {
	if err := validateFrequency(freq); err != nil {
		return decimal.Decimal{}, err
	}
	if err := validateBasis(basis); err != nil {
		return decimal.Decimal{}, err
	}
	if !settlement.After(issue) {
		return decimal.Decimal{}, errors.New("settlement must be after issue")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	// ACCRINT = par · rate · year_fraction(start, settlement, basis).
	// start = issue when calc_method=true, else firstInterest.
	start := issue
	if !calcMethod {
		start = firstInterest
	}
	yf, err := yearFraction(work, start, settlement, basis)
	if err != nil {
		return decimal.Decimal{}, err
	}
	v := decimal.Mul(work, decimal.Mul(work, par, rate), yf)
	return o.outCtx().Normalize(v), nil
}

// validateBondInputs runs the common settlement/maturity/freq/basis
// checks shared by every bond pricing function.
func validateBondInputs(settlement, maturity time.Time, freq int, basis DayCountBasis) error {
	if !maturity.After(settlement) {
		return fmt.Errorf("maturity (%s) must be after settlement (%s)",
			maturity.Format("2006-01-02"), settlement.Format("2006-01-02"))
	}
	if err := validateFrequency(freq); err != nil {
		return err
	}
	if err := validateBasis(basis); err != nil {
		return err
	}
	return nil
}
