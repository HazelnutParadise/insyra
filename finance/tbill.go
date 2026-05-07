package finance

import (
	"errors"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

// TBillEq returns the bond-equivalent yield of a Treasury bill.
//
//	TBILLEQ = 365 · discount / (360 - discount · DSM)
//
// where DSM is the number of calendar days from settlement to maturity.
// Returns an error if maturity ≤ settlement or DSM > 365.
//
// Excel equivalent: TBILLEQ(settlement, maturity, discount)
func TBillEq(settlement, maturity time.Time, discount decimal.Decimal, opts ...Options) (decimal.Decimal, error) {
	dsm, err := tbillDSM(settlement, maturity)
	if err != nil {
		return decimal.Decimal{}, err
	}
	o := resolveOpts(opts)
	work := o.workCtx()

	dsmDec := decimal.NewFromInt64(work, int64(dsm))
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
		return 0, errors.New("T-bill: settlement to maturity must not exceed 365 days")
	}
	return dsm, nil
}
