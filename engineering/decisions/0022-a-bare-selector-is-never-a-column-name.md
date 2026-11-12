# 0022 — A bare column reference is an index, never a column name

**Status:** accepted
**Date:** 2026-09-22

## Context
Review finding T-11 reported `GetCol("price")` returning nothing on a table with a column called `price`, because a bare string was read as an Excel-style index. Some entry points fell back to an uppercased column name, so the same call meant different things on different tables. CCL had the same question for a bare word in an expression.

## Decision
One column selector for the whole library: a `string` is an Excel-style index, `Name("price")` is a column name, an `int` is a 0-based position. A bare word in CCL is a column index too; the owner confirmed on 2026-09-21 that this was always the design. In both, the column names shape only the error message, which offers the `Name(...)` form when the table has that name. The owner asked on 2026-09-22 for full unification rather than the minimal fix. The rule as callers use it is in `AGENTS.md` under Key Conventions.

## Options rejected
- Resolve a bare name first as a column name. A call's meaning would depend on the data: renaming a column could silently change a result.
- Typed `ColIndex` and `ColName` string types. Go converts an untyped literal implicitly, so `GetCol("price")` would still have compiled.

## Consequences
The generic `any` parameter keeps every existing `"A"` literal compiling. The explicit `ByIndex`, `ByName` and `ByNumber` spellings stay on the core accessors. `ColumnIndex`'s uppercased-name fallback, shared by `GetCol`, `SetColToRowNames` and `SortBy`, was removed (`one-column-selector`).
