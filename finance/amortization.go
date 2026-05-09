package finance

import (
	"github.com/HazelnutParadise/insyra"
	"github.com/TimLai666/go-decimal/decimal"
)

// AmortizationRow is one row in an amortization schedule. Values are
// kept as decimal.Decimal so callers don't lose precision when reading
// back individual entries.
type AmortizationRow struct {
	Period    int             // 1-indexed period number
	Payment   decimal.Decimal // total payment in this period (constant)
	Interest  decimal.Decimal // interest portion of the payment
	Principal decimal.Decimal // principal portion of the payment
	Balance   decimal.Decimal // remaining balance after this payment
}

// AmortizationSchedule returns the full per-period amortization
// schedule of a level-payment loan. Use ScheduleTable for an
// insyra.IDataTable representation suitable for printing or export.
func AmortizationSchedule(rate decimal.Decimal, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) ([]AmortizationRow, error) {
	if err := validateNper(nper); err != nil {
		return nil, err
	}
	if err := validateTiming(timing); err != nil {
		return nil, err
	}

	o := resolveOpts(opts)
	work := o.workCtx()
	out := o.outCtx()

	pmt, err := pmtInternal(work, rate, nper, pv, fv, timing)
	if err != nil {
		return nil, err
	}

	rows := make([]AmortizationRow, 0, nper)
	balance := pv
	pmtOut := out.Normalize(pmt)

	for per := 1; per <= nper; per++ {
		var ipmt, ppmt decimal.Decimal

		if timing == PaymentEnd {
			// End-of-period: interest accrues on opening balance, then
			// payment is applied at period close.
			ipmt = neg(decimal.Mul(work, balance, rate))
			ppmt = decimal.Sub(work, pmt, ipmt)
			// balance after period = balance*(1+r) + pmt
			balance = decimal.Add(work, decimal.Mul(work, balance, onePlus(work, rate)), pmt)
		} else {
			// Begin-of-period: payment is applied first; interest then
			// accrues during the period on the post-payment balance.
			if per == 1 {
				ipmt = decimal.NewFromInt64(work, 0)
				ppmt = pmt
				balance = decimal.Add(work, balance, pmt)
				// balance grows by (1+r) over period 1
				balance = decimal.Mul(work, balance, onePlus(work, rate))
			} else {
				// "balance" here is the post-payment, post-period-growth
				// balance from the prior iteration — interest accrued in
				// the just-finished period equals balance_at_start * r,
				// where balance_at_start = balance / (1+r). Equivalently,
				// the interest charged to this payment is (balance carried
				// in - payment that would have applied) * r — which works
				// out to the formula below.
				ipmt = neg(decimal.Mul(work, balance, rate))
				ipmt, err = decimal.Div(work, ipmt, onePlus(work, rate))
				if err != nil {
					return nil, err
				}
				ppmt = decimal.Sub(work, pmt, ipmt)
				balance = decimal.Add(work, balance, pmt)
				balance = decimal.Mul(work, balance, onePlus(work, rate))
			}
		}

		rows = append(rows, AmortizationRow{
			Period:    per,
			Payment:   pmtOut,
			Interest:  out.Normalize(ipmt),
			Principal: out.Normalize(ppmt),
			Balance:   out.Normalize(balance),
		})
	}
	return rows, nil
}

// ScheduleTable returns the amortization schedule as an insyra
// DataTable with columns: Period, Payment, Interest, Principal,
// Balance. Numeric columns are stored as decimal.Decimal values so the
// table preserves the precision of the source calculation; convert via
// .String() if you need a textual export.
func ScheduleTable(rate decimal.Decimal, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (insyra.IDataTable, error) {
	rows, err := AmortizationSchedule(rate, nper, pv, fv, timing, opts...)
	if err != nil {
		return nil, err
	}

	table := insyra.NewDataTable()
	for _, r := range rows {
		table.AppendRowsByColIndex(map[string]any{
			"A": r.Period,
			"B": r.Payment,
			"C": r.Interest,
			"D": r.Principal,
			"E": r.Balance,
		})
	}
	table.SetColNameByIndex("A", "Period")
	table.SetColNameByIndex("B", "Payment")
	table.SetColNameByIndex("C", "Interest")
	table.SetColNameByIndex("D", "Principal")
	table.SetColNameByIndex("E", "Balance")
	return table, nil
}
