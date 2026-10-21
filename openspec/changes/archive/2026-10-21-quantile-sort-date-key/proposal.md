## Why

`quant.QuantileSort` keys dates with its own rule: string dates are trimmed, so "2026-01-02" and " 2026-01-02 " become one cross-section, and `time.Time` is keyed by `UnixNano`, which is undefined outside 1678-2262 so distinct instants collide. The documented rule is that dates are grouped by value. `quant.FamaMacBeth` already has the exact rule.

## What Changes

- QuantileSort and FamaMacBeth share one date-key helper with the exact rule: strings compared exactly, `time.Time` by instant, and only string, `time.Time`, bool, integer and float dates accepted.
- QuantileSort returns an error for any other date type or a NaN date. A missing (`nil`) date is still an error with its existing wording.
- QuantileSort has not been released, so nothing downstream changes. FamaMacBeth behaves as before.

## Impact

`quant/date_key.go`, `quant/quantile_sort.go`, `quant/fama_macbeth.go`, `Docs/quant.md`, both changelogs.
