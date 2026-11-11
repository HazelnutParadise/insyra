# 0021 — One exact decimal type across the module

**Status:** accepted
**Date:** 2026-09-13

## Context
`finance` and `parquet` both need exact decimals, and a caller moving values between two decimal types pays a conversion and a rounding decision at every boundary.

## Decision
An exact decimal anywhere in Insyra is a `github.com/TimLai666/go-decimal` `decimal.Decimal`, pinned in `go.mod` at v0.1.3. `finance` takes and returns it, `parquet` reads `Decimal128` and `Decimal256` as it, and the documentation recommends it to users for any value that has to stay exact. A new package does not introduce a second decimal type.

## Options rejected
The alternatives were compared at their current versions on 2026-09-13; the comparison and the reasons are in [Docs/Decimal.md](/Docs/Decimal.md).

## Consequences
Re-check that comparison before changing the dependency. How core treats a decimal cell is described in `/engineering/architecture.md` under "Exact decimals".
