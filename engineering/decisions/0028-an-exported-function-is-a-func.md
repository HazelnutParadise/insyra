# 0028 — An exported function is declared with `func` and has one name

**Status:** accepted
**Date:** 2026-09-25

## Context
`ToFloat64`, `ToFloat64Safe` and `ReadSlice2D` in the root package and `CAI` in `mkt` were variables holding a function, so any caller could reassign one for every other caller. Some had a second name, such as `Slice2DToDataTable` beside `ReadSlice2D`.

## Decision
An exported function in Insyra is declared with `func`, never held in a variable, and has one name. Where two names existed, the one in the `Read*` family stays: the owner chose `ReadSlice2D` over `Slice2DToDataTable` on 2026-09-25, and the other name stays one release as Deprecated (`exported-functions-are-functions`).

## Options rejected
- Keep both names. The one-name rule of #211 allows only the exceptions listed in `AGENTS.md`.

## Consequences
`finance.Zero` is a variable, not a function, and stays one: the owner ruled on 2026-09-26 (#248) that nobody reassigns it, and making it a function would rewrite every `finance.Zero` a caller has written.
