# Exact Decimals

When a number has to stay exact — money, interest rates, anything where `0.1 + 0.2` must be `0.3` — use [`github.com/TimLai666/go-decimal`](https://github.com/TimLai666/go-decimal). It is the one decimal type used across Insyra: every [`finance`](finance.md) function takes and returns it, a [Parquet](parquet.md) `Decimal128` or `Decimal256` column reads as it, and a column of them works with the numeric methods like any other number column.

```bash
go get github.com/TimLai666/go-decimal
```

```go
import "github.com/TimLai666/go-decimal/decimal"

ctx := decimal.Context{Scale: 2, Mode: decimal.RoundingModeHalfEven}
prices := []decimal.Decimal{
    decimal.MustParse(ctx, "19.99"),
    decimal.MustParse(ctx, "5.01"),
}
dl := insyra.NewDataList(prices) // one cell per price
```

`Context.Scale` is the number of places after the point a result keeps, and `Context.Mode` is how it rounds.

---

## How a Cell Treats a Decimal

- **One decimal is one cell.** A `[]decimal.Decimal` passed to `NewDataList` gives one cell per value, like any slice; no wrapping is needed.
- **It is a number.** `Mean`, `Sum`, `Describe` and the `stats` package read it, and `IsNumeric` returns `true`.
- **Sorting puts it among the numbers.** Two decimals compare exactly, so a difference beyond what a `float64` can hold is still seen; a decimal against a `float64` compares as floats.
- **It displays and exports as its own text.** `Show`, `ToCSV` and `ToJSON` write `19.99`, not the value's internals.
- **Searching finds it.** `Count(v)` and `FindAll(v)` match a decimal by value. A decimal cannot be a Go map key by itself, so build the key with `insyra.ToMapKey(v)` when you index a map of cell values, including the result of `Counter`.

---

## Totals Must Stay in Decimals

`Mean`, `Sum`, `Describe` and `stats` return `float64`. A `float64` holds about 16 significant digits and cannot represent most decimal fractions exactly, so a total computed that way is close, not exact:

```go
dl.Sum() // 44.989999999999995 for 19.99 + 5.01 + 19.99
```

Use the float methods for analysis — averages, spread, tests. For an amount that has to be exact, add the cells as decimals:

```go
total := decimal.NewFromInt64(ctx, 0)
for i := 0; i < dl.Len(); i++ {
    total = decimal.Add(ctx, total, dl.Get(i).(decimal.Decimal))
}
// total is 44.99
```

The cells themselves always keep their exact value; the rounding happens only in the `float64` a numeric method returns.

---

## Getting Decimals Into a Table

- **From `finance`:** its functions return `decimal.Decimal`, and `finance.ScheduleTable` builds a table whose amount columns hold them.
- **From Parquet:** a `Decimal128` or `Decimal256` column reads as `decimal.Decimal`, keeping the file's own digits and scale.
- **From CSV:** column type inference loads a number with a fractional part as `float64`, which already loses exactness. Load the file with `CSVReadOptions{RawStrings: true}` so every cell stays the text it was, then parse the amount column with `decimal.Parse(ctx, s)`.

---

## Choosing a Decimal Package

Insyra uses go-decimal for every exact decimal and recommends it for yours. The Go alternatives were compared at their current versions on 2026-09-13, by reading their source and running them.

| | go-decimal v0.1.3 | shopspring/decimal v1.4.0 | cockroachdb/apd v3.2.3 | govalues/decimal v0.1.36 |
| --- | --- | --- | --- | --- |
| Largest value | unbounded | unbounded | unbounded | 19 digits in total |
| Places a result keeps | set per call by `Context.Scale`, digits after the point | division: the package variable `DivisionPrecision`; otherwise per call | `Context.Precision`, digits in total | per call |
| Rounding modes | 9, chosen in the `Context` | 7, one method each | chosen in the `Context` | `Round`, `Trunc`, `Ceil`, `Floor` |
| Settings shared by the whole program | none | 4 package variables | none | none |
| `Sqrt` / `Exp` / `Log` / `Pow` | all four | no `Sqrt` | all four | all four |
| A call | `decimal.Add(ctx, a, b)` | `a.Add(b)` | `ctx.Add(dst, a, b)` returning `(Condition, error)` | `a.Add(b)` returning `(Decimal, error)` |

What decided it:

- **A result keeps the places you asked for.** An amount of money has a fixed number of places, and `Context.Scale` says exactly that. `apd` counts digits in total, so how many land after the point depends on how large the number is. `govalues` holds 19 digits in total and drops places to make a value fit, without an error: at `finance`'s default of 10 places, `999999999.9999999999 + 0.0000000001` came back as `1000000000.000000000`, with nine.
- **Nothing the rest of the program does changes a result.** In `shopspring/decimal`, `Div` rounds to the package variable `DivisionPrecision`, which any code in the same program can set. `1/3` gave `0.3333333333333333`; after another part of the program set it to 4, the same division gave `0.3333`.
- **The math is all there.** `finance` calls `Sqrt`, `Exp`, `Log` and `Pow`, and `shopspring/decimal` has no `Sqrt`.
- **One type across the library.** The same `decimal.Decimal` comes out of `finance` and out of a Parquet column, and goes into the numeric methods and sorting, so nothing is converted on the way.

Insyra pins go-decimal at v0.1.3 in its `go.mod`.
