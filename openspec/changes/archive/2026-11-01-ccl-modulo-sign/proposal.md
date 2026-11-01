## Why

`MOD` and `%` give the remainder the sign of the first number, so `MOD(-7, 3)` is `-1`. In Excel, Python and R it is `2`, which also keeps a cycle such as a weekday unbroken across zero. The owner ruled on 2026-11-01 (#414) that CCL follows Excel.

## What Changes

- **BREAKING** `MOD` and `%` give the remainder the sign of the second number. Results with only positive numbers do not change.

## Impact

- `internal/ccl/`, `Docs/CCL.md`, both changelogs, `ccl-type-semantics` spec.
