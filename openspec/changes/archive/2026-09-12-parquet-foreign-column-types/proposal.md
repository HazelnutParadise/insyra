# Proposal: parquet-foreign-column-types

## Why

[#371](https://github.com/HazelnutParadise/insyra/issues/371). `getVal`'s `default` arm returns `arr.String()`, the string form of the whole array, ignoring the row index it was given. Every row of such a column reads back as the same string, and nothing reports it: `Read` returns a nil error and the table's `Err()` is nil.

`parquet.Write` only ever emits seven Arrow types, so this is invisible on files this library wrote. It hits files written by anything else, which is what a Parquet reader is for. Measured on 2026-09-12 by writing a file with nine foreign column types and reading it back through `Read`:

| Column type | Every row read back as |
| --- | --- |
| Date32 | `[19000 19001 19002]` |
| Date64 | `[18518 18519 18520]` |
| Decimal128 | `[{12345 0} {67890 0} {11111 0}]` |
| uint64, Int16, Int8 | `[1 2 3]` |
| Binary | `["aa" "bb" "cc"]` |
| List | `[[0 1] [1 2] [2 3]]` |

**Dictionary is not affected**, contrary to the issue: pqarrow materialises it as a plain String array on read, and its three rows came back correctly. That matters, because a pandas `category` column was the most common case on the issue's list.

## What Changes

Every Arrow type with a faithful Go representation gets one, and a type without one reads as `nil` cells with the reason recorded on the table rather than as plausible-looking text.

- **Dates**: `Date32` and `Date64` read as `time.Time`, like the `Timestamp` arm already does.
- **Integers**: `Int8`, `Int16`, `Uint8`, `Uint16`, `Uint32` and `Uint64` read as their own Go types, like the `Int64` and `Int32` arms already do. `Uint64` stays `uint64` because a value above 2^63 does not fit an `int64`, and since `match-integers-by-value` an integer is matched by value whatever its Go type.
- **Binary**: `Binary`, `LargeBinary` and `FixedSizeBinary` read as `[]byte`; `LargeString` as `string`.
- **Decimals**: `Decimal128` and `Decimal256` read as a `github.com/TimLai666/go-decimal` `decimal.Decimal`, built with `NewFromScaledInt(num.BigInt(), scale)`, which rounds and normalises nothing. Verified exact over the full 38-digit Decimal128 range, including negatives.
- **Everything else** — `List`, `Struct`, `Map`, `Time32`, `Time64`, `Duration`, `Interval` and anything Arrow adds later — reads as `nil` cells, and the table carries `column "tags": unsupported Arrow column type list<item: int64>` through the sticky `Err()`. `getVal`'s `default` arm returns `nil`, so the per-cell paths behind `FilterWithCCL` do the same thing without threading an error through every accessor.

`decimal.Decimal` is treated the way `time.Time` already is: its own sorting rank and its own `CompareAny` branch, so a decimal column sorts by value rather than by the lexicographic order of its text, and it displays through its `String()`. Like `time.Time`, it is **not** numeric to `IsNumeric` or `ToFloat64Safe`.

## Capabilities

### New Capabilities

- `parquet-column-types`: which Arrow column types can be read, what each becomes in Go, and what happens to one that cannot.

### Modified Capabilities

(none)

## Impact

- New dependency `github.com/TimLai666/go-decimal` v0.1.3: MIT, no dependencies of its own, `go 1.22` directive, so it puts no pressure on the module's `go 1.25.12`. The representation is a `big.Int` coefficient with an `int32` scale, which is what Decimal128 needs; an `int64` coefficient could not hold it.
- `Binary`, `LargeBinary` and `FixedSizeBinary` read as a `string` holding the raw bytes rather than a `[]byte`, because `NewDataList` flattens every `reflect.Slice` and so a slice cannot be a cell at all. A Go string holds arbitrary bytes, `0x00` and invalid UTF-8 included, so `[]byte(cell)` recovers them exactly.
- `internal/algorithms` imports it, because ordering a decimal correctly needs `decimal.Cmp` and there is no interface to dispatch on. It is therefore a core dependency, not a `parquet`-only one.
- A column of one of the newly supported types changes from one repeated string to real values. Nothing that was right changes.
- `Mean`, `Sum` and the rest of the numeric path refuse a decimal column, naming the row, exactly as they refuse a `time.Time` column today. Making decimals numeric library-wide is a separate decision and is recorded as a follow-up rather than taken here.
- A read-then-write round trip is still lossy for the new types, because `inferArrowType` writes only its original seven: a `[]byte` column is written back as a string, a decimal as its text. That is out of scope here and recorded as a follow-up.

## Backport to dev (0.3.x)

Dev received: `Date32`/`Date64` as `time.Time` at UTC, `Int8`/`Int16` and `Uint8`..`Uint64` as their Go types, `LargeString` as `string`, `Decimal128`/`Decimal256` as `decimal.Decimal`; every other type as `nil` cells with the reason recorded through `SetErr` in `Read`, `Stream` and `ReadColumn`, while `FilterWithCCL` sees `nil` and records nothing; `decimal.Decimal` sorting by value; the docs table, skill note, changelog entries and the two AGENTS.md follow-ups, reworded for this line. go-decimal was already a direct dependency (`finance`), so `go.mod` is unchanged.

Adapted:
- `Err()` on this line is not sticky: each unsupported column logs a warning and replaces `Err()`, so with several of them `Err()` names the last. A test pins it.
- Binary columns (`Binary`, `LargeBinary`, `FixedSizeBinary`) keep their old reading here, the text of the whole array, and count as supported so no reason is recorded. The 0.4 intermediate of a `string` holding the raw bytes was not taken; `parquet-binary-is-bytes` brings `[]byte` directly.

Left on 0.4:
- `Binary` as a `string` of raw bytes, its test assertion and its requirement text: an intermediate step, replaced on this line by `parquet-binary-is-bytes`.
- `api-review.md` and `delivery-status.md` edits.
