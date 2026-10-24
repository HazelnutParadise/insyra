# Exact Decimals

When a number has to stay exact — money, interest rates, anything where `0.1 + 0.2` must be `0.3` — use [`github.com/TimLai666/go-decimal`](https://github.com/TimLai666/go-decimal). It is the one decimal type used across Insyra: every [`finance`](finance.md) function takes and returns it, and a [Parquet](parquet.md) `Decimal128` or `Decimal256` column reads as it.

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

- **One decimal is one cell.** A `[]decimal.Decimal` passed to `NewDataList` gives one cell per value, like any slice, and a single decimal needs no wrapping. Wrap the slice in `insyra.Cell(v)` only when the whole slice should sit in one cell.
- **It is not a number to the numeric methods.** Like a `time.Time`, a decimal cell is skipped by `Mean` and `Sum` with a warning on `Err()`, so a column holding only decimals gives `NaN`. `Describe` leaves its mean, spread and quantiles empty, and `IsNumeric` returns `false`. `stats` does not read it either: depending on the function you get an error or a `NaN` result. Convert the column first, as shown below.
- **Two decimals sort by value.** `Sort` and `SortBy` compare two decimals exactly, so `9.50` sorts before `10.00` and a difference beyond what a `float64` can hold is still seen. In a column that mixes decimals with other types, the decimals sort as a group after the numbers, strings and times.
- **It displays as its own text, but JSON loses it.** `Show` prints `19.99` and `ToCSV` writes `19.99`. `ToJSON` writes a decimal cell as `{}`, because the type has no JSON form of its own, so convert the column to strings before exporting JSON.
- **Searching matches the digits and the scale.** `Count(v)` and `FindAll(v)` find a decimal holding the same digits at the same scale, so `19.99` does not match `19.990`. A decimal cannot be a Go map key by itself, so `Counter` keys it by an `insyra.UncomparableKey`; build the key with `insyra.ToMapKey(v)` to look one up:

```go
counts := dl.Counter()
counts[insyra.ToMapKey(decimal.MustParse(ctx, "19.99"))] // 2 for a column holding 19.99, 5.01, 19.99
```

---

## Converting a Column for Analysis

go-decimal has no `Float64` method, so read each cell through its own text:

```go
floats := insyra.NewDataList()
for i := 0; i < dl.Len(); i++ {
    f, err := strconv.ParseFloat(dl.Get(i).(decimal.Decimal).String(), 64)
    if err != nil {
        log.Fatal(err)
    }
    floats.Append(f)
}
floats.Mean() // 14.996666666666664 for 19.99, 5.01, 19.99
```

A `float64` holds about 16 significant digits and cannot represent most decimal fractions exactly. That is fine for averages, spread and tests, and wrong for an amount that has to be exact: `floats.Sum()` gives `44.989999999999995` for the same three prices.

---

## Totals Must Stay in Decimals

For an amount that has to be exact, add the cells as decimals:

```go
total := decimal.NewFromInt64(ctx, 0)
for i := 0; i < dl.Len(); i++ {
    total = decimal.Add(ctx, total, dl.Get(i).(decimal.Decimal))
}
// total is 44.99
```

The cells themselves always keep their exact value; rounding happens only in a `float64` you convert to.

---

## Getting Decimals Into a Table

- **From `finance`:** its functions return `decimal.Decimal`, and `finance.ScheduleTable` builds a table whose `Payment`, `Interest`, `Principal` and `Balance` columns hold them.
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
- **One type across the library.** The same `decimal.Decimal` comes out of `finance` and out of a Parquet column, so values from the two can be compared and added without a conversion between them.

Insyra pins go-decimal at v0.1.3 in its `go.mod`.
