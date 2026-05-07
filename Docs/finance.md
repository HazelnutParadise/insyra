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
- **Discounted Cashflows**: `NPV`, `NPVExcel`, `IRR` — net present value and internal rate of return
- **Rate Conversion**: `EffectiveRate`, `NominalRate`, `ContinuousFromAnnual`, `AnnualFromContinuous`
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

---

## Error Handling

All exported functions return `(value, error)` and surface validation problems through the second return value. Common error sources:

- **`PMT/PV/FV/RATE`** — `nper ≤ 0` or invalid `PaymentTiming`
- **`IPMT/PPMT`** — `per` outside `[1, nper]`
- **`CumIPMT/CumPPMT`** — `startPeriod < 1`, `endPeriod > nper`, or `startPeriod > endPeriod`
- **`NPV/NPVExcel`** — empty `cashflows` or `rate == -1`
- **`IRR`** — fewer than two cashflows, derivative vanishes, iterate falls below `-1`, or no convergence in 100 iterations
- **`NPER`** — degenerate `rate = pmt = 0`, or argument to `Log` is non-positive
- **`RATE/IRR`** — failed convergence (in 60 / 100 iterations respectively)
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
