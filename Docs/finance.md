# [ finance ] Package

This document describes all public APIs in the `finance` package, designed for AI/automated applications to directly understand each function, type, parameter, and return value.

---

## Installation

```bash
go get github.com/HazelnutParadise/insyra/finance
```

The package depends on [`github.com/TimLai666/go-decimal`](https://github.com/TimLai666/go-decimal) for fixed-point decimal arithmetic. It is added automatically by `go mod tidy`.

---

## Overview

The `finance` package provides high-precision financial calculations for banking and corporate-finance use cases:

- **Time Value of Money**: `PMT`, `PV`, `FV`, `NPER`, `RATE` — solve any of the five Excel TVM variables given the others
- **Per-Period Split**: `IPMT`, `PPMT`, `CumIPMT`, `CumPPMT` — separate interest from principal in any period or range
- **Discounted Cashflows**: `NPV`, `NPVExcel`, `IRR`, `MIRR` — net present value, internal rate of return, modified IRR
- **Date-Indexed Cashflows**: `XNPV`, `XIRR` — same as NPV/IRR but cashflows can fall on arbitrary dates
- **Rate Conversion**: `EffectiveRate`, `NominalRate`, `ContinuousFromAnnual`, `AnnualFromContinuous`
- **Depreciation**: `SLN`, `DDB`, `SYD`, `VDB` — straight-line, declining-balance, sum-of-years digits, variable declining-balance
- **Bonds**: `Price`, `Yield`, `Duration`, `MDuration`, `AccrInt` with five day-count basis options
- **Treasury Bills**: `TBillEq`, `TBillPrice`, `TBillYield`
- **Amortization**: `AmortizationSchedule` (typed slice) and `ScheduleTable` (returns `insyra.IDataTable` for printing/export)

All functions perform their internal arithmetic on `decimal.Decimal` values, never on `float64`, so results are exact up to the chosen output precision. The output `Scale` (digits after the decimal point) and rounding `Mode` are configurable per call via `Options`.

> Compared to spreadsheet/`float64`-based references such as Excel and `numpy_financial`, the `finance` package routinely produces answers that are correct further than the 11th significant digit. For the canonical 30-year mortgage `PMT(0.005, 360, 100000)`, Excel returns `-599.5505251260807` while this package returns `-599.55052515275239459146124368…`, which matches the exact rational `-500·201^360 / (201^360 - 200^360)`.

Every exported function returns `(decimal.Decimal, error)` (or `([]AmortizationRow, error)` / `(insyra.IDataTable, error)`). Always handle `err` at the call site — invalid arguments and divide-by-zero situations are signaled this way, never via panic or warning logs.

---

## Core Types

### PaymentTiming

When during each period the annuity payment occurs.

```go
type PaymentTiming uint8

const (
    PaymentEnd   PaymentTiming = 0 // end of period (Excel type=0, default)
    PaymentBegin PaymentTiming = 1 // beginning of period (Excel type=1)
)
```

### Options

Per-call precision and rounding configuration. Pass as the last (variadic) argument; omit it to accept the defaults.

```go
type Options struct {
    Scale int32                 // decimal places kept in the result (default DefaultScale = 10)
    Mode  decimal.RoundingMode  // rounding mode (default RoundingModeHalfUp)
}

const DefaultScale int32 = 10
```

Internally, every operation runs at `Scale + 16` guard digits and the result is normalized once at the end, so chained calculations don't accumulate visible rounding error.

`decimal.RoundingMode` is one of `RoundingModeDown`, `RoundingModeUp`, or `RoundingModeHalfUp` (see go-decimal docs).

### AmortizationRow

One row in the per-period schedule.

```go
type AmortizationRow struct {
    Period    int             // 1-indexed period number
    Payment   decimal.Decimal // total payment in this period (constant)
    Interest  decimal.Decimal // interest portion of the payment
    Principal decimal.Decimal // principal portion of the payment
    Balance   decimal.Decimal // remaining balance after this payment
}
```

### Package-level helpers

```go
var Zero decimal.Decimal               // shared zero, suitable as fv / pv

func New(s string) (decimal.Decimal, error)   // parse a decimal literal
func MustNew(s string) decimal.Decimal        // panic on parse error
func FromInt(n int) decimal.Decimal           // int → Decimal
func FromFloat(f float64) (decimal.Decimal, error)  // float64 → Decimal (use sparingly)
```

`New` / `MustNew` parse at the package's internal precision (28 digits). Use them to build inputs from string literals — do **not** parse via `decimal.Decimal{}` directly.

---

## Sign Convention

Every function follows Excel/Google-Sheets sign conventions:

- Money **received** (e.g., the loan principal you take out, the FV of an investment that matures in your favor) is **positive**.
- Money **paid out** (each loan installment, deposits into a savings plan) is **negative**.

Concretely, for a $100,000 mortgage at 0.5% per period over 360 periods:

```
PMT(0.005, 360, 100000, fv=0, end)  =>  -599.55052515…
```

The negative sign reflects that you (the borrower) pay 599.55 each month. PV and FV similarly report opposite signs from PMT.

---

## Time Value of Money

### PMT

```go
func PMT(rate decimal.Decimal, nper int, pv, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (decimal.Decimal, error)
```

Periodic payment of an annuity / loan. Excel equivalent: `PMT(rate, nper, pv, fv, type)`.

**Parameters:**

- `rate`: periodic interest rate
- `nper`: total number of periods (must be ≥ 1)
- `pv`: present value (loan amount with sign)
- `fv`: future value (use `finance.Zero` if none)
- `timing`: `PaymentEnd` or `PaymentBegin`
- `opts`: optional precision overrides

**Returns:** `(pmt, err)` — `err` is non-nil for `nper ≤ 0` or invalid `timing`.

### PV

```go
func PV(rate decimal.Decimal, nper int, pmt, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (decimal.Decimal, error)
```

Present value of an annuity. Excel equivalent: `PV(rate, nper, pmt, fv, type)`.

### FV

```go
func FV(rate decimal.Decimal, nper int, pmt, pv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (decimal.Decimal, error)
```

Future value of an annuity. Excel equivalent: `FV(rate, nper, pmt, pv, type)`.

### NPER

```go
func NPER(rate, pmt, pv, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (decimal.Decimal, error)
```

Number of periods needed to satisfy the TVM equation. May return a non-integer (the last period is then partial). Excel equivalent: `NPER(rate, pmt, pv, fv, type)`.

Returns an error if `rate == 0 && pmt == 0` (degenerate), or if the resulting argument to `Log` is non-positive (no real-valued NPER exists).

### RATE

```go
func RATE(nper int, pmt, pv, fv decimal.Decimal, timing PaymentTiming,
    guess decimal.Decimal, opts ...Options) (decimal.Decimal, error)
```

Periodic interest rate that satisfies the TVM equation. Excel equivalent: `RATE(nper, pmt, pv, fv, type, guess)`.

Internally uses Newton's method with a finite-difference derivative. `guess` provides the seed; pass `finance.Zero` to use the Excel-default 0.1 (10%). Returns an error if the iteration fails to converge in 60 steps — try a different `guess`.

---

## Per-Period Split

### IPMT

```go
func IPMT(rate decimal.Decimal, per, nper int, pv, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (decimal.Decimal, error)
```

Interest portion of the `per`-th payment (`per` is 1-indexed, must satisfy `1 ≤ per ≤ nper`). Excel equivalent: `IPMT(rate, per, nper, pv, fv, type)`.

With `PaymentBegin`, `IPMT(per=1) == 0` because no time has elapsed before the first installment.

### PPMT

```go
func PPMT(rate decimal.Decimal, per, nper int, pv, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (decimal.Decimal, error)
```

Principal portion of the `per`-th payment. By definition `IPMT[per] + PPMT[per] == PMT` for every period. Excel equivalent: `PPMT(rate, per, nper, pv, fv, type)`.

### CumIPMT, CumPPMT

```go
func CumIPMT(rate decimal.Decimal, nper int, pv decimal.Decimal,
    startPeriod, endPeriod int, timing PaymentTiming,
    opts ...Options) (decimal.Decimal, error)

func CumPPMT(rate decimal.Decimal, nper int, pv decimal.Decimal,
    startPeriod, endPeriod int, timing PaymentTiming,
    opts ...Options) (decimal.Decimal, error)
```

Cumulative interest / principal paid between `startPeriod` and `endPeriod` inclusive (1-indexed). Future value is implicitly `0`. Excel equivalents: `CUMIPMT`, `CUMPRINC`.

The identity `CumIPMT(1,nper) + CumPPMT(1,nper) = PMT · nper` always holds.

---

## Discounted Cashflows

### NPV

```go
func NPV(rate decimal.Decimal, cashflows []decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Net present value of `cashflows`, where `cashflows[0]` occurs at **t = 0**:

```
NPV = Σ cashflows[i] / (1 + rate)^i
```

This convention matches `numpy_financial.npv` and is the natural input for `IRR`. It differs from Excel's `NPV`, which assumes the first value occurs at t = 1 — for that, use `NPVExcel`.

### NPVExcel

```go
func NPVExcel(rate decimal.Decimal, cashflows []decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Excel-compatible: discounts `cashflows[0]` by one period. Equivalent to `NPV(rate, append([Zero], cashflows...))`.

### IRR

```go
func IRR(cashflows []decimal.Decimal, guess decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Internal rate of return, i.e. the `r` that drives `NPV(r, cashflows) → 0`. Solves with Newton's method using the analytic derivative `-Σ i·cf[i] / (1+r)^(i+1)`. Pass `finance.Zero` for `guess` to use the Excel-default 0.1.

`cashflows` must contain at least two entries with at least one sign change. Returns an error if Newton fails to converge within 100 iterations or if iterates fall below `r = -1`. When that happens, retry with a guess closer to the expected answer.

### MIRR

```go
func MIRR(cashflows []decimal.Decimal, financeRate, reinvestRate decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Modified internal rate of return. Unlike `IRR`, it lets you discount negative (financing) cashflows and compound positive (reinvestment) cashflows at different rates:

```text
MIRR = (FV(positive_cf @ reinvestRate) / -PV(negative_cf @ financeRate))^(1/n) - 1
```

where n = `len(cashflows) - 1`. Excel equivalent: `MIRR(values, finance_rate, reinvest_rate)`.

Returns an error if the cashflow stream contains no negatives (no financing to recover) or no positives (no nth-root to take).

---

## Date-Indexed Cashflows

`XNPV` and `XIRR` discount each cashflow by its actual elapsed time from a reference date, using a fixed 365-day year (matching Excel exactly).

### XNPV

```go
func XNPV(rate decimal.Decimal, values []decimal.Decimal, dates []time.Time,
    opts ...Options) (decimal.Decimal, error)
```

```text
XNPV = Σ values[i] / (1 + rate)^((dates[i] - dates[0]) / 365)
```

`dates[0]` is the reference (t=0). Lengths of `values` and `dates` must match and there must be at least 2 entries. Excel equivalent: `XNPV(rate, values, dates)`.

### XIRR

```go
func XIRR(values []decimal.Decimal, dates []time.Time, guess decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

The rate that drives `XNPV → 0`. Newton's method, default seed 0.1 (pass `finance.Zero` to get the default). Excel equivalent: `XIRR(values, dates, [guess])`.

---

## Rate Conversion

### EffectiveRate

```go
func EffectiveRate(nominal decimal.Decimal, periodsPerYear int,
    opts ...Options) (decimal.Decimal, error)
```

Converts a nominal annual rate compounded `periodsPerYear` times per year into the effective annual rate:

```
effective = (1 + nominal/m)^m - 1
```

Excel equivalent: `EFFECT(nominal_rate, npery)`.

### NominalRate

```go
func NominalRate(effective decimal.Decimal, periodsPerYear int,
    opts ...Options) (decimal.Decimal, error)
```

Inverse of `EffectiveRate`:

```
nominal = m · ((1 + effective)^(1/m) - 1)
```

Excel equivalent: `NOMINAL(effect_rate, npery)`. Internally uses `decimal.Pow` with a fractional exponent, which routes through `Exp(exp · Log(base))`.

### ContinuousFromAnnual, AnnualFromContinuous

```go
func ContinuousFromAnnual(effective decimal.Decimal, opts ...Options) (decimal.Decimal, error)
func AnnualFromContinuous(continuous decimal.Decimal, opts ...Options) (decimal.Decimal, error)
```

Convert between effective annual rates and continuously-compounded rates:

```
continuous = ln(1 + effective)
effective  = e^continuous - 1
```

---

## Amortization

### AmortizationSchedule

```go
func AmortizationSchedule(rate decimal.Decimal, nper int, pv, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) ([]AmortizationRow, error)
```

Full per-period schedule of a level-payment loan. Returns `nper` rows where each `AmortizationRow.Balance` is the remaining balance after that period's activity.

### ScheduleTable

```go
func ScheduleTable(rate decimal.Decimal, nper int, pv, fv decimal.Decimal,
    timing PaymentTiming, opts ...Options) (insyra.IDataTable, error)
```

Same data as `AmortizationSchedule`, packaged into an `insyra.IDataTable` with five columns: `Period`, `Payment`, `Interest`, `Principal`, `Balance`. Numeric cells are stored as `decimal.Decimal` values, so precision is preserved through the table layer.

```go
table, _ := finance.ScheduleTable(rate, 12, pv, finance.Zero, finance.PaymentEnd)
table.Show() // pretty-print to terminal
```

---

## Depreciation

All depreciation functions accept the asset's `cost` and `salvage` value as `decimal.Decimal` and report depreciation in dollars at the same precision.

### SLN

```go
func SLN(cost, salvage decimal.Decimal, life int, opts ...Options) (decimal.Decimal, error)
```

Straight-line: `(cost - salvage) / life`. Excel equivalent: `SLN(cost, salvage, life)`.

### DDB

```go
func DDB(cost, salvage decimal.Decimal, life, per int, factor decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Declining-balance depreciation in the `per`-th period. `factor=2` is conventional double-declining; pass `finance.MustNew("1.5")` for 150% etc. Each period the asset depreciates by `bookValue · factor/life`, capped so the asset never falls below `salvage`. Excel equivalent: `DDB(cost, salvage, life, period, factor)`.

### SYD

```go
func SYD(cost, salvage decimal.Decimal, life, per int, opts ...Options) (decimal.Decimal, error)
```

Sum-of-years digits in the `per`-th period: `(cost - salvage) · (life - per + 1) / (life·(life+1)/2)`. Excel equivalent: `SYD(cost, salvage, life, per)`.

### VDB

```go
func VDB(cost, salvage decimal.Decimal, life int,
    startPeriod, endPeriod, factor decimal.Decimal, noSwitch bool,
    opts ...Options) (decimal.Decimal, error)
```

Cumulative depreciation between `startPeriod` and `endPeriod` (real numbers in `[0, life]`; partial periods are pro-rated linearly). When `noSwitch=false` (Excel default), the method automatically switches from declining-balance to straight-line as soon as straight-line would yield more — which is what most US tax schedules require. Excel equivalent: `VDB(cost, salvage, life, start_period, end_period, factor, no_switch)`.

---

## Day-Count Basis

Bond and ACCRINT functions take a `DayCountBasis` that mirrors Excel's `basis` parameter:

```go
type DayCountBasis uint8

const (
    Basis30_360US     DayCountBasis = 0 // US (NASD) 30/360
    BasisActualActual DayCountBasis = 1 // ISDA actual/actual
    BasisActual360    DayCountBasis = 2 // actual/360 (money-market)
    BasisActual365    DayCountBasis = 3 // actual/365 (fixed)
    Basis30_360EU     DayCountBasis = 4 // European 30/360
)
```

The `Basis30_360US` rules used by bond pricing apply the four canonical substitutions in order: last-of-Feb→30 (when both endpoints are Feb-end), last-of-Feb→30 for D1 alone, 31→30 for D1, then 31→30 for D2 if D1 already became 30. `BasisActualActual` follows ISDA's variant: each year-segment of the period contributes `daysInSegment / 365_or_366`.

---

## Bonds

All bond pricing functions assume coupon dates land on the same day-of-month as the maturity date, stepping backward by `12/frequency` months. `frequency` must be 1, 2, or 4. Prices are quoted per $100 face value; pass `redemption` separately for non-par redemption values.

### Price

```go
func Price(settlement, maturity time.Time, rate, yld, redemption decimal.Decimal,
    freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error)
```

Clean price per $100 face value. Compound discounting at `yld/freq` per period, with a fractional first-period exponent `DSC/E`. The single-coupon-remaining case automatically falls back to simple-interest discounting (matching Excel). Excel equivalent: `PRICE(settlement, maturity, rate, yld, redemption, frequency, basis)`.

### Yield

```go
func Yield(settlement, maturity time.Time, rate, pr, redemption decimal.Decimal,
    freq int, basis DayCountBasis, guess decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Inverse of `Price`: solve for the yield that reproduces `pr`. Newton's method with default seed 0.05 (pass `finance.Zero` for the default). Returns an error if the iteration fails to converge in 80 steps. Excel equivalent: `YIELD(settlement, maturity, rate, pr, redemption, frequency, basis)`.

### Duration, MDuration

```go
func Duration(settlement, maturity time.Time, coupon, yld decimal.Decimal,
    freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error)

func MDuration(settlement, maturity time.Time, coupon, yld decimal.Decimal,
    freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error)
```

Macaulay duration (years) and modified duration (`= Duration / (1 + yld/freq)`). Excel equivalents: `DURATION`, `MDURATION`.

### AccrInt

```go
func AccrInt(issue, firstInterest, settlement time.Time, rate, par decimal.Decimal,
    freq int, basis DayCountBasis, calcMethod bool,
    opts ...Options) (decimal.Decimal, error)
```

Accrued interest = `par · rate · yearFraction(start, settlement, basis)`, where `start = issue` if `calcMethod=true` (Excel's default — accumulate from issue) or `start = firstInterest` otherwise. Excel equivalent: `ACCRINT(issue, first_interest, settlement, rate, par, frequency, basis, calc_method)`.

---

## Treasury Bills

T-bill functions use ACT/360 conventions and require `0 < (maturity - settlement) ≤ 365` calendar days.

### TBillEq

```go
func TBillEq(settlement, maturity time.Time, discount decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Bond-equivalent yield: `365 · discount / (360 - discount · DSM)`. Excel equivalent: `TBILLEQ(settlement, maturity, discount)`.

### TBillPrice

```go
func TBillPrice(settlement, maturity time.Time, discount decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Price per $100 face value: `100 · (1 - discount · DSM / 360)`. Excel equivalent: `TBILLPRICE(settlement, maturity, discount)`.

### TBillYield

```go
func TBillYield(settlement, maturity time.Time, pr decimal.Decimal,
    opts ...Options) (decimal.Decimal, error)
```

Yield given a price: `((100 - pr) / pr) · (360 / DSM)`. Excel equivalent: `TBILLYIELD(settlement, maturity, pr)`.

---

## Usage Examples

### 30-Year Mortgage at High Precision

```go
package main

import (
    "fmt"

    "github.com/HazelnutParadise/insyra/finance"
)

func main() {
    rate := finance.MustNew("0.005")    // 0.5% per month
    pv   := finance.MustNew("100000")   // $100,000 loan

    // Default: 10 decimal places, HalfUp rounding.
    pmt, err := finance.PMT(rate, 360, pv, finance.Zero, finance.PaymentEnd)
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Println(pmt.String()) // -599.5505251528
}
```

### Custom Precision and Rounding

```go
import "github.com/TimLai666/go-decimal/decimal"

opts := finance.Options{
    Scale: 4,
    Mode:  decimal.RoundingModeHalfUp,
}
pmt, _ := finance.PMT(rate, 360, pv, finance.Zero, finance.PaymentEnd, opts)
fmt.Println(pmt.String()) // -599.5505
```

### Solving for Different TVM Variables

```go
// What rate makes 30 monthly $1,000 payments retire a $25,000 loan?
guess := finance.MustNew("0.01")
r, _  := finance.RATE(30, finance.MustNew("-1000"),
                      finance.MustNew("25000"),
                      finance.Zero,
                      finance.PaymentEnd, guess)

// How long until $200/month grows into $50,000 at 6%/yr (0.5%/month)?
n, _ := finance.NPER(finance.MustNew("0.005"),
                     finance.MustNew("-200"),
                     finance.Zero,
                     finance.MustNew("50000"),
                     finance.PaymentEnd)
```

### Per-Period Interest and Principal

```go
rate := finance.MustNew("0.005")
pv   := finance.MustNew("100000")

ipmt1, _ := finance.IPMT(rate, 1, 360, pv, finance.Zero, finance.PaymentEnd) // -500
ppmt1, _ := finance.PPMT(rate, 1, 360, pv, finance.Zero, finance.PaymentEnd) // -99.55…

// Total interest paid in years 1–5 (months 1–60):
totalI, _ := finance.CumIPMT(rate, 360, pv, 1, 60, finance.PaymentEnd)
```

### NPV / IRR for an Investment Project

```go
cashflows := []decimal.Decimal{
    finance.MustNew("-1000"), // initial outlay
    finance.MustNew("300"),   // year-1 cashflow
    finance.MustNew("400"),   // year-2 cashflow
    finance.MustNew("500"),
    finance.MustNew("600"),
}

discount := finance.MustNew("0.1")
npv, _ := finance.NPV(discount, cashflows)
irr, _ := finance.IRR(cashflows, finance.Zero)

fmt.Println("NPV:", npv.String())
fmt.Println("IRR:", irr.String())
```

### Rate Conversion

```go
nominal := finance.MustNew("0.06")             // 6% nominal
eff,  _ := finance.EffectiveRate(nominal, 12)  // monthly compounding
back, _ := finance.NominalRate(eff, 12)        // round-trips to nominal
fmt.Println(eff.String())  // 0.0616778119
fmt.Println(back.String()) // 0.0600000000
```

### Amortization Schedule into a DataTable

```go
rate := finance.MustNew("0.005")
pv   := finance.MustNew("100000")

table, _ := finance.ScheduleTable(rate, 360, pv, finance.Zero, finance.PaymentEnd,
                                  finance.Options{Scale: 4})
table.Show()
// Period │ Payment   │ Interest  │ Principal │ Balance
// 1      │ -599.5505 │ -500.0000 │  -99.5505 │ 99900.4495
// 2      │ -599.5505 │ -499.5022 │ -100.0483 │ 99800.4012
// …
```

### MIRR with Different Finance / Reinvestment Rates

```go
cf := []decimal.Decimal{
    finance.MustNew("-120000"),
    finance.MustNew("39000"),
    finance.MustNew("30000"),
    finance.MustNew("21000"),
    finance.MustNew("37000"),
    finance.MustNew("46000"),
}
mirr, _ := finance.MIRR(cf, finance.MustNew("0.10"), finance.MustNew("0.12"))
// mirr ≈ 0.12609413
```

### XNPV / XIRR for Irregular-Date Cashflows

```go
import "time"

values := []decimal.Decimal{
    finance.MustNew("-10000"),
    finance.MustNew("2750"),
    finance.MustNew("4250"),
    finance.MustNew("3250"),
    finance.MustNew("2750"),
}
dates := []time.Time{
    time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC),
    time.Date(2008, 3, 1, 0, 0, 0, 0, time.UTC),
    time.Date(2008, 10, 30, 0, 0, 0, 0, time.UTC),
    time.Date(2009, 2, 15, 0, 0, 0, 0, time.UTC),
    time.Date(2009, 4, 1, 0, 0, 0, 0, time.UTC),
}
xnpv, _ := finance.XNPV(finance.MustNew("0.09"), values, dates)
xirr, _ := finance.XIRR(values, dates, finance.Zero)
```

### Depreciation Schedules

```go
cost    := finance.MustNew("30000")
salvage := finance.MustNew("7500")
life    := 10

// Straight-line: same per period.
sln, _ := finance.SLN(cost, salvage, life)

// Sum-of-years digits: front-loaded, sums to (cost - salvage).
for per := 1; per <= life; per++ {
    d, _ := finance.SYD(cost, salvage, life, per)
    fmt.Printf("Year %d: %s\n", per, d.String())
}

// Variable declining-balance for the first half-year.
half, _ := finance.VDB(cost, salvage, life,
    finance.MustNew("0"), finance.MustNew("0.5"),
    finance.MustNew("2"), false)
```

### Bond Pricing and Yield

```go
settlement := time.Date(2008, 2, 15, 0, 0, 0, 0, time.UTC)
maturity   := time.Date(2017, 11, 15, 0, 0, 0, 0, time.UTC)

price, _ := finance.Price(settlement, maturity,
    finance.MustNew("0.0575"), finance.MustNew("0.065"),
    finance.MustNew("100"),    // redemption per 100
    2,                         // semi-annual coupons
    finance.Basis30_360US)
// price ≈ 94.6337

yld, _ := finance.Yield(settlement, maturity,
    finance.MustNew("0.0575"), price,
    finance.MustNew("100"), 2,
    finance.Basis30_360US, finance.Zero)
// yld round-trips back to 0.065
```

### T-Bill Quoting

```go
settlement := time.Date(2008, 3, 31, 0, 0, 0, 0, time.UTC)
maturity   := time.Date(2008, 6, 1, 0, 0, 0, 0, time.UTC)

// Discount-quoted T-bill at 9% — what's the price per 100?
pr,  _ := finance.TBillPrice(settlement, maturity, finance.MustNew("0.09"))
// 98.45

// And the bond-equivalent yield?
beq, _ := finance.TBillEq(settlement, maturity, finance.MustNew("0.09"))
```

---

## Error Handling

All exported functions return `(value, error)` and surface validation problems through the second return value. Common error sources:

- **`PMT/PV/FV/RATE`** — `nper ≤ 0` or invalid `PaymentTiming`
- **`IPMT/PPMT`** — `per` outside `[1, nper]`
- **`CumIPMT/CumPPMT`** — `startPeriod < 1`, `endPeriod > nper`, or `startPeriod > endPeriod`
- **`NPV/NPVExcel`** — empty `cashflows` or `rate == -1`
- **`IRR`** — fewer than two cashflows, derivative vanishes, iterate falls below `-1`, or no convergence in 100 iterations
- **`MIRR`** — fewer than two cashflows, no negatives (no PV) or no positives (no FV)
- **`NPER`** — degenerate `rate = pmt = 0`, or argument to `Log` is non-positive
- **`RATE/IRR/XIRR/YIELD`** — failed convergence in their respective iteration caps
- **`XNPV/XIRR`** — length mismatch between `values` and `dates`, fewer than 2 entries
- **`SLN/SYD/DDB`** — `life ≤ 0`, `per` outside `[1, life]`, or `factor = 0`
- **`VDB`** — `startPeriod / endPeriod` outside `[0, life]`, or `startPeriod > endPeriod`
- **`Price/Yield/Duration/MDuration`** — `maturity ≤ settlement`, frequency not in {1, 2, 4}, or basis not in `[0, 4]`
- **`AccrInt`** — `settlement ≤ issue`, frequency not in {1, 2, 4}, or basis not in `[0, 4]`
- **`TBill*`** — `maturity ≤ settlement` or settlement-to-maturity exceeds 365 days
- **`EffectiveRate/NominalRate`** — `periodsPerYear < 1`
- **`Pow`-based paths** (`NominalRate`, `Effective`, etc.) — propagate `decimal.Pow` errors when the base would be invalid

```go
pmt, err := finance.PMT(rate, 360, pv, finance.Zero, finance.PaymentEnd)
if err != nil {
    return fmt.Errorf("compute PMT: %w", err)
}
```

The package never logs warnings on its own and never panics from valid input — it follows the same error-first contract as `stats` (see [stats.md](./stats.md)).

---

## Best Practices

1. **Build inputs with `MustNew` / `New`**: do not start from `float64` unless you absolutely must — float literals already carry binary error before they reach `Decimal`.
2. **Pick `Scale` to match downstream display**: 2 for currency, 4 for fixed-income yields, 10+ for chained calculations that will themselves feed into more math. The default of 10 is a safe starting point.
3. **Match Excel's sign convention**: pass `pv` positive when the cashflow is incoming and expect the corresponding `pmt`/`fv` to come back negative.
4. **Use `Zero` for absent values**: pass `finance.Zero` rather than building a zero with `MustNew("0")`.
5. **Provide a `guess` to `RATE` / `IRR`** when you have prior knowledge of the answer — Newton converges noticeably faster from a closer seed.
6. **Prefer `ScheduleTable` for output**, `AmortizationSchedule` for further math: the slice keeps every row as `decimal.Decimal`, while the `IDataTable` is what you want for `Show()` or CSV export.
7. **Trust this package over Excel for high-precision work**. Excel's TVM formulas run on `float64` and lose accuracy from roughly the 11th significant digit onward; `finance` carries 28 digits of internal precision before normalizing.

---

## Related Packages

- [`insyra`](../README.md): `DataTable` / `DataList` core types used by `ScheduleTable`
- [`stats`](./stats.md): statistical analysis for downstream cashflow modeling
- [`mkt`](./mkt.md): customer / marketing analytics
- [`isr`](./isr.md): syntax-sugar wrappers (no `finance` shortcut yet — call this package directly)
