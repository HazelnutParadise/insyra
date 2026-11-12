# 0021 — One exact decimal type across the module

**Status:** accepted
**Date:** 2026-09-13

## Context
`finance` and `parquet` both need exact decimals, and a caller moving values between two decimal types pays a conversion and a rounding decision at every boundary.

## Decision
An exact decimal anywhere in Insyra is a `github.com/TimLai666/go-decimal` `decimal.Decimal`, pinned in `go.mod` at v0.1.3. `finance` takes and returns it, `parquet` reads `Decimal128` and `Decimal256` as it, and the documentation recommends it to users for any value that has to stay exact. A new package does not introduce a second decimal type.

## Options rejected
The alternatives were compared at their current versions on 2026-09-13; the full comparison is in [Docs/Decimal.md](/Docs/Decimal.md).
- `shopspring/decimal`: `Div` rounds to a package variable any code in the program can change, and it has no `Sqrt`, which `finance` needs.
- `cockroachdb/apd`: it counts digits in total, so how many land after the point depends on how large the number is.
- `govalues/decimal`: it holds 19 digits in total and drops places to make a value fit, without an error.

## Consequences
Re-check that comparison before changing the dependency. How core treats a decimal cell is described in `/engineering/architecture.md` under "Exact decimals".
