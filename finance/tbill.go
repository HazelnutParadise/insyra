package finance

import (
	"errors"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

// TBillEq returns the bond-equivalent (coupon-equivalent) yield of a Treasury
// bill, following the US Treasury / SIA "investment rate" convention:
//
//	DSM ≤ 182:  i = 365 · discount / (360 - discount · DSM)
//	DSM > 182:  i = 2·(√(n² − (2n−1)(1 − 1/P)) − n) / (2n − 1)
//	            with n = DSM/365 and price per $1  P = 1 − discount·DSM/360
//
// where DSM is the number of calendar days from settlement to maturity.
// Returns an error if maturity ≤ settlement or DSM > 365.
//
// API-compatible with Excel's TBILLEQ(settlement, maturity, discount), but the
// VALUE follows the financially-correct Treasury convention. For DSM > 182,
// Excel keeps using the simple short-bill formula and so overstates the yield
// (e.g. ~7 bp high on a 364-day 5% bill); this implementation uses the long-bill
// coupon-equivalent formula, which accounts for the intra-period coupon via
// semi-annual compounding. Verified against QuantLib's exact semi-annually
// compounded yield (agreement within ~1 bp for DSM > 182) and continuous with
// the short-bill formula at the 182-day boundary.
func TBillEq(settlement, maturity time.Time, discount decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	dsm, err := tbillDSM(settlement, maturity)
	if err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()
	dsmDec := decimal.NewFromInt64(work, int64(dsm))

	if dsm <= 182 {
		// Short bill: simple money-market bond-equivalent yield.
		num := decimal.Mul(work, decimal.NewFromInt64(work, 365), discount)
		denom := decimal.Sub(work, decimal.NewFromInt64(work, 360),
			decimal.Mul(work, discount, dsmDec))
		if isZero(denom) {
			return decimal.Decimal{}, errors.New("TBILLEQ: denominator vanished (discount · DSM = 360)")
		}
		v, err := decimal.Div(work, num, denom)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return o.outCtx().Normalize(v), nil
	}

	// Long bill (DSM > 182): US Treasury / SIA coupon-equivalent yield.
	one := decimal.NewFromInt64(work, 1)
	two := decimal.NewFromInt64(work, 2)
	dfrac, err := decimal.Div(work, decimal.Mul(work, discount, dsmDec),
		decimal.NewFromInt64(work, 360))
	if err != nil {
		return decimal.Decimal{}, err
	}
	price := decimal.Sub(work, one, dfrac) // price per $1
	if isZero(price) {
		return decimal.Decimal{}, errors.New("TBILLEQ: price vanished (discount · DSM = 360)")
	}
	n, err := decimal.Div(work, dsmDec, decimal.NewFromInt64(work, 365))
	if err != nil {
		return decimal.Decimal{}, err
	}
	twoNm1 := decimal.Sub(work, decimal.Mul(work, two, n), one) // 2n − 1
	if isZero(twoNm1) {
		return decimal.Decimal{}, errors.New("TBILLEQ: degenerate at the 182.5-day boundary")
	}
	invP, err := decimal.Div(work, one, price)
	if err != nil {
		return decimal.Decimal{}, err
	}
	oneMinusInvP := decimal.Sub(work, one, invP)
	// discriminant = n² − (2n−1)(1 − 1/P)
	disc := decimal.Sub(work, decimal.Mul(work, n, n),
		decimal.Mul(work, twoNm1, oneMinusInvP))
	root, err := decimal.Sqrt(work, disc)
	if err != nil {
		return decimal.Decimal{}, err
	}
	// i = 2·(√disc − n) / (2n − 1)
	v, err := decimal.Div(work, decimal.Mul(work, two, decimal.Sub(work, root, n)), twoNm1)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(v), nil
}

// TBillPrice returns the price per $100 face value of a Treasury bill
// at the given discount rate.
//
//	TBILLPRICE = 100 · (1 - discount · DSM / 360)
//
// Excel equivalent: TBILLPRICE(settlement, maturity, discount)
func TBillPrice(settlement, maturity time.Time, discount decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	dsm, err := tbillDSM(settlement, maturity)
	if err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	dsmDec := decimal.NewFromInt64(work, int64(dsm))
	frac, err := decimal.Div(work,
		decimal.Mul(work, discount, dsmDec),
		decimal.NewFromInt64(work, 360))
	if err != nil {
		return decimal.Decimal{}, err
	}
	one := decimal.NewFromInt64(work, 1)
	hundred := decimal.NewFromInt64(work, 100)
	v := decimal.Mul(work, hundred, decimal.Sub(work, one, frac))
	return o.outCtx().Normalize(v), nil
}

// TBillYield returns the yield of a Treasury bill given its price per
// $100 face value.
//
//	TBILLYIELD = ((100 - pr) / pr) · (360 / DSM)
//
// Excel equivalent: TBILLYIELD(settlement, maturity, pr)
func TBillYield(settlement, maturity time.Time, pr decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	dsm, err := tbillDSM(settlement, maturity)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if isZero(pr) {
		return decimal.Decimal{}, errors.New("TBILLYIELD: price cannot be zero")
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	dsmDec := decimal.NewFromInt64(work, int64(dsm))
	hundred := decimal.NewFromInt64(work, 100)
	priceFactor, err := decimal.Div(work, decimal.Sub(work, hundred, pr), pr)
	if err != nil {
		return decimal.Decimal{}, err
	}
	dayFactor, err := decimal.Div(work, decimal.NewFromInt64(work, 360), dsmDec)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.outCtx().Normalize(decimal.Mul(work, priceFactor, dayFactor)), nil
}

// tbillDSM returns the calendar days from settlement to maturity,
// validating Excel's T-bill constraints (max 365 days, settlement
// strictly before maturity).
func tbillDSM(settlement, maturity time.Time) (int, error) {
	dsm := calendarDays(settlement, maturity)
	if dsm <= 0 {
		return 0, errors.New("maturity must be after settlement")
	}
	if dsm > 365 {
		return 0, errors.New("settlement to maturity must not exceed 365 days for a T-bill")
	}
	return dsm, nil
}
